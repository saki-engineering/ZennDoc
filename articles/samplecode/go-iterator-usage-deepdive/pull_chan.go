package main

import "fmt"

type Zipped[T1, T2 any] struct {
	V1  T1
	OK1 bool

	V2  T2
	OK2 bool
}

func Zip[T1, T2 any](ch1 <-chan T1, ch2 <-chan T2) chan Zipped[T1, T2] {
	ch := make(chan Zipped[T1, T2])
	go func() {
		defer close(ch)
		for {
			var val Zipped[T1, T2]
			val.V1, val.OK1 = <-ch1
			val.V2, val.OK2 = <-ch2
			if !val.OK1 && !val.OK2 {
				return
			}
			ch <- val
		}
	}()
	return ch
}

func main() {
	ch1 := make(chan int)
	ch2 := make(chan string)
	ch := Zip(ch1, ch2)
	go func() {
		defer close(ch1)
		defer close(ch2)
		// ch2 <- "a"
		ch1 <- 1
		ch2 <- "a"
		ch1 <- 2
		ch2 <- "b"
		ch1 <- 3
	}()

	for z := range ch {
		fmt.Println(z.V1, z.V2)
	}
}
