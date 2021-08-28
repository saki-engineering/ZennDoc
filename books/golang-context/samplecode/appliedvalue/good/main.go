package main

import (
	"context"
	"good/fuga"
	"good/hoge"
)

func main() {
	ctx := context.Background()

	ctx = hoge.SetValue(ctx)
	ctx = fuga.SetValue(ctx)

	hoge.GetValueFromHoge(ctx)
	fuga.GetValueFromFuga(ctx)
}
