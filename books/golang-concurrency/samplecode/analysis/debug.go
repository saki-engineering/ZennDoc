package main

import (
	"math/rand"
	"sync"
	"time"
)

func doWork() {
	time.Sleep(time.Duration(rand.Intn(1500)) * time.Millisecond)
	var counter int
	for i := 0; i < 2*1e9; i++ {
		counter++
	}
}

func main() {
	var wg sync.WaitGroup
	n := 15

	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			doWork()
		}()
	}
	wg.Wait()
}
