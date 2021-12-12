package main

import (
	"encoding/json"
	"fmt"
)

type GoStruct struct {
	A int  `json:"a"`
	B int  `json:"b"`
	C *int `json:"c"`
}

func main() {
	jsonString1 := `{"a":1,"b":2}`
	decode(jsonString1)

	jsonString2 := `{"a":1,"b":2,"c":0}`
	decode(jsonString2)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v ", stcData)
	if stcData.C != nil {
		fmt.Printf("C:%d", *stcData.C)
	}
	fmt.Printf("\n")
}
