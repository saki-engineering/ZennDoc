package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"
)

var wg sync.WaitGroup

func main() {
	ctx, cancel := context.WithCancelCause(context.Background())
	wg.Add(1)
	go task(ctx)

	randomSleep()
	cancel(errors.New("canceled by main func"))

	wg.Wait()
}

func task(ctx context.Context) {
	defer wg.Done()

	ctx, cancel := context.WithCancelCause(ctx)
	wg.Add(1)
	go subTask(ctx)

	randomSleep()
	cancel(errors.New("canceled by task func"))
}

func subTask(ctx context.Context) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		fmt.Println(context.Cause(ctx))
	case <-doSomething():
		fmt.Println("subtask done")
	}
}

func randomSleep() {
	r := rand.Intn(3000)
	time.Sleep(time.Duration(r) * time.Millisecond)
}

func doSomething() <-chan time.Time {
	return time.NewTimer(1 * time.Second).C
}
