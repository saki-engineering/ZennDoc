package main

import (
	"fmt"
	"strings"
)

func main() {
	str := "Hellooooooooooooooooooooooooooo!"
	rd := strings.NewReader(str)

	row := make([]byte, 10)
	rd.Read(row)
	fmt.Println(string(row))
}
