package main

import (
	"fmt"
	"iter"
)

type Set struct {
	X, Y int
}

func doubleLoop(i, j int) iter.Seq[Set] {
	return func(yield func(Set) bool) {
		for x := 0; x < i; x++ {
			for y := 0; y < j; y++ {
				if !yield(Set{x, y}) {
					return
				}
			}
		}
	}
}

func doubleLoopSlice(i, j int) []Set {
	result := make([]Set, 0, i*j)
	for x := 0; x < i; x++ {
		for y := 0; y < j; y++ {
			result = append(result, Set{x, y})
		}
	}
	return result
}

func main() {
	for x := 0; x < 2; x++ {
		for y := 0; y < 3; y++ {
			fmt.Println(x, y)
		}
	}

	fmt.Println("----")

	for _, s := range doubleLoopSlice(2, 3) {
		fmt.Println(s.X, s.Y)
	}

	fmt.Println("----")

	for s := range doubleLoop(2, 3) {
		fmt.Println(s.X, s.Y)
	}
}
