package main

import (
	"fmt"
)

func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

	c := make(chan int)

	for _, s := range src {
		go func(s int, c chan int) {
			result := s * 2
			c <- result
		}(s, c)
	}

	for _ = range src {
		num := <-c
		dst = append(dst, num)
	}

	fmt.Println(dst)
	close(c)
}
