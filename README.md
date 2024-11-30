# Go to sh

PoC for generating bash script from small subset of Go.

Goのコードをシェルスクリプトに変換するやつです。限られたことしかできません。実用的なものになる予定はないです。

Supported:

- types: `int`, `string`, `[]int`, `[]string` 
- go keyword, func, if, else, for, break, continue, const, var, append, len, go
- commands: pwd, cd, export, echo, printf, read, exit, sleep.

TODO:

- jq, curl support
- Convert bash/compiler.go to compiler.sh

# 制限

## サポートしていないものがたくさんあります

- for range, make, new, ch, switch, select...

## 型

- 利用可能な型は、`int`, `string`, `[]int`, `[]string` のみです
- 定数の場合のみ`float`を扱えます(例： `bash.Sleep(0.1)` は有効)

## 関数の引数

- 関数の最後の引数以外ではスライスを受け取ることはできません

## 関数の戻り値

場合によって値の返し方が異なります。扱いを明示したい場合は以下の型(type alias)が使えます。

- `bash.StdoutString` (= string) は標準出力として関数の結果を返します (これがデフォルト動作。この場合は関数内でecho等はできません)
- `bash.StatusCode` (= byte) は関数の終了コードとして返します
- `bash.TempVarString` (= string) は_tmp変数に値をセットします
- 多値のサポートは2つ目の戻り値が `bash.StatusCode` 型のときのみ動きます

## goroutine

サブプロセスとして実行されます。いまのところ、起動したgoroutineとの通信手段は用意していません。
また、無名関数も使えないので通常の関数を使ってください。

# Usage

```bash
go run . examples/hello_world.go > hello.sh
chmod a+x hello.sh
./hello.sh
```

## Examples

- examples/hello_workd.go
- examples/read_stdin.go
- examples/fizz_buzz.go

Input:

```go
package main

import "fmt"

// Hello, world
func main() {
	fmt.Println("Hello, world!")
}
```

Output:

```bash
#!/bin/bash

# Hello, world
# function main()
  echo "Hello, world!"
# end of main
```

# License

MIT License
