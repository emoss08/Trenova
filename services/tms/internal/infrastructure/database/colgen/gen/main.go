package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"text/template"

	"github.com/emoss08/trenova/internal/infrastructure/database/structparse"
)

const (
	helperPackage    = "buncolgen"
	helperImportPath = "github.com/emoss08/trenova/pkg/buncolgen"
)

var (
	domainDir    = flag.String("domain", "", "Glob of domain directories containing entity packages")
	outputDir    = flag.String("output", "", "Output directory for generated column files")
	outputImport = flag.String("import", helperImportPath, "Import path of the output directory")
)

var excludedPackages = map[string]struct{}{
	// Email has manual field maps and model names that collide with existing
	// generated helpers in other packages until buncolgen supports package
	// qualified symbol names.
	"email": {},
}

func main() {
	flag.Parse()

	if *domainDir == "" || *outputDir == "" {
		fmt.Fprintln(os.Stderr, "Usage: buncolgen -domain=<dir> -output=<dir>")
		os.Exit(1)
	}

	colTmpl, err := template.New("columns").Parse(columnsTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing columns template: %v\n", err)
		os.Exit(1)
	}

	bridgeTmpl, err := template.New("bridge").Parse(fieldMapBridgeTemplate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing bridge template: %v\n", err)
		os.Exit(1)
	}

	pkgDirs, err := structparse.PackageDirs(*domainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading domain directories: %v\n", err)
		os.Exit(1)
	}

	var totalModels int

	for _, pkgDir := range pkgDirs {
		entry := dirEntryName(pkgDir)
		if _, excluded := excludedPackages[entry.Name()]; excluded {
			continue
		}

		result, parseErr := structparse.ParsePackage(pkgDir)
		if parseErr != nil {
			fmt.Fprintf(os.Stderr, "Error parsing %s: %v\n", entry.Name(), parseErr)
			os.Exit(1)
		}

		for _, w := range result.Warnings {
			fmt.Fprintf(os.Stderr, "Warning [%s]: %s\n", entry.Name(), w)
		}

		if len(result.Models) == 0 {
			continue
		}

		sort.Slice(result.Models, func(i, j int) bool {
			return result.Models[i].StructName < result.Models[j].StructName
		})

		if err := generateColumnsFile(colTmpl, result.Models, entry.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating columns for %s: %v\n", entry.Name(), err)
			os.Exit(1)
		}

		if err := generateBridgeFile(bridgeTmpl, result.Models, pkgDir); err != nil {
			fmt.Fprintf(os.Stderr, "Error generating bridge for %s: %v\n", entry.Name(), err)
			os.Exit(1)
		}

		totalModels += len(result.Models)
	}

	fmt.Printf("Generated column constants for %d models\n", totalModels)
}

func generateColumnsFile(
	tmpl *template.Template,
	models []structparse.Model,
	pkgName string,
) error {
	data := columnsTemplateData{
		Package:      columnsPackage(pkgName),
		Qual:         helperPackage + ".",
		HelperImport: helperImportPath,
		Models:       models,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting generated code for %s: %w", pkgName, err)
	}

	dir := filepath.Join(*outputDir, columnsPackage(pkgName))
	if err = os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", dir, err)
	}

	return os.WriteFile(filepath.Join(dir, pkgName+"_gen.go"), formatted, 0o644)
}

func generateBridgeFile(tmpl *template.Template, models []structparse.Model, pkgDir string) error {
	data := bridgeTemplateData{
		PackageName:    models[0].PackageName,
		ColumnsImport:  *outputImport + "/" + columnsPackage(filepath.Base(pkgDir)),
		ColumnsPackage: columnsPackage(filepath.Base(pkgDir)),
		Models:         models,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting bridge file for %s: %w", models[0].PackageName, err)
	}

	outputPath := filepath.Join(pkgDir, "fieldmap_gen.go")
	return os.WriteFile(outputPath, formatted, 0o644)
}

type dirEntryName string

func (d dirEntryName) Name() string {
	return filepath.Base(string(d))
}

func columnsPackage(domainPackage string) string {
	return domainPackage + "col"
}
