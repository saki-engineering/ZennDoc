package main

import "fmt"

type Node struct {
	Value    int
	Children []*Node
}

func DFS(node *Node, visited map[*Node]bool) {
	if node == nil || visited[node] {
		return
	}

	visited[node] = true
	doSomething(node)

	for _, child := range node.Children {
		DFS(child, visited)
	}
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

	visited := make(map[*Node]bool)
	DFS(root, visited)
}
