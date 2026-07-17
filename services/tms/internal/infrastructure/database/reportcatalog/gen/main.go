package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	manifestPath   = flag.String("manifest", "", "Path to the reportcatalog.yml curation manifest")
	domainDir      = flag.String("domain", "", "Path to the domain directory containing entity packages")
	domaintypesDir = flag.String("domaintypes", "", "Path to the shared domaintypes package for enum resolution")
	outputPath     = flag.String("output", "", "Output path for the generated catalog file")
)

func main() {
	flag.Parse()

	if *manifestPath == "" || *domainDir == "" || *outputPath == "" {
		fmt.Fprintln(os.Stderr, "Usage: reportcataloggen -manifest=<file> -domain=<dir> [-domaintypes=<dir>] -output=<file>")
		os.Exit(1)
	}

	manifest, manifestRaw, err := LoadManifest(*manifestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	b, err := newBuilder(manifest, *domainDir, *domaintypesDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	catalog, err := b.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	for _, warning := range b.warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
	}

	version := ComputeVersion(catalog, manifestRaw)

	source, err := EmitSource(catalog, version)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*outputPath, source, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", *outputPath, err)
		os.Exit(1)
	}

	entityCount := len(catalog.Entities)
	fieldCount := 0
	edgeCount := 0
	for i := range catalog.Entities {
		fieldCount += len(catalog.Entities[i].Fields)
		edgeCount += len(catalog.Entities[i].Edges)
	}

	fmt.Printf("Generated report catalog: %d entities, %d fields, %d edges (%s)\n",
		entityCount, fieldCount, edgeCount, version[:16]+"…")
}
