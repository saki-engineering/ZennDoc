package main

import (
	"fmt"
	"iter"
)

func iterate() iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := 0; i < 10; i++ {
			fmt.Printf("yield %d\n", i)
			ok := yield(i)
			// fmt.Printf("yield %d\n", i)
			if !ok {
				return
			}
		}
	}
}

func main() {
	for i := range iterate() {
		fmt.Printf("recv %d\n", i)
	}
}
