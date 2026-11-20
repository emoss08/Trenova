package main

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

const (
	cloudDir    = "internal/cloud"
	cloudImport = modulePath + "/" + cloudDir
)

var cloudImporterDirs = []string{cloudDir, "internal/bootstrap"}

var skippedDirs = []string{".git", "node_modules", "testdata", ".deskbench"}

func cloudImporters(root string) ([]string, error) {
	fset := token.NewFileSet()
	problems := make([]string, 0)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if slices.Contains(skippedDirs, d.Name()) || mayImportCloud(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		file, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			return fmt.Errorf("parse %s: %w", rel, parseErr)
		}
		for _, spec := range file.Imports {
			imported, unquoteErr := strconv.Unquote(spec.Path.Value)
			if unquoteErr != nil {
				return fmt.Errorf("read an import of %s: %w", rel, unquoteErr)
			}
			if imported == cloudImport || strings.HasPrefix(imported, cloudImport+"/") {
				problems = append(problems, fmt.Sprintf(
					"cloud: %s:%d imports %s; only %s may import the commercial directory",
					rel,
					fset.Position(spec.Pos()).Line,
					imported,
					strings.Join(cloudImporterDirs, " and "),
				))
			}
		}
		return nil
	})
	return problems, err
}

func mayImportCloud(rel string) bool {
	for _, dir := range cloudImporterDirs {
		if rel == dir || strings.HasPrefix(rel, dir+"/") {
			return true
		}
	}
	return false
}
