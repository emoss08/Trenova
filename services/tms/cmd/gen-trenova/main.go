/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"text/template"
	"time"

	"golang.org/x/sync/semaphore"
	"golang.org/x/time/rate"
	"golang.org/x/tools/imports"
)

var (
	typeNames  = flag.String("type", "", "comma-separated list of type names")
	output     = flag.String("output", "", "output file name; default srcdir/<type>_gen.go")
	domainPath = flag.String("domain-path", "", "path to domain folder to scan all entities")
	outputDir  = flag.String(
		"output-dir",
		"",
		"output directory for generated files (default: same as source)",
	)
	concurrency = flag.Int("concurrency", 0, "number of concurrent workers (default: num CPUs)")

	// Global buffer pool to reduce allocations
	bufferPool = sync.Pool{
		New: func() any {
			return new(bytes.Buffer)
		},
	}

	// Cache for parsed packages to avoid re-parsing
	packageCache sync.Map

	// Pre-compiled template to avoid re-parsing
	compiledTemplate *template.Template
	templateOnce     sync.Once

	// CPU throttling controls
	cpuThrottler      *rate.Limiter
	fileIOThrottler   *rate.Limiter
	parseThrottler    *rate.Limiter
	templateThrottler *rate.Limiter

	// Performance counters
	filesProcessed atomic.Int64
	bytesProcessed atomic.Int64

	// Global context for cancellation
	globalCtx    context.Context
	globalCancel context.CancelFunc
)

func main() {
	flag.Parse()

	// Initialize global context
	globalCtx, globalCancel = context.WithCancel(context.Background())
	defer globalCancel()

	// Set up signal handling for graceful shutdown
	setupSignalHandling()

	// Set concurrency limit
	if *concurrency <= 0 {
		// Default to 1/4 of CPU cores for better control
		*concurrency = runtime.NumCPU() / 4
		if *concurrency < 1 {
			*concurrency = 1
		}
	}

	// Limit GOMAXPROCS to prevent using all CPU cores
	maxProcs := runtime.NumCPU() / 4
	if maxProcs < 1 {
		maxProcs = 1
	}
	if maxProcs > 2 {
		maxProcs = 2 // More aggressive cap for WSL
	}
	runtime.GOMAXPROCS(maxProcs)

	// Initialize rate limiters for CPU throttling
	initializeRateLimiters()

	log.Printf(
		"Using %d concurrent workers with GOMAXPROCS=%d (use -concurrency flag to adjust)\n",
		*concurrency,
		maxProcs,
	)
	log.Printf("CPU throttling enabled: parse=%d/sec, file I/O=%d/sec, template=%d/sec\n",
		10*(*concurrency), 20*(*concurrency), 5*(*concurrency))

	// Check if we're in scan mode or specific type mode
	if *domainPath != "" {
		// Scan mode: find all domain entities
		start := time.Now()
		if err := scanAndGenerate(*domainPath); err != nil {
			log.Fatal(err)
		}
		elapsed := time.Since(start)
		log.Printf("Code generation completed in %v\n", elapsed)
		log.Printf(
			"Processed %d files, %s total\n",
			filesProcessed.Load(),
			formatBytes(bytesProcessed.Load()),
		)
		log.Printf("Average: %.2f files/sec, %s/sec\n",
			float64(filesProcessed.Load())/elapsed.Seconds(),
			formatBytes(int64(float64(bytesProcessed.Load())/elapsed.Seconds())))
		return
	}

	// Specific type mode: original behavior
	if len(*typeNames) == 0 {
		fmt.Println("Usage: gen-trenova -type=Type1[,Type2] OR -domain-path=/path/to/domain")
		flag.Usage()
		os.Exit(2)
	}

	types := strings.Split(*typeNames, ",")

	// Parse the current package
	pkg, err := parsePackage(".")
	if err != nil {
		log.Fatal(err)
	}

	for _, typeName := range types {
		if err := generateForType(pkg, typeName); err != nil {
			log.Fatal(err)
		}
	}
}

