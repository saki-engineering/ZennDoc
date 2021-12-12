package main

import (
	"encoding/json"
	"fmt"
)

type GoStruct struct {
	A int
	B string
	c int
	d string
}

func main() {
	fmt.Println("====== struct -> json ======")
	stcData := GoStruct{A: 1, B: "bbb", c: 2, d: "ddd"}
	encode(stcData)

	fmt.Println("====== json -> struct ======")
	jsonString := `{"A":3, "B":"bbbbbb", "c": 4, "d": "dddddd"}`
	decode(jsonString)
}

func encode(stcData GoStruct) {
	jsonData, err := json.Marshal(stcData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s\n", jsonData)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
