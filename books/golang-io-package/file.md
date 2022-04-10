---
title: "ファイルの読み書き"
---
# はじめに
Goでファイルの読み書きに関する処理は`os`パッケージ中に存在する`File`型のメソッドで行います。

この章では
- `os.File`型って一体何者？
- 読み書きってどうやってするの？
- 低レイヤでは何が起こっているの？
ということについてまとめていきます。

# ファイルオブジェクト
`os`パッケージには`os.File`型が存在し、Goでファイルを扱うときはこれが元となります。
```go
type File struct {
	*file // os specific
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/os/types.go#16]

`os.File`型の実際の実装は`os.file`型という非公開型で行われており、その内部構造については外から直接見ることができないようになっています。

:::message
このように「公開する構造体の中身を隠したい場合に、隠す中身を非公開の構造体型にしてまとめて、公開型構造体に埋め込む」という手段はGoの標準パッケージ内ではよく見られる手法です。
:::

# ファイルを開く(open)
## 読み込み権限onlyで開く
Go言語でファイルを扱い読み書きするためには、まずはそのファイルを"open"して、`os.File`型を取得しなくてはいけません。

`os.File`型を得るためには、`os.Open(ファイルパス)`関数を使います。
```go
f, err := os.Open("text.txt")
```
得られる第一返り値`f`が`os.File`型のファイルオブジェクトです。

`os.Open()`関数について、ドキュメントでは以下のように書かれています。
> `Open` opens the named file for reading. If successful, methods on the returned file can be used for reading; the associated file descriptor has mode O_RDONLY.
> 
> (訳) `Open`関数は、名前付きのファイルを読み込み専用で開きます。`Open`が成功すれば、返り値として得たファイルオブジェクトのメソッドを中身の読み込みのために使うことができます。`Open`から得たファイルは、Linuxでいう`O_RDONLY`フラグがついた状態になっています。
> 
> 出典:[pkg.go.dev - os package](https://pkg.go.dev/os#Open)

## 書き込み権限付きで開く
書き込み権限がついた状態のファイルが欲しい場合、`os.Create(ファイルパス)`関数を使います。
```go
f, err := os.Create("write.txt")
```
`Open()`と同様に、これも第一返り値`f`が`os.File`型のファイルオブジェクトです。

"create"の名前を見ると「ファイルがない状態からの新規作成にしか対応してないのか？」と思う方もいるでしょうが、引数のファイルパスには既に存在しているファイルの名前も指定することができます。今回の場合、`write.txt`が既に存在してもしなくても、上のコードは正しく動作します。

ドキュメントに記載されている`os.Create()`の説明は以下のようになっています。
> Create creates or truncates the named file. If the file already exists, it is truncated. If the file does not exist, it is created with mode 0666 (before umask). If successful, methods on the returned File can be used for I/O; the associated file descriptor has mode O_RDWR.
>
> (訳)`Create()`関数は、名前付きファイルを作成するか、中身を空にして開きます。引数として指定したファイルが既に存在している場合、中身を空にして開くほうの動作がなされます。ファイルが存在していなかった場合は、`umask 0666`のパーミッションでファイルを作成します。`Create()`が成功すれば、返り値として得たファイルオブジェクトのメソッドをI/Oのために使うことができます。`Create`から得たファイルは、Linuxでいう`O_RDWR`フラグがついた状態になっています。
>
> 出典:[pkg.go.dev - os package](https://pkg.go.dev/os#Create)

:::message
truncateは、直訳が「切り捨てる」という動詞です。Linuxの文脈では、truncateは「ファイルサイズを指定したサイズにする」という意味で使われることが多いです。これには、ファイルサイズを大きくすることも小さくすることも含まれ、例えば10byteのファイルを20byteにする処理も、訳語に反しますが"truncate"です。ファイルサイズが指定されなかった場合、ファイルサイズ0にtruncateされるととられ、今回の`Create`の場合はこちらの動作になります。
:::

# ファイル内容の読み込み(read)
同じディレクトリ中にある`text.txt`の内容をすべて読み込むという操作を考えます。

```
Hello, world!
Hello, Golang!
```


これをGoで行う場合、`os.File`型の`Read`メソッドを用いて以下のように実装できます。

```go
// os.FileオブジェクトをOpen関数か何かで事前に得ておくとする
// 変数fがファイルオブジェクトとする

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

fmt.Printf("read %d bytes:\n", count)
fmt.Println(string(data[:count]))

