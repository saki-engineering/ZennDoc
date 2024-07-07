package main

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
			case out <- num:
			}
		}

		close(out)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	// ctx, cancel := context.WithCancel(context.Background())
	ctx, cancel := context.WithCancelCause(context.Background())
	gen := generator(ctx, 1)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
	// cancel()
	cancel(errors.New("got enough data by main func"))

	wg.Wait()
}
