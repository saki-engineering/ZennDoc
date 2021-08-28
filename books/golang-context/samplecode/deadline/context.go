package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var wg sync.WaitGroup

//func generator(done chan struct{}, num int) <-chan int {
func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)

	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			// case <-done:
			case <-ctx.Done():
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
	// done := make(chan struct{})
	// gen := generator(done, 1)
	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Second))
	//ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	gen := generator(ctx, 1)

	wg.Add(1)

LOOP:
	for i := 0; i < 5; i++ {
		select {
		case result, ok := <-gen:
			if ok {
				fmt.Println(result)
			} else {
				fmt.Println("timeout")
				break LOOP
			}
			/*
				case <-time.After(time.Second): // 1秒間selectできなかったら
					fmt.Println("timeout")
					break LOOP
			*/
		}
	}
	// close(done)
	cancel()

	wg.Wait()
}
