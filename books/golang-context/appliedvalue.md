---
title: "Valueメソッドを有効に使うtips"
---
# この章について
前章でも説明した通り、contextへの値付加というのは
- keyとvalueはcontextを介した時点で全て`interface{}`型になる
- 見方を変えると「引数となりうる値を、contextで隠蔽している」という捉え方にもなる

という点で、扱い方が難しい概念です。

この章では、「contextのvalueを、危うさなしに使うにはどういう設計にしたらいいか」ということについて考察していきたいと思います。



# contextに与えるkeyの設定
## keyに設定できる値
> The provided key must be comparable.
> (訳) keyに使用する値は比較可能なものでなくてはなりません。
> 
> 出典: [pkg.go.dev - context.WithValue](https://pkg.go.dev/context@go1.17#WithValue)

これはよくよく考えてもらえば当たり前のことをいってるな、ということがわかると思います。
contextの`Value(key)`メソッドにて「引数に与えたkeyを内部に持つvalueがないかな」という作業をすることを想像すると、「引数とcontextが持っているkeyは**等しいかどうか(=比較可能かどうか)**」ということが決定できないといけないのです。

比較可能(comparable)な値の定義については、[Goの言語仕様書](https://golang.org/ref/spec#Comparison_operators)に明確に定義されています。
- bool値は比較可能であり、`true`同士と`false`同士が等しいと判定される
- 整数値(int, int64など), 浮動小数点値(float32, float64)は比較可能
- 複素数値は比較可能であり、2つの複素数の実部と虚部が共に等しい場合に等しいと判定される
- 文字列値は比較可能
- ポインタ値は比較可能であり、「どちらも同じ変数を指している場合」と「どちらも`nil`である場合」に等しいと判定される
- チャネル値は比較可能であり、「どちらも同様の`make`文から作られている場合」と「どちらも`nil`である場合」に等しいと判定される
- インターフェース値は比較可能であり、「どちらも同じdynamic type・等しいdynamic valueを持つ場合」と「どちらも`nil`である場合」に等しいと判定される
- 非インターフェース型の型`X`の値`x`と、インターフェース型`T`の値`t`は、「型`X`が比較可能でありかつインターフェース`T`を実装している場合」に比較可能であり、「`t`のdynamic typeとdynamic valueがそれぞれ`X`と`x`であった場合」に等しいと判定される
- 構造体型はすべてのフィールドが比較可能である場合にそれ自身も比較可能となり、それぞれの対応するnon-blankなフィールドの値が等しい場合に2つの構造体値が等しいと判定される
- 配列型は、その配列の基底型が比較可能である場合にそれ自身も比較可能となり、全ての配列要素が等しい場合に2つの配列値は等しいと判定される

逆に、スライス、マップ、関数値などは比較可能ではない(not comparable)ため、contextのkeyとして使うことはできません。

:::message
**dynamic type/valueとは何か？**
変数定義時に明確に型宣言されていない場合において、コンパイル時にそれに適した型・値であるdynamic type/valueが与えられます。
```go
// staticなtype・valueの例
var x interface{}  // x is nil and has static type interface{}
var v *T           // v has value nil, static type *T

// dynamicなtype・valueの例
x = 42             // x has value 42 and dynamic type int
x = v              // x has value (*T)(nil) and dynamic type *T
```
コード出典:[Go言語仕様書#Variables](https://golang.org/ref/spec#Variables)
:::


## keyの衝突
contextに与えるkeyについて、注意深く設計していないと「keyの衝突」が起こる可能性があります。

### 悪い例
#### 状況設定
`hoge`と`fuga`2つのパッケージにて、同じkeyでcontextに値を付加する関数`SetValue`を用意しました。

```go
// hoge
func SetValue(ctx context.Context) context.Context {
	return context.WithValue(ctx, "a", "b") // hoge pkgの中で("a", "b")というkey-valueを追加
}

// fuga
func SetValue(ctx context.Context) context.Context {
	return context.WithValue(ctx, "a", "c") // fuga pkgの中で("a", "c")というkey-valueを追加
}
```

そして、`main`関数内で作ったcontextに、`hoge.SetValue`→`fuga.SetValue`の順番で値を付加していきます。
```go
import (
	"bad/fuga"
	"bad/hoge"
	"context"
)

func main() {
	ctx := context.Background()

	ctx = hoge.SetValue(ctx)
	ctx = fuga.SetValue(ctx)

	hoge.GetValueFromHoge(ctx) // hoge.SetValueでセットしたkey"a"に対するValue(="b")を見たい
	fuga.GetValueFromFuga(ctx) // fuga.SetValueでセットしたkey"a"に対するValue(="c")を見たい
}
```

値を付加した後に、それぞれの`GetValueFromXXX`関数で実際にどんなvalueが格納されているのか確認しています。
```go
func GetValueFromHoge(ctx context.Context) {
	val, ok := ctx.Value("a").(string)
	fmt.Println(val, ok)
}

func GetValueFromFuga(ctx context.Context) {
	val, ok := ctx.Value("a").(string)
	fmt.Println(val, ok)
}
```

#### 結果
これを実行すると、以下のようになります。
```bash
$ go run main.go
c true  // hoge.GetValueFromHoge(ctx)からの出力
c true  // fuga.GetValueFromFuga(ctx)からの出力
```
`hoge`パッケージの中でcontextに値`"b"`を付加していたのに、`hoge.GetValueFromHoge`関数で確認できたvalueは`"c"`でした。
これは、`hoge`と`fuga`で同じkey`"a"`を利用してしまったため、key`"a"`に対応するvalueは、後からSetした`fuga`の方の`"c"`が使用されてしまうのです。

### 解決策: パッケージごとに独自の非公開key型を導入
このようなkeyの衝突を避けるために、Goでは「keyとして使用するための独自のkey型」を導入するという手段を公式で推奨しています。
`context.WithValue`関数の公式ドキュメントにも、以下のような記述があります。

> The provided key should not be of type string or any other built-in type to avoid collisions between packages using context. 
> **Users of WithValue should define their own types for keys.**
>
> (訳)異なるパッケージ間でcontextを共有したときのkey衝突を避けるために、keyにセットする値に`string`型のようなビルトインな型を使うべきではありません。
> その代わり、**ユーザーはkeyには独自型を定義して使うべき**です。
>
> 出典: [pkg.go.dev - context.WithValue](https://pkg.go.dev/context@go1.17#WithValue)

#### コード改修
`hoge`,`fuga`パッケージの中身を、それぞれ以下のように改修します。
```diff go
+// hoge

+type ctxKey struct{}

func SetValue(ctx context.Context) context.Context {
-	return context.WithValue(ctx, "a", "b")
+	return context.WithValue(ctx, ctxKey{}, "b")
}

func GetValueFromHoge(ctx context.Context) {
-	val, ok := ctx.Value("a").(string)
+	val, ok := ctx.Value(ctxKey{}).(string)
	fmt.Println(val, ok)
}
```
```diff go
+// fuga

+type ctxKey struct{}

func SetValue(ctx context.Context) context.Context {
-	return context.WithValue(ctx, "a", "c")
+	return context.WithValue(ctx, ctxKey{}, "c")
}

func GetValueFromFuga(ctx context.Context) {
-	val, ok := ctx.Value("a").(string)
+	val, ok := ctx.Value(ctxKey{}).(string)
	fmt.Println(val, ok)
}
```
`hoge`,`fuga`パッケージ共に`ctxKey`型という非公開型を導入し、それぞれ`ctxKey`型の値をkeyとしてcontextに値を付与しています。

この改修を終えた後に、先ほどと同じ`main`関数を実行したらどうなるでしょうか。

#### 結果
```bash
$ go run main.go
b true  // hoge.GetValueFromHoge(ctx)からの出力
c true  // fuga.GetValueFromFuga(ctx)からの出力
```
無事衝突することなく、`hoge.GetValueFromHoge`関数からは`hoge`パッケージで付加されたvalue`"b"`が、`fuga.GetValueFromFuga`関数からは`fuga`パッケージで付加されたvalue`"c"`が確認できました。

これは、contextに付与された値のkeyがそれぞれ
- `hoge`パッケージ内: `hoge.ctxKey`型の値
- `fuga`パッケージ内: `fuga.ctxKey`型の値

であるからです。
各パッケージ内で独自の型を作ったことにより、`hoge`と`fuga`パッケージ双方空構造体で同じ見た目の値になったとしても、型が異なるので違う値扱いになり衝突しなくなるのです。
また、独自型を非公開にすれば、keyの衝突を避けるためには「`hoge`パッケージ内で同じ型のkeyを使ってないか」「`fuga`パッケージ内で同じ型のkeyを使っていないか」というところのみ気にすればいいので、contextが断然扱いやすくなります。

また、同じパッケージ内でのkey衝突に関しても、都度空構造体をベースとした異なる型を定義してそれを利用することで簡単に回避可能です。

:::message
go-staticcheckという静的解析ツールでは、独自非公開型を定義せずにビルトイン型(`int`や`string`のように、Goに元からある型)をkeyにしている`context.WithValue`関数を見つけると、
`should not use built-in type xxxx as key for value; define your own type to avoid collisions (SA1029)`
という警告が出るようになっています。
:::

# valueとして与えてもいいデータ・与えるべきでないデータ
「contextの値として付加するべき値はどのようなものがふさわしいか？」というのは、Goコミュニティの中で盛んに議論されてきたトピックです。
数々の人が様々な使い方をして、その結果経験則として分かったことを一言でいうならば、

> **Use context Values only for request-scoped data that transits processes and APIs**, not for passing optional parameters to functions.
> 
> (訳)contextのvalueは、関数のoptionalなパラメータを渡すためにではなく、**プロセスやAPI間を渡り歩くリクエストスコープなデータを伝播するために使うべき**である。
>
> 出典: [pkg.go.dev - context](https://pkg.go.dev/context@go1.17#pkg-overview)

これについて、もっと深く具体例を出しながら論じていきましょう。

## valueとして与えるべきではないデータ
### 関数の引数
関数の引数となるべきものを、contextの値として付加するべきではありません。
「関数の引数とは何か？」ということをはっきりさせておくと、ここでは「その関数の**挙動**を決定づける因子」としておきましょう。

例えば、以下のようなコードを考えます。
```go
func doSomething(ctx context.Context) {
	isOdd(ctx) // ctxに入っているkey=numに対応する値が、奇数かどうかを判定する関数
}

func main() {
	ctx := context.Background()
	ctx = prepareContext1()
	ctx = prepareContext2()
	ctx = prepareContext3()

	doSomething(ctx)
}
```
これには問題点があります。

- コメントがないと「`isOdd`関数は、contextの『`num`』というkeyの偶奇を見ているんだな」という情報がわからない
- `doSomething`関数の引数として渡されているcontextが、いつどこで`key=num`の値を付加されているのかが非常に分かりにくい
- contextにどのような値が入っているのかがわからないので、`isOdd`関数の結果がどうなるのか予想が非常につきにくい

簡単にいうと、`isOdd`関数の**挙動**を決めるための引数がcontextの中に**隠蔽**されてしまっているため、非常に見通しがつきにくいコードになってしまっているのです。

それでは、`isOdd`関数の挙動を決める「判定対象の数値」を、`isOdd`関数の引数にしたらどうなるでしょうか。
```go
func doSomething(ctx context.Context, num int) {
	isOdd(num) // numが奇数かどうか判定する関数
}

func main() {
	ctx := context.Background()
	ctx = prepareContext1()
	ctx = prepareContext2()
	ctx = prepareContext3()

	num := 1

	doSomething(ctx, num)
}
```
こうすることで、

- `isOdd`関数が見ているのは、引数の`num`のみだということが明確
- 「`doSomething`関数内で呼ばれている`isOdd`関数の挙動を決定するのは、`main`関数内で定義されている変数`num`である」ということが明確
- コードの実行結果が、`num=1`であるため奇数判定されるだろうという予測が容易に立つ

という点で非常に良くなりました。

繰り返しますが、「関数の挙動を決める変数」というのは、引数の形で渡すべきです。contextの中に埋め込む形で隠蔽するべきではありません。

### type-unsafeになったら困るもの
再び先ほどの`isOdd`関数の例を挙げてみましょう。

contextを使った`isOdd`関数の実装は以下のようになっていました。
```go
const num ctxKey = 0

func isOdd(ctx context.Context) {
	num, ok := ctx.Value(num).(int) // 型アサーション
	if ok {
		if num%2 == 1 {
			fmt.Println("odd number")
		} else {
			fmt.Println("not odd number")
		}
	}
}

func doSomething(ctx context.Context) {
	isOdd(ctx) // ctxに入っているkey=numに対応する値が、奇数かどうかを判定する関数
}
```
`isOdd`関数の中で、contextから得られるkey=numの値が、`int`型に本当になるのかどうかを確認するアサーション作業が入っているのがわかるかと思います。
これは、「contextに渡した時点で、keyとvalueは`interface{}`型になってしまう」ゆえに起こる現象です。

```go
// WithValueで渡した時点でkeyもvalueもinterface{}型になり、元の型情報は失われてしまう
func WithValue(parent Context, key, val interface{}) Context

// 当然、取り出す時も型情報が失われたinterface{}型となる
type Context interface {
	Value(key interface{}) interface{}
}
```

`isOdd`関数の引数に判定対象`num`を入れてしまう形ならば、型アサーションを排除することができます。
これは、関数の引数としてなら、変数`num`の元の型である`int`を保全することができるからです。
```go
func isOdd(ctx context.Context, num int) {
	// 型アサーションなし
	if num%2 == 1 {
		fmt.Println("odd number")
	} else {
		fmt.Println("not odd number")
	}
}

func doSomething(ctx context.Context) {
	isOdd(ctx, 1) // 第二引数を、奇数かどうかを判定する関数
}
```

contextに渡した値は、`interface{}`型となって型情報が失われるということを意識するべきです。
そのため、type-unsafeになったら困る値をcontextに渡すべきではありません。

### 可変な値
今度は先ほどの`isOdd`関数を、以下のように使ってみましょう。
```go
func doSomethingSpecial(ctx context.Context) context.Context {
	return context.WithValue(ctx, num, 2)
}

func main() {
	ctx := context.Background()
	ctx = context.WithValue(ctx, num, 1)

	isOdd(ctx) // odd

	ctx = doSomethingSpecial(ctx)

	isOdd(ctx) // ???
}
```
`main`関数内で与えたcontextの値は当初`1`だったので、`isOdd`関数の結果は「奇数」判定されるでしょう。
しかし、その後に`doSomethingSpecial`という全然スペシャルではない関数の実行が挟まれています。
そのため、`isOdd(ctx)`という呼び出しの字面は同じでも、2回目の`isOdd`関数の結果が1回目のそれと同じになるかどうか、というのが一目ではわからなくなってしまいました。

これも先ほど述べた内容ではあるのですが、contextの中に値を付与するというのは下手したら「context中に変数を隠蔽する」ということにもなりかねます。
そのため、「contextの中には何が入っているのか？」の見通しを良くするために、contextに渡す値というのは不変値が望ましいでしょう。

### ゴールーチンセーフではない値
そもそも、contextは「複数のゴールーチン間で情報伝達をするための仕組み」でした。
そのため、contextに渡すvalueというのも、異なる並行関数で扱われることを想定して、ゴールーチンセーフなものにする必要があります。

> The same Context **may be passed to functions running in different goroutines**
> 
> (訳)同一のcontextは、**異なるゴールーチン上で動いている関数に渡される可能性があります。**
>
> 出典: [pkg.go.dev - context](https://pkg.go.dev/context@go1.17#pkg-overview)

ゴールーチンセーフでない値の例として、スライスが挙げられます。
例えば以下のようにゴールーチンを10個立てて、それらの中で個別にあるスライス`slice`に要素を一つずつ追加していったとしても、最終的な`len(slice)`の値が`10`になるとは限りません。
これは、スライスがゴールーチンセーフではなく、`append`の際の排他処理が取れていないからです。
```go
func main() {
	var wg sync.WaitGroup
	wg.Add(10)

	slice := make([]int, 0)
	for i := 0; i < 10; i++ {
		go func(i int) {
			defer wg.Done()
			slice = append(slice, i)
		}(i)
	}

	wg.Wait()
	fmt.Println(len(slice)) // 10になるとは限らない
}
```
繰り返しますが、contextにゴールーチンセーフでない値を渡すべきではありません。
その部分を担保するのは、Goの言語仕様ではなくGoを利用するプログラマ側の責任です。

## valueに与えるのがふさわしい値
それでは逆に、「contextに渡してやった方がいい値」というのはなんでしょうか。

渡すべきではない値の条件を全て避けようとすると、条件は以下のようになります。
1. 関数の挙動を変えうる引数となり得ない
2. type-unsafeを許容できる
3. 不変値
4. ゴールーチンセーフ

そして、contextというのは本来「異なるゴールーチン上で情報伝達するための機構」なのです。
これらの条件を鑑みると、自ずと使用用途は限られます。
それは「**リクエストスコープ**な値」であることです。

### リクエストスコープとは？
リクエストスコープとは、「一つのリクエストが処理されている間に共有される」という性質のことです。
例を挙げると、

- ヘッダから抜き出したユーザーID
- 認証トークン
- トレースのためにサーバー側でつける処理ID
- etc...

です。これらの値は、一つのリクエストの間に変わることがなく、リクエストを捌くために使われる複数のゴールーチン間で共有されるべき値です。
