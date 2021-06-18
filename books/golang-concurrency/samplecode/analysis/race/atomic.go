package main

import (
	"fmt"
	"sync"
)

var num = 0

func add(a int) {
	num += a
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	var mu sync.Mutex

	go func() {
		defer wg.Done()
		mu.Lock()
		add(1)
		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
		mu.Lock()
		add(-1)
		mu.Unlock()
	}()

	wg.Wait()
	fmt.Println(num)
}
