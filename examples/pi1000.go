package main

import (
	"fmt"
	"math"

	"github.com/binzume/gotosh/shell"
)

func main() {
	// Print 1000 digits of Pi
	shell.SetFloatPrecision(1000)
	fmt.Println("Pi:", math.Atan(1)*4)
}
