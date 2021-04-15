package main

import (
	"fmt"
	"strings"
)

func main() {
	var b strings.Builder
	str := "written by string"

	b.WriteString(str)
	fmt.Println(b.String())
}
