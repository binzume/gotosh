# Go to sh

PoC for generating shell script from subset of Go.

Go(のサブセット)で書かれたプログラムをシェルスクリプトに変換するプログラムです。
Goで書かれています。

シェルスクリプトを型付きの言語で書くために実装しました。
Goでコンパイルしたバイナリを実行するのが困難な環境のために作ったので、可能な限りBusyBoxのash等でも動作するようにしています。

Supported:

- Types: `int`, `string`, `[]int`, `[]string`, `struct`
- Go keywords: func, if, else, for, break, continue, const, var, struct, append, len, go

TODO:

- jq, curl support
- Convert compiler/compiler.go to compiler.sh

# Usage

See [examples](examples) folder

```bash
go run . examples/fizz_buzz.go > fizz_buzz.sh
chmod a+x fizz_buzz.sh
./fizz_buzz.sh
```

### Input(fizz_buzz.go)

```go
package main

import "fmt"

const fizz = "Fizz"
const buzz = "Buzz"

func FizzBuzz(n int) {
	for i := 1; i <= n; i++ {
		if i%15 == 0 {
			fmt.Println(fizz + buzz)
		} else if i%3 == 0 {
			fmt.Println(fizz)
		} else if i%5 == 0 {
			fmt.Println(buzz)
		} else {
			fmt.Println(i)
		}
	}
}

func main() {
	FizzBuzz(50)
}
```

### Output(fizz_buzz.sh)

```bash
#!/bin/bash

fizz="Fizz"
buzz="Buzz"
function FizzBuzz() {
  local n="$1"; shift
  i=1
  while [ $(( i<=n )) -ne 0 ]; do :
    if [ $(( i%15 == 0 )) -ne 0 ]; then :
      echo "$fizz""$buzz"
    elif [ $(( i%3 == 0 )) -ne 0 ]; then :
      echo "$fizz"
    elif [ $(( i%5 == 0 )) -ne 0 ]; then :
      echo "$buzz"
    else
      echo $i
    fi
  let "i++"; done
}

function main() {
  FizzBuzz 50
}

main "${@}"
```

## Supported functions

