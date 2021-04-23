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

	data := make([]byte, 1024)
	count, err := f.Read(data)
	if err != nil {
		fmt.Println(err)
		fmt.Println("fail to read file")
	}
	fmt.Printf("read %d bytes: %q\n", count, data[:count])
	fmt.Println(string(data[:count]))
}
