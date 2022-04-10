package main_test

import (
	"encoding/json"
	"testing"
	"time"
)

type MyDate struct {
	Timestamp time.Time
}

func (d *MyDate) UnmarshalJSON(data []byte) error {
	t, err := time.ParseInLocation(`"2006-01-02T15:04:05Z07:00"`, string(data), newYork)
	if err != nil {
		return err
	}
	d.Timestamp = t
	return nil
}

func json2time(t string) time.Time {
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp.In(newYork)
}

func json2timeVer2(t string) time.Time {
	type myStruct struct {
		Timestamp MyDate `json:"timestamp"`
	}
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp.Timestamp
}

func json2unix(t string) int64 {
	time2unix := func(t time.Time) int64 {
		return t.Unix()
	}
	return time2unix(json2time(t))
}

func TestConvertJSON(t *testing.T) {
	if got := json2time(jsonTime); got != timeTime {
		t.Errorf("json2time: got %s but want %s\n", got, timeTime)
	}
	if got := json2timeVer2(jsonTime); got != timeTime {
		t.Errorf("json2timeVer2: got %s but want %s\n", got, timeTime)
	}
	if got := json2unix(jsonTime); got != unixTime {
		t.Errorf("json2unix: got %d but want %d\n", got, unixTime)
	}
}
