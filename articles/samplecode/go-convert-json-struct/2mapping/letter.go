package main

import (
	"encoding/json"
	"fmt"
)

type GoStruct struct {
	A    int
	B    string
	Cccc int
	DDdD int
}

func main() {
	fmt.Println("====== json -> struct ======")
	jsonString := `{"A":3, "b":"bbbbbb", "cccc": 4, "ddDd": 5}`
	decode(jsonString)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
