package main

import (
	"encoding/json"
	"fmt"
)

func main() {
	jsonString := `{"a":1,"b":2,"d":4}`

	var Data interface{}
	if err := json.Unmarshal([]byte(jsonString), &Data); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%T, %+v\n", Data, Data)
}
