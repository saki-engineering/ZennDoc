package main

import (
	"fmt"
	"iter"
	"sync"
	"time"
)

var (
	searchTime  = 500 * time.Millisecond
	executeTime = 100 * time.Millisecond
)

type Node struct {
	Value    int
	Children []*Node
}

func doSomething(node *Node) {
	time.Sleep(executeTime)
	// fmt.Println(node.Value)
}

func SliceDFS(root *Node) []*Node {
	if root == nil {
		return nil
	}

	stack := []*Node{root}
	result := []*Node{}
	visited := make(map[*Node]bool)

	for len(stack) > 0 {
		// スタックの最後の要素を取得して削除
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if visited[node] {
			continue
		}

		time.Sleep(searchTime)

		visited[node] = true
		result = append(result, node)

		// 子ノードをスタックに追加（逆順で追加することで左から処理）
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
	return result
}

func SliceMain() {
	s := time.Now()
	var wg sync.WaitGroup

	root := &Node{Value: 1}
	child1 := &Node{Value: 2}
	child2 := &Node{Value: 3}
	child3 := &Node{Value: 4}
	child4 := &Node{Value: 5}

	root.Children = []*Node{child1, child2}
	child1.Children = []*Node{child3, child4}

	for _, node := range SliceDFS(root) {
		go func(node *Node) {
			defer wg.Done()
			fmt.Printf("execute kick: %s\n", time.Since(s))
			doSomething(node)
		}(node)
		wg.Add(1)
	}
	wg.Wait()
	fmt.Printf("end: %s\n", time.Since(s))
}

func IterDFS(root *Node) iter.Seq[*Node] {
	stack := []*Node{root}
	visited := make(map[*Node]bool)

	return func(yield func(*Node) bool) {
		if root == nil {
			return
		}

		for len(stack) > 0 {
			// スタックの最後の要素を取得して削除
			node := stack[len(stack)-1]
			stack = stack[:len(stack)-1]

			if visited[node] {
				continue
			}

			time.Sleep(searchTime)

			visited[node] = true
			if !yield(node) {
				return
			}

			// 子ノードをスタックに追加（逆順で追加することで左から処理）
			for i := len(node.Children) - 1; i >= 0; i-- {
				stack = append(stack, node.Children[i])
			}
		}
	}
}

func IterMain() {
	s := time.Now()
	var wg sync.WaitGroup

	root := &Node{Value: 1}
	child1 := &Node{Value: 2}
	child2 := &Node{Value: 3}
	child3 := &Node{Value: 4}
	child4 := &Node{Value: 5}

	root.Children = []*Node{child1, child2}
	child1.Children = []*Node{child3, child4}

	for node := range IterDFS(root) {
		go func(node *Node) {
			defer wg.Done()
			fmt.Printf("execute kick: %s\n", time.Since(s))
			doSomething(node)
		}(node)
		wg.Add(1)
	}
	wg.Wait()
	fmt.Printf("end: %s\n", time.Since(s))
}

func main() {
	SliceMain()
	fmt.Println("-----")
	IterMain()
}