/*
出力結果

read 28 bytes:
Hello, world!
Hello, Golang!
*/
```
`Read(b []byte)`メソッドの引数としてとる`[]byte`スライスの中に、読み込まれたファイルの内容が格納されます。

また、`Read()`メソッドの第一返り値(上での`count`変数に値が格納)には、「`Read()`メソッドを実行した結果、何byteが読み込まれたか」が`int`型で入っています。
そのため、`string(data[:count])`とすることで、ファイルから読み込まれた文字列をそのまま得ることができます。

:::message
`fmt.Println(string(data[:count]))`
↓
`fmt.Println(data[:count])`
のようにprintする内容を変更すると、「文字列」ではなくて「文字列にエンコードする前のバイト列そのまま」が得られるので注意。
(例)
文字列→`"Hello, world!\nHello, Golang!"`
エンコード前→バイト列`[72 101 108 108 111 44 32 119 111 114 108 100 33 10 72 101 108 108 111 44 32 71 111 108 97 110 103 33]`
:::

# ファイルへの書き込み(write)
ファイルに何かを書き込むときは、`os.File`型の`Write()`メソッドを利用します。

実際に`write.txt`というテキストファイルに文字列を書き込むコードを実装してみます。
```go
// fはos.Create()で得たファイルオブジェクトとします。

str := "write this file by Golang!"
data := []byte(str)
count, err := f.Write(data)
if err != nil {
    fmt.Println(err)
    fmt.Println("fail to write file")
}

/*
--------------------------------
挙動の確認
--------------------------------
*/
fmt.Printf("write %d bytes\n", count)
/*
出力結果
write 26 bytes
*/
```
```
write this file by Golang!
```
`Write`メソッドの引数としてとる`[]byte`スライス(ここでは変数`data`)に格納されている内容が、ファイルにそのまま書き込まれることになります。
ここでは引数に「文字列`write this file by Golang!`をバイト列にキャストしたもの」を使っているので、この文字列がそのまま`write.txt`に書き込まれます。

また、`Write`メソッドの第一返り値には、「メソッド実行の結果ファイルに何byte書き込まれたか」が`int`型で得られます。

:::message
`Write`メソッドを使う予定のファイルオブジェクトは、書き込み権限がついた`os.Create()`から作ったものでなくてはなりません。
`os.Open()`で開いたファイルは読み込み専用なので、これに`Write`メソッドを使うと、以下のようなエラーが出ます。
`write write.txt: bad file descriptor`
:::

# ファイルを閉じる(close)
## 基本
ファイルを閉じるときは`os.File`型`Close`メソッドを用います。
```go
f, err := os.Open("text.txt")
if err != nil {
    fmt.Println("cannot open the file")
}
defer f.Close()

// 以下read処理等を書く
```
上のコードでは、`Close()`メソッドは`defer`を使って呼んでいます。
一般的に、ファイルというのは「開いて使わなくなったら必ず閉じるもの」なので、`Close()`は`defer`での呼び出し予約と非常に相性がいいメソッドです。

## 応用
ところで、`Close`メソッドの定義をドキュメントで見てみると、以下のようになっています。
```go
func (f *File) Close() error
```
出典:[pkg.go.dev - os#File.Close](https://pkg.go.dev/os#File.Close)
このように、実は返り値にエラーがあるのです。

ファイルを開いた後に行う操作が「読み込み」だけの場合、元のファイルはそのままですから`Close()`に失敗するということはほとんどありません。
そのため、基本の節では`Close`メソッドから返ってくるエラーをさらっと握り潰してしまいました。

しかし、開いた後に行う操作が「書き込み」のような元のファイルに影響を与えるような操作だった場合、その処理が正常終了しないと`Close`できない、という状態に陥ることがあります。
そのため、`Write`メソッドを使う場合は`Close`の返り値エラーをきちんと処理すべきです。

`defer`を使いつつエラーを扱うためには、以下のように無名関数を使います。
```diff go
f, err := os.Create("write.txt")
if err != nil {
	fmt.Println("cannot open the file")
}
- defer f.Close()
+ defer func(){
+	err := f.Close()
+	if err != nil {
+		fmt.Println(err)
+	}
+ }()

