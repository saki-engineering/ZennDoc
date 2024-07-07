package main

import (
	"context"
	"errors"
	"fmt"
	"time"
)

func main() {
	ctx, _ := context.WithTimeoutCause(context.Background(), 3*time.Second, errors.New("timeout caused by main"))
	taskA(ctx)
	taskB(ctx)
}

func taskA(ctx context.Context) {
	ctx, _ = context.WithTimeoutCause(ctx, 2*time.Second, errors.New("timeout caused by taskA"))
	fmt.Println("start taskA...")

	select {
	case <-ctx.Done():
		fmt.Println(context.Cause(ctx))
	case <-_taskA():
		fmt.Println("taskA done")
	}
}

func taskB(ctx context.Context) {
	ctx, _ = context.WithTimeoutCause(ctx, 2*time.Second, errors.New("timeout caused by taskB"))
	fmt.Println("start taskB..")

	select {
	case <-ctx.Done():
		fmt.Println(context.Cause(ctx))
	case <-_taskA():
		fmt.Println("taskB done")
	}
}

func _taskA() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(1500 * time.Millisecond)
		close(ch)
	}()
	return ch
}

func _taskB() <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		time.Sleep(10000 * time.Millisecond)
		close(ch)
	}()
	return ch
}