func parsePackage(dir string) (*ast.Package, error) {
	// Check cache first
	if cached, ok := packageCache.Load(dir); ok {
		return cached.(*ast.Package), nil
	}

	// Rate limit parsing operations
	ctx := globalCtx
	if err := parseThrottler.Wait(ctx); err != nil {
		return nil, fmt.Errorf("parse throttler: %w", err)
	}

	fset := token.NewFileSet()
	// Filter function to exclude test files and generated files
	filter := func(info os.FileInfo) bool {
		// Yield CPU periodically
		runtime.Gosched()

		name := info.Name()
		return strings.HasSuffix(name, ".go") &&
			!strings.HasSuffix(name, "_test.go") &&
			!strings.HasSuffix(name, "_gen.go")
	}

	// Add small delay to prevent CPU spike
	time.Sleep(2 * time.Millisecond)

	packages, err := parser.ParseDir(
		fset,
		dir,
		filter,
		parser.ParseComments|parser.SkipObjectResolution,
	)
	if err != nil {
		return nil, err
	}

	// Return the first package found and cache it
	for _, pkg := range packages {
		packageCache.Store(dir, pkg)
		return pkg, nil
	}
	return nil, fmt.Errorf("no packages found")
}

func generateForType(pkg *ast.Package, typeName string) error {
	// Find the type declaration
	var typeSpec *ast.TypeSpec
	var structType *ast.StructType

	for _, file := range pkg.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.TypeSpec:
				if x.Name.Name == typeName {
					typeSpec = x
					if st, ok := x.Type.(*ast.StructType); ok {
						structType = st
					}
					return false
				}
			}
			return true
		})
		if typeSpec != nil {
			break
		}
	}

	if typeSpec == nil || structType == nil {
		return fmt.Errorf("type %s not found or not a struct", typeName)
	}

	// Extract metadata from struct
	metadata := extractMetadata(structType)

	// Generate code
	outputFile := *output
	if outputFile == "" {
		outputFile = strings.ToLower(typeName) + "_gen.go"
	}

	return generateCode(outputFile, pkg.Name, typeName, metadata)
}

type fieldInfo struct {
	Name       string
	DBName     string
	JSONName   string
	Type       string
	GoType     string
	BunTags    map[string]string
	Sortable   bool
	Filterable bool
}

type relationInfo struct {
	Name         string // Field name in Go struct
	Type         string // "belongs-to" or "has-many"
	GoType       string // Full Go type (e.g. "*customer.Customer" or "[]*ShipmentMove")
	RelatedType  string // The related type name (e.g. "Customer" or "ShipmentMove")
	RelatedPkg   string // Package of related type if external (e.g. "customer")
	JoinField    string // Join field from bun tag (e.g. "customer_id=id")
	LocalField   string // Local field name (e.g. "customer_id")
	ForeignField string // Foreign field name (e.g. "id")
	IsSlice      bool   // True for has-many relations
	IsPointer    bool   // True if the field is a pointer
}

type metadata struct {
	PackageName string
	TypeName    string
	LowerName   string
	TableName   string
	TableAlias  string
	IDPrefix    string
	Fields      []fieldInfo
	Relations   []relationInfo
	HasVersion  bool
	HasStatus   bool
	Imports     map[string]string // package alias -> import path
}

