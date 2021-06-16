package main

import "fmt"

func restFunc() <-chan int {
	result := make(chan int)
	go func() {
		defer close(result)
		// resultに値を送信する処理
		for i := 0; i < 5; i++ {
			result <- 1
		}
	}()
	return result
}

func main() {
	result := restFunc()
	for i := 0; i < 5; i++ {
		fmt.Println(<-result)
	}
}
