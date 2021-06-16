---
title: "(おまけ)実行ファイル分析のやり方"
---
# この章について
`go build xxx.go`コマンドでできた実行ファイルの中身をみている場面で、どうやって中を見ていたのかを説明します。

# 実行ファイルの詳細
ここでは、以下のようなハローワールドのコード`main.go`というファイル名で用意しました。
```go
package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}
```

これを実行ファイルに直すには、`go build`コマンドを打ちます。
```bash
$ ls
main.go
$ go build main.go
```
すると、カレントディレクトリ下に`main`という名前の実行ファイルができているのが確認できます。
```bash
$ ls
main.go main
```

この拡張子のない`main`というのは一体何者なのでしょうか。

## Macの場合
Macの場合は、これは`Mach-O`という形式で書かれた実行ファイルです。

:::message
ファイル形式については、`file`コマンドで以下のように確認することができます。
```bash
$ file main
main: Mach-O 64-bit executable x86_64
```
:::

`Mach-O`形式が中でどういうフォーマットになっているのかについては、以下のリンクを参照してください。
https://www.itmedia.co.jp/enterprise/articles/0711/30/news014_3.html

この実行ファイルの中身を出力するためには、`otool`コマンドというものを使用します。
```bash
// result.txtに中身を書き出す
$ otool -t -v -V mybinary > result.txt
```

`result.txt`の中を探してみると、自分で実装した`main`関数部分は、実行ファイルの中では以下のようになっていることがわかります。
```
_main.main:
00000000010a2e00	movq	%gs:0x30, %rcx
00000000010a2e09	cmpq	0x10(%rcx), %rsp
00000000010a2e0d	jbe	0x10a2e80
00000000010a2e0f	subq	$0x58, %rsp
00000000010a2e13	movq	%rbp, 0x50(%rsp)
00000000010a2e18	leaq	0x50(%rsp), %rbp
00000000010a2e1d	xorps	%xmm0, %xmm0
00000000010a2e20	movups	%xmm0, 0x40(%rsp)
00000000010a2e25	leaq	0xafd4(%rip), %rax
00000000010a2e2c	movq	%rax, 0x40(%rsp)
00000000010a2e31	leaq	0x43698(%rip), %rax
00000000010a2e38	movq	%rax, 0x48(%rsp)
00000000010a2e3d	movq	_os.Stdout(%rip), %rax
00000000010a2e44	leaq	"_go.itab.*os.File,io.Writer"(%rip), %rcx
00000000010a2e4b	movq	%rcx, (%rsp)
00000000010a2e4f	movq	%rax, 0x8(%rsp)
00000000010a2e54	leaq	0x40(%rsp), %rax
00000000010a2e59	movq	%rax, 0x10(%rsp)
00000000010a2e5e	movq	$0x1, 0x18(%rsp)
00000000010a2e67	movq	$0x1, 0x20(%rsp)
00000000010a2e70	callq	_fmt.Fprintln
00000000010a2e75	movq	0x50(%rsp), %rbp
00000000010a2e7a	addq	$0x58, %rsp
00000000010a2e7e	retq
00000000010a2e7f	nop
00000000010a2e80	callq	_runtime.morestack_noctxt
00000000010a2e85	jmp	_main.main
```

## Linuxの場合
Linuxの場合は、`go build`コマンドで作られた実行ファイルは`ELF`(Executable and Linkable Format)という形式になります。
Macでいう`otool`コマンドにあたるのは、こちらでは`readelf`コマンドです。

詳細については割愛します。



# go tool objdump
ちなみに、Go言語にも実行ファイルを逆アセンブルする`objdump`コマンドが公式に用意されています。
https://golang.org/cmd/objdump/

これで、先ほどのMacで作った`main.go`の実行ファイルを逆アセンブルしてみましょう。
```bash
// 結果をobjdump.txtに書き出す
$ go tool objdump main > objdump.txt
```

すると、先ほどと同じ部分が今回は以下のようになっています。
```
TEXT main.main(SB) /path/to/main.go
  main.go:5		0x10a2e00		65488b0c2530000000	MOVQ GS:0x30, CX							
  main.go:5		0x10a2e09		483b6110		CMPQ 0x10(CX), SP							
  main.go:5		0x10a2e0d		7671			JBE 0x10a2e80								
  main.go:5		0x10a2e0f		4883ec58		SUBQ $0x58, SP								
  main.go:5		0x10a2e13		48896c2450		MOVQ BP, 0x50(SP)							
  main.go:5		0x10a2e18		488d6c2450		LEAQ 0x50(SP), BP							
  main.go:6		0x10a2e1d		0f57c0			XORPS X0, X0								
  main.go:6		0x10a2e20		0f11442440		MOVUPS X0, 0x40(SP)							
  main.go:6		0x10a2e25		488d05d4af0000		LEAQ runtime.rodata+44608(SB), AX					
  main.go:6		0x10a2e2c		4889442440		MOVQ AX, 0x40(SP)							
  main.go:6		0x10a2e31		488d0598360400		LEAQ sync/atomic.CompareAndSwapUintptr.args_stackmap+192(SB), AX	
  main.go:6		0x10a2e38		4889442448		MOVQ AX, 0x48(SP)							
  print.go:274		0x10a2e3d		488b05f4ba0b00		MOVQ os.Stdout(SB), AX							
  print.go:274		0x10a2e44		488d0dfd4c0400		LEAQ go.itab.*os.File,io.Writer(SB), CX					
  print.go:274		0x10a2e4b		48890c24		MOVQ CX, 0(SP)								
  print.go:274		0x10a2e4f		4889442408		MOVQ AX, 0x8(SP)							
  print.go:274		0x10a2e54		488d442440		LEAQ 0x40(SP), AX							
  print.go:274		0x10a2e59		4889442410		MOVQ AX, 0x10(SP)							
  print.go:274		0x10a2e5e		48c744241801000000	MOVQ $0x1, 0x18(SP)							
  print.go:274		0x10a2e67		48c744242001000000	MOVQ $0x1, 0x20(SP)							
  print.go:274		0x10a2e70		e88b9affff		CALL fmt.Fprintln(SB)							
  main.go:6		0x10a2e75		488b6c2450		MOVQ 0x50(SP), BP							
  main.go:6		0x10a2e7a		4883c458		ADDQ $0x58, SP								
  main.go:6		0x10a2e7e		c3			RET									
  main.go:5		0x10a2e7f		90			NOPL									
  main.go:5		0x10a2e80		e8fbf5fbff		CALL runtime.morestack_noctxt(SB)					
  main.go:5		0x10a2e85		e976ffffff		JMP main.main(SB)
```
先ほどよりも情報量が増えてわかりやすいですね。

実はここに何が書かれているかについても、公式ドキュメントがあります。詳しくはこちらをご覧ください。
https://golang.org/doc/asm