func extractMetadata(structType *ast.StructType) metadata {
	// Yield CPU at the start of metadata extraction
	runtime.Gosched()

	meta := metadata{
		Fields:    make([]fieldInfo, 0, len(structType.Fields.List)),
		Relations: make([]relationInfo, 0, 4), // Pre-allocate for typical relations
		Imports:   make(map[string]string, 8), // Pre-allocate for typical imports
	}

	fieldCount := 0
	for _, field := range structType.Fields.List {
		// Yield CPU every 10 fields to prevent hogging
		fieldCount++
		if fieldCount%10 == 0 {
			runtime.Gosched()
			time.Sleep(100 * time.Microsecond)
		}
		if field.Tag == nil {
			continue
		}

		// Parse struct tags
		tag := field.Tag.Value
		tag = strings.Trim(tag, "`")

		// Extract bun tags
		bunTag := getTagValue(tag, "bun")
		bunTags := parseBunTag(bunTag)

		// Extract table info from BaseModel
		if len(field.Names) == 0 && strings.Contains(bunTag, "table:") {
			// Parse table:shipment_types,alias:st
			if tableStart := strings.Index(bunTag, "table:"); tableStart != -1 {
				tableInfo := bunTag[tableStart+6:]
				parts := strings.Split(tableInfo, ",")
				meta.TableName = parts[0]

				// Look for alias in the parts
				for _, part := range parts[1:] {
					if strings.HasPrefix(part, "alias:") {
						meta.TableAlias = strings.TrimPrefix(part, "alias:")
						meta.TableAlias = strings.TrimSpace(strings.Trim(meta.TableAlias, "\""))
					}
				}
			}
		}

		// Process regular fields
		for _, name := range field.Names {
			// Check if this is a relationship field
			if relType, hasRel := bunTags["rel"]; hasRel {
				// Extract Go type once to avoid multiple calls
				goType := extractGoType(field.Type)
				relInfo := relationInfo{
					Name:      name.Name,
					Type:      relType,
					GoType:    goType,
					JoinField: bunTags["join"],
					IsSlice:   strings.HasPrefix(goType, "[]"),
					IsPointer: strings.HasPrefix(strings.TrimPrefix(goType, "[]"), "*"),
				}

				// Parse join field (e.g., "customer_id=id")
				if relInfo.JoinField != "" {
					parts := strings.Split(relInfo.JoinField, "=")
					if len(parts) == 2 {
						relInfo.LocalField = parts[0]
						relInfo.ForeignField = parts[1]
					}
				}

				// Extract related type and package
				trimmedType := relInfo.GoType
				trimmedType = strings.TrimPrefix(trimmedType, "[]")
				trimmedType = strings.TrimPrefix(trimmedType, "*")

				if strings.Contains(trimmedType, ".") {
					parts := strings.Split(trimmedType, ".")
					relInfo.RelatedPkg = parts[0]
					relInfo.RelatedType = parts[1]
				} else {
					relInfo.RelatedType = trimmedType
				}

				meta.Relations = append(meta.Relations, relInfo)
				continue
			}

			info := fieldInfo{
				Name:     name.Name,
				DBName:   bunTags["column"],
				JSONName: getTagValue(tag, "json"),
				BunTags:  bunTags,
			}

			// If no explicit column name, use the tag value
			if info.DBName == "" && bunTag != "" && !strings.Contains(bunTag, ":") {
				info.DBName = strings.Split(bunTag, ",")[0]
			}

			// Skip fields with bun:"-" (not stored in database)
			if info.DBName == "-" {
				continue
			}

			// Extract Go type from AST
			info.GoType = extractGoType(field.Type)

			// Handle custom types more efficiently
			if ident, ok := field.Type.(*ast.Ident); ok && info.GoType == "string" &&
				ident.Obj != nil {
				// This is a type alias or custom type, keep the original name
				info.GoType = ident.Name
			}

			// Check for codegen tags
			codegenTag := getTagValue(tag, "codegen")
			if codegenTag != "" {
				// Parse codegen directives like codegen:"sortable,filterable"
				directives := strings.Split(codegenTag, ",")
				for _, directive := range directives {
					switch strings.TrimSpace(directive) {
					case "sortable":
						info.Sortable = true
					case "filterable":
						info.Filterable = true
					case "-sortable":
						info.Sortable = false
					case "-filterable":
						info.Filterable = false
					}
				}
			} else {
				// Use heuristics if no explicit codegen tags
				info.Sortable = isSortableField(info)
				info.Filterable = isFilterableField(info)
			}

			meta.Fields = append(meta.Fields, info)
		}
	}

	// Derive ID prefix from table alias
	if meta.TableAlias != "" {
		meta.IDPrefix = meta.TableAlias + "_"
	}

	// Check for special fields
	for _, field := range meta.Fields {
		if field.Name == "Version" && field.GoType == "int64" {
			meta.HasVersion = true
		}
		if field.Name == "Status" {
			meta.HasStatus = true
		}
	}

	// Extract imports from field types
	for _, field := range meta.Fields {
		if strings.Contains(field.GoType, ".") {
			parts := strings.Split(field.GoType, ".")
			if len(parts) >= 2 {
				pkg := parts[0]
				// Remove pointer prefix if exists
				pkg = strings.TrimPrefix(pkg, "*")
				pkg = strings.TrimPrefix(pkg, "[]")

				// Map common packages to their import paths
				switch pkg {
				case "pulid":
					meta.Imports[pkg] = "github.com/emoss08/trenova/shared/pulid"
				case "domain":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain"
				case "decimal":
					meta.Imports[pkg] = "github.com/shopspring/decimal"
				case "sql":
					meta.Imports[pkg] = "database/sql"
				case "shipment":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/shipment"
				case "billing":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/billing"
				case "permission":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/permission"
				case "accessorialcharge":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
				case "businessunit":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/businessunit"
				case "organization":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/organization"
				case "shipmenttype":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/shipmenttype"
				case "servicetype":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/servicetype"
				case "customer":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/customer"
				case "equipmenttype":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/equipmenttype"
				case "user":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/user"
				case "formulatemplate":
					meta.Imports[pkg] = "github.com/emoss08/trenova/internal/core/domain/formulatemplate"
				default:
					// If we can't map it, we might need to add more mappings
					fmt.Printf(
						"Warning: Unknown package alias '%s' in type '%s'\n",
						pkg,
						field.GoType,
					)
				}
			}
		}
	}

	return meta
}

