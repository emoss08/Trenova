package main

import (
	"flag"
	"fmt"
	"os"
)

var (
	manifestPath = flag.String("manifest", "", "Path to mapper override manifest")
	outputPath   = flag.String("output", "", "Path to generated Go file")
	outputDir    = flag.String("output-dir", "", "Path to generated mapper package directory")
	gqlgenPath   = flag.String("gqlgen", "", "Path to gqlgen.yml")
	modelPath    = flag.String("model", "", "Path to gqlmodel models_gen.go")
	domainDir    = flag.String("domain", "", "Path to internal/core/domain")
	resolverDir  = flag.String("resolver", "", "Path to resolver package")
	goModPath    = flag.String("gomod", "", "Path to go.mod")
)

func main() {
	flag.Parse()

	if err := run(&generatorOptions{
		ManifestPath: *manifestPath,
		OutputPath:   *outputPath,
		OutputDir:    *outputDir,
		GqlgenPath:   *gqlgenPath,
		ModelPath:    *modelPath,
		DomainDir:    *domainDir,
		ResolverDir:  *resolverDir,
		GoModPath:    *goModPath,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating GraphQL resolver mappers: %v\n", err)
		os.Exit(1)
	}
}
