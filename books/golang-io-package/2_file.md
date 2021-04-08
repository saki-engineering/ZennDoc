---
title: "fileの読み書き"
---
# はじめに
ファイルの読み書きは基本的には`os`パッケージで行います。

# ファイルオブジェクト
`os`パッケージには`os.File`型が存在し、Goでファイルを扱うときはこれが元となります。

## 読み込みonlyでopen
ファイルを開くときは`os.Open`関数を用います。
```go
f, err := os.Open("text.txt")
```
`os.Open`関数で得られる第一返り値`f`がファイルオブジェクトです。

ドキュメントの記載は以下。
> Open opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY. If there is an error, it will be of type *PathError.
> 出典:https://pkg.go.dev/os#Open

## 書き込み権限付きでopen
I/Oで使いたくて書き込み権限が欲しいなら`os.Create`関数を使います。すでに`write.txt`がある状態でも使えます。
```go
f, err := os.Create("write.txt")
```
これも第一返り値`f`がファイルオブジェクトです。

ドキュメントに記載されている`os.Create`の説明は以下のようになっています。
> Create creates or truncates the named file. If the file already exists, it is truncated. If the file does not exist, it is created with mode 0666 (before umask). If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
> 出典:https://pkg.go.dev/os#Create

:::message
O_RDWRは読み書きOKというファイルオープン時のLinuxにおけるflagらしい
:::

## close
ファイルを閉じるときはファイルオブジェクト`f`の`Close`メソッドを用います。
以下のコードのように、`Close`メソッドは`defer`で読んでおくのが賢いでしょう。
```go
f, err := os.Open("text.txt")
if err != nil {
    fmt.Println("cannot open the file")
}
defer f.Close()
```

# ファイル内容の読み込み
同じディレクトリ中にある`text.txt`の内容をすべて読み込むという操作を考えます。

```
Hello, world!
Hello, Golang!
```


この場合は、`os.File`型の`Read`メソッドを用いて以下のようにします。

```go
// os.FileオブジェクトをOpen関数か何かで事前に得ておくとする
// 変数fがFileオブジェクトとする

data := make([]byte, 1024)
count, err := f.Read(data)
if err != nil {
    fmt.Println(err)
    fmt.Println("fail to read file")
}

/*
--------------------------------
挙動の確認
--------------------------------
*/

fmt.Printf("read %d bytes: %q\n", count, data[:count])
fmt.Println(string(data[:count]))

/*
出力結果

read 28 bytes: "Hello, world!\nHello, Golang!"
Hello, world!
Hello, Golang!
*/
```
`f.Read([]byte)`の引数としてとる`[]byte`スライスの中に、読み込まれたファイルの内容が格納されます。

# 書き込み
ファイルに何かを書き込むときは、`f.Write`メソッドを利用します。
```go
// os.Createでファイルオブジェクトを得ておくとする
// os.Openではダメ。

f, err := os.Create("write.txt")
if err != nil {
    fmt.Println("cannot open the file")
}
defer f.Close()

str := "write this file by Golang!"
data := []byte(str)
count, err := f.Write(data)
if err != nil {
    fmt.Println(err)
    fmt.Println("fail to write file")
}
fmt.Printf("write %d bytes\n", count)
/*
出力結果
write 26 bytes
*/
```
`Write`メソッドの引数としてとる`[]byte`スライスに格納されている内容が、ファイルにそのまま書き込まれることになります。

:::message
`os.Open`関数で得た読み込み専用のファイルオブジェクトにWriteを使用すると、以下のようなエラーが出ます。
`write write.txt: bad file descriptor`
:::

# 低レイヤで何が起きているのか
## ファイルオブジェクト
`os.File`型の中身は以下のようになっています。
```go
type file struct {
	pfd         poll.FD
	name        string
	dirinfo     *dirInfo // nil unless directory being read
	nonblock    bool     // whether we set nonblocking mode
	stdoutOrErr bool     // whether this is stdout or stderr
	appendMode  bool     // whether file is opened for appending
}
```
重要なのは`pfd`フィールドです。
Linuxカーネルプロセス内部では、openしたファイル1つに対して非負整数1つを対応付けて管理しており、この非負整数のことをfd(ファイルディスクリプタ)と呼んでいます。
FD型のGoDoc説明にもそう明記されています。

> FD is a file descriptor. The net and os packages use this type as a field of a larger type representing a network connection or OS file.
> (出典) https://pkg.go.dev/internal/poll#FD

:::message
オープンしたファイルについてつけているのがfdで、オープンしていなくてもカーネルでは全ファイルを整数番号で把握している。これをinode番号という。
どちらかというと、fdはファイルそのものの番号というより、読み書きするインターフェースにつけられた番号といった方がいいかも？

これは、同じファイルを開いたらいつも同じfd番号になるとかいうものではない。
:::

