# Go to sh

PoC for generating bash script from small subset of Go.

Goのコードをシェルスクリプトに変換するやつです。限られたことしかできません。実用的なものになる予定はないです。

Supported:

- types: int, string
- go keyword, func, if, else, for, break, continue, const, var
- commands: pwd, cd, export, echo, printf, read, exit, sleep.


Function return value:

場合によって値の返し方が異なります。扱いを明示したい場合は以下の型が使えます。

- `bash.StdoutString` (= string) は標準出力として関数の結果を返します (この場合は関数内でecho等はできません)
- `bash.StatusCode` (= byte) は関数の終了コードとして返します
- `bash.TempVarString` (= string) は_tmp変数に値をセットします
- 多値のサポートは2つ目の戻り値が `bash.StatusCode` 型のときのみ動きます

TODO:

- error type
- slice support
- jq, curl support
- Convert bash/compiler.go to compiler.sh

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
