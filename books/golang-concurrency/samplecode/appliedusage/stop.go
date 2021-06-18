package main

import (
	"fmt"
	"time"
)

func generator(done chan struct{}) <-chan int {
	result := make(chan int)
	go func() {
		defer close(result)
	LOOP:
		for {
			select {
			case <-done:
				fmt.Println("break")
				break LOOP
			case result <- 1:
			}
		}
		fmt.Println("end")
	}()
	return result
}

func main() {
	done := make(chan struct{})

	result := generator(done)
	for i := 0; i < 5; i++ {
		fmt.Println(<-result)
	}
	close(done)

	time.Sleep(time.Second)
}
