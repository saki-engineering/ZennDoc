package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 同一人物・並列
	// ctx1を、G1-1, G1-2というゴールーチンに渡す
	// ctx1のキャンセル実行
	// →G1-1, G1-2両方キャンセルされる

	ctx0 := context.Background()

	ctx1, cancel1 := context.WithCancel(ctx0)
	go func(ctx1 context.Context) {
		select {
		case <-ctx1.Done():
			fmt.Println("G1-1 canceled")
		}
	}(ctx1)

	go func(ctx1 context.Context) {
		select {
		case <-ctx1.Done():
			fmt.Println("G1-2 canceled")
		}
	}(ctx1)

	cancel1()

	time.Sleep(time.Second)
}

// G1-1 canceled
// G1-2 canceled
