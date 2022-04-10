package main_test

import (
	"encoding/json"
	"testing"
	"time"
)

func json2time(t string) time.Time {
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp
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
	if got := json2unix(jsonTime); got != unixTime {
		t.Errorf("json2unix: got %d but want %d\n", got, unixTime)
	}
}
