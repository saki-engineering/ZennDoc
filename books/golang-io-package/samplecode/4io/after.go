package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

func main() {
	// ファイルの場合
	obj, _ := os.Open("../2file/read.txt")
	defer obj.Close()

	// netの場合
	// 送信connは3net/server.goを使用すること
	//obj, _ := net.Dial("tcp", "localhost:8080")

	TranslateIntoGerman(obj)
}

func TranslateIntoGerman(r io.Reader) {
	data := make([]byte, 300)
	len, _ := r.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	fmt.Println(result)
}
