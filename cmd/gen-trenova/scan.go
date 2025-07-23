// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
)

// DomainInfo contains information about a domain entity
type DomainInfo struct {
	PackageName string
	TypeName    string
	FilePath    string
	HasBunModel bool
}

// ScanDomainFolder scans the domain folder for entities with bun.BaseModel
func ScanDomainFolder(domainPath string) ([]DomainInfo, error) {
	var domains []DomainInfo

	err := filepath.WalkDir(domainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-go files
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files and generated files
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}

		// Parse the file
		infos, err := scanFile(path)
		if err != nil {
			return fmt.Errorf("scanning %s: %w", path, err)
		}

		domains = append(domains, infos...)
		return nil
	})

	return domains, err
}

// scanFile scans a single Go file for domain entities
func scanFile(filePath string) ([]DomainInfo, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var domains []DomainInfo
	packageName := node.Name.Name

	// Look for structs with bun.BaseModel
	ast.Inspect(node, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		// Check if this struct has bun.BaseModel
		if hasBunBaseModel(structType) {
			domains = append(domains, DomainInfo{
				PackageName: packageName,
				TypeName:    typeSpec.Name.Name,
				FilePath:    filePath,
				HasBunModel: true,
			})
		}

		return true
	})

	return domains, nil
}

// hasBunBaseModel checks if a struct has bun.BaseModel embedded
func hasBunBaseModel(structType *ast.StructType) bool {
	for _, field := range structType.Fields.List {
		// Check for embedded bun.BaseModel
		if len(field.Names) == 0 {
			// This is an embedded field
			if selectorExpr, ok := field.Type.(*ast.SelectorExpr); ok {
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					if ident.Name == "bun" && selectorExpr.Sel.Name == "BaseModel" {
						return true
					}
				}
			}
		}
	}
	return false
}