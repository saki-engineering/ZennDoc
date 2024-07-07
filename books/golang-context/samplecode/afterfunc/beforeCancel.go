package main

import (
	"context"
	"fmt"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	wg.Add(1)
	go func() {
		defer wg.Done()
	L:
		for {
			select {
			case <-ctx.Done():
				fmt.Println("ctx cleanup done")
				break L
			case <-time.Tick(time.Second):
				fmt.Println("tick")
			}
		}
	}()

	time.Sleep(3 * time.Second)
	cancel()

	time.Sleep(1 * time.Second)
	wg.Wait()
}
