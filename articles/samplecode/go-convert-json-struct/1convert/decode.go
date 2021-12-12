package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type GoStruct struct {
	A int
	B string
}

func main() {
	var stcData GoStruct
	jsonString := `{"A":1, "B":"bbb"}`

	if err := json.NewDecoder(strings.NewReader(jsonString)).Decode(&stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