`os.Open()`の中身をこれからみていきます。
```go
func Open(name string) (*File, error) {
	return OpenFile(name, O_RDONLY, 0)
}
```
ちなみに`os.Create()`も引数が違うだけで`OpenFile`を呼んでいるので同じです。
`os.OpenFile()`をReadOnlyのフラグを立ててやってます。`OpenFile`を見ると
```go
func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	// (略)
	f, err := openFileNolog(name, flag, perm)
	// (略)
}
```
`openFileNoLog`を見ると`syscall.Open(引数略)`が呼ばれています。
https://go.googlesource.com/go/+/go1.16.2/src/os/file_unix.go
ここでシステムコールopen()を呼んでいるのでカーネル接続

openをみたのでcloseをみてみましょう。
`f.Close()`メソッドは
```go
func (f *File) Close() error {
	// (略)
	return f.file.close()
}
```
内部で`file.close()`が呼ばれていて、これは
```go
func (file *file) close() error {
	// (略)
	if e := file.pfd.Close(); e != nil {
		// (略)
	}
	// (略)
}
```
内部で`pfd.Close()`が呼ばれている。これは
```go
func (fd *FD) Close() error {
	if !fd.fdmu.increfAndClose() {
		return errClosing(fd.isFile)
	}
	// Unblock any I/O.  Once it all unblocks and returns,
	// so that it cannot be referring to fd.sysfd anymore,
	// the final decref will close fd.sysfd. This should happen
	// fairly quickly, since all the I/O is non-blocking, and any
	// attempts to block in the pollDesc will return errClosing(fd.isFile).
	fd.pd.evict()
	// The call to decref will call destroy if there are no other
	// references.
	err := fd.decref()
	// Wait until the descriptor is closed. If this was the only
	// reference, it is already closed. Only wait if the file has
	// not been set to blocking mode, as otherwise any current I/O
	// may be blocking, and that would block the Close.
	// No need for an atomic read of isBlocking, increfAndClose means
	// we have exclusive access to fd.
	if fd.isBlocking == 0 {
		runtime_Semacquire(&fd.csema)
	}
	return err
}
```
https://go.googlesource.com/go/+/go1.16.2/src/internal/poll/fd_unix.go#92
ここから先がsyscall.Close()にどう繋がるのか？？？
close-on-execフラグが関連しているらしい、syscall.CloseOnExec()は使っているから
これによると、exec()したら自動クローズっぽい
Linux的にはO_CLOEXEC

## Readメソッド
まずは`os.File`の`Read`メソッドが根本的に何をやっているのか深く掘り下げてみましょう。

`Read`メソッドは、`os.File`型のフィールドの一つ`pfd`のReadメソッドを呼んでいます。
```go
// Read()メソッド
func (f *File) Read(b []byte) (n int, err error) {
	// (中略)
	n, e := f.read(b)  // ここで読み込み
	return n, f.wrapErr("read", e)
}
```
```go
// f.read(b)の中身
func (f *File) read(b []byte) (n int, err error) {
    n, err = f.pfd.Read(b) // ここで読み込み
    // (中略)
    return n, err
}
```

ここで出てきた`pfd`フィールドは、`internal/poll`パッケージの`FD`型です。(これを略してpfdというフィールド名にしている)
```go
type file struct {
	pfd         poll.FD
	// (略)
}
```

この`os.File`型の`Read`メソッドは、内部的には`poll.FD`型の`Read()`メソッドを呼び出していることになります。
この`poll.FD`型の`Read()`メソッドは以下のような処理をしています。

1. readLockする()
2. 読んだものを保存する先の`p []byte`が長さ0じゃないかを確認
3. ファイルを読み込み専用で開く(fd.pd.prepareReadは`fd_poll_runtime.go`にあって、そこのpd.prepareも同じファイルに実装、)
4. ストリームだったら(UDPみたいなパケットベースな受信とは対極の)、2^30個のbyteだけ受け取るように制限する
5. システムコールを呼び出してファイルを読む
出典:https://go.googlesource.com/go/+/go1.16.2/src/internal/poll/fd_unix.go#142

この5番が重要で、この処理は`ignoringEINTRIO(syscall.Read, fd.Sysfd, p)`と書かれています。この`syscall.Read`という関数が、macのread()システムコールを呼ぶ処理をしていて、ここでGoのカーネル低レイヤが繋がる瞬間です。

:::message
EINTRは、処理中に割り込み信号(ユーザーによるCtrl+Cなど)がはいったというエラー番号のこと。
:::

## Writeメソッド
`os.File`型の`Write()`メソッドのほうも見てみましょう。

```go
func (f *File) Write(b []byte) (n int, err error) {
	// (略)
	n, e := f.write(b)
    // (略)
}
```
これも、`f.write()`メソッドが呼び出され
```go
func (f *File) write(b []byte) (n int, err error) {
	n, err = f.pfd.Write(b)
	// (略)
}
```
この中で`f.pfd.Write(b)`メソッドが呼び出され、その中で`ignoringEINTRIO(syscall.Write, fd.Sysfd, p[nn:max])`となりシステムコールwrite()を呼んでいる。


# 執筆メモ
- ファイルってなんで閉じなきゃいけないの？
- システムコール、低レイヤでは何が起きてるの？