package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx := context.Background()
	fmt.Println(ctx.Deadline())

	fmt.Println(time.Now())
	ctx, _ = context.WithTimeout(ctx, 2*time.Second)
	fmt.Println(ctx.Deadline())
}
