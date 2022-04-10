package main_test

import "time"

var (
	newYork *time.Location = func() *time.Location {
		location, _ := time.LoadLocation("America/New_York")
		return location
	}()
	timeTime time.Time = time.Date(2022, 3, 31, 20, 0, 0, 0, newYork)
	unixTime int64     = 1648771200
	strTime  string    = "2022-03-31T20:00:00-04:00"
	jsonTime string    = `{"timestamp":"2022-03-31T20:00:00-04:00"}`
)
