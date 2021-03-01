---
title: "Goの言語仕様書精読のススメ & 英語彙集"
emoji: "📝"
type: "idea" # tech: 技術記事 / idea: アイデア
topics: [go, 初心者]
published: false
---
# この記事について
Go言語公式から提供されている[The Go Programming Language Specification](https://golang.org/ref/spec)という文章があります。
![](https://storage.googleapis.com/zenn-user-upload/tj58u1t53u3m78f00p7u0mg4we8s)
*実際のThe Go Programming Language Specificationのページ画面*

この文章、個人的にはじっくり読んでみると結構得るものが大きいな、と感じるものです。本記事では
- The Go Programming Language Specificationって何が書いてあるの？
- 読んだら何がわかるの？
- 読むときにはどういうところに注目したらいいの？
- 英語難しいから単語教えて！

という疑問に答えながら、The Go Programming Language Specification精読の布教を行います。

# The Go Programming Language Specification とは？
[The Go Programming Language Specification](https://golang.org/ref/spec)とは、Goの言語仕様が書いてある文書です。

:::message
"The Go Programming Language Specification"と名前が長いので、以下これを"GoSpec"と略して表現します。
:::
言語仕様なので、「どんな型があるのか」「どんな演算子があるのか」というおなじみの内容だけでなく、「そもそもソースコード中に使える文字は何か」や「ある型の変数に値を代入できる条件は何か」といった地固め的な内容も含まれます。

以下の表で、GoSpecの章立てとそこで定義されている内容についてまとめました。

| 章 | 定義されている内容 |
| ---- | ---- |
| Introduction | --- |
| Notation | EBNFによる表記方法 |
| Source code representation | Unicodeのどの文字を使用するか |
| Lexical elements | 字句解析におけるtokenをどう定めるか |
| Constants | 「定数」そのものとその型 |
| Variables | 変数宣言と型 |
| Types | Goで使用できる型の仕様 |
| Properties of types and values | 型と変数に絡んだ概念(型同一性など) |
| Blocks | コードブロック |
| Declarations and scope | 〇〇宣言と表現されるもの全てとそのスコープ範囲 |
| Expressions | Goで使える式 |
| Statements | Goで使える文 |
| Built-in functions | ビルトイン関数 |
| Packages | パッケージ宣言やインポート宣言といったパッケージ周りの記述方式 |
| Program initialization and execution | 変数・パッケージの初期化の挙動とプログラム実行開始位置 |
| Errors | エラーインターフェース |
| Run-time panics | ランタイムパニック |
| System considerations | その他補足事項 |

# 読んで得られたもの
- 「今更言語仕様書なんて読まなくても、私Go言語そこそこ書けるからだいじょーぶ！」
- 「こんな長くて難しい文章読んで何が嬉しいの？？？」
- 「読んだらどういうスキルが身につくの？」

と思っているそこのあなたに、筆者が感じた「GoSpecを読んで得られそうなもの」を挙げていきたいと思います。

## 重箱の底にあるような細かい仕様がわかる
早速ですが問題です。
以下のコードを実行したらなんと出力されるでしょう。
```go
var x int8 = -128
fmt.Println(x/-1)
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/F3bYa1i2RXt)。

驚くことに、答えは`-128`です。
「負の数を負で割ったのに答えが負なの？？Why？？？」と思うでしょう。私も最初はそう思いました。
この挙動については、GoSpecの[Arithmetic operators](https://golang.org/ref/spec#Arithmetic_operators)の章に、以下のように記述されています。
>  if the dividend x is the most negative value for the int type of x, the quotient q = x / -1 is equal to x (and r = 0) due to two's-complement integer overflow.
>
> (和訳) 割られる数`x`がその型で表現可能な最小の負値である場合、2の補数表現でのオーバーフロー防止のために、`x/-1`の商`q`の値は`x`と等しくなります。(そして余り`r`も`0`になります。)

今回の例の場合、2の補数表現を使って8bit(=`int8`)で表現できる数の範囲は`-128`~`127`です。そのため、数学でやるように商を`128`としてしまうと、解が`int8`に収まらないのです。そのため、このような特殊な挙動を定義しているのです。

こういうところからGo Quiz[^1]のネタができるんだなあという感想を持ちました。
[^1]: Go Quizについては[Goクイズ Advent Calendar 2020](https://qiita.com/advent-calendar/2020/goquiz)がまとまったコンテンツとして存在します。

## デバッグ力がつく
### 例その1
「なんでこんな記述あるんだ？？？」というほど、当たり前のように思える文章に出会うことがあります。
例えば、以下の記述を見てみましょう。
> The return parameters of the function are passed by value back to the caller when the function returns.
>
> (和訳) 関数の戻り値は、return文が呼ばれ復帰するときに呼び出し元に値渡しされる。

`v := f()`とかいう記述のときに、関数`f()`の戻り値の値が`return`文が呼ばれたときの値になり、それが`v`に入るのは当たり前のように感じます。
しかし、以下のような関数に副作用があるケースを考えてみましょう。

```go
var s = 1

func f() int {
	s++
	return s
}


func main() {
	v := f()
	v++
	fmt.Println(s)
	fmt.Println(v)
}
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/Uao0ZWq9Tn7)。

このとき、出力される値は`2`と`3`です。
`v := f()`が呼び出されたときの戻り値`s`の値は`2`です。そのため`v`には`2`が格納されます。その後`v`がインクリメントされていますが、`v := f()`では**値渡し**をされているので、このインクリメントによって`s`が影響を受けることはありません。そのためこのような結果になります。

といったように、書いてある内容を検証する際に、「この場合はどうだ？」「あの場合はどうだ？」と考えるうちにぶっとんだ例を思いつけるように訓練されていきます。
おそらくこれはデバックするときに使う思考回路と似たものがあるのではないでしょうか。

### 例その2
例えば、関数の引数として可変数引数を渡せるときに、その引数として何も指定しなかった場合、どうなるでしょうか。

```go
func main() {
	// こう呼び出したときに引数pは何になるのか？
	// nil？ それとも空スライス？
	f()
}

func f(p ...string) {
	fmt.Println(p == nil)
}
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/sls09uIEk_o)。

このときの挙動は[Expressions-Passing arguments to ... parameters](https://golang.org/ref/spec#Passing_arguments_to_..._parameters)の部分に明確に記されています。

> If f is invoked with no actual arguments for p, the value passed to p is nil.
>
> (和訳)もし関数`f`が、可変数引数`p`に何も渡さない状態で呼び出された場合、`p`に渡される値は`nil`となります。

このように「可変数引数に何も渡さないと`nil`になる」などの、特殊なパターンでのデフォルト値・ゼロ値を頭に入れておくことで、例外分岐やデバックがスムーズになりそうな感じがします。


## コンパイラが何を見ているのかがなんとなくわかる
GoSpecは「人間がGoコードを解釈する」ために書かれたのではなく、「コンパイラがGoコードを解釈する」ために書かれた側面が強いです。

例えば随所に散りばめられているEBNF記述です。以下は[If statements](https://golang.org/ref/spec#If_statements)の節に記載された、`if`文をどのように記述するのかを[EBNF](https://ja.wikipedia.org/wiki/EBNF)というメタ文法記法で表現したものです。
```
IfStmt = "if" [ SimpleStmt ";" ] Expression Block [ "else" ( IfStmt | Block ) ] .
```
これを見て「Goで`if`文ってこんな感じに書けばいいんだ！」といってコードをゴリゴリ書けちゃう人は本当にごく少数じゃないでしょうか。
この`IfStmt`の記述は、「コンパイラがこのEBNFに基づいて構文解析をしています」ということを主張する内容です。つまり「まず`"if"`という文字列があって、次に条件を示す`SimpleStmt`が規定の書き方であって……」ということをコンパイラはやっているのです。なんとなく雰囲気がわかるでしょうか。

また、他にもGoSpec本文中には「字句解析で使うトークンの種類」だったり「この演算はコンパイル時にこんな感じに最適化されることがあります」といったコンパイラのための記述がたくさんあります。
そのため、読み進めていくとコンパイラの気持ちがなんとなくわかるようになっていく……かもしれません。

# あまり期待しない方がいいもの
そして、「これを期待して読むのはちょっとやめた方がいいんじゃないかな？」ということも一応述べておきます。
## Go初学者の方が文法をこれをみて学ぼうとすること
繰り返しますが、これは「コンパイラがGoコードを解釈する」ために書かれたものであって、「人間がGoコードを解釈する」ためのものではありません。
GoSpecはインターネット上でタダで見られる文章ではあるのですが、Goの基本的な書き方を普通に知りたいというかたは、お金を浮かせようとはせずに普通に市販の本を買った方がいいでしょう。


# 読むときに気を付けること
ここからは、ただ読むだけではなく、読んできちんと得るものを得るためにはどういう意識を持った方がいいか？ということについて、筆者が感じたことを書いていきます。
## その記述があることで何が嬉しいのか？ということを意識する
GoSpecは言語仕様書であり、全ての記述には意味があるはずです。

例えば[Types](https://golang.org/ref/spec#Types)の章で導入される「全ての型にはunderlying typeがある」という概念は、一見すると「なんでこんなものを導入するんだ？？」と思うかもしれません。
> Each type T has an underlying type: If T is one of the predeclared boolean, numeric, or string types, or a type literal, the corresponding underlying type is T itself. Otherwise, T's underlying type is the underlying type of the type to which T refers in its type declaration.
>
> (和訳) 全ての型`T`にはそれぞれunderlying typeが存在します。`T`がブール型・数値型・文字列型・型リテラルの場合、underlying typeは`T`そのものになります。それ以外の場合、underlying typeは`T`の型定義に使われている型のunderlying typeと同じになります。

しかし、このunderlying typeはassignability(代入可能性)を定義するために不可欠なものです(後述)。

このように「この記述はどこで役に立つの？」という意識を持つことで、全体像の理解につながります。


## 常に例を考える
某書籍に「**例示は理解の試金石**」という言葉があります。
https://twitter.com/hyuki/status/985985934452649984
これは本当にそうで、読んだ内容を元に「こういうコードはこの記述を元に確かにこういう挙動をする」という例が作れるかどうかで理解の深さが段違いだという実感があります。

### 具体例作りの具体例
例えば、「ある値`x`が型`T`の変数に代入可能(assignable)である」ことの条件の一部として、以下のような定義があります。
> 1. `x`の型`V`と型`T`が同一(identical)のunderlying typeをもち、かつ型`V`と型`T`の少なくともどちらか一つがdefined typeでないこと
> 2. `x`がuntypedな定数であり、かつ型`T`で表現可能(representable)であること

例えば、以下のように定義された`MyIntSlice`型に`[]int`を代入するのはOKです。
```go
type MyIntSlice []int

func main() {
	var src = []int{0, 1, 2}
	var dst = MyIntSlice{3, 4, 5}
	
	dst = src
	
	fmt.Printf("%T, %v\n", dst, dst)
}
/// main.MyIntSlice, [0 1 2]
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/gb7uN9cztj4)。

なぜこのような挙動になるのかというと、`MyIntSlice`と`[]int`のunderlying typeはどちらも`[]int`で同じであり、かつ`[]int`はdefined typeでないからです。
これはassignableの定義1にぴったり当てはまります。

ですが、以下の`MyInt`型変数に`int`型を代入するのはNGです。
`MyInt`と`int`のunderlying typeはどちらも`int`で一致しますが、どちらもdefined typeなので代入できないのです。
```go
type MyInt int

func main() {
	var src int = 1
	var dst MyInt = 2
	
	dst = src
	
	fmt.Printf("%T, %v\n", dst, dst)
}
// ./prog.go:13:6: cannot use src (type int) as type MyInt in assignment
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/Vg475rldLVu)。


では、以下のように一部をコメントアウトしたらどうなるでしょうか？
```diff go
type MyInt int

func main() {
-	var src int = 1
	var dst MyInt = 2
	
-	dst = src
	
	fmt.Printf("%T, %v\n", dst, dst)
}
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/KcrKqbs-4bt)。

