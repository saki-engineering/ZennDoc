package main

import (
	"slogger"

	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(slogger.Analyzer) }
