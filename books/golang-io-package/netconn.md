---
title: "ネットワーク"
---
# はじめに
この章ではネットワークについて扱います。
「ネットワークにI/Oがなんの関係があるの？」と思う方もいるかもしれませんが、「サーバーからデータを受け取る」「クライアントからデータを送る」というのは、言い換えると「コネクションからデータを読み取る・書き込む」ともいえるのです。

`net`パッケージのドキュメントには以下のように記載されています。
> Package net provides a portable interface for **network I/O**, including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
> (訳)`net`パッケージでは、TCP/IP, UDP, DNS, UNIXドメインソケットを含むネットワークI/Oのインターフェース(移植性あり)を提供します。
> 出典:[pkg.go.dev - net package](https://pkg.go.dev/net)

ネットワークをI/Oと捉える言葉が明示されているのがわかります。

ここからは、TCP通信で短い文字列を送る・受け取るためのGoのコードについて解説していきます。

# ネットワークコネクション
ネットワーク通信においては、「クライアント-サーバー」間を繋ぐコネクションが形成されます。
このコネクションパイプをGoで扱うインターフェースが`net.Conn`インターフェースです。
```go
type Conn interface {
    Read(b []byte) (n int, err error)
    Write(b []byte) (n int, err error)
    Close() error
    LocalAddr() Addr
    RemoteAddr() Addr
    SetDeadline(t time.Time) error
    SetReadDeadline(t time.Time) error
    SetWriteDeadline(t time.Time) error
}
```
出典:[pkg.go.dev - net#Conn](https://golang.org/pkg/net/#Conn)

`net.Conn`インターフェースは8つのメソッドセットで構成されており、これを満たす構造体としては`net`パッケージの中だけでも`net.IPConn`, `net.TCPConn`, `net.UDPConn`, `net.UnixConn`があります。

# コネクションを取得
## サーバー側から取得する
サーバー側から`net.Conn`インターフェースを取得するためには、以下のような手順を踏みます。

1. `net.Listen(通信プロトコル, アドレス)`関数から`net.Listener`型の変数(`ln`)を得る
2. `ln`の`Accept()`メソッドを実行する

```go
ln, err := net.Listen("tcp", ":8080")
if err != nil {
    fmt.Println("cannot listen", err)
}
conn, err := ln.Accept()
if err != nil {
    fmt.Println("cannot accept", err)
}
```
`conn`が`net.Conn`インターフェースの変数で、今回の場合、その実体はTCP通信のために使う`net.TCPConn`型構造体です。
![](https://storage.googleapis.com/zenn-user-upload/o27hayivyrxb3f1sice2v688pifa =250x)

## クライアント側から取得する
クライアント側から`net.Conn`インターフェースを取得するためには、`net.Dial(通信プロトコル, アドレス)`関数を実行します。

```go
conn, err := net.Dial("tcp", "localhost:8080")
if err != nil {
    fmt.Println("error: ", err)
}
```
この`conn`も実体は`net.TCPConn`型です。
![](https://storage.googleapis.com/zenn-user-upload/qpd98ckv0y2foe9uc312hzx9y591 =280x)


# サーバー側からのデータ発信
サーバー側から、TCPコネクションを使って文字列`"Hello, net pkg!"`を一回送信する処理は、`net.TCPConn`の`Write`メソッドを利用して以下のように実装されます。
```go
// コネクションを得る
ln, err := net.Listen("tcp", ":8080")
if err != nil {
    fmt.Println("cannot listen", err)
}
conn, err := ln.Accept()
if err != nil {
    fmt.Println("cannot accept", err)
}

// ここから送信

str := "Hello, net pkg!"
data := []byte(str)
_, err = conn.Write(data)
if err != nil {
    fmt.Println("cannot write", err)
}
```
`Write`メソッドの挙動は、`os.File`型の`Write`メソッドのものとそう変わりません。
引数にとった`[]byte`列の内容をコネクションに書き込み、そして何byte書き込めたかの値が第一返り値になります。

# クライアント側がデータ受信
クライアントがTCPコネクションから、文字列データを受け取るコードを`net.TCPConn`の`Read`メソッドを利用して書きます。
```go
// コネクションを得る
conn, err := net.Dial("tcp", "localhost:8080")
if err != nil {
    fmt.Println("error: ", err)
}

// ここから読み取り

data := make([]byte, 1024)
count, _ := conn.Read(data)
fmt.Println(string(data[:count]))

// 出力結果
// Hello, net pkg!
```
`Read`メソッドの挙動も`os.File`の`Read`メソッドと同じです。
引数にとった`[]byte`列の中に、コネクションから読み取った内容を入れて、そして何byte読めたかの値が第一返り値になります。


# 低レイヤで何が起きているのか
ここからは、`os.File`型のときにやったのと同様のコードリーディングを行います。
ネットワークまわりのI/Oの実装では、どのようなシステムコールにつながっているのでしょうか。低レイヤの話に深く潜り込んでいきます。

## ネットワークコネクション(net.TCPConnの正体)
`net.TCPConn`構造体の正体は、非公開の構造体`net.conn`型です。
```go
type TCPConn struct {
	conn
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/net/tcpsock.go#85]

そしてこの`net.conn`型の中身は、`netFD`型構造体そのものです。
```go
type conn struct {
	fd *netFD
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/net/net.go#170]

この`netFD`型は一体何なのでしょうか。これも定義を見てみましょう。
```go
type netFD struct {
	pfd poll.FD
	// immutable until Close
	family      int
	sotype      int
	isConnected bool // handshake completed or use of association with peer
	net         string
	laddr       Addr
	raddr       Addr
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/net/fd_posix.go#17]

前章で出てきた`poll.FD`型の`pfd`フィールドがここでも登場しました。これは一体どういうことでしょうか。

実はLinuxの設計思想として **"everything-is-a-file philosophy"** というものがあります。これは、キーボードからの入力も、プリンターへの出力も、ハードディスクやネットワークからのI/Oもありとあらゆるものを全て「**OSのファイルシステム上にあるファイルへのI/Oとして捉える**」という思想です。
今回のようなネットワークからのデータ読み取り・書き込みも、OS内部的には通常のファイルI/Oと変わらないのです。そのため、ネットワークコネクションに対しても、通常ファイルと同様にfdが与えられるのです。
![](https://storage.googleapis.com/zenn-user-upload/av0ff3br57ap2iygje9p6qf1hnzw)

## コネクションオープン
では、通信するネットワークに対応するfdはどのように決まるのでしょうか。
また、コネクションに対応したfdが入った`net.Conn`(ここでは`net.TCPConn`型構造体)はどのようにして得られるのでしょうか。

これを理解するためには、

- クライアント側で`net.Dial()`を実行
- サーバー側で`net.Listen()`→`ln.Accept()`を実行

それぞれにおいて裏で何が起きているのか、コードを読んで深掘りしていきましょう。

### クライアント側からのコネクションオープン
まずは、クライアント側から`net.Conn`を得るために呼ぶ`net.Dial(通信プロトコル, アドレス)`の中身をみてみます。
すると、今私たちが欲しい「コネクションに割り当てられたfdをもつ`net.TCPConn`」を作っているのは、実質`net.Dialer`型の`DialContext`メソッドであることがわかります。

```go
func Dial(network, address string) (Conn, error) {
	var d Dialer
	return d.Dial(network, address)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/net/dial.go#317]
```go
func (d *Dialer) Dial(network, address string) (Conn, error) {
	return d.DialContext(context.Background(), network, address) // net.TCPConnを作っているのはここ
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/net/dial.go#347]

`net.Dialer`型の`DialContext`メソッドは、「引数として渡されたプロトコル・URL・ポート番号に対応した`net.Conn`を作る」ためのメソッドです。
> DialContext connects to the address on the named network using the provided context.
> 出典:[pkg.go.dev - net#Dialer.DialContext](https://pkg.go.dev/net@go1.16.3#Dialer.DialContext)

この`DialContext`メソッドでやっていることは中々複雑なのですが、核としては
1. `syscall.Socket`経由でシステムコールsocket()を呼んで、URLやポート番号からfdをゲットする
2. 1で得たfdを`poll.FD`型にする
3. 2で得た`poll.FD`型の`fd`を使い`newTCPConn(fd)`を実行→これが`TCPConn`になる

という流れです。
![](https://storage.googleapis.com/zenn-user-upload/36rx6fx5dw4qd3o6lriieeji4ur2 =320x)
結局のところ、システムコールsocket()を内部で呼んで得たfdを`TCPConn`型にラップしている、ということです。

### サーバー側からのコネクションオープン
サーバー側で`net.Listen()`→`ln.Accept()`という手順を踏んだ場合は何が起こっているのでしょうか。
`net.Listen()`関数の実装を確認してみます。
```go
func Listen(network, address string) (Listener, error) {
	var lc ListenConfig
	return lc.Listen(context.Background(), network, address)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/net/dial.go#704]

`net.ListenConfig`型の`Listen`メソッドを内部で呼んでいます。
この`Listenメソッド`の中身も中々複雑ですが、核は

1. `syscall.Socket`経由でシステムコールsocket()を呼んで、URLやポート番号からfdをゲットする
2. 1で得たfdを内部フィールドに含んだ`TCPListener`型を生成し、返り値にする

となっています。
ここでも、コネクションに対応したfdを得るからくりはsocket()システムコールです。

ですがまだ`net.Listener`が得られただけで、実際に通信に使う`TCPConn`構造体がまだです。
実は、この「リスナーからコネクションを得る」ためのメソッドが`Accept()`メソッドなのです。その中身をみてみます。
```go
func (l *TCPListener) Accept() (Conn, error) {
	// (略)
	c, err := l.accept()
	// (略)
	return c, nil
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/net/tcpsock.go#257]

内部では非公開メソッド`accept()`を呼んでいました。その中身は以下のようになっています。
```go
func (ln *TCPListener) accept() (*TCPConn, error) {
	// リスナー本体からfdを取得
	fd, err := ln.fd.accept()
	// (略)

	// fdからTCPConnを作成
	tc := newTCPConn(fd)
	// (略)

	return tc, nil
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/net/tcpsock_posix.go#138]

要するに、「リスナーからコネクションを得る」=「リスナーからfdを取り出して、それを`TCPConn`にラップする」ということなのです。

![](https://storage.googleapis.com/zenn-user-upload/drawttp5tipfm1qkwadcsiyx6j5w =300x)


## Readメソッド
`net.TCPConn`型の`Read()`の中身を掘り下げます。

先述した通り、`net.TCPConn`型の実体は非公開構造体`conn`です。そのため、`conn`型の`Read`メソッドがそのまま`net.TCPConn`型の`Read`メソッドとして機能します。

:::message
`(c *TCPConn) Read`が定義されていなくても、内部フィールド構造体の`(c *conn) Read`がそのまま`TCPConn`型のメソッドとして機能する挙動のことをメソッド委譲といいます。
:::

その`conn`型の`Read`メソッドは、内部ではフィールド`fd`(`netFD`型)の`Read`メソッドを呼んでいます。
```go
func (c *conn) Read(b []byte) (int, error) {
	// (略)
	n, err := c.fd.Read(b)
	// (略)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/net/net.go#179]

`netFD`型の`Read()`メソッドの中身では、`pfd`フィールド(`poll.FD`型)の`Read`メソッドを呼んでいます。
```go
func (fd *netFD) Read(p []byte) (n int, err error) {
	n, err = fd.pfd.Read(p)
	// (略)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/net/fd_posix.go#54]

この`poll.FD`型の`Read`メソッドというのは、前章のファイルI/Oでも出てきたものです。ここから先は通常ファイルのI/Oと同じく、対応したfdのファイルの中身を読み込むためのシステムコール`syscall.Read`につながります。
"everything-is-a-file"思想の名の通り、ネットワークコネクションからのデータ読み取りも、OSの世界においてはファイルの読み取りと変わらず`read`システムコールで処理されるのです。

`net.TCPConn`型の`Read`メソッドの処理手順をまとめます。
1. `net.conn`型の`Read`メソッドを呼ぶ
2. 1の中で`net.netFD`型の`Read`メソッドを呼ぶ
3. 2の中で`poll.FD`型の`Read`メソッドを呼ぶ
4. 3の中で`syscall.Read`メソッドを呼ぶ
5. OSカーネルのシステムコールで読み込み処理

## Writeメソッド
`net.TCPConn`型の`Write()`メソッドのほうも`Read`メソッドと同様の流れで実装されています。
1. `net.conn`型の`Write`メソッドを呼ぶ
2. 1の中で`net.netFD`型の`Write`メソッドを呼ぶ
3. 2の中で`poll.FD`型の`Write`メソッドを呼ぶ
4. 3の中で`syscall.Write`メソッドを呼ぶ
5. OSカーネルのシステムコールで書き込み処理

![](https://storage.googleapis.com/zenn-user-upload/pt2qg55vm9759qmjuif9mc88mm0u)

# まとめ
前章・本章とファイル・ネットワークのI/Oについて取り上げました。
しかし、I/Oする対象こそ違えど、内部的な構造は両方とも
- fdがある(=ファイルへのI/Oと見れる)
- `Read()`メソッド、`Write()`メソッドのシグネチャが同じ
- 裏でシステムコールread()/write()を呼んでいる

等々、似ているところがあります。

次章では、これらI/Oをまとめてひっくるめて扱う抽象化の手段を紹介します。