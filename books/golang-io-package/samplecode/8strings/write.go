package main

import (
	"fmt"
	"strings"
)

func main() {
	var b strings.Builder
	src := []byte("world!!!!!!!!")

	b.Write(src)
	fmt.Println(b.String())
}
