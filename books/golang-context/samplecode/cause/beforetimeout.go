package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	taskA(ctx)
	taskB(ctx)
}

func taskA(ctx context.Context) {
	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
	fmt.Println("start taskA...")

	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
	case <-_taskA():
		fmt.Println("taskA done")
	}
}

func taskB(ctx context.Context) {
	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
	fmt.Println("start taskB..")

	select {
	case <-ctx.Done():
		fmt.Println(ctx.Err())
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
