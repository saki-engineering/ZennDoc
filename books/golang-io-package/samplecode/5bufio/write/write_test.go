package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"testing"
)

const Bsize = 4096

// nbyteごとに書き込む
func WriteOS(w io.Writer, n int) {
	data := []byte(strings.Repeat("a", n))

	t := Bsize / n
	for i := 0; i < t; i++ {
		w.Write(data)
	}
}

func BenchmarkWrite1(b *testing.B) {
	f, _ := os.Create("write1.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		WriteOS(f, 1)
	}
}

func BenchmarkWrite32(b *testing.B) {
	f, _ := os.Create("write2.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		WriteOS(f, 32)
	}
}

func BenchmarkWrite256(b *testing.B) {
	f, _ := os.Create("write3.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		WriteOS(f, 256)
	}
}

func BenchmarkWrite4096(b *testing.B) {
	f, _ := os.Create("write4.txt")
	defer f.Close()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		WriteOS(f, 4096)
	}
}

func BenchmarkBWrite1(b *testing.B) {
	f, _ := os.Create("write5.txt")
	defer f.Close()
	bf := bufio.NewWriter(f)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		WriteOS(bf, 1)
	}
	bf.Flush()
}

func BenchmarkBWrite32(b *testing.B) {
	f, _ := os.Create("write6.txt")
	defer f.Close()
	bf := bufio.NewWriter(f)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		WriteOS(bf, 32)
	}
	bf.Flush()
}

func BenchmarkBWrite256(b *testing.B) {
	f, _ := os.Create("write7.txt")
	defer f.Close()
	bf := bufio.NewWriter(f)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		WriteOS(bf, 256)
	}
	bf.Flush()
}

func BenchmarkBWrite4096(b *testing.B) {
	f, _ := os.Create("write8.txt")
	defer f.Close()
	bf := bufio.NewWriter(f)

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		WriteOS(bf, 4096)
	}
	bf.Flush()
}
