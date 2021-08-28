package main

import (
	"context"
	"fmt"
	"sync"
)

var wg sync.WaitGroup

//func generator(ctx context.Context, num int, userID int, authToken string, traceID int) <-chan int {
func generator(ctx context.Context, num int) <-chan int {
	out := make(chan int)
	go func() {
		defer wg.Done()

	LOOP:
		for {
			select {
			case <-ctx.Done():
				break LOOP
			case out <- num:
			}
		}

		close(out)
		userID, authToken, traceID := ctx.Value("userID").(int), ctx.Value("authToken").(string), ctx.Value("traceID").(int)
		fmt.Println("log: ", userID, authToken, traceID)
		fmt.Println("generator closed")
	}()
	return out
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "userID", 2)
	ctx = context.WithValue(ctx, "authToken", "xxxxxxxx")
	ctx = context.WithValue(ctx, "traceID", 3)

	gen := generator(ctx, 1)
	// gen := generator(ctx, 1, 2, "xxxxxxxx", 3)

	wg.Add(1)

	for i := 0; i < 5; i++ {
		fmt.Println(<-gen)
	}
	cancel()

	wg.Wait()
}
