package main

import (
	"bufio"
	"io"
	"os"
	"testing"
)

const Fsize = 4096 * 500

// nbyteごとに読む
func ReadOS(r io.Reader, n int) {
	data := make([]byte, n)

	t := Fsize / n
	for i := 0; i < t; i++ {
		r.Read(data)
	}
}

func BenchmarkRead1(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(f, 1)
	}
}

func BenchmarkRead32(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(f, 32)
	}
}

func BenchmarkRead256(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(f, 256)
	}
}

func BenchmarkRead4096(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(f, 4096)
	}
}

func BenchmarkBRead1(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()
	bf := bufio.NewReader(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(bf, 1)
	}
}

func BenchmarkBRead32(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()
	bf := bufio.NewReader(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(bf, 32)
	}
}

func BenchmarkBRead256(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()
	bf := bufio.NewReader(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(bf, 256)
	}
}

func BenchmarkBRead4096(b *testing.B) {
	f, _ := os.Open("read.txt")
	defer f.Close()
	bf := bufio.NewReader(f)

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		ReadOS(bf, 4096)
	}
}
