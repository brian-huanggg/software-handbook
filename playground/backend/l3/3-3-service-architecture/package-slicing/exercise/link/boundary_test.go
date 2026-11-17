//go:build exercise

package link

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

type fakeRecorder struct{ codes []string }

func (f *fakeRecorder) Record(code string) { f.codes = append(f.codes, code) }

func TestResolveRecordsThroughInterface(t *testing.T) {
	rec := &fakeRecorder{}
	s := New(rec)
	s.Create("go", "https://go.dev")
	if url, err := s.Resolve("go"); err != nil || url != "https://go.dev" {
		t.Fatalf("Resolve(go) = %q, %v", url, err)
	}
	if len(rec.codes) != 1 {
		t.Fatalf("recorded %v, want one click", rec.codes)
	}
}

func TestBoundary(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if strings.HasSuffix(strings.Trim(imp.Path.Value, `"`), "/analytics") {
				t.Errorf("%s still imports %s", name, imp.Path.Value)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			ts, ok := n.(*ast.TypeSpec)
			if !ok {
				return true
			}
			if ts.Name.Name == "Store" {
				t.Errorf("type Store is still exported")
			}
			if st, ok := ts.Type.(*ast.StructType); ok && ts.Name.Name == "Service" {
				for _, fld := range st.Fields.List {
					for _, id := range fld.Names {
						if id.IsExported() {
							t.Errorf("Service.%s is exported", id.Name)
						}
					}
				}
			}
			return true
		})
	}
}
