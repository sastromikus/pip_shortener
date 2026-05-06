package main

import (
	"github.com/sastromikus/pip_shortener/cmd/linter/nopanicexit"

	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(nopanicexit.Analyzer)
}
