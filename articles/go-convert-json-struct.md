---
title: "Goにおけるjsonの扱い方を整理・考察してみた ~ データスキーマを添えて"
emoji: "🧪"
type: "tech" # tech: 技術記事 / idea: アイデア
topics: ["go", "json"]
published: false
---
# この記事について
この記事は[Go Advent Calendar 2021](https://qiita.com/advent-calendar/2021/go) 13日目の記事です。
記事のテーマは`encoding/json`パッケージにおけるjsonエンコード・デコードの扱い方についてです。

普段私は、
- `Marshal`と`Unmarshal`ってどっちがGo→jsonでどっちがjson→Goなんだっけ？
- タグでマッピング規則をいじれるのはエンコードとデコードどっちだっけ？
- 非公開フィールドってどういう扱いになるんだっけ？

などというところがうろ覚えでパッと出てこないので、この際なのでまとめてしまおうということで前半部分を書きました。

また記事後半では、私が今年読んだとある本の内容に基づいて**Goでのjsonデコードが持つ性質**を好き勝手に考察してみました。

25日に向けてちょうど折り返しの位置ですが特におもしろネタ要素はなく、普通に私の書きたいことをひたすら真面目に理屈っぽく書いてしまいました。ごめんなさい(ゝω・)ﾃﾍﾍﾟﾛ

## 使用する環境・バージョン
- OS: macOS Catalina 10.15.7
- go version go1.17.2 darwin/amd64

## 想定読者
- Goの構造体型について基本的な理解がある人
- jsonが何かわかる人
- (`io.Reader`と`io.Writer`について知っていると読みやすいですが必須ではないです)





# 用語定義
まずは「エンコード・デコードとは何を意味するのか？」というところについて明確にしておきましょう。

どちらとも「データを変換するものだ」という理解はあると思いますが、これらは「何から何に」変換するものなのでしょうか。
この「何」という部分を認識するために、「そもそもデータはどう扱われているのか」について掘り下げてみましょう。

## 2種類のデータの扱い方
プログラムがデータを扱う方法は、大きく分けて2つあります。

一つは「何らかの構造を持つことを前提とした」扱い方です。
例えばリスト・配列・構造体……といったある種のデータ構造がこのパターンに該当します。
これらのデータ構造は、例えば「配列のn番目」「keyがxxxの値」といった形で簡単に参照・操作できるように最適化されています。
ここではこれを「**インメモリ表現**」と呼びたいと思います。

そしてもう一つは「ただのバイト列としての」扱い方です。
これは例えば「ファイルへの書き込み」「ネットワークからの受信」などといった、データそのものに何が書いてあるのかを気にする必要がない・気にするべきではないパターンにおいて使われます。
以下、これを「**バイト列表現**」と呼びます。

![](https://storage.googleapis.com/zenn-user-upload/55e1dbb34999-20211211.png)

インメモリ表現・バイト列表現[^1]、この2つをもとに、エンコード・デコードは「どちらからどちらへの」変換なのか確認してみましょう。
[^1]:実はこの表現も元ネタ本からいただいたものだったりします。

## エンコーディング(encoding)
エンコーディングは**インメモリ表現からバイト列表現への変換**のことを指します。
例えば、以下の動作はエンコードと呼ばれます。

- Go構造体からjsonを生成する
- 平文から暗号文を生成する

jsonと暗号文という全く別の文脈で同じ単語が使われることに違和感を覚える方もいるかもしれません。
しかし、いずれもデータを伝送するときに使われる表現であり、「中に何が書いてあるのか」ということは大して重要ではないという点で共通点があります(暗号文に至っては「何が書かれているのか」を意識してはいけない状態)。
そのためこの2つを「バイト列表現への変換」として同一視するのは、そこまでおかしなことではないのです。

エンコーディングと同じ意味の言葉として、**シリアライゼーション**(serialization)・**マーシャリング**(marshalling)がありますが、この記事全体ではエンコーディングと統一して述べます。

## デコーディング(decoding)
デコーディングは**バイト列表現からインメモリ表現への変換**のことを指します。
例えば、以下の動作はデコードと呼ばれます。

- jsonからGo構造体を生成する
- 暗号文から平文を生成する

デコーディングと同じ意味の言葉としては、**デシリアライゼーション**(deserialization)・**アンマーシャリング**(unmarshalling)があります。




# Goで行うエンコード・デコード
エンコード・デコードの定義について確認できたところで、Goの中でそれを行うにはどうしたらいいのかについて説明したいと思います。
「Go構造体をどのようなバイト列表現に変換するのか」というところについては様々な種類がありますが、本記事ではjsonの場合を扱います。

![](https://storage.googleapis.com/zenn-user-upload/afacbaa84ec8-20211212.png)

## Go構造体からjsonへのエンコード
例えば、以下のような構造体があったとします。
```go
type GoStruct struct {
	A int
	B string
}
stcData := GoStruct{A: 1, B: "bbb"}
```
この構造体に格納されているデータを、jsonの形に変換するにはどうしたらいいでしょうか。

### `json.Marshal`関数の利用
一つの方法として、標準パッケージ[`encoding/json`](https://pkg.go.dev/encoding/json)に含まれている[`json.Marshal`](https://pkg.go.dev/encoding/json#Marshal)関数を使う方法があります。

```go
func main() {
	stcData := GoStruct{A: 1, B: "bbb"}

	// Marshal関数でjsonエンコード
	// ->返り値jsonDataにはエンコード結果が[]byteの形で格納される
	jsonData, err := json.Marshal(stcData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s\n", jsonData)
}
```

このコードを実行した結果は以下のようになります。
```bash
$ go run main.go
{"A":1,"B":"bbb"}
```
Go構造体のフィールド名をキーに、フィールド値をvalueにしたjsonが生成されていることがわかります。

| Go構造体のフィールド名 | →エンコード→ | jsonキー名 |
| :---: | :---: | :---: |
| `A` | → | `A` |
| `B` | → | `B` |

### Encoderの利用
`encoding/json`パッケージ内には[`Encoder`](https://pkg.go.dev/encoding/json#Encoder)という構造体が存在し、それの[`Encode`](https://pkg.go.dev/encoding/json#Encoder.Encode)メソッドを用いることでもjsonエンコードを行うことができます。

手順としては以下のようになります。
1. [`json.NewEncoder`](https://pkg.go.dev/encoding/json#NewEncoder)関数から、引数で指定された場所にjsonを出力するエンコーダーを作成する
2. `Encode(Go構造体)`メソッドを実行してエンコード

```go
type GoStruct struct {
	A int
	B string
}

func main() {
	stcData := GoStruct{A: 1, B: "bbb"}

	// 標準出力にjsonエンコード結果を出す
	err := json.NewEncoder(os.Stdout).Encode(stcData)
	if err != nil {
		fmt.Println(err)
	}
}
```

`Marshal`関数とエンコーダーの違いとしては、前者はエンコード結果が`[]byte`になるのに対し後者は`io.Writer`の形で自由に指定することができるという点です。
例えばエンコード結果をそのまま`os.File`や`http.ResponseWriter`に書き込みたいという場合に、`Marshal`関数の場合は返り値でバイトスライスを得る→それを各々のWriteメソッドに渡してやるという2ステップが必要ですが、エンコーダーを使った場合は`Encode`メソッド一発で済ませることができます。
```go
file, err := os.Create(filename) // fileはos.File型
var stcData = GoStruct{A: 1, B: "bbb"}

// Marshal関数を使用した場合
jsonData, _ := json.Marshal(stcData)
file.Write(jsonData)

// Encoderを使用した場合
json.NewEncoder(file).Encode(stcData)
```
:::message
上記のコードは、簡単のためエラー処理を省略して書いています。
:::

## jsonからGo構造体へのデコード
次にエンコードの逆、jsonからGo構造体にデコードすることを考えてみましょう。

以下のようなjsonの内容を、先ほどのGo構造体にマッピングするにはどうしたらいいでしょうか。
```json
{"A":1, "B":"bbb"}
```
```go
type GoStruct struct {
	A int
	B string
}
```

### `json.Unmarshal`関数の利用
`Marshal`関数があったように、`encoding/json`パッケージ内には[`Unmarshal`](https://pkg.go.dev/encoding/json#Unmarshal)関数が存在しますのでそれを使うというのが第一の選択肢です。

```go
func main() {
	var stcData GoStruct
	jsonString := `{"A":1, "B":"bbb"}`

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
```

このコードを実行した結果は以下のようになります。
```bash
$ go run main.go
{A:1 B:bbb}
```
jsonのキー`"a"`の内容が構造体の`A`フィールドに、キー`"b"`の内容が構造体の`B`フィールドに入ったことが確認できました。

| jsonキー名 | →デコード→ | Go構造体のフィールド名 |
| :---: | :---: | :---: |
| `A` | → | `A` |
| `B` | → | `B` |

### Decoderの利用
`Unmarshal`関数はデコード対象となるjson文字列を`[]byte`の形で指定していましたが、これを`io.Reader`の形で得ているのならばデコーダーを用意するという方法があります。

手順としてはEncoderの時と同様です。
1. [`json.NewDecoder`](https://pkg.go.dev/encoding/json#NewDecoder)関数にデコード対象を`io.Reader`の形で渡し、[`json.Decoder`](https://pkg.go.dev/encoding/json#Decoder)を得る
2. [`Decode(Go構造体)`](https://pkg.go.dev/encoding/json#Decoder.Decode)メソッドを実行してデコード
```go
type GoStruct struct {
	A int
	B string
}

func main() {
	var stcData GoStruct
	jsonString := `{"A":1, "B":"bbb"}`

	// io.Reader型にしたjsonStringの内容をデコードする
	if err := json.NewDecoder(strings.NewReader(jsonString)).Decode(&stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
```
デコード対象となるデータを`io.Reader`で指定するということで、ストリームと相性がいい方法です。





# Go構造体フィールド-jsonキーのマッピング規則
エンコード・デコードの過程において、Goの構造体フィールドとjsonキーの間で対応づけが行われます。
```go
stcData := GoStruct{A: 1, B: "bbb"}

// Go構造体のA,Bフィールド<->jsonのキーA,Bが対応

jsonString := `{"A":1, "B":"bbb"}`
```
現状では同名のフィールド・キーが対応づいてますが、これを柔軟に変えたいという場面も存在します。
ここからは、構造体フィールド・jsonプロパティのマッピングルールについて探っていきます。

:::details 本章で使用するエンコード・デコード処理
また、本章ではエンコード・デコードを行うコードは全て以下のものを使用します。
```go
type GoStruct struct {
	// フィールドを定義する
}

// Go構造体->json
func encode(stcData GoStruct) {
	jsonData, err := json.Marshal(stcData)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%s\n", jsonData)
}

// json->Go構造体
func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
```
:::

## 大文字・小文字の区別
ここまで扱ってきたjsonは`{"A":1, "B":"bbb"}`という、キーに大文字が含まれているものを使用してきました。
しかし、jsonのキーが大文字というところに違和感を覚える方もいるでしょう。できれば`{"a":1, "b":"bbb"}`のようにキーを小文字にしたいです。
このような場合に対応することは、具体的には「大文字・小文字が一致していないGo構造体フィールドとjsonキーを対応づけること」は可能なのでしょうか。

実際に検証してみましょう。
大文字小文字が一致していないGo構造体に向けて、jsonデコードをしてみます。
```go
type GoStruct struct {
	A    int
	B    string
	Cccc int
	DDdD int
}

func main() {
	fmt.Println("====== json -> struct ======")
	// json -> Go構造体
	// A -> A
	// b -> B (小文字から大文字)
	// cccc -> Cccc (頭文字だけ不一致)
	// ddDd -> DDdD (全ての文字が不一致)
	jsonString := `{"A":3, "b":"bbbbbb", "cccc": 4, "ddDd": 5}`
	decode(jsonString)
}
```

これを実行した結果は以下のようになります。
jsonキーとGo構造体フィールド、互いの大文字・小文字が一致していなくても問題なくデコードが行われることが確認できました。
```bash
$ go run main.go
====== json -> struct ======
{A:3 B:bbbbbb Cccc:4 DDdD:5}
```
| jsonキー名 | →デコード→ | Go構造体のフィールド名 |
| :---: | :---: | :---: |
| `A` | → | `A` |
| `b` | → | `B` |
| `cccc` | → | `Cccc` |
| `ddDd` | → | `DDdD` |

この挙動については、`json.Unmarshal`関数の公式ドキュメントにも明記されています。
> To unmarshal JSON into a struct, Unmarshal matches incoming object keys to the keys (snip) , preferring an exact match **but also accepting a case-insensitive match**.
> 
> (訳)jsonをGo構造体にアンマーシャル(=デコード)する際には、`Unmarshal`関数はそのjsonキーとGo構造体のフィールドを、(中略)フィールド名が完全一致するもの同士、**なければ大文字小文字の区別なしで一致するもの同士で対応づけ**します。
>
> 出典:[pkg.go.dev - encoding/json#Unmarshal](https://pkg.go.dev/encoding/json#Unmarshal)

## 非公開フィールドの扱い
今までは、Go構造体の中には公開フィールドのみを用意してきました。
一方で、Go構造体の中に非公開フィールドがあった場合にはそれはどう扱われるのでしょうか。

先ほど「構造体フィールドとjsonキーの大文字・小文字は区別しない」という話を取り上げたので、非公開フィールドにもマッピングが行えそうにもみえます。
しかし、「非公開フィールドなので、そもそも`encoding/json`パッケージから見えない=エンコード/デコードできないのでは？」とも思えます。

実際にコードを動かして真偽を確かめてみましょう。
```go
type GoStruct struct {
	A int
	B string
	c int		// 非公開フィールド
	d string	// 非公開フィールド
}

func main() {
	// 非公開フィールドc,dもjsonエンコードされるのか？
	stcData := GoStruct{A: 1, B: "bbb", c: 2, d: "ddd"}
	encode(stcData)

	// cキー、dキーの内容は、Go構造体の非公開フィールドにマッピングされるのか？
	jsonString := `{"A":3, "B":"bbbbbb", "c": 4, "d": "dddddd"}`
	decode(jsonString)
}
```
実行結果は以下のようになります。
```bash
$ go run main.go
====== struct -> json ======
{"A":1,"B":"bbb"}
====== json -> struct ======
{A:3 B:bbbbbb c:0 d:}
```

| Go構造体のフィールド名 | →エンコード→ | jsonキー名 |
| :---: | :---: | :---: |
| `A`(公開フィールド) | → | `A` |
| `B`(公開フィールド) | → | `B` |
| `c`(**非**公開フィールド) | → | - (生成されない) |
| `d`(**非**公開フィールド) | → | - (生成されない) |

| jsonキー名 | →デコード→ | Go構造体のフィールド名 |
| :---: | :---: | :---: |
| `A` | → | `A` |
| `B` | → | `B` |
| `c` | → | - (非公開フィールド`c`には値が反映されていない) |
| `d` | → | - (非公開フィールド`d`には値が反映されていない) |

Go構造体からjsonへのエンコードでも、jsonからGo構造体へのデコードであっても、非公開フィールドは無視されることが確認できました。

ここからわかることについてまとめると、以下のようになります。
- Go構造体の非公開フィールドは、例えjsonの形であっても外に公開できない
- jsonからGo構造体の非公開メソッドにアクセス・マッピングすることもできない

`encoding/json`の公式ドキュメントやGo公式ブログの[JSON and Go](https://go.dev/blog/json)でも、このことについて以下のように言及されています。

> Each **exported struct field becomes a member of the object**, using the field name as the object key.
>
> (訳) Go構造体のそれぞれの**公開フィールドは、jsonエンコード時にはオブジェクトプロパティになり**、その際にはフィールド名がキーの名前になります。
>
> 出典:[pkg.go.dev - encoding/json#Marshal](https://pkg.go.dev/encoding/json#Marshal)

> The json package only accesses the exported fields of struct types (those that begin with an uppercase letter).
> Therefore **only the the exported fields of a struct will be present in the JSON output**.
>
> (訳) `encoding/json`パッケージは、構造体型の公開フィールド(=フィールド名が大文字から始まるもの)のみにアクセスすることができます。
> そのため、**jsonエンコード結果に含まれるのは、Goの公開フィールドの値のみ**です。
>
> 出典:[The Go Blog: JSON and Go](https://go.dev/blog/json)


## タグによるマッピング名の変更
ここまで紹介してきた方法は、大文字小文字の違いはあれど「jsonキーとGo構造体のフィールド名が一致している」状態のエンコード・デコードでした。
しかし、全く違う名前のjsonキーとGo構造体フィールドをマッピングしたい場合にはどうしたらいいでしょうか。

`encoding/json`パッケージでは、そのような場合には構造体フィールドにタグをつけることで対応します。

### エンコーディング(Go構造体->json)の場合
例えば、Go構造体に以下のようにタグ付けを行います。
```go
type GoStruct struct {
	A int    `json:"first"`
	B string `json:"second"`
}
```

この状態で、Go構造体`stcData`をjsonエンコードすると以下のようになります。
```go
func main() {
	stcData := GoStruct{A: 1, B: "bbb"}
	encode(stcData)
}
```
```bash
$ go run main.go
{"first":1,"second":"bbb"}
```
`json:"first"`のタグがついたフィールド`A`はjsonでは`first`と、`json:"second"`のタグがついたフィールド`B`はjsonでは`second`とエンコードされました。

| Go構造体のフィールド名 | →エンコード→ | jsonキー名 |
| :---: | :---: | :---: |
| `A`(タグ: `json:"first"`) | → | `first` |
| `B`(タグ: `json:"second"`) | → | `second` |

### デコーディング(json->Go構造体)の場合
エンコーディングの結果を確かめたところで、今度はデコーディングについても検証してみましょう。
Go構造体のタグづけは、デコードにも影響するのでしょうか。

```go
// タグづけを済ませたGo構造体に向けてデコードしたい
type GoStruct struct {
	A int    `json:"first"`
	B string `json:"second"`
}

func main() {
	// jsonキーをGo構造体フィールド名と一致させたパターン
	jsonString1 := `{"A":3, "B":"bbbbbb"}`
	decode(jsonString1)

	// jsonキーをタグと完全一致させたパターン
	jsonString2 := `{"first":4, "second":"bbb"}`
	decode(jsonString2)

	// jsonキーをタグと文字だけ一致させたパターン(大文字・小文字の違いはあり)
	jsonString3 := `{"First":5, "Second":"b"}`
	decode(jsonString3)
}
```
検証にあたって、以下3パターンのjsonを用意しました。
1. jsonキーをGo構造体フィールド名と一致させたパターン(jsonキー: `A` -> Go構造体フィールド: `A`)
2. jsonキーをタグと完全一致させたパターン(jsonキー: `first` -> Go構造体タグ: `first`)
3. 大文字小文字の違いはあれど、jsonキーをタグと文字だけ一致させたパターン(jsonキー: `First` -> Go構造体タグ: `first`)

これらをデコードした結果は、以下のようになります。
```bash
$ go run main.go
====== json -> struct (field) ======
{A:0 B:}

====== json -> struct (tag, lower) ======
{A:4 B:bbb}

====== json -> struct (tag, upper) ======
{A:5 B:b}
```
| jsonキー名 | →デコード→ | Go構造体のフィールド名 |
| :---: | :---: | :---: |
| `A` | → | - (`json:"first"`タグがついた`A`フィールドには反映されない) |
| `B` | → | - (`json:"second"`タグがついた`B`フィールドには反映されない) |
| `first` | → | `A`(タグ: `json:"first"`) |
| `second` | → | `B`(タグ: `json:"second"`) |
| `First` | → | `A`(タグ: `json:"first"`) |
| `Second` | → | `B`(タグ: `json:"second"`) |

タグづけの結果は、デコードにも影響するということが確認できました。

検証の結果判明したことをまとめると、以下のようになります。
- 一度タグをつけてしまうとデコード先となる構造体は、フィールド名ではなくタグ名の方が優先されて決定される。
- タグ名とjsonキーも、大文字小文字の違いは問わず、文字が一致していればマッピングは行われる

## マッピング対象選定の優先順位
さて、ここまでjson-Go構造体のマッピングを行うにあたり、以下のようなケースを考察してきました。
- タグ名・フィールド名とjsonキーの大文字・小文字が一致していなかった場合
- Go構造体フィールドにタグがついていた場合

これらの優先順位について、具体的には「同名のタグがついた構造体フィールド、大文字小文字一致のフィールド、文字だけ一致のフィールド」のようにマッピング条件を満たすフィールドが複数あった場合に、どれが優先されるのかについて調べます。

結果から述べてしまうと、Go公式ブログ記事[JSON and Go](https://go.dev/blog/json)に答えが書いてあります。
`Foo`という名前のjsonキーをマッピングするGo構造体は、以下の順番で選ばれます。
1. `json:"Foo"`タグがついたもの
2. `Foo`フィールド
3. `foo`,`fOO`など、大文字・小文字を無視して`Foo`という名前を持つフィールド





# データスキーマと互換性
さて、Goとjsonの話から脱線しまして、ここで「**データスキーマ**」について述べたいと思います。

## スキーマの種類
「スキーマ」とはよく聞く言葉ではありますが、ここでは
> スキーマ: データ内にどんなフィールドが含まれているのかを保証するもの

と捉えてみましょう。

ちなみに、「スキーマがデータ構造を保証するのはいつなのか？」によってスキーマの種類が2種類存在します。

### スキーマオンリード(schema on read)
「**データ構造は暗黙的なものであり、スキーマはデータの読み取り時にのみ解釈される**」という考え方です。

一例として、NoSQLの一種であるドキュメントDBはスキーマオンリードであるといえます。
![](https://storage.googleapis.com/zenn-user-upload/7827775fef92-20211212.png)
実際にDynamoDBは指定のプライマリーキーさえ存在していれば、他のattributeに制限が加わることはありません。
上図のように、attributeが違う2つのデータを同じテーブルに入れることをDBが拒むことはないわけです。
言い換えるならば「write操作のときに、スキーマによるデータ構造保証は行われない」のです。

ですがDBの中のデータを読み出すときには、開発者は「ある種のattributeをデータが持っていることを前提とした処理」をすることが大半だと思います。
これはつまり「read操作のときにスキーマによる構造保証を期待している」ということです。
NoSQLは「writeのときにデータ構造の縛りがないこと」を根拠にしばしばスキーマレスだと表現されることが多いのですが、正確にいえばオンリードのスキーマを持っているのです。

スキーマオンリードについてまた別の例えをすると、「動的型付け言語が、データ型のチェックを実行時に行うようなもの」とも見ることができます。

### スキーマオンライト(schema on write)
「**データ構造は明示的なものであり、書き込まれるデータは全てスキーマに従ったものであることが保証される**」という考え方です。

![](https://storage.googleapis.com/zenn-user-upload/6a930353d991-20211212.png)
一般的なRDBはスキーマオンライトにあたります。
テーブル作成時に指定されたスキーマに従わないデータを、開発者が書き込むことすらできないわけです。
つまりこれは「書き込み段階でスキーマによるデータ構造を強制する」ということです。

スキーマオンリードのときと同様に、言語を用いた別の例えをするならば「静的型付け言語が、データ型のチェックを実行前、コンパイル時に行うようなもの」とも見ることができます。

## 互換性とは
さて、スキーマを用いれば「ある時点におけるデータ構造」というのは保証できますが、「そのデータ型がずっと使えるのか」という点については少し考える必要があります。

プログラム然り、データ然り、「いついかなる時も同じである」ということはほとんどないでしょう。
わかりやすい例として、プログラムにはバージョンがあり、そのバージョン違いによって生成されるデータ・読み込みたいデータの構造も異なってくるはずです。
つまり、スキーマというのは可変のものであり、場合によっては古いスキーマと新しいスキーマが混在する、ということも考えなくてはならないのです。

そのときに出てくるのが**互換性**という概念です。

### 前方互換性
前方互換性とは「**新しいバージョンによって作られたものを、古いバージョンが扱える**」ことをいいます。

例えるならば「Go1.18で書かれたコードを、Go1.0で実行することができるか」という問題です。
この問題の答えは「不可能」ですので、「Goには前方互換性はない」ということになります。

![](https://storage.googleapis.com/zenn-user-upload/8b05148d46d0-20211211.png)

### 後方互換性
後方互換性とは「**古いバージョンによって作られたものを、新しいバージョンが扱える**」ことをいいます。

例えるならば「Go1.0で書かれたコードを、Go1.18で実行することができるか」という問題です。
これは「可能」ですので、「Goには後方互換性がある」ということになります。

![](https://storage.googleapis.com/zenn-user-upload/827eb3634fc6-20211211.png)

:::message
Goでは、メジャーバージョン1の間は後方互換性が保たれた開発が行われます。
:::

## スキーマと互換性
ここまでの話を一度まとめてみましょう。

- スキーマとは、データ構造を保証するものである
- スキーマが保証するのは「ある時点のデータ構造」であり、スキーマ自体が変わってしまう・新旧のスキーマが混在することもある
- 「あるバージョンのものから生まれた成果物を、違うバージョンのものが扱えるか」という概念を互換性という

すると、「互換性」をスキーマを使って表現するとどうなる？ということを考えたくなります。
前方互換性・後方互換性をスキーマに当てはめると、以下のようになります。
- 前方互換性: 新しいスキーマによって作られたデータを、古いスキーマが読むことができる
- 後方互換性: 古いスキーマによって作られたデータを、新しいスキーマが読むことができる

![](https://storage.googleapis.com/zenn-user-upload/a93df10d530f-20211211.png)





# `encoding/json`デコードにおけるスキーマと互換性
さて、なぜいきなり「スキーマ」「互換性」という話をしたかというと、この話を某本で読んだときに「これGo構造体とjsonで同じことを考えてみたい」と思ったからなのです。

https://www.oreilly.co.jp/books/9784873118703/

Go構造体の型定義は、「その構造体に、どのような型のどのような名前のデータが入っているか」を定めるという点である種のスキーマと捉えることができます。
そこで、ここからは「古いjsonを新しいGo構造体に」もしくは「新しいjsonを古いGo構造体に」マッピングできるか検証し、Goでのjsonデコードで担保できる互換性について考察したいと思います。

:::message
「デコード時の互換性について検証するなら、エンコード時は？」と思った方もいるかもしれません。
しかし、互換性にて担保したい「スキーマがデータを読むことができること」というのは、その性質上「読み込み時にデータ構造を確定させる」つまり「バイト列表現からインメモリ表現への変換」でしか登場しないシチュエーションです。
そのため、スキーマ互換性について考えて意味があるのはデコード時のみです。エンコードは「データに構造を要求しない、バイト列表現への変換」であるため、「スキーマによる構造決定」という概念が入り込む余地はありません。
:::

## 状況設定
Goとjsonに対して「古い」「新しい」という概念が出てきましたが、ここではその新旧を以下のように設定します。

| フィールド | 旧スキーマ | 新スキーマ | 備考 |
| :--- | :---: | :---: | :---: |
| `A` - `int`型 | ○ | ○ |  |
| `B` - `int`型 | ○ | ○ |  |
| `C` - `int`型 | ○ |   | スキーマ更新に伴って削除されるフィールド |
| `D` - `int`型 |   | ○ | スキーマ更新に伴って追加されるフィールド |

## `encoding/json`デコードでの前方互換性
前方互換性は「新しいスキーマによって作られたデータを、古いスキーマが読むことができる」性質です。

ここでは「新しいスキーマから作られたjson」を「古いスキーマのGo構造体が扱えるのか」について検証します。

### 検証と結果
```go
// 古いスキーマをもとに定義されたGo構造体
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"`
}

func main() {
	// 新しいスキーマで生成されたjson
	jsonString := `{"a":1,"b":2,"d":4}`
	decode(jsonString)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
```

このコードの実行結果は以下のようになります。
```bash
$ go run main.go
{A:1 B:2 C:0}
```
| 新スキーマ(json) | →デコード→ | 旧スキーマ(Go構造体) | 備考 |
| :---: | :---: | :---: | :---: |
| `"a":1` | → | `A:1` |  |
| `"b":2` | → | `B:2` |  |
| - | → | `C:0` | スキーマ更新に伴って削除されるフィールド |
| `"d":4` | → | - | スキーマ更新に伴って追加されるフィールド |

### 考察
ここでは各フィールドがどのように対応付いたかということ触れるよりも、そもそも「コンパイル・実行が成功した」という点について注目して述べたいと思います。

```json
// (再掲)新しいスキーマで生成されたjson
{"a":1,"b":2,"d":4}
```
```go
// (再掲)古いスキーマをもとに定義されたGo構造体
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"`
}
```
jsonの中には、どのGo構造体フィールドにも対応付かない`d`キーがありました。
それでもデコードが成功したということは、「**Go構造体にマッピングできないjsonキーがあった場合には、単にそれを無視する**」という挙動を`json.Unmarshal`はするということです。

:::message
この「Go構造体にないキーがjsonに含まれていた場合にはデコード時に無視」という挙動は、「大きいjsonから、一部のフィールドだけ抜き出してGoコード内で使いたい」という場合に威力を発揮します。
:::

さて、「jsonにはキーがあるけどGo構造体にはフィールドがない」というパターンの逆である、「jsonにはキーがないけどGo構造体にはフィールドがある」というパターンについてもみていきましょう。今回の場合、それは「jsonには`c`キーがないのに、Go構造体にはそれと紐づくはずの`C`フィールドがある」ということがどう影響したのか、ということです。
先ほどの結果からもわかるとおり、Go構造体の`C`フィールドには、**`int`型のゼロ値である0が格納されています**。
```bash
{A:1 B:2 C:0}
```
これについては`json.Unmarshal`の仕様というよりはGoそのものの仕様です。
変数宣言時に初期化されたGo構造体の値が`json.Unmarshal`関数によって上書きされないのであれば、初期化時のゼロ値がそのまま残ることになります。

ここで、某本での前方互換性の定義を思い出します。
再掲になりますが、スキーマの観点で前方互換性は「新しいスキーマによって作られたデータを、古いスキーマが読むことができる」と表されます。
これを行うためには、具体的には以下のような挙動にならなくてはならないと本の中では論じられています。

- スキーマ更新に伴って削除されるフィールドは、必須属性がついていないものでなくてはならない
- スキーマ更新に伴って追加されたフィールドを、古いコードは無視しなければならない

:::message
もし必須フィールドをスキーマ更新時に削除してしまった場合、古いコードは「必須フィールドが削除された新しいデータ」を正しく扱うことができません。
:::

これを、Goにおけるjsonデコードが満たしているかどうか、対応づけてみましょう。

| 新スキーマでの変更点 | 某本 | Goのjsonデコード(新json→旧構造体) | 結論 |
| :--- | :--- | :--- | :--- |
| フィールド削除 | 必須属性がついていないものでなくてはならない | 古いコードにある削除されたフィールドにはゼロ値が入る | 「削除されたフィールドがオプションかどうか」は別で確認が必要 |
| フィールド追加 | 追加フィールドを古いコードは無視しなければならない | 追加フィールドを無視してデコード | Goの`json.Unmarshal`は要件を完全に満たしている |

Go構造体には必須フィールド、つまり「構造体のとあるフィールドが非ゼロ値であることを強要する」機構はありません。
そのため、某本で言われているような「スキーマ内に必須フィールドがあって、そこに非ゼロの値が埋まっているかどうか」というのは、`json.Unmarshal`関数以外の場所で別途個別に確認することが求められます。

:::message
ここでは「必須フィールドが埋まっている=非ゼロ値」としましたが、ときにはゼロ値を値なしとみなしたくない場合もあるかと思います。
「ゼロ値と同じ値が入っている状況」と「そもそも値がない状況」を区別したい場合にとるべき手法については後述します。
:::

## `encoding/json`デコードでの後方互換性
今度は後方互換性「古いスキーマによって作られたデータを、新しいスキーマが読むことができる」性質を検証してみましょう。

具体的には「古いスキーマから作られたjson」を「新しいスキーマのGo構造体が扱えるのか」についてみてみます。

### 検証と結果
```go
// 新しいスキーマをもとに定義されたGo構造体
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	//C int `json:"c"`
	D int `json:"d"`
}

// 古いスキーマで生成されたjson
func main() {
	jsonString := `{"a":1,"b":2,"c":3}`
	decode(jsonString)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%+v\n", stcData)
}
```
このコードの実行結果は以下のようになります。
```bash
$ go run main.go
{A:1 B:2 D:0}
```

| 旧スキーマ(json) | →デコード→ | 新スキーマ(Go構造体) | 備考 |
| :---: | :---: | :---: | :---: |
| `"a":1` | → | `A:1` |  |
| `"b":2` | → | `B:2` |  |
| `"c":3` | → | - | スキーマ更新に伴って削除されるフィールド |
| - | → | `D:0` | スキーマ更新に伴って追加されるフィールド |

### 考察
今回も、jsonキーとGo構造体フィールド間で不一致があるのにも関わらず、デコード自体は成功しました。
それには、`json.Unmarshal`が以下のような挙動をしたからです。

- Go構造体のみに追加された新フィールド`D`にはゼロ値が入る
- 新スキーマからは削除されjsonにだけ残っていた`c`キーは、デコード時には無視される

```json
// (再掲)古いスキーマで生成されたjson
{"a":1,"b":2,"c":3}
```
```go
// (再掲)新しいスキーマをもとに定義されたGo構造体
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	//C int `json:"c"`
	D int `json:"d"`
}
```

また、某本での記述についてもみてみましょう。
後方互換性の定義「古いスキーマによって作られたデータを、新しいスキーマが読むことができる」を満たすためには、具体的には以下のような挙動にならなくてはならないと論じられています。

- スキーマ更新に伴って削除されたフィールドに関するデータは、デコード時に無視しなければならない
- スキーマ更新に伴って追加するフィールドに、必須属性をつけてはならない

:::message
もしスキーマ更新時に追加する新フィールドに必須属性をつけてしまった場合、新しいコードは「新スキーマにて追加された必須フィールドが含まれていない古いデータ」を正しく扱うことができません。
:::

この某本での後方互換性の定義を、Goにおけるjsonデコードが満たしているかどうか比べてみましょう。

| 新スキーマでの変更点 | 某本 | Goのjsonデコード(旧json→新構造体) | 結論 |
| :--- | :--- | :--- | :--- |
| フィールド削除 | 削除されたフィールドに関するデータは、デコード時に無視しなければならない | 削除済みフィールドのデータは無視してデコード | Goの`json.Unmarshal`は要件を完全に満たしている |
| フィールド追加 | 追加フィールドに必須属性をつけてはならない | 古いjsonから得られなかった追加フィールドにはゼロ値が入る | jsonからデータが得られなかった場合にゼロ値になることを許容する必要がある |

フィールド削除とフィールド追加における挙動が、前方互換性のときと逆になっているのがお分かりいただけると思います。





# `encoding/json`デコード 〜 応用編
ここからは、少し工夫したデコードのやり方について紹介します。

## フィールドを削除しない場合の扱い方
### 状況説明
後方互換性について論じる際に、「`C`フィールドを新しいスキーマでは削除する」というのを、「実際に`C`フィールドをコメントアウトさせる」という方法で実装しました。
```go
// Cフィールドを削除
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	//C int `json:"c"`
	D int `json:"d"`
}
```
しかし、「Goコードそのものの後方互換性の問題から、直接コメントアウト削除させたくない」という場合にはどうしたらいいでしょうか。
```go
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"` // コメントアウトせずにDeprecated扱いにしたい
	D int `json:"d"`
}

func main() {
	jsonString := `{"a":1,"b":2,"c":3}`
	decode(jsonString)
}

func decode(jsonString string) {
	// (略: 上記で使用したものと同じ)
}
```
このままデコードを実行すると、以下のようにjsonの`c`キーの値までGo構造体に読み込まれてしまいます。
いらない`C`フィールドの値がゼロ値ではない値で埋まっている、というのは少々誤解を招く仕様かと思うので、何とかしたいです。
```bash
// DeprecatedにしたいフィールドCに非ゼロ値が入ってしまう
$ go run main.go
{A:1 B:2 C:3 D:0}
```

### 解決策
jsonの`c`キーをデコード時に無視するためには、タグを変えてやる必要があります。
`C`フィールドにつけるタグを、`json:"c"`から`json:"-"`に変更してみましょう。
```go
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"-"`
	D int `json:"d"`
}

func main() {
	jsonString := `{"a":1,"b":2,"c":3}`
	decode(jsonString)
}
```
すると、「Go構造体の`C`フィールドがゼロ値のまま」という所望の結果が得られます。
```bash
// jsonキー`"c": 3`がデコード時に無視される
$ go run main.go
{A:1 B:2 C:0 D:0}
```

## 「ゼロ値」と「値なし」の区別をつける方法
ゼロ値を値なしとみなしたくない場合、つまり「ゼロ値と同じ値が入っている状況」と「そもそも値がない状況」を区別したい場合について論じたいと思います。

例えばフィールド`C`にそのような条件を課したいとします。
この場合、`C`フィールドを`int`ではなくて`*int`のようにポインタ型にしてしまいましょう。
```go
type GoStruct struct {
	A int  `json:"a"`
	B int  `json:"b"`
	C *int `json:"c"`
}
```

この型定義を使って、「値が0の`c`キーが存在するjson」と「そもそも`c`キーが存在しないjson」をデコードしてみましょう。
```go
func main() {
	// そもそもcキーが存在しないjson
	jsonString1 := `{"a":1,"b":2}`
	decode(jsonString1)

	// 値が0のcキーが存在するjson
	jsonString2 := `{"a":1,"b":2,"c":0}`
	decode(jsonString2)
}

func decode(jsonString string) {
	var stcData GoStruct

	if err := json.Unmarshal([]byte(jsonString), &stcData); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("%+v ", stcData)
	if stcData.C != nil {
		fmt.Printf("C:%d", *stcData.C)
	}
	fmt.Printf("\n")
}
```
結果は以下のようになります。
```bash
$ go run main.go
// そもそもcキーが存在しないjson
{A:1 B:2 C:<nil>} 

// 値が0のcキーが存在するjson
{A:1 B:2 C:0xc0000b4110} C:0
```
json内での`c`キーの有無によって、Go構造体の`C`フィールドの値が違っているのが分かります。
- jsonに`c`キーがあった場合: `c`キーの値が格納されている場所へのポインタ
- jsonに`c`キーがなかった場合: `nil`

`encoding/json`パッケージ内にも、jsonの`null`値をデコードすることについて以下のように述べられています。
> Because null is often used in JSON to mean “not present,” **unmarshaling a JSON null into any other Go type has no effect on the value and produces no error**.
>
> (訳) null値はjsonの中で「値がない」ことを示すためによく使われるため、**null値のjsonをデコードしても、対象となったGoの構造体値には何の影響も及ぼさない&エラーも発生させないようになっています**。
>
> 出典:[pkg.go.dev - encoding/json](https://pkg.go.dev/encoding/json#Unmarshal)

## 任意の構造のjsonをデコードする方法
### 状況説明
ある種の構造体型に向けてデコードすると、先にも述べたとおり「そのjsonキーに対応する構造体フィールドがなかった場合にはそれは無視されてしまう」という挙動をします。
```go
// (再掲) jsonキーのdはデコード時に無視される
type GoStruct struct {
	A int `json:"a"`
	B int `json:"b"`
	C int `json:"c"`
}

func main() {
	jsonString := `{"a":1,"b":2,"d":4}`
	decode(jsonString)
}

// 結果
// {A:1 B:2 C:0}
```
事前に`a`,`b`,`c`というフィールドがあることを仮定してGo構造体を定義しているので、デコード結果もそれに引きづられる形で`d`キーが無視されてしまいます。
これはつまり`encoding/json`でのデコードというのは「デコード先のGo構造体型の構造に引きづられる」というスキーマオンライトの一面があると見ることができます。

ですが「どんなjsonキーがあったとしても、取りこぼすことなくデコードしたい」「事前に来るjsonの構造がわからない」という場合にはどうしたらいいのでしょうか。
このような場合には、静的型付け言語であるGoではなす術がないのでしょうか。

### 解決策
実は、[Go公式Blog - JSON and Go](https://go.dev/blog/json)の中の "Decoding arbitrary data" という項目に、まさにこのようなシチュエーションのときに役立つ手法について述べられています。

それは「`json.Unmarshal`関数にて指定するデコード先に、`interface{}`を指定する」という方法です。
```go
func main() {
	jsonString := `{"a":1,"b":2,"d":4}`

	var Data interface{}
	if err := json.Unmarshal([]byte(jsonString), &Data); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf("%T, %+v\n", Data, Data)
}
```
```bash
$ go run main.go
map[string]interface {}, map[a:1 b:2 c:3]
```
`interface{}`へのデコードにすると、結果はjsonの中身がそのまま反映されたmapになります。
mapのkeyはjsonキーがそのまま`string`の形で入っており、valueは`interface{}`型で対応プロパティが入っています。

このmapから欲しい値を取得した後に型アサーションをすれば、Goの静的型付けの恩恵を受けながらも擬似的なスキーマオンリードのデコードをすることができます。




# まとめ
というわけで、Goにおけるjsonエンコード・デコードについて色々と考察してきました。

本記事は「Goでjsonを扱うにあたって、何か特別新しい書き方・事実が見つかった！」という内容ではないですが、「Go構造体やjsonデコードというのをちょっと違った切り口で見るとこんな感じになりますよ」というサンプルを提供できればと思い書きました。
私自身もまだ元ネタとなった本を全て噛み砕けているわけではなく、まだまだ理論としては完成形じゃないなーという感覚があるので、同じテーマで違う見方、もっと深い考察ができた！という方がいれば、ぜひコメント欄にてシェアしていただければ嬉しいです。

あと今日12/13は、実は私の誕生日です。
記事を落とさずちゃんとリリースできたことも含めて祝ってください。

# 参考文献
- [Go公式Blog - JSON and Go](https://go.dev/blog/json)
- [オライリー データ指向アプリケーションデザイン(元ネタの本)](https://www.oreilly.co.jp/books/9784873118703/)