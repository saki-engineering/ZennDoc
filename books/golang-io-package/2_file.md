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

# 執筆メモ
- ファイルってなんで閉じなきゃいけないの？
- システムコール、低レイヤでは何が起きてるの？