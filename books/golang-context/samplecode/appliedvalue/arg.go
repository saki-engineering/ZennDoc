package main

import (
	"context"
	"fmt"
)

type ctxKey int

const (
	num ctxKey = iota
)

func isOdd(ctx context.Context) {
	num, ok := ctx.Value(num).(int)
	if ok {
		if num%2 == 1 {
			fmt.Println("odd number")
		} else {
			fmt.Println("not odd number")
		}
	}
}

func doSomething(ctx context.Context) {
	isOdd(ctx)
}

func main() {
	ctx := context.Background()
	ctx = context.WithValue(ctx, num, 1)

	doSomething(ctx)
}
