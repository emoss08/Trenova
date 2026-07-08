package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	manifestPath = flag.String("manifest", "", "Path to projection override manifest")
	schemaDir    = flag.String("schema", "", "Path to GraphQL schema directory")
	outputPath   = flag.String("output", "", "Path to generated Go file")
	gqlgenPath   = flag.String("gqlgen", "", "Path to gqlgen.yml")
	domainDir    = flag.String("domain", "", "Path to internal/core/domain")
	buncolgenDir = flag.String("buncolgen", "", "Path to pkg/buncolgen")
	goModPath    = flag.String("gomod", "", "Path to go.mod")
)

func main() {
	flag.Parse()

	if *manifestPath == "" || *schemaDir == "" || *outputPath == "" {
		fmt.Fprintln(
			os.Stderr,
			"Usage: projectiongen -manifest=<file> -schema=<dir> -output=<file> [-gqlgen=<file>] [-domain=<dir>] [-buncolgen=<dir>] [-gomod=<file>]",
		)
		os.Exit(1)
	}

	if err := run(generatorOptions{
		ManifestPath: *manifestPath,
		SchemaDir:    *schemaDir,
		OutputPath:   *outputPath,
		GqlgenPath:   *gqlgenPath,
		DomainDir:    *domainDir,
		BuncolgenDir: *buncolgenDir,
		GoModPath:    *goModPath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating GraphQL projections: %v\n", err)
		os.Exit(1)
	}
}
