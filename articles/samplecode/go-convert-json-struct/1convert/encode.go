package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type GoStruct struct {
	A int
	B string
}

func main() {
	stcData := GoStruct{A: 1, B: "bbb"}

	err := json.NewEncoder(os.Stdout).Encode(stcData)
	if err != nil {
		fmt.Println(err)
	}
}
