package main

import "fmt"

type MyStruct struct {
	mystruct
}

type mystruct struct {
	id int
}

func (n *mystruct) Write() {
	fmt.Println(n.id)
}

func main() {
	a := MyStruct{mystruct{id: 1000}}
	a.Write()
}
