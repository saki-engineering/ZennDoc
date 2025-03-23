package main

import "fmt"

type Set struct {
	X, Y int
}

func doubleLoop(i, j int) <-chan Set {
	ch := make(chan Set)
	go func() {
		defer close(ch)
		for x := 0; x < i; x++ {
			for y := 0; y < j; y++ {
				ch <- Set{x, y}
			}
		}
	}()
	return ch
}

func main() {
	ch := doubleLoop(2, 3)
	for s := range ch {
		fmt.Println(s.X, s.Y)
	}
}
