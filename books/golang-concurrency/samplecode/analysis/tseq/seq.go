package main

import (
	"fmt"
	"math/rand"
	"time"
)

func RandomWait(i int) {
	fmt.Printf("No.%d start\n", i+1)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	fmt.Printf("No.%d done\n", i+1)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		RandomWait(i)
	}
}
