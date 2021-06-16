package main

import (
	"fmt"
	"sync"
	"time"
)

func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

	var mu sync.Mutex

	for _, s := range src {
		go func(s int) {
			result := s * 2
			mu.Lock()
			dst = append(dst, result)
			mu.Unlock()
		}(s)
	}

	time.Sleep(time.Second)
	fmt.Println(dst)
}
