package hoge

import (
	"context"
	"fmt"
)

type ctxKey int

const (
	a ctxKey = iota
)

func SetValue(ctx context.Context) context.Context {
	return context.WithValue(ctx, a, "b")
}

func GetValueFromHoge(ctx context.Context) {
	val, ok := ctx.Value(a).(string)
	fmt.Println(val, ok)
}