やっていることとしては、`MyInt`型に`2`を代入していることです。さっきもコンパイルエラーになったんだから今回もダメなんじゃないの？というかたは残念不正解です。

正解は`main.MyInt, 2`と出力され、きちんと期待通りの動作が得られます。これは変数`dst`に代入している`2`は、扱いとしては`int`型ではなく「untypedな定数」となるからです。
assignableの定義2から、`2`は`MyInt`に代入可能というわけです。

### 具体例でわかったこと
このように、代入可能性の例を作る過程において
- ある型をみてそのunderlying typeがわかるかどうか
- defined type, untyped, 表現可能性といった概念の理解

が問われるわけで、その結果「『ある変数にある値を代入できるか？』というのは『当たり前』で済ませる概念ではない」という感覚が養えるのです。

## 言語仕様書内で出てくる固有概念を体にしみつかせる
では、前述したように具体例をスラスラ出せるようにするためにはどうしたらいいでしょう。
それには、例えば「untypedな定数とは何？具体的にはどう作るの？」「defined typeってなんだっけ？」とかそういう固有概念がパッと出てくるようにしないとダメなんです。

固有概念の定義についてきっちり体に染み込ませておけば、「これとそれは違う」という小さな差異にも気付ける・頭を巡らせられるようになります。
そうすると「じゃあここを変えたらどういう挙動になる？」と検証するような具体例が作れるようになります。

