package main_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

var (
	timeTime time.Time = time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)
	// strTime  string    = "2022-04-01T09:00:00+09:00"
	// jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
	strTime  string = "2022/04/01 00:00:00.000 +0000"
	jsonTime string = `{"timestamp":"2022/04/01 00:00:00.000 +0000"}`
)

type MyDate struct {
	// ここを構造体にしないと、メソッドの移譲ができない
	Timestamp time.Time
}

func (d MyDate) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Timestamp.Format("2006/01/02 15:04:05.000 -0700"))), nil
}

func (d *MyDate) UnmarshalJSON(data []byte) error {
	t, err := time.ParseInLocation(`"2006/01/02 15:04:05.000 -0700"`, string(data), time.UTC)
	if err != nil {
		return err
	}
	d.Timestamp = t
	return nil
}

func time2str(t time.Time) string {
	// 特に変わったことはなし
	return t.Format("2006/01/02 15:04:05.000 -0700")
}

func str2time(t string) time.Time {
	// 特に変わったことはなし
	parsedTime, _ := time.ParseInLocation("2006/01/02 15:04:05.000 -0700", t, time.UTC)
	return parsedTime
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

func TestTime2Str(t *testing.T) {
	if got := time2str(timeTime); got != strTime {
		t.Errorf("got %s but want %s", got, strTime)
	}
}

func TestStr2Time(t *testing.T) {
	if got := str2time(strTime); got != timeTime {
		t.Errorf("got %s but want %s", got, timeTime)
	}
}

func TestTime2JSON(t *testing.T) {
	if got := time2json(timeTime); got != jsonTime {
		t.Errorf("got %s but want %s", got, jsonTime)
	}
}

func TestJSON2Time(t *testing.T) {
	if got := json2time(jsonTime); got != timeTime {
		t.Errorf("got %s but want %s", got, timeTime)
	}
}
