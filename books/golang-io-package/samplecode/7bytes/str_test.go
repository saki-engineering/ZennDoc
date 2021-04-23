package main

import (
	"io"
	"strings"
	"testing"
)

func TranslateIntoGerman(r io.Reader) string {
	data := make([]byte, 300)
	len, _ := r.Read(data)
	str := string(data[:len])

	result := strings.ReplaceAll(str, "Hello", "Guten Tag")
	return result
}

func Test_Strings(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{
			name: "normal",
			arg:  "Hello, World!",
			want: "Guten Tag, World!",
		},
		{
			name: "repeat",
			arg:  "Hello World, Hello Golang!",
			want: "Guten Tag World, Guten Tag Golang!",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TranslateIntoGerman(strings.NewReader(tt.arg)); got != tt.want {
				t.Errorf("got %v, but want %v", got, tt.want)
			}
		})
	}
}
