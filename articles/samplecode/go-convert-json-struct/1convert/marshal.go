package main

import (
	"encoding/json"
	"fmt"
)

type GoStruct struct {
	A int
	B string
}

func main() {
	stcData := GoStruct{A: 1, B: "bbb"}

	jsonData, err := json.Marshal(stcData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s\n", jsonData)
}
