package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"runtime/trace"
	"sync"
	"time"
)

func doWork(done chan struct{}) {
	time.Sleep(time.Millisecond * 500)
	done <- struct{}{}
}

func _main() {
	// タスクを定義
	_, task := trace.NewTask(context.Background(), "main")
	defer task.End()

	done := make(chan struct{})

	go doWork(done)

	<-done

	var wg sync.WaitGroup
	n := 5
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
		}()
	}
	wg.Wait()
}

func main() {
	// トレースを始める
	f, err := os.Create("syncBlock.out")
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
