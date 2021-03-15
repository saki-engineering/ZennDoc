package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.Create("write.txt")
	if err != nil {
		fmt.Println("cannot open the file")
	}
	defer f.Close()

	str := "write this file by Golang!"
	data := []byte(str)
	count, err := f.Write(data)
	if err != nil {
		fmt.Println(err)
		fmt.Println("fail to write file")
	}
	fmt.Printf("read %d bytes\n", count)
}
