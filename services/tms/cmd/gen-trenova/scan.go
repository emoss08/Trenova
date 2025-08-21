/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package main

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/semaphore"
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
	var filesScanned atomic.Int64
	var directoriesScanned atomic.Int64

	// First, collect all files to process with CPU yielding
	var files []string
	var walkCount int

	err := filepath.WalkDir(domainPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Yield CPU periodically during walk
		walkCount++
		if walkCount%50 == 0 {
			runtime.Gosched()
			time.Sleep(500 * time.Microsecond)
		}

		// Track directories
		if d.IsDir() {
			directoriesScanned.Add(1)
			return nil
		}

		// Skip non-go files
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		// Skip test files and generated files
		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	fmt.Printf("Found %d Go files in %d directories\n", len(files), directoriesScanned.Load())

	// Process files concurrently with better control
	concurrency := runtime.NumCPU() / 4
	if concurrency < 1 {
		concurrency = 1
	}
	if concurrency > 2 {
		concurrency = 2 // More aggressive limit for WSL
	}

	// Use semaphore for better concurrency control
	sem := semaphore.NewWeighted(int64(concurrency))
	ctx := context.Background()

	// Batch processing to reduce overhead
	batchSize := 10
	if len(files) < 50 {
		batchSize = 5
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	var domains []DomainInfo
	var processedFiles atomic.Int64

	// Process files in batches
	for i := 0; i < len(files); i += batchSize {
		end := i + batchSize
		if end > len(files) {
			end = len(files)
		}
		batch := files[i:end]

		wg.Add(1)
		go func(batchFiles []string, batchNum int) {
			defer wg.Done()

			// Acquire semaphore for batch
			if err := sem.Acquire(ctx, 1); err != nil {
				fmt.Printf("Failed to acquire semaphore: %v\n", err)
				return
			}
			defer sem.Release(1)

			// Add delay between batches to prevent CPU spikes
			time.Sleep(time.Duration(batchNum*2) * time.Millisecond)

			// Process files in batch
			for _, path := range batchFiles {
				// Yield CPU between files
				runtime.Gosched()

				infos, err := scanFile(path)
				if err != nil {
					// Log error but don't fail the entire scan
					fmt.Printf("Warning: failed to scan %s: %v\n", path, err)
					continue
				}

				filesScanned.Add(1)
				processedFiles.Add(1)

				if len(infos) > 0 {
					mu.Lock()
					domains = append(domains, infos...)
					mu.Unlock()
				}

				// Small delay between files in batch
				if processedFiles.Load()%5 == 0 {
					time.Sleep(time.Millisecond)
				}
			}
		}(batch, i/batchSize)
	}

	wg.Wait()
	fmt.Printf("Scanned %d files, found %d domain entities\n", filesScanned.Load(), len(domains))
	return domains, nil
}

// scanFile scans a single Go file for domain entities
func scanFile(filePath string) ([]DomainInfo, error) {
	// Add small delay to prevent rapid file access
	time.Sleep(500 * time.Microsecond)

	// Get file info for size checking
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	// Skip very large files that might cause issues
	if fileInfo.Size() > 1024*1024 { // 1MB
		fmt.Printf("Skipping large file: %s (%d bytes)\n", filePath, fileInfo.Size())
		return nil, nil
	}

	fset := token.NewFileSet()

	// Yield CPU before parsing
	runtime.Gosched()

	// Parse only what we need - skip function bodies and object resolution
	node, err := parser.ParseFile(
		fset,
		filePath,
		nil,
		parser.SkipObjectResolution|parser.ParseComments,
	)
	if err != nil {
		return nil, err
	}

	// Yield CPU after parsing
	runtime.Gosched()

	var domains []DomainInfo
	packageName := node.Name.Name

	// Optimize by only looking at top-level declarations
	declCount := 0
	for _, decl := range node.Decls {
		// Yield CPU periodically
		declCount++
		if declCount%10 == 0 {
			runtime.Gosched()
		}

		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
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
		}
	}

	return domains, nil
}

// hasBunBaseModel checks if a struct has bun.BaseModel embedded
func hasBunBaseModel(structType *ast.StructType) bool {
	// Early return for empty structs
	if structType.Fields == nil || len(structType.Fields.List) == 0 {
		return false
	}

	// Limit check to prevent excessive processing
	fieldCount := 0
	for _, field := range structType.Fields.List {
		fieldCount++
		if fieldCount > 100 { // Reasonable limit for struct fields
			break
		}

		// Yield CPU periodically
		if fieldCount%20 == 0 {
			runtime.Gosched()
		}

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
