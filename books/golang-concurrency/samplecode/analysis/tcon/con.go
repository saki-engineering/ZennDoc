package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

func RandomWait(i int) {
	fmt.Printf("No.%d start\n", i+1)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	fmt.Printf("No.%d done\n", i+1)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			RandomWait(i)
		}(i)
	}
	wg.Wait()
}
