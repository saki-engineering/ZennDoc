package main_test

import (
	"testing"
	"time"
)

var (
	newYork  *time.Location
	timeTime time.Time
	unixTime int64  = 1648771200
	strTime  string = "2022-03-31T20:00:00-04:00"
	jsonTime string = `{"timestamp":"2022-03-31T20:00:00-04:00"}`
)

func TestMain(m *testing.M) {
	location, _ := time.LoadLocation("America/New_York")
	newYork = location

	timeTime = time.Date(2022, 3, 31, 20, 0, 0, 0, newYork)

	m.Run()
}
