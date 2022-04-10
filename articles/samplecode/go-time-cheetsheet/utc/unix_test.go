package main_test

import (
	"encoding/json"
	"testing"
	"time"
)

func unix2time(t int64) time.Time {
	return time.Unix(t, 0).In(time.UTC)
}

func unix2str(t int64) string {
	time2str := func(t time.Time) string {
		return t.Format("2006-01-02T15:04:05Z07:00")
	}
	return time2str(unix2time(t))
}

func unix2json(t int64) string {
	time2json := func(t time.Time) string {
		type myStruct struct {
			Timestamp time.Time `json:"timestamp"`
		}
		b, _ := json.Marshal(myStruct{t})

		return string(b)
	}
	return time2json(unix2time(t))
}

func TestConvertUnix(t *testing.T) {
	if got := unix2time(unixTime); got != timeTime {
		t.Errorf("unix2time: got %s but want %s\n", got, timeTime)
	}
	if got := unix2str(unixTime); got != strTime {
		t.Errorf("unix2str: got %s but want %s\n", got, strTime)
	}
	if got := unix2json(unixTime); got != jsonTime {
		t.Errorf("unix2json: got %s but want %s\n", got, jsonTime)
	}
}
