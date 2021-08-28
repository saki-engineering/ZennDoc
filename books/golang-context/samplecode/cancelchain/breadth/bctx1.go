package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 兄弟・並列
	// cxt1とctx2は兄弟(親はctx0)
	// ctx1のキャンセルだけ実行
	// →G1だけキャンセルされる

	ctx0 := context.Background()

	ctx1, cancel1 := context.WithCancel(ctx0)
	go func(ctx1 context.Context) {
		select {
		case <-ctx1.Done():
			fmt.Println("G1 canceled")
		}
	}(ctx1)

	ctx2, _ := context.WithCancel(ctx0)
	go func(ctx2 context.Context) {
		select {
		case <-ctx2.Done():
			fmt.Println("G2 canceled")
		}
	}(ctx2)

	cancel1()

	time.Sleep(time.Second)
}

// G1 canceled
