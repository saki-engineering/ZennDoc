package main

import (
	"fmt"
	"sync"
	"time"
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
				// case out <- num: これが時間がかかっているという想定
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

LOOP:
	for i := 0; i < 5; i++ {
		select {
		case result := <-gen:
			fmt.Println(result)
		case <-time.After(time.Second): // 1秒間selectできなかったら
			fmt.Println("timeout")
			break LOOP
		}
	}
	close(done)

	wg.Wait()
}
