# Go to sh

PoC for generating shell script from subset of Go.

Goのコードをシェルスクリプトに変換するやつです。

シェルスクリプトを型付きの言語で書きたくて実装しました。
普通にGoのバイナリを実行するのが困難な環境のために作ったので、slice以外はBusyBoxのashで動作するようにしています。

Supported:

- Types: `int`, `string`, `[]int`, `[]string` 
- Go keyword, func, if, else, for, break, continue, const, var, append, len, go

TODO:

- jq, curl support
- Convert bash/compiler.go to compiler.sh

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
	FizzBuzz(100)
}
```

### Output(fizz_buzz.sh)

```bash
#!/bin/sh

fizz="Fizz"
buzz="Buzz"
function FizzBuzz() {
  local n="$1"; shift
  i=1
  while [ $(( i<=n )) -ne 0 ]; do
    if [ $(( i%15==0 )) -ne 0 ]; then
      echo "$fizz""$buzz"
    elif [ $(( i%3==0 )) -ne 0 ]; then
      echo "$fizz"
    elif [ $(( i%5==0 )) -ne 0 ]; then
      echo "$buzz"
    else
      echo "$i"
    fi

  let "i++"; done
}

# function main()
  FizzBuzz 100
# end of main
```

## Supported functions

- [bash.*](bash/builtin.go)
- [fmt.Print](https://pkg.go.dev/fmt#Print)
- [fmt.Println](https://pkg.go.dev/fmt#Println)
- [fmt.Printf](https://pkg.go.dev/fmt#Printf)
- [fmt.Sprint](https://pkg.go.dev/fmt#Sprint)
- [fmt.Sprintln](https://pkg.go.dev/fmt#Sprintln)
- [fmt.Sprintf](https://pkg.go.dev/fmt#Sprintf)
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
- [os.Getwd](https://pkg.go.dev/os#Getwd)
- [os.Chdir](https://pkg.go.dev/os#Chdir)
- [os.Getpid](https://pkg.go.dev/os#Getpid)
- [os.Getppid](https://pkg.go.dev/os#Getppid)
- [os.Getuid](https://pkg.go.dev/os#Getuid)
- [os.Geteuid](https://pkg.go.dev/os#Geteuid)
- [os.Getgid](https://pkg.go.dev/os#Getgid)
- [os.Getegid](https://pkg.go.dev/os#Getegid)
- [os.Hostname](https://pkg.go.dev/os#Hostname)
- [os.Getenv](https://pkg.go.dev/os#Getenv)
- [os.Setenv](https://pkg.go.dev/os#Setenv)

# 制限

## サポートしていないものがたくさんあります

- defer, range, make, new, chan, switch, select, struct, map...

## 型

- 利用可能な型は、`int`, `string`, `[]int`, `[]string` のみです
- 定数の場合のみ`float`を扱えます(例： `bash.Sleep(0.1)` は有効)

sliceの実装はbash専用です。zshの場合は `setopt KSH_ARRAYS` を追加する必要があると思います。

## 関数

### 引数

関数の最後の引数以外ではスライスを受け取ることはできません。またすべての値はsliceなども含めて値渡しです。

### 戻り値

関数の結果は標準出力として返します。なので基本的に値を返す関数の内部で標準出力に何かを出力することはできません。
標準出力以外で値を返したい場合は以下の型(type alias)が使えます。(名前しか見てないので同名のtypeを定義しても動作します)

- `bash.TempVarString` (= string) は _tmpN 変数を使って値を返します。複数の値を返す必要がある場合に使います
- `bash.StatusCode` (= byte) は関数の終了コードとして返します

多値の戻り値は以下の組み合わせに対応しています。

- (*, StatusCode)
- (TempVarString, TempVarString, ..., StatusCode)

## レシーバ

レシーバのある関数(メソッド)も使えますが、ポインタが無いのでメソッド内で自身の値を書き換えることはできません。

## goroutine

サブプロセスとして実行されます。いまのところ、起動したgoroutineとの通信手段は用意していません。
また、無名関数も使えないので通常の関数を使ってください。

# License

MIT License
