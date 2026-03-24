package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

var (
	seedsDir     = flag.String("seeds", "", "Path to seeds directory")
	outputFile   = flag.String("output", "", "Output file path")
	registerFile = flag.String("register", "", "Output path for register_gen.go")
	pkgName      = flag.String("package", "seeder", "Package name for generated file")
)

type SeedInfo struct {
	Name            string
	ConstName       string
	SourceFile      string
	Directory       string
	PackageName     string
	ConstructorName string
}

func main() {
	flag.Parse()

	if *seedsDir == "" || *outputFile == "" {
		fmt.Fprintln(
			os.Stderr,
			"Usage: seedgen -seeds=<dir> -output=<file> [-register=<file>] [-package=<pkg>]",
		)
		os.Exit(1)
	}

	seeds, err := findSeeds(*seedsDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding seeds: %v\n", err)
		os.Exit(1)
	}

	if err := generateFile(seeds, *outputFile, *pkgName); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated %s with %d seed IDs\n", *outputFile, len(seeds))

	if *registerFile != "" {
		if err := generateRegisterFile(seeds, *registerFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating register file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generated %s with %d seed registrations\n", *registerFile, len(seeds))
	}
}

func findSeeds(dir string) ([]SeedInfo, error) {
	var seeds []SeedInfo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		if strings.HasSuffix(path, "_test.go") || strings.HasSuffix(path, "_gen.go") {
			return nil
		}

		base := filepath.Base(path)
		if base == "register.go" {
			return nil
		}

		seedDir := filepath.Base(filepath.Dir(path))

		fileSeeds, err := parseFile(path, seedDir)
		if err != nil {
			return fmt.Errorf("parsing %s: %w", path, err)
		}

		seeds = append(seeds, fileSeeds...)
		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(seeds, func(i, j int) bool {
		return seeds[i].Name < seeds[j].Name
	})

	return seeds, nil
}

func parseFile(path string, dir string) ([]SeedInfo, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var seeds []SeedInfo

	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		if sel.Sel.Name != "NewBaseSeed" {
			return true
		}

		if len(call.Args) < 1 {
			return true
		}

		lit, ok := call.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}

		name := strings.Trim(lit.Value, `"`)
		seeds = append(seeds, SeedInfo{
			Name:            name,
			ConstName:       toConstName(name),
			SourceFile:      filepath.Base(path),
			Directory:       dir,
			PackageName:     dir,
			ConstructorName: "New" + name + "Seed",
		})

		return true
	})

	return seeds, nil
}

func toConstName(name string) string {
	var result strings.Builder
	result.WriteString("Seed")

	nextUpper := true
	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if nextUpper {
				result.WriteRune(unicode.ToUpper(r))
				nextUpper = false
			} else {
				result.WriteRune(r)
			}
		} else {
			nextUpper = true
		}
	}

	return result.String()
}

func generateFile(seeds []SeedInfo, outputPath, pkgName string) error {
	tmpl := template.Must(template.New("seeds").Parse(seedsTemplate))

	var baseSeeds, devSeeds, testSeeds []SeedInfo
	for _, s := range seeds {
		switch s.Directory {
		case "base":
			baseSeeds = append(baseSeeds, s)
		case "development":
			devSeeds = append(devSeeds, s)
		case "test", "testing":
			testSeeds = append(testSeeds, s)
		}
	}

	var buf bytes.Buffer
	data := struct {
		Package string
		Seeds   []SeedInfo
		Base    []SeedInfo
		Dev     []SeedInfo
		Test    []SeedInfo
	}{
		Package: pkgName,
		Seeds:   seeds,
		Base:    baseSeeds,
		Dev:     devSeeds,
		Test:    testSeeds,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0o644)
}

func generateRegisterFile(seeds []SeedInfo, outputPath string) error {
	var baseSeeds, devSeeds, testSeeds []SeedInfo
	for _, s := range seeds {
		switch s.Directory {
		case "base":
			baseSeeds = append(baseSeeds, s)
		case "development":
			devSeeds = append(devSeeds, s)
		case "test", "testing":
			testSeeds = append(testSeeds, s)
		}
	}

	sortBySourceFile := func(s []SeedInfo) {
		sort.Slice(s, func(i, j int) bool {
			return s[i].SourceFile < s[j].SourceFile
		})
	}
	sortBySourceFile(baseSeeds)
	sortBySourceFile(devSeeds)
	sortBySourceFile(testSeeds)

	var ordered []SeedInfo
	ordered = append(ordered, baseSeeds...)
	ordered = append(ordered, devSeeds...)
	ordered = append(ordered, testSeeds...)

	packages := make(map[string]bool)
	for _, s := range ordered {
		packages[s.PackageName] = true
	}

	var importPackages []string
	for pkg := range packages {
		importPackages = append(importPackages, pkg)
	}
	sort.Strings(importPackages)

	tmpl := template.Must(template.New("register").Parse(registerTemplate))

	var buf bytes.Buffer
	data := struct {
		Seeds    []SeedInfo
		Packages []string
	}{
		Seeds:    ordered,
		Packages: importPackages,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return err
	}

	return os.WriteFile(outputPath, buf.Bytes(), 0o644)
}

const seedsTemplate = `// Code generated by seedgen. DO NOT EDIT.
// Run "go generate" to regenerate this file.

package {{.Package}}

type SeedID string

func (s SeedID) String() string {
	return string(s)
}

const (
{{- range .Seeds}}
	{{.ConstName}} SeedID = "{{.Name}}" // from {{.SourceFile}}
{{- end}}
)

var AllSeedIDs = []SeedID{
{{- range .Seeds}}
	{{.ConstName}},
{{- end}}
}

var BaseSeedIDs = []SeedID{
{{- range .Base}}
	{{.ConstName}},
{{- end}}
}

var DevelopmentSeedIDs = []SeedID{
{{- range .Dev}}
	{{.ConstName}},
{{- end}}
}

var TestSeedIDs = []SeedID{
{{- range .Test}}
	{{.ConstName}},
{{- end}}
}

func ValidateSeedID(id SeedID) bool {
	for _, valid := range AllSeedIDs {
		if id == valid {
			return true
		}
	}
	return false
}
`

const registerTemplate = `// Code generated by seedgen. DO NOT EDIT.
// Run "go generate" to regenerate this file.

package seeds

import (
	"github.com/emoss08/trenova/internal/infrastructure/database/seeder"
{{- range .Packages}}
	"github.com/emoss08/trenova/internal/infrastructure/database/seeds/{{.}}"
{{- end}}
)

func Register(r *seeder.Registry) {
{{- range .Seeds}}
	r.MustRegister({{.PackageName}}.{{.ConstructorName}}())
{{- end}}
}
`
