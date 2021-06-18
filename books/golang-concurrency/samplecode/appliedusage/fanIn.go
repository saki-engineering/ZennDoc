package main

import (
	"fmt"
	"sync"
	"time"
)

func generator(done chan struct{}, a int) <-chan int {
	gen := make(chan int)
	go func() {
		defer close(gen)
	LOOP:
		for {
			select {
			case <-done:
				break LOOP
			case gen <- a:
			}
		}
		fmt.Printf("closing gen%d\n", a)
	}()
	return gen
}

func fanIn1(done chan struct{}, c1, c2 <-chan int) <-chan int {
	result := make(chan int)

	go func() {
		defer fmt.Println("closed fanin")
		defer close(result)
		for {
			// caseはfor文で回せないので(=可変長は無理)
			// 統合元のチャネルがスライスでくるとかだとこれはできない
			select {
			case <-done:
				fmt.Println("done")
				return
			case num := <-c1:
				fmt.Println("send 1")
				result <- num
			case num := <-c2:
				fmt.Println("send 2")
				result <- num
			default:
				fmt.Println("continue")
				continue
			}
		}
	}()

	return result
}

func fanIn2(done chan struct{}, cs ...<-chan int) <-chan int {
	result := make(chan int)

	var wg sync.WaitGroup
	wg.Add(len(cs))

	for i, c := range cs {
		go func(c <-chan int, i int) {
			defer wg.Done()

			for num := range c {
				select {
				case <-done:
					fmt.Println("wg.Done", i)
					return
				case result <- num:
					fmt.Println("send", i)
				}
			}
		}(c, i)
	}

	go func() {
		// selectでdoneが閉じられるのを待つと、
		// 全てのルーチンが終わった保証がない
		wg.Wait()
		fmt.Println("closing fanin")
		close(result)
	}()

	return result
}

func main() {
	done := make(chan struct{})

	gen1 := generator(done, 1)
	gen2 := generator(done, 2)

	//result := fanIn1(done, gen1, gen2)
	result := fanIn2(done, gen1, gen2)
	for i := 0; i < 5; i++ {
		<-result
	}
	close(done)
	fmt.Println("main close done")
	for {
		if _, ok := <-result; !ok {
			break
		}
	}

	time.Sleep(time.Second * 1)
}

// Unbuffered channels are blocking. Each send must have a receive.

/*
fanIn1でこういうときもある
continue
send 2
send 1
send 1
send 2
send 2
send 2
closing gen2
closing gen1
main close done
send 1
send 1
send 1
send 1
send 2
send 1
done
closed fanin
*/