例えば、GoSpec内では[定数](https://golang.org/ref/spec#Constants)と[変数](https://golang.org/ref/spec#Variables)は別章に分けられています。つまり2つは「違うもの」なのです。
この意識があると、以下のように「シフト回数に負定数使っちゃダメって書いてあるけど、じゃあ変数使っててそれが負だった場合どうなる？」という発想が出てきやすいです。
```go
func main() {
	const negConst int = -1
	var negVar int = -1
	
	_ = 1 << negVar  // panic: runtime error: negative shift amount
	_ = 1 << negConst  // invalid negative shift count: -1
}
```
The Go Playgroundでの実行は[こちら](https://play.golang.org/p/q1RT8LgyUFw)。




# 読み進めるにあたって重要な概念
ここからは、筆者が実際にGoSpecを読み進めるにあたって「この知識・英単語を知ってるとスムーズかも？」と思ったものを書き留めたものです。**よく知っている内容があったら適宜飛ばして読んでください。**
## コンパイラまわりの知識
コンパイラは、原始プログラム(=ソースコード)から目的プログラムを生成するツールです。
つまり、人間が書いたソースコードを、機械語翻訳されたオブジェクトコードを生成するのがコンパイラが行う処理です。
この処理過程は以下の5段階に分けることができます。

1. 字句解析
ソースコードを字句単位に分割します。
例えば、`a := 1 + 2`というソースコードを字句に分割したら、`a` `:=` `1` `+` `2`となります。
2. 構文解析
分割された字句が、文法的に正しく並んでいるかをチェックします。一般的に構文木が作れるかどうかで確認されます。
3. 意味解析
構文が確定した後に、その字句が変数なのか、型はあっているのかといった意味内容面をチェックします。
4. 最適化
効率よくプログラムを実行するための最適化を行います。
例えば、乗算`x*2`を左シフト`x<<1`に直して効率化するなど。
5. 機械語生成

:::message
ちなみにGoのコンパイラgcについてはこちらの記事が入門内容としてわかりやすいです。
:::
https://www.ebiebievidence.com/posts/2020/12/golang-compiler/

これを前提にしたらわかる！という英単語がいくつかあります。

### token
字句解析で分割生成される「字句」「単語」「トークン」のこと。lexical tokenともいう。
:::message
ちなみに字句解析は英訳するとlexical analysis.
:::
[GoSpec](https://golang.org/ref/spec#Tokens)によると、Goではtokenは4種類に分けられる、とされています。

- keywords
- operators and punctuation(演算子と句読点)
- literals
- identifiers

`x` `:=` `1` `+` `23` `;`と分けられて、それぞれ identifier, operator, literal, operator, literal, punctuation となる。

### keyword
いわゆる予約語のこと。

そのままキーワードと訳すなかれ。プログラミングの世界においては「予約語」をこういうことが多いです。少なくとも[C言語](https://en.cppreference.com/w/c/keyword)と[JavaScript](https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Lexical_grammar#keywords)はそう言っているのが確認できます。

### literal
リテラルは、ざっくりいうとソースコード内に直接書かれた値のことです。
(例)`1`(int), `2.71828`(float), `1.e+0i`(complex), `'a'`(rune), `"abc"`(string)

このように例であげたリテラル以外にも、Goで「リテラル」と呼ばれるものは存在します。
例えば型リテラルです。`[]int`とか`*int`とかはそれぞれintスライス型、intポインタ型を表す型リテラルです。

なんでこれがint型リテラルやstringリテラルと同じ型"リテラル"という名前なの？という説明としてはこちらのtenntennさんのツイートがわかりやすいです。
https://twitter.com/tenntenn/status/1362352486611185666

Goでリテラルと呼ばれるものを挙げると以下の通り。
- 整数リテラル
- 小数点リテラル
- 複素数リテラル
- runeリテラル
- 文字列リテラル
- 型リテラル
- compositeリテラル
- 構造体リテラル
- 配列リテラル
- スライスリテラル
- マップリテラル
- 関数リテラル

~~いっぱいありすぎでは？？~~

### identifier
変数名や型名を表す識別子・名前のこと。
ざっくりいうなら、トークン(字句)の中で、リテラルでも演算子でも予約語でもないものというのが手っ取り早い(要検証)。

このidentifierという単語は、GoSpec本文中に今後よく出てくることになります。本当に。IT用語じゃなくて一般名詞だと思ってしまうと読解が難しくなってしまうので注意しましょう。


## EBNFの概念
EBNFはプログラム言語の文法を記述するためのメタ言語の一つです。記述の終わりは`.`をつけることでで表されます。

GoSpecの[Notation](https://golang.org/ref/spec#Notation)章には、EBNFの記法をEBNF自体で表して定義しています。
```
Production  = production_name "=" [ Expression ] "." .
Expression  = Alternative { "|" Alternative } .
Alternative = Term { Term } .
Term        = production_name | token [ "…" token ] | Group | Option | Repetition .
Group       = "(" Expression ")" .
Option      = "[" Expression "]" .
Repetition  = "{" Expression "}" .
```
これは要するに以下の内容をいっているだけにすぎません。

- 何か新しいものに名前をつけて定義したいときは、`production_name = (定義内容)`と書けばOK
- `|`で区切ることで、orを表現できる
- "要素"を一個以上並べてフレーズ(=Alternative[^2])を作ることができる
- "要素"の定義は「EBNFで定義された何かの名前 or token or `()`か`{}`か`[]`で囲ったもの」
- `()`で囲うことで要素をグループ化できる
- `[]`で囲うと、中身を0回または1回繰り返すことを示す
- `{}`で囲うと、中身を0回以上繰り返すことを示す

[^2]:Alternativeは直訳が「選択肢」です。つまり、`|`で区切ることで「A or B」とするときの「A」や「B」を選択肢とみている故の単語チョイスです。
:::message
`Alternative`の定義でも使われている`xxx {xxx}`という表現は、EBNFにおいて「中身を1回以上繰り返す」というイディオムみたいなものです。
:::

### 具体例
`Alternative`の一例として、[Struct types](https://golang.org/ref/spec#Struct_types)の節で登場する`FieldDecl`を紹介します。
```
FieldDecl     = (IdentifierList Type | EmbeddedField) [ Tag ] .
```
この`FieldDecl`は、`(IdentifierList Type | EmbeddedField)`というグループ要素と`[Tag]`という要素を2個羅列しているAlternativeだと読めます。



## Go特有の概念
underlying typeやassignableといった、Go特有の固有概念を辞書順に並べました。
「この概念があることでどこで嬉しいのか？」「何に使われているのか？」というところを意識した方がいいのがこのへんの単語です。

### addressable
あるオペランドが[addressable](https://golang.org/ref/spec#Address_operators)であるということは、
- 変数
- ポインタ変数をデリファレンスしたもの
- スライス`a`のインデックスを指定して`a[i]`としたもの
- addressableな構造体変数のフィールドセレクタ`stc.fld`
- addressableな配列`a`のインデックスを指定して`a[i]`としたもの

であると定義されている。
要するに、addressableであるということは「そのポインタが取得できる」というオペランドの性質のことをいいます。

オペランドにある演算や操作を行うための前提条件として、addressableであるということが随所で出てきます。
(例) [slice expression](https://golang.org/ref/spec#Slice_expressions), [インクリメント/デクリメント](https://golang.org/ref/spec#IncDec_statements)

### assign to a variable
「変数に値を代入する」という言い回し。

> (例文) It is the most recent value **assigned to** the variable.

### assignable(assignability)
ある値`x`が型`T`にassignableであるということは、「型`T`の変数に代入することができる」という値`x`の性質のことです。
定義内容については[GoSpec本文](https://golang.org/ref/spec#Assignability)を参照のこと。

### base type
何型に対してのポインタなのかというのを、[base type](https://golang.org/ref/spec#Pointer_types)と表現されます。type identityの定義にからむ概念となります。
例えば、`int`型へのポインタ`*int`型のbase typeは`int`型です。

また、メソッドについているレシーバーの型名についても「レシーバーの[base type](https://golang.org/ref/spec#Method_declarations)」と表現されます。
(例) `Read`メソッド`func (p T) Read(p []byte) (n int, err error)` のbase typeは型`T`。

### bind
識別子(identifier)に変数とか定数とかパッケージ名とか、何かを紐付けすることをbindするといいます。
> (例文) A constant declaration **binds** a list of identifiers (the names of the constants) to the values of a list of constant expressions.

また、メソッドを型に対応させることもbindすると表現されます。
> (例文) The method is said to **be bound** to its receiver base type and the method name is visible only within selectors for type T or *T.

### constants
定数。

「リテラルと何が違うの？」と思った方々、奇遇ですね、私と同じです。
> **A constant value** is represented by a rune, integer, floating-point, imaginary, or string literal, an identifier denoting a constant, a constant expression, a conversion with a result that is a constant, or the result value of some built-in functions such as unsafe.Sizeof applied to any value, cap or len applied to some expressions, real and imag applied to a complex constant and complex applied to numeric constants. 
(出典) https://golang.org/ref/spec#Constants

要するに、定数は

- 各種リテラル
- 定数を示す識別子
- constant expression(`const a = 2`みたいな表現)
- `len(a)`みたいなある種の組み込み関数

などで表せるもので、リテラルはその中の一つにすぎないということです。

### conversion
いわゆる「[型キャスト](https://golang.org/ref/spec#Conversions)」のこと。GoSpec本文中ではtype castingではなくconversionという表現に統一されています。
随所にこの単語は出てくるため、意味を把握しておくとスムーズ。

:::message
インターフェース型を特定型の変数に代入する[型アサーション](https://golang.org/ref/spec#Type_assertions)との違いに注意。
:::

### default type
untypedな定数に、型を暗黙的に与える必要が出てきたときに当てがわれる型のことを「そのuntypedな定数がもつ[default type](https://golang.org/ref/spec#Constants)」と表現します。

例えば、`i := 0`と書かれたときに、識別子`i`の型は明示的には定義されていません。このときに、`i`の型について議論しなくてはいけないときには、これは一旦0のdefault typeである`int`型と解釈されます。

GoSpec本文中に登場する回数は5回。忘れた頃に出てきて困惑する単語です。

### defined type
`int`や`string`といった、Goで元々用意されている型と、ユーザーによって`type`を使い定義された型の2つをdefined typeと呼びます。

のびしーさん(@shino_nobishii)が作られたこちらの資料で非常にわかりやすく説明されているため、詳細はそちらに譲ります。
https://docs.google.com/presentation/d/1JSsrv404ZDJSnxp4UcJ3iVYDJAiT3RyETMe9AvbkxuY/edit#slide=id.p

### dynamic type
インターフェース型の変数に、その中に代入された値にしたがって実行時に与えられる動的な型のことを[dynamic type](https://golang.org/ref/spec#Variables)という。
型アサーションを説明する章で重要になり、多用される概念です。

対義語はstatic type。

### expression
式。評価された結果、値(value)になるもののことを[expression](https://golang.org/ref/spec#Expressions)と表現される。

例えば`a[x]`という表現は、「配列orスライス`a`の`x`番目の値」などと解釈でき、値を得ることができるため、これはIndex expressionsという名前がついている。

:::message
値を返さない文(statement)との違いに注意。
:::

### identical<->different
2つの型が同じか/違うかどうかという概念のこと。

例えば、以下の型定義を見ると
```go
type MyInt = int	// identical
type MyString string	// different
```
`MyInt`は型エイリアスなので`int`型と同一(identical)、`MyString`は新しくこの名前の型を宣言しているので`string`型とは違う(different)型となります。

conversion(型キャスト)やassignable(代入可能性)を定めるのに型の同一性(type identity)は重要な概念となります。
詳細な定義に関しては[GoSpec本文](https://golang.org/ref/spec#Type_identity)を参照してください。

### interface
インターフェース型だけじゃなくて、インターフェースに付属するメソッドセットのことも[interface](https://golang.org/ref/spec#Method_sets)と呼ぶことがあります。

(例文) An interface type specifies a method set called its **interface**.

### numeric
直訳は数値。
GoSpec本文中では「`rune`, `int`, `float`, `complex`」の型をまとめてnumeric typeと呼ぶことが多く、何回も出てくるワードです。

### predeclared identifier
直訳すると「宣言済みの識別子」。

予約語(keyword)とは別の概念になります。例えば`iota`や`true`はpredeclared identifierですがkeywordではありません。
predeclared identifierのリストは[GoSpec本文](https://golang.org/ref/spec#Predeclared_identifiers)を参照してください。

### promoted
[struct type](https://golang.org/ref/spec#Struct_types)の節にのみに登場する概念。

例えば、`MyStruct`型の構造体が
```go
type MyStruct {
	Ownfield int
	EmbeddedMyStruct
}
```
のように`EmbeddedMyStruct`型の埋め込みフィールドを持っていたとします。
このとき、`EmbeddedMyStruct`型のフィールドorメソッド`f`が、`MyStruct`型の正当なセレクタになるのであれば、`Mystruct`型の変数`x`から`x.f`という風に直接これを呼び出すことができるようになります。
このとき、フィールドorメソッド`f`がpromotedされたと表現されます。

### representable(representability)
定数`x`が型`T`でrepresentableであるとは、ざっくりいうと「型`T`で表現可能な値の範囲に入っている」ということです。

例えば、`1000`は`int8`ではrepresentableではありませんが、`int16`ではrepresentableです。
:::message
`int8`と`int16`の範囲はそれぞれ-128~127、-32768~32767です。
:::

representableの詳細な定義は[GoSpec本文](https://golang.org/ref/spec#Representability)を参照のこと。

### selector
構造体フィールドやメソッドを表す表現`x.f`の`f`の部分のことを「セレクタ」と呼びます。

### signature
関数における「引数と返り値のセット」のことをGoSpec本文中ではsignatureと表現しています。
> (例文) Such a declaration provides the signature for a function implemented outside Go, such as an assembly routine.

### statement
文。`if`や`for`といった、値を返さない表現のこと。
:::message
値を返す式(expression)との違いに注意。
:::

### static type
dynamic typeに対して、直接定義された型や、`new`を使って与えられた静的な型をstatic typeと呼びます。
> (例文) The static type (or just type) of a variable is the type given in its declaration, the type provided in the new call or composite literal, or the type of an element of a structured variable.

### underlying array
スライスの内部実装の中には、「ある配列へのポインタ」と「ある配列の中で参照できる部分の長さ(=len)」と「ある配列の長さ(=cap)」が含まれています。
その「ある配列」のことをunderlying arrayと表現しています。

詳しくは以下のGo Blogをご覧ください。
https://blog.golang.org/slices-intro

### underlying type
Goでは全ての型に[underlying type](https://golang.org/ref/spec#Types)というものが定義されていて、それがtype identityやassignableの定義設定に大いに関わってきます。

DQNEO(@DQNEO)さんがまとめたこちらのスライドがわかりやすいので、詳細説明はそちらに譲ります。
https://speakerdeck.com/dqneo/go-language-underlying-type

### unique(uniqueness)
ある識別子がunique(同一)であるかどうかというのはきちんと[Uniqueness of identifiers](https://golang.org/ref/spec#Uniqueness_of_identifiers)という節で定義されています。
いわゆる「構造体の中に同じ名前のフィールドやメソッドがあってはいけませんよ」という決まりの「同じ」という部分の正確な定義を行っているということです。
> (例文) In a method set, each method must have a **unique** non-blank method name.

### untyped
型が明確に宣言されていない定数のことをuntypedと呼びます。

例えば、以下の2つの定数のうち、`Pi`はfloat型と明示されているのでtypedですが、`zero`はこの時点ではuntypedです。
```go
const Pi float64 = 3.14159265358979323846
const zero = 0.0         // untyped floating-point constant
```
そのため、定数`zero`の型の解釈が必要になった場合は、default typeが割り当てられることになります。




## プログラム全般について
Goだけではなくて、IT関係の英文を読むときに一般によく出てきそうな単語をまとめました。
### argument
値のある実引数のことをGoSpec本文ではargumentと表す傾向にあります。
> (例文) A new, initialized channel value can be made using the built-in function make, which takes the channel type and an optional capacity as **arguments**: `make(chan int, 100)`

例文でいうと、`make`関数の第2引数は、すでに`100`という実値が与えられているのでargumentとなったのだと思います。
:::message
parameterとの比較に注意です。
:::

### assignment
名詞で「代入」の意。

> (例文) Strings can be concatenated using the `+` operator or the `+=` **assignment** operator.

### base
n進数の基数のこと。
> (例文) An optional prefix sets a non-decimal **base**: 0b or 0B for binary, 0, 0o, or 0O for octal, and 0x or 0X for hexadecimal.

また、後に出てくるbase prefixは、2進数であることを表す`0b`や8進数を表す`0o`といった接頭辞を示します。

### byte-wise
「1byteごとに」という意味。

> (例文) String values are comparable and ordered, lexically **byte-wise**.

### compatibility
互換性。特に、backward compatibilityで後方互換性という意味になります。

### concurrent programming
並列処理のこと。
> (例文) Go has explicit support for **concurrent programming**.

### decimal
「小数の」という意味も存在しますが、GoSpec本文中では「10進数の」という意味で用いられています。
> (例文) A single 0 is considered a **decimal** zero.

### implementation-specific
「実装依存」の意。
> (例文) There is a predeclared numeric types with **implementation-specific sizes**: `uint     either 32 or 64 bits`

### indice
indexと同じ意味。

"index"の単語のほうがGoSpec本文中には多く出てくるのですが、"indice"もたまに出てきてびっくりします。

### operand
オペランドのこと。日本語にすると「被演算子」。

例えば`1+2`という式があったときに、`+`が演算子で`1`と`2`がオペランド(被演算子)となります。
> (例文) The comparison operators == and != must be fully defined for **operands** of the key type.

### parameter
決まった値がまだ与えられていない仮引数に用いられる傾向にある言葉です。
> (例文) A function type denotes the set of all functions with the same **parameter** and result types.

例文の場合、「関数型は同じ引数と返り値をもつ全ての関数を表すことができる」という内容で、ここでの引数に具体的な値はまだ想定されていません。そのため、parameterが使われているのかと思います。

:::message
argumentとの比較に注意です。
:::

### parenthesize
「括弧にいれる」という動詞。
> (例文) Parameter and result lists are always **parenthesized** except that if there is exactly one unnamed result it may be written as an unparenthesized type.

### pass by value
「値渡し」することをこう表現します。
> (例文) The parameters of the call are **passed by value**.

### pointer indirection
indirectionは間接参照の意味です。pointer indirectionはポインタの間接参照、つまりポインタ変数をデリファレンスして値を得る操作のことを指します。
> (例文) For an operand `x` of pointer type `*T`, the **pointer indirection** `*x` denotes the variable of type `T` pointed to by `x`.

### portability
「移植性」という名詞。
> (例文) To avoid **portability** issues all numeric types are defined types and thus distinct except byte, which is an alias for uint8, and rune, which is an alias for int32. 

### precision
数値の精度。`int`型や`float`型を何bitで表すか？という話です。
> (例文) Although numeric constants have arbitrary **precision** in the language, a compiler may implement them using an internal representation with limited precision.

### property
直訳すると「属性」。jsonや構造体のプロパティを見るとわかるように、何かにつける名前のようなものです。
> (例文) Programs are constructed from packages, whose **properties** allow efficient management of dependencies.

筆者は「パッケージのプロパティって何やねん！？」となりました。ここでいうとパッケージをそれと識別するためのパッケージ名ですね。

### radix, mantissa
それぞれ浮動小数点の基数、仮数のこと。

### recursively
「再帰的に」という副詞。
> (例文) An interface type T may not embed itself or any interface type that embeds T, **recursively**.

### result value
「戻り値」の意。
> (例文) Two function types are identical if they have the same number of parameters and **result values**, corresponding parameter and result types are identical, and either both functions are variadic or neither is.

### two's complement arithmetic
2の補数表現のこと。

### unary, binary
それぞれ「単一項の」「二項の」という意味の形容詞です。
主に引数・戻り値の数を言及するときに出てくる単語です。
> (例文) Primary expressions are the operands for **unary** and **binary** expressions.

:::message
関連単語にvariadicがあります。
:::

### uniform pseudo-random
「一様擬似乱数」という名詞。

### variadic
「(関数が)可変長引数の」という形容詞です。
> (例文) A function with such a parameter is called **variadic**.
:::message
関連単語にunary/binaryがあります。
:::



## 文字的なこと
Unicodeの「文字」に関する英単語です。
### canonicalized
Unicode文字には正規化という概念が存在し、それを表す単語です。
> (例文) The text is not **canonicalized**, so a single accented code point is distinct from the same character constructed from combining an accent and a letter.

例文では「Goではソースコードの正規化は行わない」と記述されています。つまり、「â」1文字と、「a+ ̂」の2文字の組み合わせでâにしたものは別物として扱うということです。~~日本語話者ではわからないよこんなの。~~

### upper/lower case
それぞれ「大文字」「小文字」の意。
> (例文) The first character of the identifier's name is a Unicode **upper case** letter (Unicode class "Lu");


## 英語辞書
ただの英単語帳です。~~大学受験とかで使えそう~~
### allocate
動詞で「割り当てる」。メモリ割り当て関連の話で頻出する単語です。
> (例文) A slice created with make always **allocates** a new, hidden array to which the returned slice value refers.

### correspond
「一致する」だけではなく「相応する、相当する」という意味もあり、後者がGoSpecではよく出てきます。
> (例文) If `T` is one of the predeclared boolean, numeric, or string types, or a type literal, the **corresponding** underlying type is `T` itself.

### denote
「示す」という動詞。めちゃくちゃ出てくる。死ぬほど出てくる。
> (例文) A type may be **denoted** by a type name.

### dividend, divisor
それぞれ「割られる数」「割る数」という意味です。
::: message
`q = x / -1`のとき、`x`はが割られる数で`-1`は割る数。
:::

### embed
「埋め込む」という動詞。
GoSpec本文だと、構造体の埋め込みフィールド周りの話でよく出てきます。
> (例文) Further rules apply to structs containing **embedded** fields, as described in the section on struct types.

### inhabit
「ファイルを同じディレクトリ内にいれる」というのをinhabitと表現するんですね。
> (例文) An implementation may require that all source files for a package **inhabit** the same directory.

### innermost
「最も内側の」という形容詞。
> (例文) The scope of a constant or variable identifier declared inside a function begins at the end of the ConstSpec or VarSpec (ShortVarDecl for short variable declarations) and ends at the end of the **innermost** containing block.

:::message
関連用語としてleftmostがあります。
:::

### invoke/ invocation
動詞で「呼び出す」の意。名詞形にするとinvocation。
> (例文) A function literal can be assigned to a variable or **invoked** directly.

### leftmost
「一番左の」という形容詞。
> (例文) The <- operator associates with the **leftmost** chan possible:
:::message
関連用語としてinnermostがあります。
:::

### legal <-> illegal
それぞれ直訳すると「合法な」「違法な」という形容詞。
illegalは「やってはいけない書き方」を表す単語としてGoSpec本文中に頻出します。
> (例文) It is **illegal** to define a label that is never used.

### parentheses, square brackets, or curly braces
それぞれ(), [], {}のカッコを表します。

### respective
各自の、それぞれという形容詞。
> (例文) The types of the elements and keys must be assignable to the **respective** field, element, and key types of the literal type

### retrieve
「検索する」という動詞。
add-retrieve-removeの三拍子の中ではあまり出てこないので忘れがち。
> (例文) Elements may be added during execution using assignments and **retrieved** with index expressions;

### stand for
表す、象徴するという熟語。
> (例文) If present, each name **stands for** one item (parameter or result) of the specified type and all non-blank names in the signature must be unique.

### successive
「連続する」という形容詞。
"success"は「成功」という意味だけではないというのが英語的ポイントです。
> (例文) Within a constant declaration, the predeclared identifier iota represents **successive** untyped integer constants. 

### take O as argument
「引数をとる」の動詞はtakeを使うんですね。
> (例文) A new, empty map value is made using the built-in function make, which **takes** the map type and an optional capacity hint **as arguments**.

### take the type
英語では型はtakeするものだそうです。
> (例文) If the type is present, all constants **take the type** specified

### valid <-> invalid
直訳するとそれぞれ「有効」「無効」という形容詞です。
これも、ある書き方が正しい記述方式かどうか、ということを議論する文脈で用いられることが多いです。
> (例文) The type assertion is invalid since it is not possible for `x` to store a value of type `T`.

### yield
「産出する」という動詞。
> (例文) The expression **yields** a function equivalent to Mv but with an explicit receiver as its first argument;

# まとめ
以上、「The Go Programming Language Specificationを読もうぜ！」という布教文章でした。

ちなみに筆者は[#gospecreading](https://gospecreading.connpass.com)というイベントに2~3回参加して、そのときの知識内容をもとに書いただけの素人です。
プロの方で追記修正して欲しいことがあれば、コメントいただければ対応します。記事の内容がより厚くなるようなご指摘は大歓迎ですのでどしどし教えてください。