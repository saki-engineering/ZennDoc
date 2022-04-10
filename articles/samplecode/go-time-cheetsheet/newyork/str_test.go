package main_test

import (
	"testing"
	"time"
)

func str2time(t string) time.Time {
	parsedTime, _ := time.ParseInLocation("2006-01-02T15:04:05Z07:00", t, newYork)
	return parsedTime
}

func str2unix(t string) int64 {
	time2unix := func(t time.Time) int64 {
		return t.Unix()
	}
	return time2unix(str2time(t))
}

func TestConvertStr(t *testing.T) {
	if got := str2time(strTime); got != timeTime {
		t.Errorf("str2time: got %s but want %s\n", got, timeTime)
	}
	if got := str2unix(strTime); got != unixTime {
		t.Errorf("str2unix: got %d but want %d\n", got, unixTime)
	}
}
