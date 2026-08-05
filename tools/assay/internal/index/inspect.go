package index

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

// packageDefinesTestMain reports whether any test file in dir declares its own
// TestMain, in the internal or the external test package. Injecting the harness
// next to one would be a duplicate definition, so such packages take the
// per-process path. Files excluded by build tags still count: conservative here
// costs a slower collection, never a broken build.
func packageDefinesTestMain(dir string) (bool, error) {
	files, err := filepath.Glob(filepath.Join(dir, "*_test.go"))
	if err != nil {
		return false, err
	}

	needle := []byte("func TestMain")
	for _, file := range files {
		source, readErr := os.ReadFile(file)
		if readErr != nil {
			return false, readErr
		}
		// The byte scan is a prefilter; the parse confirms, because "func TestMain"
		// inside a string or comment must not push a package onto the slow path.
		if !bytes.Contains(source, needle) {
			continue
		}

		parsed, parseErr := parser.ParseFile(token.NewFileSet(), file, source, parser.SkipObjectResolution)
		if parseErr != nil {
			return false, parseErr
		}
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if ok && fn.Recv == nil && fn.Name.Name == "TestMain" {
				return true, nil
			}
		}
	}

	return false, nil
}

// packageName reads the package clause of a production file in dir. The graph
// deliberately does not carry package names — adding the field would invalidate
// every cached graph for one consumer — and the harness has to be injected under
// the package's real name, which need not match the directory's.
func packageName(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || filepath.Ext(name) != ".go" || strings.HasSuffix(name, "_test.go") {
			continue
		}

		parsed, parseErr := parser.ParseFile(
			token.NewFileSet(), filepath.Join(dir, name), nil, parser.PackageClauseOnly)
		if parseErr != nil {
			continue
		}

		return parsed.Name.Name, nil
	}

	return "", os.ErrNotExist
}
