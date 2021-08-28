package db

import (
	"math/rand"
	"time"
)

func RandomWait() <-chan struct{} {
	done := make(chan struct{})
	go func() {
		time.Sleep(time.Duration(rand.Intn(5000)) * time.Millisecond)
		close(done)
	}()
	return done
}
