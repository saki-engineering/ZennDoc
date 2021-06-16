package main

import (
	"fmt"
	"time"
)

func main() {
	for i := 0; i < 3; i++ {
		/*
			go func() {
				fmt.Println(i)
			}()
		*/
		go func(i int) {
			fmt.Println(i)
		}(i)
	}
	time.Sleep(time.Second * 5)
}
