package handlers

import (
	"context"
	"errors"
	"fmt"
	"time"
	"usage/auth"
	"usage/db"
)

type MyHandleFunc func(context.Context, MyRequest)

var GetGreeting MyHandleFunc = func(ctx context.Context, req MyRequest) {
	var res MyResponse

	// doSomething()
	// トークンからユーザー検証→ダメなら即return
	userID, err := auth.VerifyAuthToken(ctx)
	if err != nil {
		res = MyResponse{Code: 403, Err: err}
		fmt.Println(res)
		return
	}

	dbReqCtx, cancel := context.WithTimeout(ctx, 2*time.Second)

	//data, _ := db.DefaultDB.Search(ctx, userID)
	rcvChan := db.DefaultDB.Search(dbReqCtx, userID)
	data, ok := <-rcvChan
	cancel()

	// DBリクエストがタイムアウトしていたら408で返す
	if !ok {
		res = MyResponse{Code: 408, Err: errors.New("DB request timeout")}
		fmt.Println(res)
		return
	}

	// レスポンスの作成
	res = MyResponse{
		Code: 200,
		Body: fmt.Sprintf("From path %s, Hello! your ID is %d\ndata → %s", req.path, userID, data),
	}

	// レスポンス内容をconnに書き込み
	fmt.Println(res)
}

var NotFoundHandler MyHandleFunc = func(ctx context.Context, req MyRequest) {
	res := MyResponse{Code: 404, Err: errors.New("not found")}
	fmt.Println(res)
}
