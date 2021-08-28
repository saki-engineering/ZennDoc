package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	// 同一人物・直列
	// (ctx0→)ctx1→ctx2という親子関係
	// ctx2のキャンセルだけ実行
	// →G2-1とG2-2だけキャンセル、G1は生きてる

	ctx0 := context.Background()

	ctx1, _ := context.WithCancel(ctx0)
	go func(ctx1 context.Context) {
		ctx2, cancel2 := context.WithCancel(ctx1)

		go func(ctx2 context.Context) {
			go func(ctx2 context.Context) {
				select {
				case <-ctx2.Done():
					fmt.Println("G2-2 canceled")
				}
			}(ctx2)

			select {
			case <-ctx2.Done():
				fmt.Println("G2-1 canceled")
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
