package main_test

import (
	"encoding/json"
	"testing"
	"time"
)

func time2unix(t time.Time) int64 {
	return t.Unix()
}

func time2str(t time.Time) string {
	return t.Format("2006-01-02T15:04:05Z07:00")
}

func time2json(t time.Time) string {
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}
	b, _ := json.Marshal(myStruct{timeTime})

	return string(b)
}

func TestConvertTime(t *testing.T) {
	if got := time2unix(timeTime); got != unixTime {
		t.Errorf("time2unix: got %d but want %d\n", got, unixTime)
	}
	if got := time2str(timeTime); got != strTime {
		t.Errorf("time2str: got %s but want %s\n", got, strTime)
	}
	if got := time2json(timeTime); got != jsonTime {
		t.Errorf("time2json: got %s but want %s\n", got, jsonTime)
	}
}
