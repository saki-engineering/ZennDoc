---
title: "はじめに"
---

# この本について
この本では、Go言語で扱えるI/Oについてまとめています。

Go言語I/Oを扱うためのパッケージとしては、ドンピシャのものとしては`io`パッケージがあります。
しかし、例えば実際にファイルを読み書きしようとするときに使うのは、`os`パッケージの`os.File`型まわりのメソッドです。
標準入力・出力を扱おうとすると`fmt`パッケージが手っ取り早いですし、また速さを求める場面では`bufio`パッケージのスキャナを使うということもあるでしょう。
このように、「I/O」といってもGoでそれに関わるパッケージは幅広いのが現状です。

また、ファイルオブジェクト`f`に対して`f.Read()`とかいう「おまじない」と唱えるだけで、なんでファイルの中身が取得できるの？一体裏で何が起こっているの？という疑問を感じている方もいるかと思います。

ここでは
- `os`や`io`とかいっぱいあるけど、それぞれどういう関係なの？
- 標準入力・出力を扱うときに`fmt`と`bufio`はどっちがいいの？
- そもそも`bufio`パッケージって何者？
- GoでやったI/Oまわりの操作は、実現のために裏で何が起こっているの？
こういったことを一から解説していきます。

## 使用する環境・バージョン
- OS: macOS Mojave 10.14.5
- go version go1.16.2 darwin/amd64

## 読者に要求する前提知識
- Goの基本的な文法の読み書きができること
- 基本情報技術者試験くらいのIT前提知識