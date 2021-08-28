package session

import "context"

type ctxKey int

const (
	sessionID ctxKey = iota
)

var sequence int = 1

func SetSessionID(ctx context.Context) context.Context {
	idCtx := context.WithValue(ctx, sessionID, sequence)
	sequence += 1
	return idCtx
}

func GetSessionID(ctx context.Context) int {
	id := ctx.Value(sessionID).(int)
	return id
}
