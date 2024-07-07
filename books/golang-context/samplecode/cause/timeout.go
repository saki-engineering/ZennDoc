package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var wg sync.WaitGroup

func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)

	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-ctx.Done():
				fmt.Println("ctx.Err() : ", ctx.Err())
				fmt.Println("context.Cause(ctx) : ", context.Cause(ctx))
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
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	ctx, cancel := context.WithTimeoutCause(context.Background(), time.Second, errors.New("1s timeout"))
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
		}
	}
	cancel()

	wg.Wait()
}
