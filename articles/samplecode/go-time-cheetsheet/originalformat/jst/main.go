package main

import (
	"encoding/json"
	"fmt"
	"time"
)

type MyDate struct {
	// ここを構造体にしないと、メソッドの移譲ができない
	Timestamp time.Time
}

func (d MyDate) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Timestamp.Format("2006/01/02 15:04:05.000 -0700"))), nil
}

func (d *MyDate) UnmarshalJSON(data []byte) error {
	t, err := time.Parse(`"2006/01/02 15:04:05.000 -0700"`, string(data))
	if err != nil {
		return err
	}
	d.Timestamp = t
	return nil
}

func time2json(t time.Time) string {
	type MyStruct struct {
		Timestamp MyDate `json:"timestamp"`
	}
	// b, _ := json.Marshal(MyStruct{MyDate{t}})
	b, _ := json.Marshal(MyStruct{MyDate{t}})
	return string(b)
}

func json2time(t string) time.Time {
	type MyStruct struct {
		Timestamp MyDate `json:"timestamp"`
	}
	var myStc MyStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp.Timestamp
}

func main() {
	var timeTime = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	fmt.Println(time2json(timeTime))

	var jsonTime string = `{"timestamp":"2022/04/01 09:00:00.000 +0900"}`
	fmt.Println(json2time(jsonTime))
}