func extractGoType(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if pkg, ok := t.X.(*ast.Ident); ok {
			return pkg.Name + "." + t.Sel.Name
		}
	case *ast.ArrayType:
		return "[]" + extractGoType(t.Elt)
	case *ast.StarExpr:
		return "*" + extractGoType(t.X)
	case *ast.MapType:
		return "map[" + extractGoType(t.Key) + "]" + extractGoType(t.Value)
	case *ast.InterfaceType:
		// Handle empty interface
		if t.Methods == nil || len(t.Methods.List) == 0 {
			return "any"
		}
		return "any" // For non-empty interfaces too
	case *ast.StructType:
		// Handle inline struct types
		return "struct{}"
	case *ast.FuncType:
		// Handle function types
		return "func()"
	case *ast.ChanType:
		// Handle channel types
		return "chan " + extractGoType(t.Value)
	}
	return "any"
}

func getTagValue(tag, key string) string {
	// Look for key:"value" pattern
	keyPrefix := key + ":\""
	if start := strings.Index(tag, keyPrefix); start != -1 {
		valueStart := start + len(keyPrefix)
		if end := strings.Index(tag[valueStart:], "\""); end != -1 {
			return tag[valueStart : valueStart+end]
		}
	}
	return ""
}

func parseBunTag(tag string) map[string]string {
	result := make(map[string]string)
	if tag == "" {
		return result
	}

	// Handle table definition
	if strings.Contains(tag, "table:") {
		if start := strings.Index(tag, "table:"); start != -1 {
			tableTag := tag[start+6:]
			if end := strings.Index(tableTag, " "); end != -1 {
				tableTag = tableTag[:end]
			}
			parts := strings.Split(tableTag, ",")
			if len(parts) > 0 {
				result["table"] = parts[0]
			}
			for _, part := range parts[1:] {
				if after, ok := strings.CutPrefix(part, "alias:"); ok {
					result["alias"] = after
				}
			}
		}
	}

	// Handle regular column tags
	parts := strings.Split(tag, ",")
	if len(parts) > 0 && !strings.Contains(parts[0], ":") {
		result["column"] = parts[0]
	}

	for _, part := range parts {
		if strings.Contains(part, ":") {
			kv := strings.SplitN(part, ":", 2)
			key := strings.TrimSpace(kv[0])
			value := strings.TrimSpace(kv[1])
			result[key] = value
		}
	}

	return result
}

