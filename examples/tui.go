package main

// Transpile:
// go run . examples/tinytui/tinytui.go examples/tui.go  > tui.sh

import (
	"fmt"

	"github.com/binzume/gotosh/examples/tinytui"
	"github.com/binzume/gotosh/shell"
)

func getCh() string {
	k := ""
	shell.Do(`read -r -d '' -s -n 1 -t 1 k`)
	return k
}

func main() {

	for r := 0; r < 256; r++ {
		tinytui.SetColor24(r, 0, 255)
		fmt.Print("#")
	}

	for i := 0; i <= 7; i++ {
		tinytui.SetColor(i)
		tinytui.BoxF(i*10, 2, 8, 4, tinytui.Fill)
	}
	tinytui.ResetColor()

	sw, sh := tinytui.ConsoleSize()
	fmt.Println("W", sw, "H", sh)
	x := 10
	y := 5
	for {
		k := getCh()
		if k != "" {
			k = string(k[0])
			tinytui.BoxC(x, y, 20, 10, " ")
			if (k == "a") && x > 1 {
				x -= 1
			} else if (k == "d") && x <= sw-20 {
				x += 1
			} else if (k == "w") && y > 1 {
				y -= 1
			} else if (k == "s") && y < sh-10 {
				y += 1
			} else if k == "q" {
				break
			}
			tinytui.Box(x, y, 20, 10)
		} else {
			shell.Sleep(0.1)
		}
	}
}
