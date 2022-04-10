package main_test

import "time"

var (
	timeTime time.Time = time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T00:00:00Z"
	jsonTime string    = `{"timestamp":"2022-04-01T00:00:00Z"}`
)
