package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.Open("read.txt")
	if err != nil {
		fmt.Println("cannot open the file")
	}
	defer f.Close()

	fmt.Println("read.txt", f.Fd())
}
