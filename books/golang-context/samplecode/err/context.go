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
				// タイムアウトで止まったのか？
				// それともキャンセルされて止まったのか？
				// Doneメソッドだけでは判定不可
				if err := ctx.Err(); errors.Is(err, context.Canceled) {
					fmt.Println("canceled")
				} else if errors.Is(err, context.DeadlineExceeded) {
					fmt.Println("deadline")
				}
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
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	gen := generator(ctx, 1)

	wg.Add(1)

LOOP:
	for i := 0; i < 5; i++ {
		select {
		case result, ok := <-gen:
			if ok {
				fmt.Println(result)
			} else {
				break LOOP
			}
		}
	}
	cancel()

	wg.Wait()
}
