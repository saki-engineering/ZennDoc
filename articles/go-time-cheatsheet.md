---
title: "Goで時刻を扱うチートシート"
emoji: "⏰"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go"]
published: false
---
# この記事について
![](https://storage.googleapis.com/zenn-user-upload/4808d9ec1f3a-20220410.png)

上のチートシートは、Goで時刻を扱う際に出てくる表現法と、それらを互いに変換するためにはどうしたらいいのかを一枚の図にまとめたものです。
この記事では、このチートシートに出てくる処理の詳細について掘り下げて説明しています。

## 使用する環境・バージョン
- OS: macOS Catalina 10.15.7
- go version go1.18 darwin/amd64
- OSのタイムゾーン: JST(日本標準時、UTC+0900)

## 想定読者
この記事では、以下のような方を想定しています。
- Goの基本的な文法は分かっている人
- 異なる時刻の表現法を、Goではどのように変換することになるかを知りたい人

逆に、以下のような方は読んでも物足りないか、ここからは得たい情報が得られないかもしれません。
- 一般的にサーバーサイドで、どうすればタイムゾーンを正しく扱うことができるかを知りたい人
- タイムゾーン実装のベストプラクティスが知りたい人





# 時刻を表すための方式
一言で「時刻」といっても、それをどういう形で扱うのかは様々です。

(例)2022年4月1日 午前9時0分0秒 日本標準時JSTの場合

- `time.Time`型 : `time.Date(2022, 4, 1, 0, 0, 0, 0, time.Local)`
- UNIX時間: `1648771200`
- 文字列: `"2022-04-01T09:00:00+09:00"`
- JSON文字列: `{"timestamp":"2022-04-01T09:00:00+09:00"}`

それぞれについて説明します。

## `time.Time`型
Goの標準パッケージ`time`に用意されている構造体です。
`time.Date`関数に日付、時間等を渡してやることで生成することができます。
```go
func Date(year int, month Month, day, hour, min, sec, nsec int, loc *Location) Time
```
出典:[pkg.go.dev - time.Date](https://pkg.go.dev/time#Date)

Goのプログラムの中で、人がわかりやすい形で時刻を扱うのなら真っ先に選択肢に入ってくるのがこの構造体でしょう。

## UNIX時間
1970年1月1日午前0時0分0秒から何秒経過したかの値で時刻を表す方式です。

(例)
- 1970年1月1日午前0時0分0秒 : UNIX時間だと`0`
- 1970年1月1日午前0時0分1秒 : UNIX時間だと`1`
- 1970年1月1日午前1時0分0秒 : UNIX時間だと`3600`

## 文字列 / JSON文字列
`"2022-04-01T00:00:00Z"`のように、時刻の情報が文字列の形で与えられるということもあります。
これは、そのデータ構造の中で時刻を表すための特別な型が存在しないときに起こるパターンです。

代表的な例がJSONです。
[RFC8259](https://datatracker.ietf.org/doc/html/rfc8259)では、JSONの値として使えるのは以下の型だと規定されています。
- オブジェクト
- 配列
- 数値
- 文字列
- 真偽値(`true`/`false`)
- `null`

時刻を表すための特別な型はJSONにはなく、そのためJSONの中では時刻を文字列として扱わざるを得ません。






# Goにおける時刻型の変換
それではここからは、上に挙げた時刻表現をどう互いに変換するのかについて見ていきます。

## `time.Time`型からの変換
### `time.Time`型 -> UNIX時間
`time.Time`型には`Unix`メソッドが用意されており、それを用いることで簡単にUNIX時間を得ることができます。
```go
func (t Time) Unix() int64
```
出典:[pkg.go.dev - time.Time.Unix](https://pkg.go.dev/time#Time.Unix)

```go
func time2unix(t time.Time) int64 {
	// レシーバーtのUNIX時間を返す
	return t.Unix()
}

func main() {
	var timeTime = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	fmt.Println(time2unix(timeTime)) // 1648771200
}
```

### `time.Time`型 -> 文字列
`time.Time`型には`Format`メソッドというものが用意されています。
```go
func (t Time) Format(layout string) string
```
出典:[pkg.go.dev - time.Time.Format](https://pkg.go.dev/time#Time.Format)

この`Format`メソッドの引数`layout`にて、変換後の文字列のフォーマットを指定します。

```go
func time2str(t time.Time) string {
	// レシーバーtを、"YYYY-MM-DDTHH-MM-SSZZZZ"という形の文字列に変換する
	return t.Format("2006-01-02T15:04:05Z07:00")
}

func main() {
	var timeTime = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	fmt.Println(time2str(timeTime))
	// 2022-04-01T09:00:00+09:00
}
```

:::message
引数`layout`に渡す時刻は、「2006年1月2日15時4分5秒 アメリカ山地標準時MST(GMT-0700)」のものを使うことになっています。

また、`time`パッケージには引数`layout`に渡すためのメジャーな表現を定数として定義してくれているので、そちらを使うのもいいでしょう。
```go
const (
	Layout      = "01/02 03:04:05PM '06 -0700" // The reference time, in numerical order.
	ANSIC       = "Mon Jan _2 15:04:05 2006"
	UnixDate    = "Mon Jan _2 15:04:05 MST 2006"
	RubyDate    = "Mon Jan 02 15:04:05 -0700 2006"
	RFC822      = "02 Jan 06 15:04 MST"
	RFC822Z     = "02 Jan 06 15:04 -0700" // RFC822 with numeric zone
	RFC850      = "Monday, 02-Jan-06 15:04:05 MST"
	RFC1123     = "Mon, 02 Jan 2006 15:04:05 MST"
	RFC1123Z    = "Mon, 02 Jan 2006 15:04:05 -0700" // RFC1123 with numeric zone
	RFC3339     = "2006-01-02T15:04:05Z07:00"
	RFC3339Nano = "2006-01-02T15:04:05.999999999Z07:00"
	Kitchen     = "3:04PM"
	// Handy time stamps.
	Stamp      = "Jan _2 15:04:05"
	StampMilli = "Jan _2 15:04:05.000"
	StampMicro = "Jan _2 15:04:05.000000"
	StampNano  = "Jan _2 15:04:05.000000000"

```
出典:[pkg.go.dev - time.Constants](https://pkg.go.dev/time#pkg-constants)
:::

### `time.Time`型 -> JSON文字列
JSONのようなkey-valueの形を作るためには、独自構造体を定義してそれをJSONエンコードする必要があります。
```go
func time2json(t time.Time) string {
	// 独自構造体myStructを定義して、
	// そのTimestampフィールドをJSONキー"timestamp"に対応付けする
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}

	// myStruct構造体をJSONエンコードして返す
	b, _ := json.Marshal(myStruct{t})
	return string(b)
}

func main() {
	var timeTime = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	fmt.Println(time2json(timeTime))
	// {"timestamp":"2022-04-01T09:00:00+09:00"}
}
```

### `time.Time`型からの変換まとめ
ここまで`time.Time`型からUNIX時間・文字列・JSONに変換する方法について紹介しました。
これらが全て正しく動作することはユニットテストでも確認することができます。
```go
var (
	// この4つは全て同じ時刻を表している
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T09:00:00+09:00"
	jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
)

// time.Time型 -> UNIX時間
func time2unix(t time.Time) int64 {
	return t.Unix()
}

// time.Time型 -> 文字列
func time2str(t time.Time) string {
	return t.Format("2006-01-02T15:04:05Z07:00")
}

// time.Time型 -> JSON文字列
func time2json(t time.Time) string {
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}
	b, _ := json.Marshal(myStruct{timeTime})

	return string(b)
}

// 4つの表現方法が等しいかどうか確認
func TestConvertTime(t *testing.T) {
	if got := time2unix(timeTime); got != unixTime {
		t.Errorf("time2unix: got %d but want %d\n", got, unixTime)
	}
	if got := time2str(timeTime); got != strTime {
		t.Errorf("time2str: got %s but want %s\n", got, strTime)
	}
	if got := time2json(timeTime); got != jsonTime {
		t.Errorf("time2json: got %s but want %s\n", got, jsonTime)
	}
}

// === RUN   TestConvertTime
// --- PASS: TestConvertTime (0.00s)
// PASS
```

## UNIX時間からの変換
### UNIX時間 -> `time.Time`型
UNIX時間から`time.Time`型に変換するためには、`time`パッケージに用意された`Unix`関数を用います。
```go
func Unix(sec int64, nsec int64) Time
```
出典:[pkg.go.dev - time.Unix](https://pkg.go.dev/time#Unix)

```go
func unix2time(t int64) time.Time {
	// 秒単位のUNIX時間がt, ナノ秒が0の時刻を持つtime.Time型を返す
	return time.Unix(t, 0)
}

func main() {
	var unixTime int64 = 1648771200
	fmt.Println(unix2time(unixTime))
	// 2022-04-01 09:00:00 +0900 JST
}
```

### UNIX時間 -> 文字列
UNIX時間から直接文字列を生成する方法は存在しません。
一旦`time.Time`型を経由して、「UNIX時間 -> `time.Time`型 -> 文字列」とする必要があります。
```go
func unix2str(t int64) string {
	// UNIX時間 -> time.Time型の関数
	unix2time := func(t int64) time.Time {
		return time.Unix(t, 0)
	}
	// time.Time型 -> 文字列の関数
	time2str := func(t time.Time) string {
		return t.Format("2006-01-02T15:04:05Z07:00")
	}
	return time2str(unix2time(t))
}

func main() {
	var unixTime int64 = 1648771200
	fmt.Println(unix2str(unixTime))
	// 2022-04-01T09:00:00+09:00
}
```

### UNIX時間 -> JSON文字列
UNIX時間からJSON文字列を生成する際も、一旦`time.Time`型を経由させるしかありません。
```go
func unix2json(t int64) string {
	// UNIX時間 -> time.Time型の関数
	unix2time := func(t int64) time.Time {
		return time.Unix(t, 0)
	}
	// time.Time型 -> JSON文字列の関数
	time2json := func(t time.Time) string {
		type myStruct struct {
			Timestamp time.Time `json:"timestamp"`
		}
		b, _ := json.Marshal(myStruct{t})

		return string(b)
	}
	return time2json(unix2time(t))
}

func main() {
	var unixTime int64 = 1648771200
	fmt.Println(unix2json(unixTime))
	// {"timestamp":"2022-04-01T09:00:00+09:00"}
}
```

### UNIX時間からの変換まとめ
紹介した変換方式が正しく動作するのか、ユニットテストで検証しましょう。
```go
var (
	// この4つは全て同じ時刻を表している
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T09:00:00+09:00"
	jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
)

func unix2time(t int64) time.Time {
	return time.Unix(t, 0)
}

func unix2str(t int64) string {
	time2str := func(t time.Time) string {
		return t.Format("2006-01-02T15:04:05Z07:00")
	}
	return time2str(unix2time(t))
}

func unix2json(t int64) string {
	time2json := func(t time.Time) string {
		type myStruct struct {
			Timestamp time.Time `json:"timestamp"`
		}
		b, _ := json.Marshal(myStruct{t})

		return string(b)
	}
	return time2json(unix2time(t))
}

func TestConvertUnix(t *testing.T) {
	if got := unix2time(unixTime); !got.Equal(timeTime) {
		t.Errorf("unix2time: got %s but want %s\n", got, timeTime)
	}
	if got := unix2str(unixTime); got != strTime {
		t.Errorf("unix2str: got %s but want %s\n", got, strTime)
	}
	if got := unix2json(unixTime); got != jsonTime {
		t.Errorf("unix2json: got %s but want %s\n", got, jsonTime)
	}
}

// === RUN   TestConvertUnix
// --- PASS: TestConvertUnix (0.00s)
// PASS
```

## 文字列からの変換
### 文字列 -> `time.Time`型
文字列から`time.Time`型に変換するには、`time`パッケージ内にある`Parse`関数を使います。
```go
func Parse(layout, value string) (Time, error)
```
出典:[pkg.go.dev - time.Parse](https://pkg.go.dev/time#Parse)

引数`layout`に、変換対象となる文字列がどのような表現形式になっているのかを指定して変換を行います。
この`layout`引数も、`t.Format`メソッドと同様に「2006年1月2日15時4分5秒 アメリカ山地標準時MST(GMT-0700)」の時刻文字列を使用することになっています。

```go
func str2time(t string) time.Time {
	// YYYY-MM-DDTHH:MM:SSZZZZの形式で渡される文字列tをtime.Time型に変換して返す
	parsedTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", t)
	return parsedTime
}

func main() {
	var strTime string = "2022-04-01T09:00:00+09:00"
	fmt.Println(str2time(strTime))
	// 2022-04-01 09:00:00 +0900 JST
}
```

### 文字列 -> UNIX時間
UNIX時間から文字列に直接変換する術がなかったのと同様に、その逆変換である文字列 -> UNIX時間も一度`time.Time`型を経由する必要があります。
```go
func str2unix(t string) int64 {
	// 文字列 -> time.Time型の関数
	str2time := func(t string) time.Time {
		parsedTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", t)
		return parsedTime
	}
	// time.Time型 -> UNIX時間の関数
	time2unix := func(t time.Time) int64 {
		return t.Unix()
	}
	return time2unix(str2time(t))
}

func main() {
	var strTime string = "2022-04-01T09:00:00+09:00"
	fmt.Println(str2unix(strTime))
	// 1648771200
}
```

### 文字列 -> JSON文字列
この変換を行いたいというユースケースは、おそらくあまりないのではないでしょうか……。というわけで割愛します。

### 文字列からの変換まとめ
ここまで紹介した変換が正しく動作するかを検証するユニットテストはこちらです。
```go
var (
	// この3つは全て同じ時刻を表している
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T09:00:00+09:00"
)

func str2time(t string) time.Time {
	parsedTime, _ := time.Parse("2006-01-02T15:04:05Z07:00", t)
	return parsedTime
}

func str2unix(t string) int64 {
	time2unix := func(t time.Time) int64 {
		return t.Unix()
	}
	return time2unix(str2time(t))
}

func TestConvertStr(t *testing.T) {
	if got := str2time(strTime); !got.Equal(timeTime) {
		t.Errorf("str2time: got %s but want %s\n", got, timeTime)
	}
	if got := str2unix(strTime); got != unixTime {
		t.Errorf("str2unix: got %d but want %d\n", got, unixTime)
	}
}

// === RUN   TestConvertStr
// --- PASS: TestConvertStr (0.00s)
// PASS
```

## JSON文字列からの変換
### JSON文字列 -> `time.Time`型
JSONから何らかの値を読み込むためには、独自構造体を定義してそこに向かってJSONでコードを行う必要があります。
```go
func json2time(t string) time.Time {
	// 独自構造体myStructを定義して
	// そのTimestampフィールドにJSONキー"timestamp"を対応付け
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}

	// JSONをmyStruct構造体にデコードして、そのTimestampフィールドを取り出して返す
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp
}

func main() {
	var jsonTime string = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
	fmt.Println(json2time(jsonTime))
	// 2022-04-01 09:00:00 +0900 JST
}
```

### JSON文字列 -> UNIX時間
UNIX時間からJSON文字列に直接変換できなかったように、その逆も`time.Time`型を経由させる必要があります。
```go
func json2unix(t string) int64 {
	// JSON文字列 -> time.Time型の関数
	json2time := func(t string) time.Time {
		type myStruct struct {
			Timestamp time.Time `json:"timestamp"`
		}
		var myStc myStruct
		json.Unmarshal([]byte(t), &myStc)
		return myStc.Timestamp
	}
	// time.Time型 -> UNIX時間の関数
	time2unix := func(t time.Time) int64 {
		return t.Unix()
	}
	return time2unix(json2time(t))
}

func main() {
	var jsonTime string = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
	fmt.Println(json2unix(jsonTime))
	// 1648771200
}
```

### JSON文字列 -> 文字列
「文字列 -> JSON」同様に、これもユースケースが見えないので割愛します。

### JSON文字列からの変換まとめ
紹介した変換方法の動作を検証するユニットテストです。
```go
var (
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	unixTime int64     = 1648771200
	jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
)

func json2time(t string) time.Time {
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp
}

func json2unix(t string) int64 {
	time2unix := func(t time.Time) int64 {
		return t.Unix()
	}
	return time2unix(json2time(t))
}

func TestConvertJSON(t *testing.T) {
	if got := json2time(jsonTime); !got.Equal(timeTime) {
		t.Errorf("json2time: got %s but want %s\n", got, timeTime)
	}
	if got := json2unix(jsonTime); got != unixTime {
		t.Errorf("json2unix: got %d but want %d\n", got, unixTime)
	}
}

// === RUN   TestConvertJSON
// --- PASS: TestConvertJSON (0.00s)
// PASS
```

## まとめ
ここまで紹介した変換方法をまとめた図が以下です。
![](https://storage.googleapis.com/zenn-user-upload/2580c18a8610-20220410.png)



# 扱う時刻のタイムゾーンと実行環境のタイムゾーンが異なる場合
さて、今まで私たちはJST(日本標準時)を扱ってきました。
```go
// JST(日本標準時)での表記
var (
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T09:00:00+09:00"
	jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
)
```
しかし、「**実行環境のタイムゾーンとは異なる時刻**」を扱う際には少々注意が必要です。
ここからは、それを別のタイムゾーンの時刻にして同様の変換処理を行なっていきたいと思います。

## 別のタイムゾーンでの検証
### UTC(協定標準時)の場合
以下のように、扱う時刻をUTC(協定標準時)のものに変えて、これまで用意したユニットテストを実行してみます。
```go
var (
	timeTime time.Time = time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)
	unixTime int64     = 1648771200
	strTime  string    = "2022-04-01T00:00:00Z"
	jsonTime string    = `{"timestamp":"2022-04-01T00:00:00Z"}`
)
```
```bash
$ go test
--- FAIL: TestConvertUnix (0.00s)
    unix_test.go:34: unix2time: got 2022-04-01 09:00:00 +0900 JST but want 2022-04-01 00:00:00 +0000 UTC
    unix_test.go:37: unix2str: got 2022-04-01T09:00:00+09:00 but want 2022-04-01T00:00:00Z
    unix_test.go:40: unix2json: got {"timestamp":"2022-04-01T09:00:00+09:00"} but want {"timestamp":"2022-04-01T00:00:00Z"}
FAIL
exit status 1
```

以下3つの関数でおかしな挙動をしていることが確認できます。
- `unix2time`: UNIX時間から`time.Time`型への変換
- `unix2str`: UNIX時間から文字列への変換
- `unix2json`: UNIX時間からJSON文字列への変換

`unix2str`と`unix2json`は、内部で`unix2time`を挟んでいることを考えると、実質的には「UNIX時間から`time.Time`型への変換」がうまくいってないのが根本原因と考えていいでしょう。

### タイムゾーン`America/New_York`の場合
UTC(協定標準時)のようなメジャーな時間ではなく、今度はニューヨーク時間(通常UTC-0500、サマータイムUTC-0400)で検証してみましょう。
```go
var (
	newYork *time.Location = func() *time.Location {
		location, _ := time.LoadLocation("America/New_York")
		return location
	}()
	timeTime time.Time = time.Date(2022, 3, 31, 20, 0, 0, 0, newYork)
	unixTime int64     = 1648771200
	strTime  string    = "2022-03-31T20:00:00-04:00"
	jsonTime string    = `{"timestamp":"2022-03-31T20:00:00-04:00"}`
)
```
```bash
$ go test
--- FAIL: TestConvertJSON (0.00s)
    json_test.go:27: json2time: got 2022-03-31 20:00:00 -0400 -0400 but want 2022-03-31 20:00:00 -0400 EDT
--- FAIL: TestConvertStr (0.00s)
    str_test.go:22: str2time: got 2022-03-31 20:00:00 -0400 -0400 but want 2022-03-31 20:00:00 -0400 EDT
--- FAIL: TestConvertUnix (0.00s)
    unix_test.go:34: unix2time: got 2022-04-01 09:00:00 +0900 JST but want 2022-03-31 20:00:00 -0400 EDT
    unix_test.go:37: unix2str: got 2022-04-01T09:00:00+09:00 but want 2022-03-31T20:00:00-04:00
    unix_test.go:40: unix2json: got {"timestamp":"2022-04-01T09:00:00+09:00"} but want {"timestamp":"2022-03-31T20:00:00-04:00"}
FAIL
exit status 1
```
UNIX時間からの変換である以下3つがおかしな挙動をしているのは、UTC(協定標準時)の時と同様です。
- `unix2time`: UNIX時間から`time.Time`型への変換
- `unix2str`: UNIX時間から文字列への変換
- `unix2json`: UNIX時間からJSON文字列への変換

それに加えて、新たに2つの関数でも想定外の挙動をしています。
- `str2time`: 文字列から`time.Time`型への変換
- `json2time`: JSON文字列から`time.Time`型への変換

## 実行環境のタイムゾーンに挙動が依存する関数・メソッド
ここまでの話をまとめると、扱うタイムゾーンによって挙動が変わるのは以下の操作です。
- UNIX時間から`time.Time`型への変換(UTC、ニューヨーク時間の場合)
- 文字列から`time.Time`型への変換(ニューヨーク時間の場合)
- JSON文字列から`time.Time`型への変換(ニューヨーク時間の場合)

そして実際、これら3つの処理の中で使っている`time`パッケージの関数・メソッドがシステムタイムゾーン依存の挙動をするのです。

### UNIX時間から`time.Time`型への変換 - `time.Unix`関数
UNIX時間から`time.Time`型への変換する際に使用している`time.Unix`関数は、「返り値の`time.Time`型は、そのプログラムを実行しているシステムタイムゾーンのものにする」という仕様になっています。

> Unix returns the local Time corresponding to the given Unix time
> (訳)`Unix`関数は、与えられたUNIX時間に対応するローカル時間を返却します。
> 出典:[pkg.go.dev - time.Unix](https://pkg.go.dev/time#Unix)

そのため、今回のローカルタイムゾーンであるJST(日本標準時)以外の`time.Time`を返却したいのであれば、`time.Time`型に用意された`In`メソッドを使用してタイムゾーンを明示的に指定してやる必要があります。

```go
func (t Time) In(loc *Location) Time
```
出典:[pkg.go.dev - time.In](https://pkg.go.dev/time#Time.In)

この`In`メソッドは、レシーバーの`time.Time`からタイムゾーンだけを変えた`time.Time`を返してくれます。
つまり、`In`メソッドの使用前と使用後でUNIX時間は変わりません。

(例)
- 2022年4月1日 9時0分0秒 JST(日本標準時) -(`In`メソッド)-> 2022年4月1日 0時0分0秒 UTC(協定標準時)

この`In`メソッドを用いて、UNIX時間から`time.Time`型への変換関数`unix2time`を修正すると以下のようになります。
```go
// UTC(協定標準時)の場合
func unix2time(t int64) time.Time {
	return time.Unix(t, 0).In(time.UTC)
}

// ニューヨーク時間の場合
func unix2time(t int64) time.Time {
	return time.Unix(t, 0).In(newYork)
}
```

### 文字列から`time.Time`型への変換 - `time.Parse`関数
文字列から`time.Time`型への変換に使用している`time.Parse`関数では、タイムゾーンを以下のように扱っています。
- 入力として与えられた文字列にタイムゾーン・オフセットの情報がなかったら、それはUTC(協定標準時)の時刻とみなしてパースする。
- 入力として与えられた文字列に、オフセットの情報(例:+0900)のみがあり、タイムゾーンの情報(例:JST)がなかった場合
	- 実行環境のタイムゾーンのオフセット(今回だとJST=+0900)と一致していた場合、実行環境が使用しているタイムゾーン(今回だとJST)の時刻とみなしてパースする
	- 実行環境のタイムゾーンのオフセット(今回だとJST=+0900)と一致していない場合、**タイムゾーンが確定できないので仮のタイムゾーン名でパースする**
		- (例)JST(+0900)の実行環境で、`-0400`のオフセットを受け取った場合 -> オフセット`-0400`、仮タイムゾーン名`-0400`でパースする
- 入力として与えられた文字列に、タイムゾーンの情報(例:JST)のみがあり、オフセットの情報(例:+0900)がなかった場合
	- 実行環境のタイムゾーン(今回だとJST=+0900)と一致していた場合、そのタイムゾーンのオフセット(今回だと`+0900`)を補完してパースする
	- 与えられたタイムゾーンがUTCだった場合には、UTC+0000としてパースする
	- それ以外の場合、与えられたタイムゾーンで、仮オフセット`+0000`としてパースする
		- (例)JST(+0900)の実行環境で、`EDT`のタイムゾーンの時刻を受け取った場合 -> タイムゾーン名`EDT`、仮オフセット`+0000`でパースする

今回関係あるのが「実行環境のタイムゾーンのオフセット(今回だとJST=+0900)と一致していない場合」という箇所です。
つまり、JSTの実行環境で、ニューヨーク時間(夏)のオフセットである`-0400`を与えられたとしても、`time.Time`型生成時にタイムゾーン`EDT`を補完してくれないのです。

```bash
// テストのFAILメッセージからも、タイムゾーンが補完されずに-0400という仮のものになっているのが確認できます。
--- FAIL: TestConvertStr (0.00s)
    str_test.go:22: str2time: got 2022-03-31 20:00:00 -0400 -0400 but want 2022-03-31 20:00:00 -0400 EDT
```

:::message
UTC-0400のタイムゾーンは
- EDT(東部夏時間) - サマータイム時のニューヨーク
- AST(大西洋標準時) - 通常時のカナダ・ケベック州の一部

のように複数あるので、オフセット`-0400`からタイムゾーンがそもそも確定できないのです。
そのため、無理やりタイムゾーンを保管しないで仮のタイムゾーン名`-0400`として扱うのはまあまあ合理的なのではないでしょうか。
:::

タイムゾーン・オフセットを指定して文字列をパースするためには、`time.Parse`関数ではなく`time.ParseInLocation`関数を用いる必要があります。
```go
func ParseInLocation(layout, value string, loc *Location) (Time, error)
```
出典:[pkg.go.dev - time.ParseInLocation](https://pkg.go.dev/time#ParseInLocation)

これを用いて、文字列から`time.Time`型への変換関数`str2time`を直すと以下のようになります。
```go
// ニューヨーク時間の場合
func str2time(t string) time.Time {
	parsedTime, _ := time.ParseInLocation("2006-01-02T15:04:05Z07:00", t, newYork)
	return parsedTime
}
```

### JSON文字列から`time.Time`型への変換
JSON文字列から`time.Time`型への変換する際には、`encoding/json`パッケージの`json.Unmarshal`関数を利用しています。
これは内部的には`time.Time`型の`UnmarshalJSON`メソッド→`time.Time`型の`Parse`メソッドを利用しています。

つまり、「文字列->`time.Time`型」のときと同様に「実行環境のタイムゾーンのオフセット(今回だとJST=+0900)と一致していない場合」にはタイムゾーンの補完がなされないということです。

これを直すためには、デコード結果に`In`メソッドを用いて明示的にタイムゾーンを変換してやるという方法が一つあります。
```go
func json2time(t string) time.Time {
	type myStruct struct {
		Timestamp time.Time `json:"timestamp"`
	}
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	return myStc.Timestamp.In(newYork)
}
```

:::message
後述する「RFC3339以外の時刻フォーマットでJSONエンコード・デコードを行う」ときのように、独自の時刻型を定義して、その独自型にタイムゾーンをfixする機構を含ませたカスタムの`UnmarshalJSON`メソッドを実装するという方法もあります。
```go
// 独自構造体を定義
type MyDate struct {
	Timestamp time.Time
}

// json.Unmarshal関数が内部的に使うUnmarshalJSONメソッドをカスタムでオーバーライド
func (d *MyDate) UnmarshalJSON(data []byte) error {
	// ParseInLocation関数を利用することで、タイムゾーンを考慮したJSONデコードができるようになる
	t, err := time.ParseInLocation(`"2006-01-02T15:04:05Z07:00"`, string(data), newYork)
	if err != nil {
		return err
	}
	d.Timestamp = t
	return nil
}

func json2timeVer2(t string) time.Time {
	type myStruct struct {
		// タイムスタンプをtime.Time型ではなく
		// 独自UnmarshalJSONメソッドを持たせた自作構造体型に変更
		Timestamp MyDate `json:"timestamp"`
	}
	var myStc myStruct
	json.Unmarshal([]byte(t), &myStc)
	// ここでInメソッドを用いてタイムゾーンをいじる必要がなくなった
	return myStc.Timestamp.Timestamp
}
```
:::

## 実行環境のタイムゾーンとは異なる時刻を扱うときのまとめ
![](https://storage.googleapis.com/zenn-user-upload/f0be26de02c7-20220410.png)



# 応用編 - RCF3339以外の時刻フォーマットでJSONエンコード/でコードを行う
さて、ここからは応用編ということで、JSONに含まれている時刻文字列を自己流にいじることを考えていきたいと思います。
```go
var (
	timeTime time.Time = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	
	// RFC3339のフォーマット
	// strTime  string  = "2022-04-01T09:00:00+09:00"
	// jsonTime string  = `{"timestamp":"2022-04-01T09:00:00+09:00"}`

	// 独自の時刻文字列フォーマット
	strTime  string = "2022/04/01 09:00:00.000 +0900"
	jsonTime string = `{"timestamp":"2022/04/01 09:00:00.000 +0900"}`
)
```

ここまでは、以下2つの`encoding/json`パッケージ内の関数を何気なく使っていたかと思います。
- `json.Marshal`関数 : `time.Time`型 -> JSON文字列への変換
- `json.Unmarshal`関数 : JSON文字列 -> `time.Time`型への変換

しかしこれらは、以下のような仕様が存在するのです。
- `json.Marshal`関数 : `time.Time`型はRFC3339で定義されたフォーマット(`YYYY-MM-DDTHH:MM:SSZZZZ`)に変換する
- `json.Unmarshal`関数 : RFC3339で定義されたフォーマット(`YYYY-MM-DDTHH:MM:SSZZZZ`)の文字列を`time.Time`型に変換する

```go
// time.Time型 -> JSON文字列への変換
func time2json(t time.Time) string {
	// (一部抜粋)
	b, _ := json.Marshal(myStruct{t})
}

// JSON文字列 -> time.Time型への変換
func json2time(t string) time.Time {
	// (一部抜粋)
	json.Unmarshal([]byte(t), &myStc)
}

func main() {
	var timeTime = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	fmt.Println(time2json(timeTime))
	// {"timestamp":"2022-04-01T09:00:00+09:00"}
	// RFC3339のフォーマットがJSONに使われる

	var jsonTime string = `{"timestamp":"2022-04-01T09:00:00+09:00"}`
	fmt.Println(json2time(jsonTime))
	// デコード対象のJSONの中でRFC3339のフォーマットを使う必要がある
	// 2022-04-01 09:00:00 +0900 JST
}
```

これを、
- `time.Time`型 -> JSON文字列への変換時に、独自の文字列フォーマットで出力されるようにしたい
- JSON文字列 -> `time.Time`型への変換時に、独自の文字列フォーマットが時刻文字列として認識されるようにしたい

というように変えたい場合には、`json.Marshal`関数/`json.Unmarshal`関数の挙動を変更してやる必要があるのです。

## 独自構造体の定義
`json.Marshal`関数が`time.Time`型をJSONエンコードする際には、内部で`time.Time`型の`MarshalJSON`メソッドを利用しています。
```go
func (t Time) MarshalJSON() ([]byte, error)
```
出典:[pkg.go.dev - time.MarshalJSON](https://pkg.go.dev/time#Time.MarshalJSON)

また、`json.Unmarshal`関数が時刻文字列を`time.Time`型にデコードする際には、内部で`time.Time`型の`UnmarshalJSON`メソッドを利用しています。
```go
func (t *Time) UnmarshalJSON(data []byte) error
```
出典:[pkg.go.dev - time.UnmarshalJSON](https://pkg.go.dev/time#Time.UnmarshalJSON)

`time.Time`型をエンコード/デコードしようとする限り、RFC3339フォーマットを利用するようになっているこれらのメソッドが使われることになってしまいます。
そのため、`time.Time`型をそのまま使うのをやめて、独自の時刻構造体を定義してしまいます。

```go
type MyDate struct {
	Timestamp time.Time
}
```

## 独自構造体のエンコード/デコード挙動をカスタムする
独自構造体`MyDate`型ができたところで、この`MyDate`型がJSONエンコード・デコードされる際にはどのような処理・挙動をするのかというところを作っていきましょう。
そのためには、`MyDate`型の`MarshalJSON`メソッド・`UnmarshalJSON`メソッドを作っていけばOKです。

### JST(日本標準時)を扱う場合
`MarshalJSON`メソッドは、JSONエンコード時に呼ばれるメソッドです。
そのため、ここでは「`MyDate`構造体の`Timestamp`フィールドを、エンコード時に使いたいフォーマットに文字列変換して返す」ように作ります。
```go
func (d MyDate) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Timestamp.Format("2006/01/02 15:04:05.000 -0700"))), nil
}
```

`UnmarshalJSON`メソッドは、JSONデコード時に呼ばれるメソッドです。
そのため、ここでは「受け取った入力を、使いたいフォーマットを利用してパースして、`MyDate`構造体の`Timestamp`フィールドに収める」ように作ります。
```go
func (d *MyDate) UnmarshalJSON(data []byte) error {
	t, err := time.Parse(`"2006/01/02 15:04:05.000 -0700"`, string(data))
	if err != nil {
		return err
	}
	d.Timestamp = t
	return nil
}
```

### UTC(協定標準時)の場合
実行環境のタイムゾーンとは異なる時刻を扱う場合も同様に考えてみましょう。
```go
var (
	timeTime time.Time = time.Date(2022, 4, 1, 0, 0, 0, 0, time.UTC)

	// RFC3339のフォーマット
	// strTime  string    = "2022-04-01T09:00:00+09:00"
	// jsonTime string    = `{"timestamp":"2022-04-01T09:00:00+09:00"}`

	// 独自の時刻文字列フォーマット
	strTime  string = "2022/04/01 00:00:00.000 +0000"
	jsonTime string = `{"timestamp":"2022/04/01 00:00:00.000 +0000"}`
)
```

`MarshalJSON`メソッド内で使用している`time.Time.Format`メソッドは、特に実行環境のタイムゾーンに依存した挙動をすることはないため問題ありません。
しかし、`UnmarshalJSON`メソッド内で使用している`time.Parse`関数は実行環境のタイムゾーンによって挙動が変わります。

そのため、`MyDate`型の`UnmarshalJSON`メソッドで使用する関数を、`time.Parse`関数から`time.ParseInLocation`関数に変えましょう。
```go
func (d *MyDate) UnmarshalJSON(data []byte) error {
	t, err := time.ParseInLocation(`"2006/01/02 15:04:05.000 -0700"`, string(data), time.UTC)
	if err != nil {
		return err
	}
	d.Timestamp = t
	return nil
}
```

:::message
ニューヨーク時間のときも、UTCと同様の修正で対応可能です。
:::

## `time.Time`型 <-> JSON文字列の変換関数を修正
これで、独自フォーマットでのエンコード・デコードに対応した`MyDate`構造体の準備ができました。
ここからは、JSONエンコード・デコード時にこの`MyDate`型を使うように、変換関数を修正します。

```go
// time.Time型 -> JSON文字列への変換関数
func time2json(t time.Time) string {
	type MyStruct struct {
		// ここをtime.Time型からMyDate型に変更
		Timestamp MyDate `json:"timestamp"`
	}
	b, _ := json.Marshal(MyStruct{MyDate{t}})
	return string(b)
}

// JSON文字列 -> time.Time型への変換関数
func json2time(t string) time.Time {
	type MyStruct struct {
		// ここをtime.Time型からMyDate型に変更
		Timestamp MyDate `json:"timestamp"`
	}
	var myStc MyStruct
	json.Unmarshal([]byte(t), &myStc)
	// 返り値にはMyStruct構造体のTimestampフィールドを採用する
	return myStc.Timestamp.Timestamp
}
```

これで、独自フォーマットを利用したJSONエンコード・デコードの実装は完了です。
```go
func main() {
	var timeTime = time.Date(2022, 4, 1, 9, 0, 0, 0, time.Local)
	fmt.Println(time2json(timeTime))
	// {"timestamp":"2022/04/01 09:00:00.000 +0900"}
	// 独自フォーマットがJSONに使われる

	var jsonTime string = `{"timestamp":"2022/04/01 09:00:00.000 +0900"}`
	fmt.Println(json2time(jsonTime))
	// 独自フォーマットの時刻文字列をデコードできている
	// 2022-04-01 09:00:00 +0900 JST
}
```

# まとめ
というわけで、Goでの時刻表現と、それらを変換するための処理方法一覧を紹介してきました。
![](https://storage.googleapis.com/zenn-user-upload/4808d9ec1f3a-20220410.png)

タイムスタンプや時刻というのは、タイムゾーンや時差、サマータイムや秒数の単位はミリなのかナノなのか等、考えることが多くなかなか悩まされることが多い概念です。
この記事とチートシートで、処理の本質部分ではない変換部分はさくっと終わらせて、開発者が本来頭を使うべき上記の非機能要件に集中できるようになれれば幸いです。
