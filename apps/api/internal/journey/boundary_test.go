package journey_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// AC4 of #20: core journey code must not depend on a vendor-specific or
// mock-specific HIS shape. All HIS access goes through the his.Client port
// (ADR-0005); the mock-his Go module must never be imported from apps/api —
// the only allowed coupling to Mock HIS is the HTTP contract in
// packages/contracts/openapi/mock-his.yaml. Lives here (moved from the
// now-retired visit module, ADR-0009) since journey is the module that most
// centrally owns the HIS boundary today.
func TestNoModuleImportsMockHIS(t *testing.T) {
	fset := token.NewFileSet()
	err := filepath.WalkDir("..", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
		for _, imp := range file.Imports {
			if strings.HasPrefix(strings.Trim(imp.Path.Value, `"`), "carepath/apps/mock-his") {
				t.Errorf("%s imports %s — HIS access must go through internal/his, not the mock module",
					path, imp.Path.Value)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk internal/: %v", err)
	}
}
