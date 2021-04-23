package main

import (
	"bytes"
	"fmt"
)

func main() {
	var b bytes.Buffer // A Buffer needs no initialization.
	b.Write([]byte("World"))

	plain := make([]byte, 10)
	b.Read(plain)

	fmt.Println("buffer: ", b.String())
	fmt.Println("output:", string(plain))
}
