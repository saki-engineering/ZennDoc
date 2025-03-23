package main

import "fmt"

type Node struct {
	Value    int
	Children []*Node
}

func DFS(root *Node) []*Node {
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

		visited[node] = true
		result = append(result, node)

		// 子ノードをスタックに追加（逆順で追加することで左から処理）
		for i := len(node.Children) - 1; i >= 0; i-- {
			stack = append(stack, node.Children[i])
		}
	}
	return result
}

func doSomething(node *Node) {
	fmt.Println(node.Value)
}

func main() {
	// Create a sample tree
	root := &Node{Value: 1}
	child1 := &Node{Value: 2}
	child2 := &Node{Value: 3}
	child3 := &Node{Value: 4}
	child4 := &Node{Value: 5}

	root.Children = []*Node{child1, child2}
	child1.Children = []*Node{child3, child4}

	for _, s := range DFS(root) {
		doSomething(s)
	}
}
