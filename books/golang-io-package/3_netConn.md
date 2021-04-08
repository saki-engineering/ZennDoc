---
title: "ネットワーク"
---
# はじめに
ネットワークについては基本的には`net`パッケージで行います。
`net`パッケージのドキュメントには以下のように記載されています。
> Package net provides a portable interface for **network I/O**, including TCP/IP, UDP, domain name resolution, and Unix domain sockets.
出典: https://pkg.go.dev/net

# net.Connについて
クライアント ---- サーバー
この間のコネクション・パイプを扱うインターフェースがGoだと`net.Conn`インターフェースです。

## サーバー側からのコネクション取得
listen→Acceptをすることでコネクションを得ることができます。
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

## クライアント側からのコネクション取得
Dialをすることで得られる
```go
conn, err := net.Dial("tcp", "localhost:8080")
if err != nil {
    fmt.Println("error: ", err)
}
```

:::message
今回得られる`net.Conn`の実態は`net.TCPConn`型です。
:::


# サーバー側からの発信
サーバー側から、TCPコネクションを使って文字列`"Hello, net pkg!"`を一回送信するコードを書きます。
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
`Write`メソッドを使いました。

# クライアントが受信
TCPコネクションから、文字列を読み込むコードを書きます。
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
// Hello, net pkg!
```
`Read`メソッドを使いました。

# 低レイヤの話
## net.TCPConnの正体
`net.TCPConn`の正体は`net.conn`型です。
```go
type TCPConn struct {
	conn
}
```
そして、この`net.conn`型は`netFD`型そのものでした。
```go
type conn struct {
	fd *netFD
}
```
https://go.googlesource.com/go/+/go1.16.2/src/net/net.go

この`netFD`型の定義は以下。
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
https://go.googlesource.com/go/+/go1.16.2/src/net/fd_posix.go

つまり、netConnはファイルと同じfdである。ではポート番号とかurlとかはどこにいった？
(シスこーるsocket()の引数に渡してfdにする)

それを見るために、まずはnet.Dialで受信用のconnを得る中身をみてみる。
```go
func Dial(network, address string) (Conn, error) {
	var d Dialer
	return d.Dial(network, address)
}
```
https://pkg.go.dev/net#Dial
`Dialer`という型の`Dial()`メソッドを呼んでいます。この`Dialer`とは何か？
```go
type Dialer struct {
	Timeout time.Duration
	Deadline time.Time
	LocalAddr Addr
	DualStack bool
	FallbackDelay time.Duration
	KeepAlive time.Duration
	Resolver *Resolver
	Cancel <-chan struct{}
	Control func(network, address string, c syscall.RawConn) error
}
```
:::message
ソケットの構成要素はローカルアドレス(内部PCのプライベートIPに対応)・外部アドレス(通信先のIP)・ポート番号でも同様、の4つの構成要素がある
:::

この型のメソッド`Dialer.Dial()`をみるとこう。
```go
func (d *Dialer) Dial(network, address string) (Conn, error) {
	return d.DialContext(context.Background(), network, address)
}
```
`DialContext()`がconnの正体か。この中でやっていることは
1. `Dialer`の`d`と、引数`network, address`から`sysDialer`変数を作る。
```go
sd := &sysDialer{
		Dialer:  *d,
		network: network,
		address: address,
	}
```
2. `sysDialer`のメソッドを、`dialParallel()→dialSerial(アドレスリスト)`と呼ぶか`dialSerial(アドレスリスト)`単独で呼ぶかする
3. `dialSerial`の中で、`sysDialer`のメソッド`dialSingle(アドレス)`を呼ぶ
4. ローカルアドレスと通信先アドレスを引数に使って、`sysDialer`のメソッド`sd.dialTCP`を呼ぶ
5. `sysDialer`のメソッド`sd.doDialTCP`を呼ぶ
6. `sd.doDialTCP`の中で、`internetSocket(ctx, sd.network, laddr, raddr, syscall.SOCK_STREAM, 0, "dial", sd.Dialer.Control)`を呼んでfdを得る
7. fdから`newTCPConn(fd)`を作って返す(newTCPConnはtcpsock.goの中にある)
(これらはほとんどnet/dial.goとtcpsock_posix.goの中)

そして、`internetSocket()`の仕組みは以下。(ipsock_posix.goの中)
1. `socket(ctx, net, family, sotype, proto, ipv6only, laddr, raddr, ctrlFn)`を呼ぶ(sock_posix.goの中)
2. `socket`の中で`sysSocket(family, sotype, proto)`を呼んで、sysfdを得る(sysSocketはsock_cloexec.goの中)
    1. `socketFunc(family, sotype|syscall.SOCK_NONBLOCK|syscall.SOCK_CLOEXEC, proto)`を呼んでる
    2. そして、socketFuncはsyscall.Socketのエイリアス(hook_unix.go)
3. `newFD(s, family, sotype, net)`を呼んでfdをえようとする(fd_unix.goの中) 

結局のところ、システムコールsocket()を内部で呼んでfdをえてよしなにしているだけ。

さーばーからの発信、`net.Listen`はどうか？
```go
func Listen(network, address string) (Listener, error) {
	var lc ListenConfig
	return lc.Listen(context.Background(), network, address)
}
```
ListenCOnfigという型のListenメソッドを呼んでいる。これの中身は以下。
1. `ListenConfig`と引数`network, address`から`sysListener`変数を作る
```go
sl := &sysListener{
		ListenConfig: *lc,
		network:      network,
		address:      address,
	}
