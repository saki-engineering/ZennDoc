package main_test

import "time"

var (
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T09:00:00+09:00"
	jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
)
