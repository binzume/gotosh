package tinytui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/binzume/gotosh/shell"
)

const Space = " "
const Fill = "█"
const LineH = "─"
const LineV = "│"
const LineC = "┼"
const LineBL = "└"
const LineBR = "┘"
const LineTL = "┌"
const LineTR = "┐"
const LineTT = "┬"
const LineRT = "┤"
const LineLT = "├"
const LineBT = "┴"
const Bullet = "·"

const BrailleZero = '⠀'

const ASCII_LineH = "-"
const ASCII_LineV = "|"
const ASCII_LineC = "+"

const Black = 0
const Red = 1
const Green = 2
const Yellow = 3
const Blue = 4
const Magenta = 5
const Cyan = 6
const White = 7
const Default = 9

func GOTOSH_FUNC_strings_Index(s, f string) int {
	i := 0
	_ = shell.Do(`_tmp=${s#*$f}; i=$((${#s}-${#_tmp}-${#f}))`)
	if i < 0 {
		return -1
	}
	return i
}

func GOTOSH_FUNC_strings_Repeat(s string, n int) string {
	ret := ""
	for i := 0; i < n; i++ {
		ret += s
	}
	return ret
}

func ConsoleSize() (int, int) {
	sz, _ := shell.Exec("stty", "size")
	if p := strings.Index(sz, " "); p > 0 {
		h, _ := strconv.Atoi(sz[0:p])
		w, _ := strconv.Atoi(sz[p+1 : len(sz)])
		return w, h
	}
	return 80, 25
}

func Clear() {
	fmt.Print("\x1b[2J")
}

func SetColor(c int) {
	fmt.Printf("\x1b[3%dm", c)
}

func SetBgColor(c int) {
	fmt.Printf("\x1b[4%dm", c)
}

func SetColor24(r, g, b int) {
	fmt.Printf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

func SetBgColor24(r, g, b int) {
	fmt.Printf("\x1b[48;2;%d;%d;%dm", r, g, b)
}

func ResetColor() {
	fmt.Printf("\x1b[0m")
}

func MoveTo(x, y int) {
	fmt.Printf("\x1b[%d;%dH", y, x)
}

func MoveRight(n int) {
	fmt.Printf("\x1b[%dC", n)
}

func Box(x, y, w, h int) {
	MoveTo(x, y)
	if strings.Contains(os.Getenv("LANG"), ".UTF-8") {
		fmt.Print(LineTL + strings.Repeat(LineH, w-2) + LineTR)
		for i := y + 1; i < y+h; i++ {
			MoveTo(x, i)
			fmt.Print(LineV)
			MoveRight(w - 2)
			fmt.Print(LineV)
		}
		MoveTo(x, y+h)
		fmt.Print(LineBL + strings.Repeat(LineH, w-2) + LineBR)
	} else {
		fmt.Print(ASCII_LineC + strings.Repeat(ASCII_LineH, w-2) + ASCII_LineC)
		for i := y + 1; i < y+h; i++ {
			MoveTo(x, i)
			fmt.Print(ASCII_LineV)
			MoveRight(w - 2)
			fmt.Print(ASCII_LineV)
		}
		MoveTo(x, y+h)
		fmt.Print(ASCII_LineC + strings.Repeat(ASCII_LineH, w-2) + ASCII_LineC)
	}
}

func BoxC(x, y, w, h int, c string) {
	MoveTo(x, y)
	fmt.Print(strings.Repeat(c, w))
	for i := y + 1; i < y+h; i++ {
		MoveTo(x, i)
		fmt.Print(c)
		MoveRight(w - 2)
		fmt.Print(c)
	}
	MoveTo(x, y+h)
	fmt.Print(strings.Repeat(c, w))
}

func BoxF(x, y, w, h int, c string) {
	l := strings.Repeat(c, w)
	for i := y; i < y+h; i++ {
		MoveTo(x, i)
		fmt.Print(l)
	}
}
