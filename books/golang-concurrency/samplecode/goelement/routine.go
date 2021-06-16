package main

import (
	"fmt"
	"math/rand"
	"time"
)

func getLuckyNum() {
	fmt.Println("...")

	// ランダム占い時間
	rand.Seed(time.Now().Unix())
	time.Sleep(time.Duration(rand.Intn(3000)) * time.Millisecond)

	num := rand.Intn(10)
	fmt.Printf("Today's your lucky number is %d!\n", num)
}

func main() {
	fmt.Println("what is today's lucky number?")
	go getLuckyNum()

	time.Sleep(time.Second * 5)
}
