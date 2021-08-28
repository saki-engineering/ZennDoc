package hoge

import (
	"context"
	"fmt"
)

func SetValue(ctx context.Context) context.Context {
	return context.WithValue(ctx, "a", "b")
}

func GetValueFromHoge(ctx context.Context) {
	val, ok := ctx.Value("a").(string)
	fmt.Println(val, ok)
}