- [shell.*](shell/builtin.go)
- [shell.NArg](shell/builtin.go)
- [shell.Arg](shell/builtin.go)
- [shell.Exec](shell/builtin.go)
- [shell.Do](shell/builtin.go)
- [shell.ReadLine](shell/builtin.go)
- [shell.Sleep](shell/builtin.go)
- [shell.UnixTimeMs](shell/builtin.go)
- [fmt.Print](https://pkg.go.dev/fmt#Print)
- [fmt.Println](https://pkg.go.dev/fmt#Println)
- [fmt.Printf](https://pkg.go.dev/fmt#Printf)
- [fmt.Sprint](https://pkg.go.dev/fmt#Sprint)
- [fmt.Sprintln](https://pkg.go.dev/fmt#Sprintln)
- [fmt.Sprintf](https://pkg.go.dev/fmt#Sprintf)
- [fmt.Fprint](https://pkg.go.dev/fmt#Fprint)
- [fmt.Fprintln](https://pkg.go.dev/fmt#Fprintln)
- [fmt.Fprintf](https://pkg.go.dev/fmt#Fprintf)
- [strings.ReplaceAll](https://pkg.go.dev/strings#ReplaceAll)
- [strings.ToUpper](https://pkg.go.dev/strings#ToUpper)
- [strings.ToLower](https://pkg.go.dev/strings#ToLower)
- [strings.TrimSpace](https://pkg.go.dev/strings#TrimSpace)
- [strings.TrimPrefix](https://pkg.go.dev/strings#TrimPrefix)
- [strings.TrimSuffix](https://pkg.go.dev/strings#TrimSuffix)
- [strings.Split](https://pkg.go.dev/strings#Split)
- [strings.Join](https://pkg.go.dev/strings#Join)
- [strings.Contains](https://pkg.go.dev/strings#Contains)
- [strings.IndexAny](https://pkg.go.dev/strings#IndexAny)
- [strconv.Atoi](https://pkg.go.dev/strconv#Atoi)
- [strconv.Itoa](https://pkg.go.dev/strconv#Itoa)
- [os.Exit](https://pkg.go.dev/os#Exit)
- [os.Chdir](https://pkg.go.dev/os#Chdir)
- [os.Getwd](https://pkg.go.dev/os#Getwd)
- [os.Getpid](https://pkg.go.dev/os#Getpid)
- [os.Getppid](https://pkg.go.dev/os#Getppid)
- [os.Getuid](https://pkg.go.dev/os#Getuid)
- [os.Geteuid](https://pkg.go.dev/os#Geteuid)
- [os.Getgid](https://pkg.go.dev/os#Getgid)
- [os.Getegid](https://pkg.go.dev/os#Getegid)
- [os.Hostname](https://pkg.go.dev/os#Hostname)
- [os.Getenv](https://pkg.go.dev/os#Getenv)
- [os.Setenv](https://pkg.go.dev/os#Setenv)
- [os.Pipe](https://pkg.go.dev/os#Pipe)
- [os.Open](https://pkg.go.dev/os#Open)
- [os.Create](https://pkg.go.dev/os#Create)
- [os.Mkdir](https://pkg.go.dev/os#Mkdir)
- [os.MkdirAll](https://pkg.go.dev/os#MkdirAll)
- [os.Remove](https://pkg.go.dev/os#Remove)
- [os.RemoveAll](https://pkg.go.dev/os#RemoveAll)
- [os.Rename](https://pkg.go.dev/os#Rename)

Constatns:

- os.Stdin
- os.Stdout
- os.Stderr
- runtime.Compiler // "gotosh" になっています
- runtime.GOARCH
- runtime.GOOS
- shell.IsShellScript // トランスパイル後はtrueになるので、シェルスクリプト専用の処理への切り替えに使えます

`GOTOSH_FUNC_` プレフィックスが付いた関数を定義することで、任意のパッケージの関数を実装することができます。 (以下は `strings.Index()` を実装する例。内部の実装用なので後で変わる可能性が高いです)

```go
// Implements strings.Index()
func GOTOSH_FUNC_strings_Index(s, f string) int {
	fl := len(f)
	end := len(s) - fl + 1
	for i := 0; i < end; i++ {
		if s[i:i+fl] == f {
			return i
		}
	}
	return -1
}

func main() {
	fmt.Println(strings.Index("hello, world", "ld")) // GOTOSH_strings_Index() will be invoked
}
```


# 制限

Goの文法をすべてサポートすることは目指していないので、

- defer, range, make, new, chan, switch, select, map...

他にもサポートしていない文法が多いです。

## 型

- 利用可能な型は、`int`, `string`, `[]int`, `[]string` のみです
- 定数の場合のみ`float` 等を扱えます(例： `shell.Sleep(0.1)` は有効)
- ポインタは無いのですべての値渡しです

### struct

structのサポートはまだ途中です

- フィールド名と型のペアが並んだ単純なstructのみサポートしています
- sliceも入れることはできません
- struct中にstructを直接定義できません
- 初期化時にフィールド名を指定できません

```go
// OK
type A struct {
	a int
}
// OK
type B struct {
	a A
	b string
}

// Not supported
type C struct {
	c1, c2 string
	c3 struct {
		d int
	}
}

var b1 = B{A{1}, "abc"} // OK
var b2 = B{b: "abc"} // Not supproted
```

### slice

sliceの実装はbash専用です。zshの場合は `setopt KSH_ARRAYS` を追加する必要があると思います。
また、関数の最後の引数以外ではスライスを渡すことはできません。

## 関数

### 引数

sliceなども含めて全ての値は値渡しです。

### 戻り値

関数の結果は標準出力として返します。なので基本的に値を返す関数の内部で標準出力に何かを出力することはできません。
標準出力以外で値を返すことを強制したい場合は以下の型(type alias)が使えます。(名前しか見てないので同名のtypeを定義しても動作します)

- `shell.TempVarString` (= string) は _tmpN 変数を使って値を返します。複数の値を返す必要がある場合に使います
- `shell.StatusCode` (= byte) は関数の終了コードとして返します

多値の戻り値は以下の組み合わせに対応しています。

- (*, StatusCode)
- (TempVarString, TempVarString, ..., StatusCode)

### レシーバ

レシーバのある関数(メソッド)も使えますが、ポインタが無いのでメソッド内で自身の値を変更することはできません。

## goroutine

サブプロセスとして実行されます。無名関数はまだサポートされていないので通常の名前付きの関数を呼び出してください。

また、チャネルも使えないので、`os.Pipe()` (fifoが作られます)で作ったreader/writer等で通信してください。

# Security

なるべく気を使っていますが、ash(またはPOSIXシェル)で動作させるために `eval` している箇所があります(引数はint型なので大丈夫なはずですが)。
また、あまりテストされていないのでスクリプト生成時のエスケープ漏れなどもあるかもしれません。
信頼されない入力値を受け取るプログラムには使わないほうが無難です。

# License

MIT License
