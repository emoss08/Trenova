package index

import (
	"bytes"
	"errors"
	"go/ast"
	"go/build"
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

// packageName resolves the name to inject the harness under. go/build evaluates
// build constraints, so a //go:build ignore generator declaring package main in
// a library directory — gen.go, build.go, exactly the names that sort first —
// cannot mislead it the way reading the alphabetically-first file did. A
// test-only directory has no production name at all; the internal test package
// clause serves instead, with any _test suffix stripped so the harness lands in
// the internal test package.
func packageName(dir string) (string, error) {
	pkg, err := build.ImportDir(dir, 0)
	if err == nil && pkg.Name != "" {
		return pkg.Name, nil
	}
	var noGo *build.NoGoError
	if err != nil && !errors.As(err, &noGo) {
		return "", err
	}

	entries, readErr := os.ReadDir(dir)
	if readErr != nil {
		return "", readErr
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, "_test.go") {
			continue
		}
		parsed, parseErr := parser.ParseFile(
			token.NewFileSet(), filepath.Join(dir, name), nil, parser.PackageClauseOnly)
		if parseErr != nil {
			continue
		}

		return strings.TrimSuffix(parsed.Name.Name, "_test"), nil
	}

	return "", os.ErrNotExist
}