func isSortableField(field fieldInfo) bool {
	// Check if explicitly marked in bun tags
	if sortable, ok := field.BunTags["sortable"]; ok {
		return sortable == "true"
	}

	// Default rules based on field type and common patterns
	// Timestamps are typically sortable
	if strings.HasSuffix(field.Name, "At") && field.GoType == "int64" {
		return true
	}

	// Enum fields like Status are often sortable
	if field.GoType == "domain.Status" {
		return true
	}

	// Code/Name fields are commonly sortable for display
	if (field.Name == "Code" || field.Name == "Name") && field.GoType == "string" {
		return true
	}

	return false
}

func isFilterableField(field fieldInfo) bool {
	// Check if explicitly marked in bun tags
	if filterable, ok := field.BunTags["filterable"]; ok {
		return filterable == "true"
	}

	// Primary keys and foreign keys are always filterable
	if field.Name == "ID" || strings.HasSuffix(field.Name, "ID") {
		return true
	}

	// Timestamps are commonly filtered
	if strings.HasSuffix(field.Name, "At") && field.GoType == "int64" {
		return true
	}

	// Status fields are commonly filtered
	if field.GoType == "domain.Status" || field.Name == "Status" {
		return true
	}

	// Short string fields are often used for filtering
	if field.GoType == "string" && field.BunTags["type"] != "TEXT" {
		if dbType := field.BunTags["type"]; strings.HasPrefix(dbType, "VARCHAR") {
			return true
		}
	}

	return false
}

// formatWithImports formats the source code and fixes imports using goimports
func formatWithImports(src []byte) ([]byte, error) {
	// Rate limit CPU-intensive formatting
	ctx := globalCtx
	if err := cpuThrottler.Wait(ctx); err != nil {
		return nil, fmt.Errorf("format throttler: %w", err)
	}

	// Yield CPU before intensive operation
	runtime.Gosched()

	// Use goimports to format and fix imports
	formatted, err := imports.Process("", src, &imports.Options{
		Fragment:  false,
		AllErrors: false,
		Comments:  true,
		TabIndent: true,
		TabWidth:  8,
	})

	// Yield CPU after intensive operation
	runtime.Gosched()

	return formatted, err
}

func generateCode(outputFile, packageName, typeName string, meta metadata) error {
	meta.PackageName = packageName
	meta.TypeName = typeName
	meta.LowerName = strings.ToLower(typeName[:1]) + typeName[1:]

	tmpl, err := template.New("combined").Funcs(template.FuncMap{
		"toLower":   strings.ToLower,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"hasField": func(fields []fieldInfo, name string) bool {
			for _, f := range fields {
				if f.Name == name {
					return true
				}
			}
			return false
		},
		"getFieldType": func(fields []fieldInfo, name string) string {
			for _, f := range fields {
				if f.Name == name {
					return f.GoType
				}
			}
			return "any"
		},
	}).Parse(templateV1)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, meta); err != nil {
		return err
	}

	// Format the generated code and fix imports
	formatted, err := formatWithImports(buf.Bytes())
	if err != nil {
		// Fallback to basic formatting if goimports fails
		formatted, err = format.Source(buf.Bytes())
		if err != nil {
			return fmt.Errorf("formatting generated code: %w", err)
		}
	}

	// Write to file
	return os.WriteFile(outputFile, formatted, 0o644)
}

// generateCodeOptimized uses pre-compiled template and buffer pooling
func generateCodeOptimized(outputFile, packageName, typeName string, meta metadata) error {
	// Rate limit template execution
	ctx := globalCtx
	if err := templateThrottler.Wait(ctx); err != nil {
		return fmt.Errorf("template throttler: %w", err)
	}

	meta.PackageName = packageName
	meta.TypeName = typeName
	meta.LowerName = strings.ToLower(typeName[:1]) + typeName[1:]

	// Get buffer from pool
	buf := bufferPool.Get().(*bytes.Buffer)
	defer func() {
		buf.Reset()
		bufferPool.Put(buf)
	}()

	// Yield CPU before template execution
	runtime.Gosched()

	// Use pre-compiled template
	if err := compiledTemplate.Execute(buf, meta); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	// Format the generated code and fix imports
	formatted, err := formatWithImports(buf.Bytes())
	if err != nil {
		// Fallback to basic formatting if goimports fails
		formatted, err = format.Source(buf.Bytes())
		if err != nil {
			return fmt.Errorf("formatting generated code: %w", err)
		}
	}

	// Rate limit file I/O
	if err := fileIOThrottler.Wait(ctx); err != nil {
		return fmt.Errorf("file I/O throttler: %w", err)
	}

	// Track metrics
	bytesProcessed.Add(int64(len(formatted)))
	filesProcessed.Add(1)

	// Write to file with small delay
	time.Sleep(time.Millisecond)
	return os.WriteFile(outputFile, formatted, 0o644)
}