// 以下write処理等を書く
```

# 低レイヤで何が起きているのか
Goのコード上で`os.Open()`だったり`f.Read()`だったりを「おまじない」のように唱えることで、実際のファイルを扱うことができるのは一体どうなっているのでしょうか。
これをよく知るためには、OSカーネルへと続く低レイヤなところに視点を下ろす必要があります。
本章では`os`パッケージのコードを深く掘り下げることでこれを探っていきます。

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
(`os.File`型の中身が、非公開の構造体`os.file`型であるのは前述した通りです)
出典:[https://go.googlesource.com/go/+/go1.16.3/src/os/file_unix.go#54]

この中で重要なのは`pfd`^[pfdはおそらくpollパッケージのFD型の略です。]フィールドです。

Linuxカーネルプロセス内部では、openしたファイル1つに非負整数識別子1つを対応付けて管理しており、この非負整数のことをfd(ファイルディスクリプタ)と呼んでいます。
`poll`パッケージの`FD`型はこのfdをGo言語上で具現化した構造体なのです。

> FD is a file descriptor. The net and os packages use this type as a field of a larger type representing a network connection or OS file.
> (訳)`FD`型はファイルディスクリプタです。`net`や`os`パッケージでは、ネットワークコネクションやファイルを表す構造体の内部フィールドとしてこの型を使用しています。
> 出典:[pkg.go.dev - internal/poll package](https://pkg.go.dev/internal/poll#FD)

`FD`型の定義は以下のようになっていて、この`Sysfd`という`int`型のフィールドがfdの数字そのものを表しています。
```go
type FD struct {
    // System file descriptor. Immutable until Close.
    Sysfd int

    // Whether this is a streaming descriptor, as opposed to a
    // packet-based descriptor like a UDP socket. Immutable.
    IsStream bool

    // Whether a zero byte read indicates EOF. This is false for a
    // message based socket connection.
    ZeroReadIsEOF bool

    // contains filtered or unexported fields
}
```
出典:[pkg.go.dev - internal/poll#FD](https://golang.org/pkg/internal/poll/#FD)

:::message
ちなみにカーネルでは、openしていない全てのファイルに対しても整数の識別子をつけて管理しており、これをinode番号といいます。
fdはそれとは区別された概念で、こちらは「プロセス中でopenしたファイルに対して順番に割り当てられる番号」です。

そのため、同じファイルを開いたらいつもfdが同じ番号になる、という代物ではありません。
あるプログラムで`read.txt`を開いたらfdが3になったけど、別のときに別のプログラムで`read.txt`を開いたらfdが4になる、ということは普通に存在します。
:::

## ファイルオープン
`os.Open()`実装の中身をこれからみていきます。

まず、`os.Open`自体は、同じ`os`パッケージの`OpenFile`関数を呼んでいるだけです。
```go
func Open(name string) (*File, error) {
	return OpenFile(name, O_RDONLY, 0)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/os/file.go#310]

:::message
ちなみに`os.Create()`も内部で`OpenFile`を呼んでいます。ただし、ファイルに書き込み権限をつけるため、関数に渡している引数が違います。

というより`OpenFile`関数そのものが「ファイルを特定の権限で開く」ための一般的な操作を規定したもので、`os.Open`や`os.Create`はこれをユーザーがよく使う引数でラップしただけ、というのが本来の位置付けです。
:::

`os.OpenFile`関数の中身を見ると、非公開関数`openFileNolog`を呼んでいるのがわかります。
```go
func OpenFile(name string, flag int, perm FileMode) (*File, error) {
	// (略)
	f, err := openFileNolog(name, flag, perm)
	// (略)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/os/file.go#329]

この`openFileNoLog`関数をみると、内部では`syscall.Open()`という`syscall`パッケージの関数が呼ばれています。
```go
func openFileNolog(name string, flag int, perm FileMode) (*File, error) {
	// (略)
	var r int
	for {
		var e error
		r, e = syscall.Open(name, flag|syscall.O_CLOEXEC, syscallMode(perm))
		if e == nil {
			break
		}
		// (略:EINTRエラーを握り潰す処理)
	}
	// (略)
	return newFile(uintptr(r), name, kindOpenFile), nil
}
```
出典:[https://go.googlesource.com/go/+/go1.16.2/src/os/file_unix.go#205]
:::message
EINTRは、処理中に割り込み信号(ユーザーによるCtrl+Cなど)があったというエラー番号のこと。
:::

`openFileNolog`関数の返り値とするために、`syscall.Open`から得られた返り値`r`をfdとする`os.File`型を生成しています。
言い換えると、「ファイルのfdを得る」という根本的な操作をしているのは`syscall.Open`関数です。

この`syscall`パッケージでは、OSカーネルへのシステムコールをGoのソースコードから呼び出すためのインターフェースを定義しています。
> Package syscall contains an interface to the low-level operating system primitives.
> 出典:[pkg.go.dev - syscall package](https://pkg.go.dev/syscall)

そしてこの`syscall.Open`関数は、OSの`open`システムコールを呼び出すためのラッパーなのです。後の処理はカーネルがやってくれます。

Linuxの場合、システムコール`open()`は、指定したパスのファイルを指定したアクセスモードで開き、返り値としてfdを返すものです。
```c
#include <sys/types.h>
#include <sys/stat.h>
#include <fcntl.h>


int open(const char *pathname, int flags);
```
この引数`flags`に入れられるフラグとして`O_RDONLY`や`O_RDWR`があり、これによってopenしたファイルが読み込み専用になったり、読み書き可能になったりします。


## Readメソッド
次に、`os.File`型の`Read`メソッドを掘り下げてみましょう。

先述した通り、`os.File`型の実体は非公開の`os.file`型です。
そしてこの`os.file`型の`Read`メソッドは、非公開メソッド`read`メソッドを経由して、その構造体のフィールドの一つ`pfd`(`poll.FD`型)の`Read`メソッドを呼んでいます。
```go
// os.file型の公開Readメソッドの中身
func (f *File) Read(b []byte) (n int, err error) {
	// (中略)
	n, e := f.read(b)  // ここで読み込み(非公開readメソッドを呼び出し)
	return n, f.wrapErr("read", e)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/os/file.go#113]
```go
// os.file型の非公開readメソッドの中身
func (f *File) read(b []byte) (n int, err error) {
    n, err = f.pfd.Read(b) // ここで読み込み
    // (中略)
    return n, err
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/os/file_posix.go#30]

この`poll.FD`型の`Read()`メソッドの内部実装で、`ignoringEINTRIO(syscall.Read, fd.Sysfd, p)`というコードが存在します。
ここで呼ばれている`syscall.Read`関数が、OSカーネルの`read`システムコールのラッパーです。ここでGoと低レイヤとつながるのです。
出典:[https://go.googlesource.com/go/+/go1.16.2/src/internal/poll/fd_unix.go#162]

順番をまとめると、`os.File`型の`Read`メソッドは以下のような実装となっています。
1. `os.file`型の`Read`メソッドを呼ぶ
2. 1の中で`os.file`型の`read`メソッドを呼ぶ
3. 2の中で`poll.FD`型の`Read`メソッドを呼ぶ
4. 3の中で`syscall.Read`メソッドを呼ぶ
5. OSカーネルのシステムコールで読み込み処理

## Writeメソッド
`os.File`型の`Write()`メソッドのほうも`Read`メソッドと同様の流れで実装されています。
1. `os.file`型の`Write`メソッドを呼ぶ
2. 1の中で`os.file`型の`write`メソッドを呼ぶ
3. 2の中で`poll.FD`型の`Write`メソッドを呼ぶ
4. 3の中で`syscall.Write`メソッドを呼ぶ
5. OSカーネルのシステムコールで書き込み処理

![](https://storage.googleapis.com/zenn-user-upload/rnaugoc5gva6ra9kkxzexjjgno2o)

## (おまけ)ファイルクローズ
ここまで見てきたファイル操作の裏には、どれもシステムコールがありました。
なので「ファイルの`Close()`メソッドも、裏ではclose()のシステムコールを呼んでいるんでしょ？」と推測する方もいるかもしれません。

しかし実は、`os.File`型の`Close()`メソッドを掘り下げても、closeシステムコールに繋がる`syscall.Close`は出てきません。
これはなぜかというと、ファイルオープンの時点で「ファイルオープンしたプロセスが終了したら、自動的にファイルを閉じてください」という`O_CLOEXEC`フラグを立てているからなのです。
```go
// (再掲)
func openFileNolog(name string, flag int, perm FileMode) (*File, error) {
	// (略)
	// 第二引数が「フラグ」
	r, e = syscall.Open(name, flag|syscall.O_CLOEXEC, syscallMode(perm))
	// (略)
}
```
そのため、`Close()`メソッドがやっているのは
- エラー処理
- 対応する`os.File`型を使えなくする後始末

という側面が強いです。

# まとめ
ここまでは、ファイルの読み書きについて取り上げました。
ただし、「I/O」というのはファイルだけのものではありません。

次章では、「ファイルではないI/O」について扱いたいと思います。