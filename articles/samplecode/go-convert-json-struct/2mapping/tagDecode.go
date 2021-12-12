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
	fmt.Println("====== json -> struct (field) ======")
	jsonString1 := `{"A":3, "B":"bbbbbb"}`
	decode(jsonString1)

	fmt.Println("====== json -> struct (tag, lower) ======")
	jsonString2 := `{"first":4, "second":"bbb"}`
	decode(jsonString2)

	fmt.Println("====== json -> struct (tag, upper) ======")
	jsonString3 := `{"First":5, "Second":"b"}`
	decode(jsonString3)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
