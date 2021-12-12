package main

import (
	"encoding/json"
	"fmt"
)

type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"-"`
	D int `json:"d"`
}

func main() {
	jsonString := `{"a":1,"b":2,"c":3}`
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
