package main

import (
	"context"
	"fmt"
	"sync"
)

var wg sync.WaitGroup

type ctxRequestKey int

const (
	requestID ctxRequestKey = iota
)

func DBAccess(ctx context.Context) <-chan string {
	result := make(chan string)
	go func() {
		defer wg.Done()

		ID := RequestID(ctx)

		fmt.Printf("getting data for request ID %d\n", ID)
		result <- "result"

		close(result)
		fmt.Println("DB result chan closed")
	}()
	return result
}

func server(ctx context.Context) <-chan string {
	result := make(chan string)
	go func() {
		defer wg.Done()

		wg.Add(1)

		// ctxの中身は変えられない(keyがprivateな型だから)

		// do something
		DBresponse := DBAccess(ctx)
		select {
		case res := <-DBresponse:
			result <- res
		}

		close(result)
		fmt.Println("server result chan closed")
	}()
	return result
}

func RequestID(c context.Context) int {
	return c.Value(requestID).(int)
}

func main() {
	wg.Add(1)

	ctx := context.WithValue(context.Background(), requestID, 1)

	response := server(ctx)

	fmt.Println("response: ", <-response)

	wg.Wait()
}
