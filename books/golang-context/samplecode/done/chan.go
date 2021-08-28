package main

import (
	"fmt"
	"sync"
)

var wg sync.WaitGroup

func generator(done chan struct{}, num int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-done:
				break LOOP
			case out <- num:
			}
		}

		close(out)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	done := make(chan struct{})
	gen := generator(done, 1)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
	close(done)

	wg.Wait()
}
