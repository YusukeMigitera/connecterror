package main

import (
	"connecterror"

	"golang.org/x/tools/go/analysis/unitchecker"
)

func main() { unitchecker.Main(connecterror.Analyzer) }
