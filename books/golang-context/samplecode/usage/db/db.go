package db

import "context"

type DB struct{}

type Data string

var DefaultDB DB

func (db DB) Search(ctx context.Context, userID int) <-chan Data {
	result := make(chan Data)
	go func() {
		select {
		case <-RandomWait():
			result <- "datadatadatadata"
		case <-ctx.Done():
			close(result)
		}
		return
	}()
	return result
}
