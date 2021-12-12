package main

import (
	"encoding/json"
	"fmt"
)

type GoStruct struct {
	A int    `json:"first"`
	B string `json:"second"`
}

func main() {
	fmt.Println("====== struct -> json ======")
	stcData := GoStruct{A: 1, B: "bbb"}
	encode(stcData)
}

func encode(stcData GoStruct) {
	jsonData, err := json.Marshal(stcData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s\n", jsonData)
}
