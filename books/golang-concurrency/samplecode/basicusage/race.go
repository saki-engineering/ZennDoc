package main

import (
	"fmt"
	"time"
)

func main() {
	src := []int{1, 2, 3, 4, 5}
	dst := []int{}

	// srcの要素毎にある何か処理をして、結果をdstにいれる
	for _, s := range src {
		go func(s int) {
			// 何か(重い)処理をする
			result := s * 2

			// 結果をdstにいれる
			dst = append(dst, result)
		}(s)
	}

	time.Sleep(time.Second)
	fmt.Println(dst)
}
