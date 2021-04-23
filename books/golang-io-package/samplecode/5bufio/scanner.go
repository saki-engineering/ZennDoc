package main

import (
	"bufio"
	"fmt"
	"os"
)

func main() {
	f, _ := os.Open("text.txt")
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		fmt.Println(line)
	}
}
