package main

import (
	"fmt"
	"time"
)

func main() {
	ch1 := make(chan int)

	timeout := time.After(1 * time.Second)
	for {
		select {
		case s := <-ch1:
			fmt.Println(s)
		case <-timeout:
			fmt.Println("time out")
			return
		default:
			fmt.Println("default")
			time.Sleep(time.Millisecond * 100)
		}
	}
}
