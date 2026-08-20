package main

import (
	"fmt"

	"github.com/binzume/gotosh/curl"
)

func main() {
	body, err := curl.Get(
		"https://www.binzume.net/",
		"Accept: text/html",
		"User-Agent: gotosh",
	)
	fmt.Println("error:", err)
	fmt.Println(body)
}
