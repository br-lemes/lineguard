package lineguard_test

import (
	"testing"

	"github.com/br-lemes/lineguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, lineguard.Analyzer, "rules")
}

func TestSuggestedFixes(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.RunWithSuggestedFixes(t, testdata, lineguard.Analyzer, "fixes")
}
