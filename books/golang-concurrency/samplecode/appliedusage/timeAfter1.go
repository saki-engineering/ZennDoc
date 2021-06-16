package main

import (
	"fmt"
	"time"
)

func main() {
	ch1 := make(chan int)

	for {
		select {
		case s := <-ch1:
			fmt.Println(s)
		case <-time.After(1 * time.Second):
			fmt.Println("time out")
			return
			/*
				// これがあるとダメ
				default:
					fmt.Println("default")
					time.Sleep(time.Millisecond * 100)
			*/
		}
	}
}
