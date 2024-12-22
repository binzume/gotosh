# Go to sh

Go(のサブセット)で書かれたプログラムをシェルスクリプトに変換(トランスパイル)するプログラムです。
Goで実装されています。

シェルスクリプトを型付きの言語で書くために実装しました。
Goでコンパイルしたバイナリを実行するのが困難な環境のために作ったので、可能な限りBusyBoxのash等でも動作するようにしています。

Supported:

- Types: `int`, `string`, `float64`, `[]int`, `[]string`, `struct`
- Go keywords: func, if, else, for, break, continue, const, var, struct, append, len, go

TODO:

- jq, curl support
- Convert compiler/compiler.go to goto.sh

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
  local i=1
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
  : $(( i++ )); done
}

function main() {
  FizzBuzz 50
}

main "${@}"
```

## Supported functions

- [shell.NArg](shell/builtin.go)
- [shell.Arg](shell/builtin.go)
- [shell.Exec](shell/builtin.go)
- [shell.Do](shell/builtin.go)
- [shell.SetFloatPrecision](shell/builtin.go)
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
- [os.Open](https://pkg.go.dev/os#Open)
- [os.Create](https://pkg.go.dev/os#Create)
- [os.Mkdir](https://pkg.go.dev/os#Mkdir)
- [os.MkdirAll](https://pkg.go.dev/os#MkdirAll)
- [os.Remove](https://pkg.go.dev/os#Remove)
- [os.RemoveAll](https://pkg.go.dev/os#RemoveAll)
- [os.Rename](https://pkg.go.dev/os#Rename)
- [os.Pipe](https://pkg.go.dev/os#Pipe)
- [math.Sqrt](https://pkg.go.dev/math#Sqrt)
- [math.Pow](https://pkg.go.dev/math#Pow)
- [math.Exp](https://pkg.go.dev/math#Exp)
- [math.Log](https://pkg.go.dev/math#Log)
- [math.Sin](https://pkg.go.dev/math#Sin)
- [math.Cos](https://pkg.go.dev/math#Cos)
- [math.Atan](https://pkg.go.dev/math#Atan)

Constatns:

- os.Stdin
- os.Stdout
- os.Stderr
- runtime.Compiler // "gotosh" になっています
- runtime.GOARCH
- runtime.GOOS
- shell.IsShellScript // トランスパイル後はtrueになるので、シェルスクリプト専用の処理への切り替えに使えます

`GOTOSH_FUNC_` プレフィックスが付いた関数を定義することで、他のパッケージの関数を実装することができます。 (以下は `strings.Index()` を実装する例。暫定的な処置なので将来変わるかもしれません)

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

Goの文法をすべてサポートしているわけではありません。以下のキーワードは未サポートです。

- map, interface, chan, make, new, defer, switch, case, select...

また、サポートされていても制限がある場合や挙動が異なる場合があります。

## 型

- 利用可能な型は、`int`, `string`, `float32/64`, `[]int`, `[]string` のみです
- floatの計算のためには `bc` コマンドが必要です
- ポインタは無いのですべての値渡しです

### struct

structのサポートはまだ途中です。埋め込み等が無い単純なstructのみサポートしています。

- sliceも入れることはできません
- struct中にstructを直接定義できません
- 初期化時にフィールド名を指定できません
- structを返す関数を式の途中で使えません(一度変数に代入してください)

```go
// OK
type A struct {
	a int
}
// OK
type B struct {
	a A
	b1, b2 string
}

// Not supported
type C struct {
	c1 string
	c2 struct {
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

- `shell.TempVarString` (= string) は _tmpN 変数を使って値を返します
- `shell.StatusCode` (= byte) は関数の終了コードとして返します

多値の戻り値をそのまま他の関数に渡すことはできません。例： `fmt.Println(functionReturnsMultiValues())`

### レシーバ

レシーバのある関数(メソッド)も使えますが、ポインタが無いのでメソッド内で自身の値を変更することはできません。

## goroutine

サブプロセスとして実行されます。無名関数はまだサポートされていないので通常の名前付きの関数を呼び出してください。

また、チャネルも使えないので、`os.Pipe()` (fifoが作られます)で作ったreader/writer等で通信してください。

## 特殊な関数

トランスパイラ自体を制御する関数です。トランスパイル時に処理されるので定数のみ渡せます。

### shell.Do()

トランスパイル時に渡された文字列をシェルスクリプトとして出力します。単一の文字列リテラルのみ利用できます。

### shell.SetFloatPrecision()

float型の精度を指定します。例えば、以下のプログラムをトランスパイルして実行すると円周率を1000桁出力します。

```go
	shell.SetFloatPrecision(1000)
	fmt.Println("Pi:", math.Atan(1)*4)
```

# Security

ashなどbash以外でも動作させるために実行時に `eval` している箇所が多くなっています。
また、あまりテストされていないのでスクリプト生成時のエスケープ漏れなどもありそうです。
信頼されない入力値を受け取るプログラムには使わないか、入力を注意深くバリデートしてください。

# License

MIT License
