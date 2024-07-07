package main

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	wg.Add(1)
	go task(ctx)

	randomSleep()
	cancel()

	wg.Wait()
}

func task(ctx context.Context) {
	defer wg.Done()

	ctx, cancel := context.WithCancel(ctx)
	wg.Add(1)
	go subTask(ctx)

	randomSleep()
	cancel()
}

func subTask(ctx context.Context) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	case <-doSomething():
		fmt.Println("subtask done")
	}
}

func randomSleep() {
	r := rand.Intn(3000)
	time.Sleep(time.Duration(r) * time.Millisecond)
}

func doSomething() <-chan time.Time {
	return time.NewTimer(time.Second).C
}
