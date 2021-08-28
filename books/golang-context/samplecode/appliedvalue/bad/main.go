package main

import (
	"bad/fuga"
	"bad/hoge"
	"context"
)

func main() {
	ctx := context.Background()

	ctx = hoge.SetValue(ctx)
	ctx = fuga.SetValue(ctx)

	hoge.GetValueFromHoge(ctx)
	fuga.GetValueFromFuga(ctx)
}