```
2. アドレスを引数に使って、`sysListener`のメソッド`sl.listenTCP(ctx, la)`を呼ぶ
    1. `sl.listenTCP(ctx, la)`(tcpsock_posix.goの中)は、内部で`internetSocket(ctx, sl.network, laddr, nil, syscall.SOCK_STREAM, 0, "listen", sl.ListenConfig.Control)`を呼んでfdを得る
    2. fdを使って`TCPListener`型(fdとListenConfigが入った構造体)を返り値にする
3. `sl.listenTCP(ctx, la)`の返り値のlistenerがそのままmain関数のリスナー

そしてここで得たリスナーをAcceptするのはどうなってる？
`TCPListener`型のメソッド`Accept()`をみてみる。
```go
func (l *TCPListener) Accept() (Conn, error) {
	// (略)
	c, err := l.accept()
	// (略)
	return c, nil
}
```
小文字のメソッド`accept()`を呼んでいる。その中身は以下。(tcpsock_posix.go)
```go
func (ln *TCPListener) accept() (*TCPConn, error) {
	fd, err := ln.fd.accept()
	if err != nil {
		return nil, err
	}
	tc := newTCPConn(fd)
	if ln.lc.KeepAlive >= 0 {
		setKeepAlive(fd, true)
		ka := ln.lc.KeepAlive
		if ln.lc.KeepAlive == 0 {
			ka = defaultTCPKeepAlive
		}
		setKeepAlivePeriod(fd, ka)
	}
	return tc, nil
}
```
要するに、listenerからfdを取得して、それを使って新規の`TCPConn`を作っているだけ。


## conn.Write()の処理
今回のconnは`net.TCPConn`型なので、これの`Write()`メソッドの中身をみてみます。
```go
func (c *conn) Write(b []byte) (int, error) {
	// (略)
	n, err := c.fd.Write(b)
	// (略)
}
```
:::message
TCPConn型にはconn型しか埋め込まれていないので、`func (c *TCPConn) Write(b []byte) (int, error)`が定義されていなくても、`func (c *conn) Write(b []byte) (int, error)`がそのままTCPConn型のWriteメソッドとして使用可能です。
(メソッド委譲)
:::

`netFD`型の`Write()`メソッドが呼ばれています。この中身は
```go
func (fd *netFD) Write(p []byte) (nn int, err error) {
	nn, err = fd.pfd.Write(p)
	// (略)
}
```
`poll.FD`型の`Write`メソッドに繋がる。ここからはファイルの話と合流

## conn.Read()の処理
`net.TCPConn`型の`Read()`の中身は
```go
func (c *conn) Read(b []byte) (int, error) {
	// (略)
	n, err := c.fd.Read(b)
	// (略)
}
```
`netFD`型の`read()`メソッドが呼ばれています。この中身は
```go
func (fd *netFD) Read(p []byte) (n int, err error) {
	n, err = fd.pfd.Read(p)
	// (略)
}
```
https://go.googlesource.com/go/+/go1.16.2/src/net/fd_posix.go
`poll.FD`型の`Read`メソッドに繋がり、ファイルの話と合流します。

# 執筆メモ
このずがいい
https://ascii.jp/elem/000/001/276/1276572/

ネットワークのソケットは、ネットワーク通信に用いるファイル・ディスクリプタ（file descriptor）です。