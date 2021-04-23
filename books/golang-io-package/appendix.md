---
title: "おわりに"
---
# おわりに
というわけで、GoでI/Oに関わるものを片っ端から書き連ねました。
完全にごった煮状態の本ですがいかがでしたでしょうか。

I/Oは、根本を理解しようとすると低レイヤの知識まで必要になってくるのでなかなか難しいですが、この本が皆さんの理解の一助になっていれば幸いです。

コメントによる編集リクエスト・情報提供は大歓迎ですので、どしどし書き込んでいってください。
連絡先: [作者Twitter @saki_engineer](https://twitter.com/saki_engineer)

# 参考文献
## 書籍 Linux System Programming
https://learning.oreilly.com/library/view/linux-system-programming/0596009585/

オライリーの本です。
Linuxでの低レイヤ・カーネル内部まわりの話がこれでもかというほど書かれています。
今回この本を執筆するにあたって、1~4章のI/Oの章を大いに参考にしました。

## 書籍 Software Design 2021年1月号
https://gihyo.jp/magazine/SD/archive/2021/202101

この本のGo特集第2章が、[tenntennさん(@tenntenn)](https://twitter.com/tenntenn)が執筆された`io`章です。
このZenn本では`io.Reader`と`io.Writer`しか取り上げませんでしたが、Software Designの記事の方には他の`io`便利インターフェースについても言及があります。

## Web連載 Goならわかるシステムプログラミング
https://ascii.jp/serialarticles/1235262/

[渋川よしきさん(@shibu_jp)](https://twitter.com/shibu_jp)が書かれたWeb連載です。
Goの視点からみた低レイヤの話がとても詳しく書かれています。

以下の回を大いに参考にしました。
- [第2回 低レベルアクセスへの入り口（1）：io.Writer](https://ascii.jp/elem/000/001/243/1243667/)
- [第3回 低レベルアクセスへの入り口（2）：io.Reader前編](https://ascii.jp/elem/000/001/252/1252961/)
- [第4回 低レベルアクセスへの入り口（3）：io.Reader後編](https://ascii.jp/elem/000/001/260/1260449/)
- [第5回 Goから見たシステムコール](https://ascii.jp/elem/000/001/267/1267477/)
- [第6回 GoでたたくTCPソケット（前編）](https://ascii.jp/elem/000/001/276/1276572/)
- [第7回 GoでたたくTCPソケット（後編）](https://ascii.jp/elem/000/001/403/1403717/)

## Qiita記事 Go言語を使ったTCPクライアントの作り方
https://qiita.com/tutuz/items/e875d8ea3c31450195a7

Go Advent Calender 2020 10日目に[Tsuji Daishiro(@d_tutuz)](https://twitter.com/d_tutuz)さんが書かれた記事です。
TCPネットワークにおけるシステムコールは、この本ではsocket()しか取り上げませんでしたが、この記事ではさらに詳しいところまで掘り下げています。

## GopherCon 2019: Dave Cheney - Two Go Programs, Three Different Profiling Techniques
動画
https://www.youtube.com/watch?v=nok0aYiGiYA
サマリー記事
https://about.sourcegraph.com/go/gophercon-2019-two-go-programs-three-different-profiling-techniques-in-50-minutes/

[Dave Cheneyさん(@davecheney)](https://twitter.com/davecheney)によるGoCon2019のセッション(英語)です。
前半部分が「ユーザースペースでバッファリングしたI/Oは早いぞ」という内容です。
セッション中に実際にコードを書いて、それを分析ツールでどこが遅いのかを確かめながらコードを改善していく様子がよくわかります。

## 記事 How to read and write with Golang bufio
https://www.educative.io/edpresso/how-to-read-and-write-with-golang-bufio

`bufio.Writer`を使った際の内部バッファの挙動がイラスト付きでわかりやすく書かれています。