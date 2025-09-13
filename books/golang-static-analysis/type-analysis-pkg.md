---
title: "型チェック - typesパッケージ"
---
# この章について
ここからは、構文解析ではなく、もう一段階深堀ったソースコード分析を行う**型チェック**を解説していきたいと思います。


# 構文解析ではできないこと
ASTを用いた構文解析には型の情報は含まれていないため、型を用いたルールチェックロジックは書くことができません。

例えば、以下の二項演算式を見てみます。
```go
10 + "aaaa"
```
この式は右辺の型(`int`)と左辺の型(`string`)が異なるため、コンパイルすることはできません。
しかしASTの観点で見ると、`+`の演算子の右辺と左辺にそれぞれ識別子があるという二項演算で満たすべき形にはなっているため、おかしな書き方/シンタックスではないと判定されてしまいます。

つまり、Goの文法でおかしな構造になっていないかどうかという構文解析からさらに深く踏み入ったチェック項目として、「型がおかしくないかどうか」というプログラムの意味内容まで触れた項目も存在するのです。
それを行うのが型チェックと呼ばれる分析です。









# `go/types`パッケージを用いた型解析
型チェックを行うためのパッケージ[`go/types`パッケージ](https://pkg.go.dev/go/types)がGo標準には用意されています。
その`go/types`パッケージ内にどんな構造体・メソッド・関数が用意されており、どんなことができるのかということを理解することで、型チェックでどのような解析ロジックを書くことができるのかよりイメージしやすくなるかと思います。

そのためここからは、その`go/types`パッケージの内容について詳しく説明していきます。

## `go/types`パッケージで出来る処理
まずは実際に解析ツールを実装するにあたって有用になる、`go/types`パッケージに定義されている関数を紹介します。
- `AssertableTo`関数
- `AssignableTo`関数
- `Comparable`関数
- `ConvertibleTo`関数
- `Identical`関数
- `IdenticalIgnoreTags`関数
- `Implements`関数
- `IsInterface`関数
- `Satisfies`関数

### [`AssertableTo`関数](https://pkg.go.dev/go/types#AssertableTo)
```go
func AssertableTo(V *Interface, T Type) bool
```
第一引数`V`のインターフェース値が、第二引数`T`の型に型アサーションが可能かどうかを判定する関数です。

例えば、`any`型の変数`x`を、`int`型に型アサーションするときには`x.(int)`と書くことになります。
```go
var x any
x = 10

value, ok := x.(int)
```
`AssertableTo`関数では、このようなアサーションが成功するかしないかを判定することができます。

### [`AssignableTo`関数](https://pkg.go.dev/go/types#AssignableTo)
```go
func AssignableTo(V, T Type) bool
```
第一引数`V`の型の値が、第二引数`T`の型に代入可能かどうかを判定する関数です。

例えば、このような代入が型安全に行えるかどうかを事前に判定することができます。
```go
var x int = 10
var y interface{} = x  // int型の値をinterface{}型に代入
```

### [`Comparable`関数](https://pkg.go.dev/go/types#Comparable)
```go
func Comparable(T Type) bool
```
第一引数`T`の型が比較可能かどうかを判定する関数です。

Goの言語仕様において「比較可能」というのは、「`==`や`!=`で値の等価性を判定できる」ことを意味しています。
例えば、スライスやマップ、関数型などは比較できませんが、基本型や構造体などは比較可能です。
```go
var a []int
var b []int
// a == b  // これはコンパイルエラー（スライスは比較不可能）

var c int = 10
var d int = 20
// c == d  // これはOK（intは比較可能）
```

### [`ConvertibleTo`関数](https://pkg.go.dev/go/types#ConvertibleTo)
```go
func ConvertibleTo(V, T Type) bool
```
第一引数`V`の型の値が、第二引数`T`の型に変換可能かどうかを判定する関数です。

例えば、以下のような型変換が可能かどうかを判定できます。
```go
var x int = 10
var y float64 = float64(x)  // int型からfloat64型への変換
```

### [`Identical`関数](https://pkg.go.dev/go/types#Identical)
```go
func Identical(x, y Type) bool
```
第一引数`x`と第二引数`y`の型が同一（identical）かどうかを判定する関数です。

Goでは型の同一性は厳密に定義されており、たとえ基盤となる型が同じでも、名前付き型は異なる型として扱われます：
```go
type MyInt int
var a int = 10
var b MyInt = 20
// aとbは異なる型（intとMyInt）
```

### [`IdenticalIgnoreTags`関数](https://pkg.go.dev/go/types#IdenticalIgnoreTags)
```go
func IdenticalIgnoreTags(x, y Type) bool
```
構造体のフィールドタグを無視して、第一引数`x`と第二引数`y`の型が同一かどうかを判定する関数です。

構造体のフィールドタグは型の同一性には影響しないため、タグが異なっても構造が同じであれば同一とみなします。
```go
type User1 struct {
    Name string `json:"name"`
}

type User2 struct {
    Name string `yaml:"name"`
}
// User1とUser2はタグは異なるが、構造が同じなので同じ型と判定される
```

### [`Implements`関数](https://pkg.go.dev/go/types#Implements)
```go
func Implements(V Type, T *Interface) bool
```
第一引数`V`の型が、第二引数`T`のインターフェースを実装しているかどうかを判定する関数です。

例えば、自作の構造体が`io.Writer`インターフェースを実装しているかどうかを判定できます。
```go
type MyWriter struct{}

func (m MyWriter) Write(p []byte) (n int, err error) {
    return len(p), nil
}
// MyWriterはio.Writerインターフェースを実装している
```

### [`IsInterface`関数](https://pkg.go.dev/go/types#IsInterface)
```go
func IsInterface(t Type) bool
```
第一引数`t`の型がインターフェース型かどうかを判定する関数です。

### [`Satisfies`関数](https://pkg.go.dev/go/types#Satisfies)
```go
func Satisfies(V Type, T *Interface) bool
```
第一引数`V`の型が、第二引数`T`のインターフェースの制約を満たすかどうかを判定する関数です。

`Implements`関数と似ていますが、`Satisfies`はより広い概念で、型制約の満足度を判定します。特にジェネリクスの型制約などで使用されます。

:::message
ここに登場した「型の同一性」「代入可能性」などの概念も、Goの言語仕様に定義されている内容です。
ここの意味がわからなかった……という方は、一度Goの言語仕様書を眺めてみるとより深く理解できるかと思います。
- [The Go Programming Language Specification](https://go.dev/ref/spec)

また、いきなり言語仕様書を読むのは難しすぎて無理だったという方には、筆者が過去に書いたリファレンス解説記事があるのでそちらもご覧ください。
- [Goの言語仕様書精読のススメ & 英語彙集](https://zenn.dev/hsaki/articles/gospecdictionary)
:::

## `types.Type`型と`types.Object`型
ここまで紹介してきた`go/types`パッケージの便利な関数は、全て引数に`types.Type`型や`types.Object`型を取っています。
そのため、自作解析ツール内で型に関する処理をしたい場合には、ASTノードから`types.Type`型/`types.Object`型に変換してからこれらの関数を利用することになります。

ASTノードからこれらの構造体への変換方法は後ほど紹介するとして、まずはこの`types.Type`型と`types.Object`型が何者なのかを説明します。

### `types.Type`インターフェース
[`types.Type`インターフェース](https://pkg.go.dev/go/types#Type)は、解析で使うことになるGoにおける型情報を表現したインターフェースです。
```go
type Type interface {
	// Underlying returns the underlying type of a type.
	// Underlying types are never Named, TypeParam, or Alias types.
	//
	// See https://go.dev/ref/spec#Underlying_types.
	Underlying() Type

	// String returns a string representation of a type.
	String() string
}
```

解析をする上で、プログラム中の識別子がスライス型なのかポインタ型なのかなど、どんな種別の型の識別子なのかは気になるポイントとなってきます。
スライス型ではないなら`a[i]`のようなインデックス式は書けませんし、ポインタ型ではないなら`*a`のようなでリファレンス処理はエラーになるべきです。
このような判断を解析ツール内で行うために、静的解析ツール開発者はASTノードの型情報が欲しくなるのです。

このような識別子が持つGoの型情報を、`go/types`パッケージでは以下の14種類に分類しており、これらは全て`types.Type`インターフェースを満たす構造体として定義されています。
```
Type = *Basic
     | *Pointer
     | *Array
     | *Slice
     | *Map
     | *Chan
     | *Struct
     | *Tuple
     | *Signature
     | *Alias
     | *Named
     | *Interface
     | *Union
     | *TypeParam
```
引用: [`go/types`: The Go Type Checker](https://github.com/golang/example/tree/master/gotypes#types)

### `types.Object`インターフェース
[`types.Object`インターフェース](https://pkg.go.dev/go/types#Object)は、型情報を持ちうるオブジェクトの総称を表しています。
```go
type Object interface {
	Parent() *Scope // scope in which this object is declared; nil for methods and struct fields
	Pos() token.Pos // position of object identifier in declaration
	Pkg() *Package  // package to which this object belongs; nil for labels and objects in the Universe scope
	Name() string   // package local object name
	Type() Type     // object type
	Exported() bool // reports whether the name starts with a capital letter
	Id() string     // object name if exported, qualified name if not exported (see func Id)

	// String returns a human-readable string of the object.
	// Use [ObjectString] to control how package names are formatted in the string.
	String() string
	// contains filtered or unexported methods
}
```

この`Object`インターフェースを満たす構造体は、`go/types`パッケージ内で以下の8種類が定義されています。
```
Object = *Func         // function, concrete method, or abstract method
       | *Var          // variable, parameter, result, or struct field
       | *Const        // constant
       | *TypeName     // type name
       | *Label        // statement label
       | *PkgName      // package name, e.g. json after import "encoding/json"
       | *Builtin      // predeclared function such as append or len
       | *Nil          // predeclared nil
```
引用: [`go/types`: The Go Type Checker](https://github.com/golang/example/tree/master/gotypes#objects)

### `types.Type`と`types.Object`の具体例
`types.Type`インターフェースも`types.Object`インターフェースもいささか抽象度が高いので、最初はどのような概念を表したくてこの2つが用意されたのか分かりにくいかと思います。
そのため、実際のソースコードがどのようにこの2つのインターフェースの概念に変換されるのか、具体例を見てみましょう。

`testdata/src/a.go`の中には、以下のような変数宣言がありました。
```go
var gopher int
```
この一行の中には、`gopher`と`int`という2つの識別子が存在しています。

:::message
`var`は予約語なので、識別子の名前として使うことはできません。
:::

この2つの識別子は、以下の`types.Type`/`types.Object`の情報を持つASTノードになります。
- `gopher`
	- Type: intという基本型を表す[`types.Basic`](https://pkg.go.dev/go/types#Basic)構造体
	- Object: 変数を表す[`types.Var`](https://pkg.go.dev/go/types#Var)構造体
- `int`
	- Type: intという基本型を表す[`types.Basic`](https://pkg.go.dev/go/types#Basic)構造体
	- Object: 型名を表す[`types.TypeName`](https://pkg.go.dev/go/types#TypeName)構造体