// compileTemplate compiles the template once for reuse
func compileTemplate() (*template.Template, error) {
	return template.New("combined").Funcs(template.FuncMap{
		"toLower":   strings.ToLower,
		"hasPrefix": strings.HasPrefix,
		"hasSuffix": strings.HasSuffix,
		"hasField": func(fields []fieldInfo, name string) bool {
			for _, f := range fields {
				if f.Name == name {
					return true
				}
			}
			return false
		},
		"getFieldType": func(fields []fieldInfo, name string) string {
			for _, f := range fields {
				if f.Name == name {
					return f.GoType
				}
			}
			return "any"
		},
	}).Parse(templateV1)
}

// scanAndGenerate scans the domain folder and generates for all entities
func scanAndGenerate(domainPath string) error {
	fmt.Printf("Scanning domain folder: %s\n", domainPath)

	domains, err := ScanDomainFolder(domainPath)
	if err != nil {
		return fmt.Errorf("scanning domains: %w", err)
	}

	fmt.Printf("Found %d domain entities\n", len(domains))

	// Group by directory to parse packages efficiently
	domainsByDir := make(map[string][]DomainInfo)
	for _, d := range domains {
		dir := filepath.Dir(d.FilePath)
		domainsByDir[dir] = append(domainsByDir[dir], d)
	}

	// Pre-compile template once
	templateOnce.Do(func() {
		var err error
		compiledTemplate, err = compileTemplate()
		if err != nil {
			log.Fatalf("Failed to compile template: %v", err)
		}
	})

	// Convert map to slice for better control
	type dirWork struct {
		dir     string
		domains []DomainInfo
	}

	work := make([]dirWork, 0, len(domainsByDir))
	for dir, domains := range domainsByDir {
		work = append(work, dirWork{dir: dir, domains: domains})
	}

	// Use semaphore for better concurrency control
	sem := semaphore.NewWeighted(int64(*concurrency))

	// Process directories with proper concurrency control
	var wg sync.WaitGroup
	workChan := make(chan dirWork, len(work))
	errorsChan := make(chan error, len(work))

	// Fill work channel
	for _, w := range work {
		workChan <- w
	}
	close(workChan)

	// Start workers with CPU-friendly processing
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			for w := range workChan {
				// Acquire semaphore
				if err := sem.Acquire(globalCtx, 1); err != nil {
					log.Printf("Worker %d: failed to acquire semaphore: %v", workerID, err)
					continue
				}

				// Process with CPU yielding
				func() {
					defer sem.Release(1)

					// Add progressive backoff based on worker ID
					backoff := time.Duration(workerID*10+5) * time.Millisecond
					time.Sleep(backoff)

					// Yield CPU before processing
					runtime.Gosched()

					if err := processDirectory(domainPath, w.dir, w.domains); err != nil {
						select {
						case errorsChan <- fmt.Errorf("processing %s: %w", w.dir, err):
						default:
							// Channel full, log directly
							log.Printf("Error processing %s: %v", w.dir, err)
						}
					}

					// Yield CPU after processing
					runtime.Gosched()
					time.Sleep(5 * time.Millisecond)
				}()
			}
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errorsChan)

	// Collect any errors
	var hasErrors bool
	for err := range errorsChan {
		log.Printf("Error: %v", err)
		hasErrors = true
	}

	if hasErrors {
		return fmt.Errorf("encountered errors during generation")
	}

	return nil
}

