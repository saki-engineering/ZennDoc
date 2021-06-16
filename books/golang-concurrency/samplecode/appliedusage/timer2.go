package main

import (
	"fmt"
	"time"
)

func main() {
	ch1 := make(chan int)

	t := time.NewTimer(1 * time.Second)
	defer t.Stop()

	for {
		select {
		case s := <-ch1:
			fmt.Println(s)
		case <-t.C:
			fmt.Println("time out")
			return
		default:
			fmt.Println("default")
			time.Sleep(time.Millisecond * 100)
		}
	}
}
