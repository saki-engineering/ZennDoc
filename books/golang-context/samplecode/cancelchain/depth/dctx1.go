package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 親子・直列
	// (ctx0→)ctx1→ctx2→ctx3という親子関係
	// ctx2のキャンセルだけ実行
	// →G2とG3だけキャンセル、G1は生きてる

	ctx0 := context.Background()

	ctx1, _ := context.WithCancel(ctx0)
	go func(ctx1 context.Context) {
		ctx2, cancel2 := context.WithCancel(ctx1)

		go func(ctx2 context.Context) {
			ctx3, _ := context.WithCancel(ctx2)

			go func(ctx3 context.Context) {
				select {
				case <-ctx3.Done():
					fmt.Println("G3 canceled")
				}
			}(ctx3)

			select {
			case <-ctx2.Done():
				fmt.Println("G2 canceled")
			}
		}(ctx2)

		cancel2()

		select {
		case <-ctx1.Done():
			fmt.Println("G1 canceled")
		}

	}(ctx1)

	time.Sleep(time.Second)
}

// G2 canceled
// G3 canceled
