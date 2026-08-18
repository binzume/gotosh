package compiler

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed runtime/*.json runtime/*.sh
var embeddedRuntime embed.FS

type runtimeDefinition struct {
	ArgTypes []Type   `json:"arg_types"`
	RetTypes []Type   `json:"ret_types"`
	Requires []string `json:"requires,omitempty"`
	Body     string   `json:"-"`
}

func loadRuntimeFS(runtimeFS fs.FS, source string, defs map[string]runtimeDefinition) error {
	entries, err := fs.ReadDir(runtimeFS, ".")
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		jsonPath := entry.Name()
		name := strings.TrimSuffix(jsonPath, ".json")
		shPath := name + ".sh"
		data, err := fs.ReadFile(runtimeFS, jsonPath)
		if err != nil {
			return err
		}
		body, err := fs.ReadFile(runtimeFS, shPath)
		if err != nil {
			return fmt.Errorf("runtime %s: matching %s: %w", source, shPath, err)
		}
		var def runtimeDefinition
		if err := json.Unmarshal(data, &def); err != nil {
			return fmt.Errorf("runtime %s: %w", filepath.Join(source, jsonPath), err)
		}
		def.Body = string(body)
		if strings.TrimSpace(def.Body) == "" {
			return fmt.Errorf("runtime %s: empty shell body", filepath.Join(source, shPath))
		}
		defs[name] = def
	}
	return nil
}

func (s *state) loadRuntimeDefinitions() error {
	if s.runtimeDefs != nil {
		return nil
	}
	runtimeFS, err := fs.Sub(embeddedRuntime, "runtime")
	if err != nil {
		return err
	}
	defs := map[string]runtimeDefinition{}
	if err := loadRuntimeFS(runtimeFS, "embedded runtime", defs); err != nil {
		return err
	}
	if dirs := os.Getenv("GOTOSH_RUNTIME_DIR"); dirs != "" {
		for _, dir := range filepath.SplitList(dirs) {
			if err := loadRuntimeFS(os.DirFS(dir), dir, defs); err != nil {
				return err
			}
		}
	}
	s.runtimeDefs = defs
	for name, def := range defs {
		s.funcs[name] = shExpression{
			expr:       "GOTOSH_RT_" + strings.ReplaceAll(name, ".", "__"),
			argTypes:   def.ArgTypes,
			retTypes:   def.RetTypes,
			primaryIdx: 0,
			stdout:     true,
		}
	}
	return nil
}

func (s *state) emitUsedRuntime() {
	var names []string
	for name, def := range s.funcs {
		if def.funcUsed && s.runtimeDefs[name].Body != "" {
			names = append(names, name)
		}
	}
	sort.Strings(names)

	visiting := map[string]bool{}
	emitted := map[string]bool{}
	var emit func(string)
	emit = func(name string) {
		if emitted[name] {
			return
		}
		if visiting[name] {
			return
		}
		def, ok := s.runtimeDefs[name]
		fn, exists := s.funcs[name]
		if !ok || !exists {
			return
		}
		visiting[name] = true
		for _, required := range def.Requires {
			emit(required)
		}
		delete(visiting, name)
		emitted[name] = true

		s.Writeln("")
		s.Writeln(fn.expr + "() {")
		for _, line := range strings.Split(strings.TrimSuffix(def.Body, "\n"), "\n") {
			s.Writeln("  " + line)
		}
		s.Writeln("}")
	}
	for _, name := range names {
		emit(name)
	}
}
