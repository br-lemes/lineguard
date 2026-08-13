// +gocover:ignore:file plugin registration entrypoint
package plugin

import (
	"github.com/br-lemes/lineguard"
	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

func init() {
	register.Plugin("lineguard", New)
}

func New(settings any) (register.LinterPlugin, error) {
	return plugin{}, nil
}

type plugin struct{}

func (plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{lineguard.Analyzer}, nil
}

func (plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
