package fuga

import (
	"context"
	"fmt"
)

type ctxKey struct{}

func SetValue(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxKey{}, "c")
}

func GetValueFromFuga(ctx context.Context) {
	val, ok := ctx.Value(ctxKey{}).(string)
	fmt.Println(val, ok)
}
