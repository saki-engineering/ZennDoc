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

func doSomethingSpecial(ctx context.Context) context.Context {
	return context.WithValue(ctx, num, 2)
}

func main() {
	ctx := context.Background()
	ctx = context.WithValue(ctx, num, 1)

	isOdd(ctx)

	ctx = doSomethingSpecial(ctx)

	isOdd(ctx)
}
