package main

import (
	"net"
	"fmt"
)

func main() {
	// クライアントがconnインターフェースくれっていうのはDial
	// これは*net.TCPConn型
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println("error: ", err)
	}

	data := make([]byte, 1024)
	count, _ := conn.Read(data)
	fmt.Println(string(data[:count]))
}