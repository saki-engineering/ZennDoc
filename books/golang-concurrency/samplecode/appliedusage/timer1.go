package main

import (
	"fmt"
	"time"
)

func main() {
	ch1 := make(chan int)

	for {
		t := time.NewTimer(1 * time.Second)
		defer t.Stop()

		select {
		case s := <-ch1:
			fmt.Println(s)
		case <-t.C:
			fmt.Println("time out")
			return
			// これがあるとダメ
			/*
				default:
					fmt.Println("default")
					time.Sleep(time.Millisecond * 100)
			*/
		}
	}
}
