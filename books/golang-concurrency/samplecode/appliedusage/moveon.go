package main

import (
	"fmt"
)

type Conn int

func (c Conn) DoQuery(s string) Result {
	return Result(c)
}

type Result int

func Query(conns []Conn, query string) Result {
	ch := make(chan Result, len(conns))
	for _, conn := range conns {
		go func(c Conn) {
			select {
			case ch <- c.DoQuery(query):
			default:
			}
		}(conn)
	}
	return <-ch
}

func main() {
	conns := []Conn{1, 2, 3, 4, 5}
	query := "myquery"

	result := Query(conns, query)
	fmt.Println(result)
}
