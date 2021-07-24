package main

import "fmt"

func add(m int, n int) int

// 以下でGoとアセンブリを実行できる
// cd go-assembly/add2
// go build -o a.out
// ./a.out
func main() {
	i := add(1, 2)
	fmt.Println(i) // 3が出力される

	i = add(3, 4)
	fmt.Println(i) // 7が出力される
}
