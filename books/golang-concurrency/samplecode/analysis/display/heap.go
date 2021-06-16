package main

import (
	"context"
	"log"
	"os"
	"runtime/trace"
)

func UseMemory(ctx context.Context, cap int) {
	// regionを始める
	defer trace.StartRegion(ctx, "useMemory").End()

	slice := make([]int, cap)
	for i := 0; i < 1000; i++ {
		slice = append(slice, i)
	}
}

func _main() {
	// タスクを定義
	ctx, task := trace.NewTask(context.Background(), "main")
	defer task.End()

	for i := 0; i < 5; i++ {
		UseMemory(ctx, 1e5)
	}
}

func main() {
	// トレースを始める
	f, err := os.Create("heap.out")
	if err != nil {
		log.Fatalln("Error:", err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Fatalln("Error:", err)
		}
	}()

	if err := trace.Start(f); err != nil {
		log.Fatalln("Error:", err)
	}
	defer trace.Stop()

	_main()
}

// 途中でGCするので、heap領域の変化がよくわかる
