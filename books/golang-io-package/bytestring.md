---
title: "bytesパッケージとstringsパッケージ"
---
# はじめに
`io.Reader`と`io.Writer`を満たす型として、`bytes`パッケージの`bytes.Buffer`型が存在します。
また、`strings`パッケージの`strings.Reader`型は`io.Reader`を満たします。

本章では、これらの型について解説します。

# bytesパッケージのbytes.Buffer型
まずは、`bytes.Buffer`型の構造体の中身を確認してみましょう。
```go
type Buffer struct {
	buf      []byte
	// (略)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/bytes/buffer.go#20]

構造として特筆すべきなのは、中身にバイト列を持っているだけです。
これだけだったら「そのまま`[]byte`を使えばいいじゃないか」と思うかもしれませんが、パッケージ特有の型を新しく定義することによって、メソッドを好きに付けられるようになります。

というわけで、`bytes.Buffer`には`Read`メソッド、`Write`メソッドがついています。
これによって`io.Reader`と`io.Writer`を満たすようになっています。

## Writeメソッド
`Write`メソッドは、レシーバーのバッファの「中に」データを書き込むためのメソッドです。

使用例を以下に示します。
```go
// bytes.Bufferを用意
// (bytes.Bufferは初期化の必要がありません)
var b bytes.Buffer
b.Write([]byte("Hello"))

// バッファの中身を確認してみる
fmt.Println(b.String())

// (出力)
// Hello
```
参考:[pkg.go.dev - bytes#Buffer-Example](https://pkg.go.dev/bytes#example-Buffer)

## Readメソッド
`Read`メソッドは、レシーバーバッファから「中を」読み取るためのメソッドです。
```go
// 中にデータを入れたバッファを用意
var b bytes.Buffer
b.Write([]byte("World"))

// plainの中にバッファの中身を読み取る
plain := make([]byte, 10)
b.Read(plain)

// 読み取り後のバッファの中身と読み取り結果を確認
fmt.Println("buffer: ", b.String())
fmt.Println("output:", string(plain))

// buffer:  
// output: World
```
バッファの中からは`World`というデータが見えなくなり、きちんと変数`plain`に読み込みが成功しています。

# stringsパッケージのstrings.Reader型
`strings`パッケージは、文字列を置換したり辞書順でどっちが先か比べたりという単なる便利屋さんだけではないのです。
`bytes.Buffer`型と同じく、文字列型をパッケージ独自型でカバーすることで、`io.Reader`に代入できるようにした型も定義されているのです。

そんな独自型`strings.Reader`型は、構造体内部に文字列を内包しています。
```go
type Reader struct {
	s        string
	// (略)
}
```
出典:[https://go.googlesource.com/go/+/go1.16.3/src/strings/reader.go#17]

これは`Read`メソッドをもつ、`io.Reader`インターフェースを満たす構造体です。

:::message
`strings.Reader`型に`Write`メソッドはないので、`io.Writer`は満たしません。
:::

## Readメソッド
`Read`メソッドは、文字列から「中を」読み取るためのメソッドです。
使用例を示します。
```go
// NewReader関数から
// strings.Reader型のrdを作る
str := "Hellooooooooooooooooooooooooooo!"
rd := strings.NewReader(str)

// rowの中に読み取り
row := make([]byte, 10)
rd.Read(row)

// 読み取り結果確認
fmt.Println(string(row)) // Helloooooo
```

# これらの使いどころ
「バイト列や文字列を`io.Reader`・`io.Writer`に入れられるようにしたところで何が嬉しいの？」という疑問を持った方もいるかと思います。
ここからはそんな疑問に対して、ここで紹介した型の使い所を一つ紹介したいと思います。

## テストをスマートに書く
`io`の章で書いた`TranslateIntoGerman`関数を思い出してください。
```go
// 引数rで受け取った中身を読み込んで
// Hello → Guten Tagに置換する関数
func TranslateIntoGerman(r io.Reader) string {
	data := make([]byte, 300)
	len, _ := r.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	return result
}
```

この関数のテストを書くとき、皆さんならどうするでしょうか。
「`io.Reader`を満たすものしか引数に渡せない……テストしたい内容が書いてあるファイルを1個1個用意するか…？」と思ったこともいるでしょう。

ですが「ファイルを1個1個用意する」とかいう面倒な方法をせずとも、`strings.Reader`型を使うことで、テスト内容をコード内で用意することができるのです。
```go
func Test_Translate(t *testing.T) {
	// テストしたい内容を文字列ベースで用意
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "normal",
			arg:  "Hello, World!",
			want: "Guten Tag, World!",
		},
		{
			name: "repeat",
			arg:  "Hello World, Hello Golang!",
			want: "Guten Tag World, Guten Tag Golang!",
		},
	}

	// TranslateIntoGerman関数には
	// strings.NewReader(tt.args)で用意したstrings.Reader型を渡す
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TranslateIntoGerman(strings.NewReader(tt.arg)); got != tt.want {
				t.Errorf("got %v, but want %v", got, tt.want)
			}
		})
	}
}
```