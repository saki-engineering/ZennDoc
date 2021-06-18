---
title: "並行処理で役立つデバッグ&分析手法"
---
# この章について
並行処理を実装しているときに役に立ちそうなデバッグツールを、ここでまとめて紹介します。

- `runtime/trace`によるトレース
- `GODEBUG`環境変数によるデバッグ
- Race Detector

# traceについて
`runtime/trace`パッケージを使うことで、どうゴールーチンが動いているのかGUIで可視化することができます。
https://pkg.go.dev/runtime/trace@go1.16.4

traceパッケージでできることは、[ドキュメント](https://pkg.go.dev/runtime/trace@go1.16.4#hdr-Tracing_runtime_activities)によると以下5つです。

- ゴールーチンのcreation/blocking/unblockingイベントのキャプチャ
- システムコールのenter/exit/blockイベントのキャプチャ
- GC関連のイベントがどこで起きたかをチェック
- ヒープ領域増減のチェック
- プロセッサのstart/stopのチェック

:::message
実行中のCPU・メモリ占有率の調査についてはtraceの対象外です。これらを調べたい場合は`go tool pprof`コマンドを使用してください。
:::

## 部品
traceパッケージでは、ログをとりたいコードブロックの種類が2つ存在します。
- region
- task

### region
regionは、「Gの実行中の間の」ログをとるための部品です。Gをまたぐことはできません。regionをネストすることができます。

### task
タスクは、関数やGを跨ぐような、例えば「httpリクエスト捌き」みたいなくくりのログをとるための部品です。

regionとtaskの違いについては、言葉で説明するよりかは実際にtraceを実行しているコードをみるとわかりやすいかと思います。

# traceの実行
ここから先は、とあるコードをtraceで分析・パフォーマンスを改善する様子をお見せしようと思います。

## 改善前の処理をtraceできるようにする
以下のような「ランダム時間sleepする」処理を5回連続するプログラムを考えます。
```go
func RandomWait(i int) {
	fmt.Printf("No.%d start\n", i+1)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	fmt.Printf("No.%d done\n", i+1)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		RandomWait(i)
	}
}
```
これをtraceするために、taskとregionを定義していきます。

```diff go
func RandomWait(ctx context.Context, i int) {
+	// regionを始める
+	defer trace.StartRegion(ctx, "randomWait").End()

	fmt.Printf("No.%d start\n", i+1)
	time.Sleep(time.Duration(rand.Intn(500)) * time.Millisecond)
	fmt.Printf("No.%d done\n", i+1)
}

-func main() {
+func _main() {
+	// タスクを定義
+	ctx, task := trace.NewTask(context.Background(), "main")
+	defer task.End()

	rand.Seed(time.Now().UnixNano())
	for i := 0; i < 5; i++ {
		num := i
		RandomWait(ctx, num)
	}
}

+func main() {
+	// トレースを始める
+	// 結果出力用のファイルもここで作成
+	f, err := os.Create("tseq.out")
+	if err != nil {
+		log.Fatalln("Error:", err)
+	}
+	defer func() {
+		if err := f.Close(); err != nil {
+			log.Fatalln("Error:", err)
+		}
+	}()
+
+	if err := trace.Start(f); err != nil {
+		log.Fatalln("Error:", err)
+	}
+	defer trace.Stop()
+
+	_main()
+}
```
これを`go run [ファイル名]`で実行すると、カレントディレクトリ下に新たに`tseq.out`というファイルが作成されます。

## trace結果をみる
trace結果をみるためには、`go tool`コマンドを使います。
```bash
$ go tool trace tseq.out
2021/05/31 14:10:03 Parsing trace...
2021/05/31 14:10:03 Splitting trace...
2021/05/31 14:10:03 Opening browser. Trace viewer is listening on http://127.0.0.1:50899
```
すると、ブラウザが開いてGUI画面が立ち上がります。
ここを`User-defined tasks`→`Count 1`か`1000ms`→`(goroutine view)`の順番にクリックしていきます。
![](https://storage.googleapis.com/zenn-user-upload/e72e8001bd0da72d359ac519.png)

すると、「いつどんなtask/regionが実行されていたか」というのが視覚的に確認できます。
![](https://storage.googleapis.com/zenn-user-upload/9238d217217075b776cb9838.png)

## 並行処理するように改善
トレースする`_main`を以下のように改善してみた。
```diff go
func _main() {
	// タスクを定義
	ctx, task := trace.NewTask(context.Background(), "main")
	defer task.End()

	rand.Seed(time.Now().UnixNano())
+	var wg sync.WaitGroup
+	for i := 0; i < 5; i++ {
+		wg.Add(1)
+		go func(i int) {
+			defer wg.Done()
+			RandomWait(ctx, i)
+		}(i)
+	}
+	wg.Wait()
}
```
![](https://storage.googleapis.com/zenn-user-upload/43a4c174aa48a5cb604e5979.png)
![](https://storage.googleapis.com/zenn-user-upload/e2cee30260cbc8c6b6c7c617.png)

trace結果をみると、実行が明らかに効率的 & 早くなっていることがわかります。




# GODEBUG環境変数の使用
`GODEBUG`環境変数によって、ランタイムの動作を設定値に従って変更させることができます。

例えば、以下のようなコードを用意しました。
```go
func doWork() {
	// 何か重くて時間がかかる操作
}

func main() {
	var wg sync.WaitGroup
	n := 15

	// doWorkを、n個のゴールーチンでそれぞれ実行
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			doWork()
		}()
	}
	wg.Wait()
}
```
このプログラムを実行する際に、`GODEBUG`環境変数を使ってオプションをつけてやることができます。

```bash
$ GODEBUG=optionname1=val1,optionname2=val2 go run main.go
```

`GODEBUG`環境変数につけられるオプション一覧は`runtime`パッケージの公式ドキュメントに記載があります。
https://golang.org/pkg/runtime/#hdr-Environment_Variables

## schedtraceオプション
上記のプログラムを、`GODEBUG`の`schedtrace`オプションをつけて実行してみます。
```bash
$ GOMAXPROCS=2 GODEBUG=schedtrace=1000 go run main.go
```
:::message
`GOMAXPROCS`環境変数は、使用するCPUの上限を制限する機能があり、今回はMAX2個にしています。
:::
`schedtrace=1000`と指定することによって、「1000msごとにデバッグを表示する」ようにしました。

実行した様子は以下のようになりました。
```bash
// (一部抜粋)
SCHED 0ms: gomaxprocs=2 idleprocs=0 threads=5 spinningthreads=0 idlethreads=1 runqueue=0 [0 0]
SCHED 1009ms: gomaxprocs=2 idleprocs=0 threads=4 spinningthreads=0 idlethreads=1 runqueue=2 [3 4]
SCHED 2019ms: gomaxprocs=2 idleprocs=0 threads=4 spinningthreads=0 idlethreads=1 runqueue=11 [0 2]
SCHED 3029ms: gomaxprocs=2 idleprocs=0 threads=4 spinningthreads=0 idlethreads=1 runqueue=5 [2 3]
SCHED 4020ms: gomaxprocs=2 idleprocs=2 threads=8 spinningthreads=0 idlethreads=1 runqueue=0 [0 0]
```

それぞれのフィールドの意味は
- SCHED xxxms: プログラム開始からxxx[ms]
- gomaxprocs: 仮想CPU数
- idleprocs: アイドル状態になっているプロセッサ数
- threads: 使用しているスレッド数
- spinningthread: `spinning`状態のスレッド
- idlethread: アイドル状態のスレッド数
- runqueue: グローバルキュー内にあるG数
- `[2,3]`: Pのローカルキュー中にあるG数(今回Pは`GOMAXPROCS=2`個あるので、ローカルキューも2個存在)

:::message
スレッドが`spinning`状態であるとは、「グローバルキューやnetpollから実行するGを見つけられず、仕事をしていない状態」のことをいいます。
参考:[runtime/proc.go](https://github.com/golang/go/blob/f2eea4c1dc37886939c010daff89c03d5a3825be/src/runtime/proc.go#L56-L58)
:::

## scheddetailオプション
さらに詳細な情報を手に入れるために、`scheddetail`オプションもつけてプログラムを実行することもできます。
```bash
$ GOMAXPROCS=2 GODEBUG=schedtrace=1000,scheddetail=1  go run main.go
// (略)
SCHED 0ms: gomaxprocs=2 idleprocs=1 threads=4 spinningthreads=0 idlethreads=2 runqueue=0 gcwaiting=0 nmidlelocked=0 stopwait=0 sysmonwait=0
  P0: status=0 schedtick=0 syscalltick=0 m=-1 runqsize=0 gfreecnt=0 timerslen=0
  P1: status=1 schedtick=3 syscalltick=0 m=0 runqsize=0 gfreecnt=0 timerslen=0
  M3: p=-1 curg=-1 mallocing=0 throwing=0 preemptoff= locks=0 dying=0 spinning=false blocked=true lockedg=-1
  M2: p=-1 curg=-1 mallocing=0 throwing=0 preemptoff= locks=0 dying=0 spinning=false blocked=true lockedg=-1
  M1: p=-1 curg=-1 mallocing=0 throwing=0 preemptoff= locks=2 dying=0 spinning=false blocked=false lockedg=-1
  M0: p=1 curg=1 mallocing=0 throwing=0 preemptoff= locks=2 dying=0 spinning=false blocked=false lockedg=-1
  G1: status=2(chan receive) m=0 lockedm=-1
  G2: status=4(force gc (idle)) m=-1 lockedm=-1
  G3: status=4(GC sweep wait) m=-1 lockedm=-1
  G4: status=4(GC scavenge wait) m=-1 lockedm=-1
  G17: status=1() m=-1 lockedm=-1
// (略)
```
このように、`P`,`M`,`G`がその時どういう状態だったのかが詳細に出力されます。




# Race Detector
Goには、Race Conditionが起きていることを検出するための公式のツール**Race Detector**が存在します。

公式ドキュメントはこちら。
https://golang.org/doc/articles/race_detector

## 使ってみる
実際にそれを使っている様子をお見せしましょう。

まずは、以下のように「グローバル変数`num`に対して、加算を並行に2回行う」コードを書きます。
```go
var num = 0

func add(a int) {
	num += a
}

func main() {
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		add(1)
	}()
	go func() {
		defer wg.Done()
		add(-1)
	}()

	wg.Wait()
	fmt.Println(num)
}
```
加算は非アトミックな処理であるためデータの競合が起こります。

これをRace Detectorの方でも検出してみましょう。
やり方は簡単です。プログラム実行の際に`-race`オプションをつけるだけです。
```bash
$ go run -race main.go
==================
WARNING: DATA RACE
Read at 0x00000122ec90 by goroutine 8:
  main.add()
      /path/to/main.go:11 +0x6f
  main.main.func2()
      /path/to/main.go:24 +0x5f

Previous write at 0x00000122ec90 by goroutine 7:
  main.add()
      /path/to/main.go:11 +0x8b
  main.main.func1()
      /path/to/main.go:20 +0x5f

Goroutine 8 (running) created at:
  main.main()
      /path/to/main.go:22 +0xc8

Goroutine 7 (finished) created at:
  main.main()
      /path/to/main.go:18 +0xa6
==================
0 //(fmt.Printlnの内容)
Found 1 data race(s)
exit status 66
```

`Found 1 data race(s)`と表示され、データ競合を確認することができました。

このように、実行時に`-race`オプションをつけることによって、「**実際にデータ競合が起こったときに**」そのことを通知してくれます。

:::message
データ競合が実際に発生しなかった場合は何も起こりません。
そのため、「データ競合が起こる**可能性のある**危ないコードだ」という警告はRace Detectorからは得ることができない、ということに注意です。
:::

## プログラムを修正
それでは、データ競合が起こらないようにコードを直していきましょう。
加算を行う前に排他制御を行うことで、アトミック性を確保します。

```diff go
func main() {
	var wg sync.WaitGroup
	wg.Add(2)

+	var mu sync.Mutex

	go func() {
		defer wg.Done()
+		mu.Lock()
		add(1)
+		mu.Unlock()
	}()
	go func() {
		defer wg.Done()
+		mu.Lock()
		add(-1)
+		mu.Unlock()
	}()

	wg.Wait()
	fmt.Println(num)
}
```
:::message
4章でも記述した通り`sync.Mutex`は本来低レイヤでの使用を想定したものであり、排他制御を使ったメモリ共有よりもチャネルを使える場面であるならばそちらを選ぶべき、ということは強調しておきます。
:::

これもRace Detectorにかけてみましょう。
```bash
$ go run -race main.go
0
```
特に何も検知されることなく実行終了しました。デバッグ成功です。