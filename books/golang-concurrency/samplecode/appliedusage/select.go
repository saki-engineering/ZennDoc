package main

import (
	"fmt"
	"time"
)

func main() {
	gen1, gen2 := make(chan int), make(chan int)
	/*
		// gen1に送信
		go func() {
			gen1 <- 1
		}()
		// gen2に送信
		go func() {
			gen2 <- 2
		}()
	*/

	// ダメな例
	if n1, ok := <-gen1; ok {
		fmt.Println(n1)
	} else if n2, ok := <-gen2; ok {
		fmt.Println(n2)
	} else {
		fmt.Println("neither cannot use")
	}

	/*
		// いい例
		select {
		case num := <-gen1:
			fmt.Println(num)
		case num := <-gen2:
			fmt.Println(num)
		default:
			fmt.Println("neither chan cannot use")
		}
	*/

	time.Sleep(time.Second * 1)
}
