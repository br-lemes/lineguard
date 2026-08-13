// +gocover:ignore:file untested command entrypoint
package main

import (
	"github.com/br-lemes/lineguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(lineguard.Analyzer)
}
