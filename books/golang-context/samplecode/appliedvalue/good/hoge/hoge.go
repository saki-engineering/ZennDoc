package hoge

import (
	"context"
	"fmt"
)

type ctxKey struct{}

func SetValue(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, "b")
}

func GetValueFromHoge(ctx context.Context) {
	val, ok := ctx.Value(ctxKey{}).(string)
	fmt.Println(val, ok)
}
