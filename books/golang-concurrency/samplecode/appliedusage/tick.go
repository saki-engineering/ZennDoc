package main

import (
	"fmt"
	"time"
)

func main() {
	t := time.NewTicker(time.Millisecond * 100)
	defer t.Stop()

	for i := 0; i < 5; i++ {
		select {
		//case <-time.After(time.Millisecond * 100):
		case <-t.C:
			fmt.Println("tick")
		}
	}
}
