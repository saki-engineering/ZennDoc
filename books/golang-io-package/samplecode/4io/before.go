package main

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func main() {
	// ファイルの場合
	/*
		obj, _ := os.Open("../2file/read.txt")
		defer obj.Close()

		FileTranslateIntoGerman(obj)
	*/

	// netの場合
	// 送信connは3net/server.goを使用すること
	obj, _ := net.Dial("tcp", "localhost:8080")
	NetTranslateIntoGerman(obj)
}

func FileTranslateIntoGerman(f *os.File) {
	data := make([]byte, 300)
	len, _ := f.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	fmt.Println(result)
}

func NetTranslateIntoGerman(conn net.Conn) {
	data := make([]byte, 300)
	len, _ := conn.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	fmt.Println(result)
}
