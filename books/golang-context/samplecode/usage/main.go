package main

import (
	"usage/server"
)

// 設定
// userIDを兼ねた/pathにアクセスすることで、ユーザーページが見れる
// pathにできるのはaかb
// 3文字以上の認可トークンをつけなければ弾く

// サーバーを起動したら、標準入力にパス、トークンを入れる

func main() {
	srv := server.DefaultServer
	srv.ListenAndServe()
}
