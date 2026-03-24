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
)

var (
	domainDir = flag.String("domain", "", "Path to domain directory containing entity packages")
	outputDir = flag.String("output", "", "Output directory for generated column files")
)

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

	entries, err := os.ReadDir(*domainDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading domain directory: %v\n", err)
		os.Exit(1)
	}

	var totalModels int

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pkgDir := filepath.Join(*domainDir, entry.Name())
		result, parseErr := ParsePackage(pkgDir)
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

func generateColumnsFile(tmpl *template.Template, models []ModelInfo, pkgName string) error {
	data := columnsTemplateData{
		Models: models,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return fmt.Errorf("formatting generated code for %s: %w", pkgName, err)
	}

	outputPath := filepath.Join(*outputDir, pkgName+"_gen.go")
	return os.WriteFile(outputPath, formatted, 0o644)
}

func generateBridgeFile(tmpl *template.Template, models []ModelInfo, pkgDir string) error {
	data := bridgeTemplateData{
		PackageName: models[0].PackageName,
		Models:      models,
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
