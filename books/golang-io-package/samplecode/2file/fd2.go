package main

import (
	"fmt"
	"os"
)

func main() {
	f1, err := os.Open("write.txt")
	if err != nil {
		fmt.Println("cannot open the file")
	}
	defer f1.Close()

	f2, err := os.Open("read.txt")
	if err != nil {
		fmt.Println("cannot open the file")
	}
	defer f2.Close()

	fmt.Println("write.txt", f1.Fd())
	fmt.Println("read.txt", f2.Fd())
}