// processDirectory processes all domain types in a single directory
func processDirectory(domainPath, dir string, domains []DomainInfo) error {
	// Check context for cancellation
	select {
	case <-globalCtx.Done():
		return globalCtx.Err()
	default:
	}

	fmt.Printf("Processing package in %s (%d entities)\n", dir, len(domains))

	// Progressive delay based on number of entities
	delay := time.Duration(5+len(domains)) * time.Millisecond
	if delay > 50*time.Millisecond {
		delay = 50 * time.Millisecond
	}
	time.Sleep(delay)

	// Parse package once for all types in this directory
	pkg, err := parsePackage(dir)
	if err != nil {
		return fmt.Errorf("failed to parse package: %w", err)
	}

	// Extract all type information at once to avoid repeated AST traversal
	typeMap := make(map[string]*ast.StructType, len(domains))
	fileCount := 0
	for _, file := range pkg.Files {
		extractTypesFromFile(file, typeMap)

		// Yield CPU between files with progressive delay
		fileCount++
		runtime.Gosched()
		if fileCount%3 == 0 {
			time.Sleep(2 * time.Millisecond)
		}
	}

	// Generate for each domain type with CPU-friendly processing
	for idx, domain := range domains {
		// Check context for cancellation
		select {
		case <-globalCtx.Done():
			return globalCtx.Err()
		default:
		}

		structType, found := typeMap[domain.TypeName]
		if !found {
			log.Printf("Warning: type %s not found in parsed AST", domain.TypeName)
			continue
		}

		fmt.Printf("  Generating for %s.%s\n", domain.PackageName, domain.TypeName)

		// Determine output file
		outputFile := filepath.Join(dir, strings.ToLower(domain.TypeName)+"_gen.go")
		if *outputDir != "" {
			// Use custom output directory
			relPath, _ := filepath.Rel(domainPath, dir)
			outputFile = filepath.Join(
				*outputDir,
				relPath,
				strings.ToLower(domain.TypeName)+"_gen.go",
			)
		}

		// Generate directly with the struct type
		metadata := extractMetadata(structType)
		metadata.PackageName = domain.PackageName
		metadata.TypeName = domain.TypeName

		if err := generateCodeOptimized(outputFile, domain.PackageName, domain.TypeName, metadata); err != nil {
			log.Printf("Warning: failed to generate for %s: %v", domain.TypeName, err)
			continue
		}

		// Add progressive delay between generations
		if idx < len(domains)-1 {
			delay := time.Duration(3+idx%5) * time.Millisecond
			time.Sleep(delay)
			runtime.Gosched()
		}
	}

	return nil
}

// extractTypesFromFile extracts struct types from a file without full traversal
func extractTypesFromFile(file *ast.File, typeMap map[string]*ast.StructType) {
	for _, decl := range file.Decls {
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

			typeMap[typeSpec.Name.Name] = structType
		}
	}
}

// initializeRateLimiters sets up rate limiters for CPU throttling
func initializeRateLimiters() {
	// Base rates that scale with concurrency
	parseRate := rate.Limit(10 * (*concurrency))   // AST parsing operations per second
	fileIORate := rate.Limit(20 * (*concurrency))  // File I/O operations per second
	templateRate := rate.Limit(5 * (*concurrency)) // Template executions per second
	cpuRate := rate.Limit(30 * (*concurrency))     // General CPU-intensive ops per second

	// Create rate limiters with burst capacity
	parseThrottler = rate.NewLimiter(parseRate, int(parseRate))
	fileIOThrottler = rate.NewLimiter(fileIORate, int(fileIORate))
	templateThrottler = rate.NewLimiter(templateRate, int(templateRate))
	cpuThrottler = rate.NewLimiter(cpuRate, int(cpuRate))
}

// setupSignalHandling sets up graceful shutdown on interrupt
func setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\nReceived interrupt signal, shutting down gracefully...")
		globalCancel()

		// Give goroutines time to finish
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()
}

// formatBytes formats bytes into human-readable format
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
