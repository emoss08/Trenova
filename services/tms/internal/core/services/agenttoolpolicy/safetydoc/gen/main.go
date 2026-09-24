package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/safetydoc"
)

func main() {
	output := flag.String("output", "", "where to write the tool safety document")
	flag.Parse()

	if err := run(*output); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating the AI tool safety document: %v\n", err)
		os.Exit(1)
	}
}

func run(output string) error {
	if output == "" {
		return errors.New("-output is required")
	}

	catalog, err := registered.Catalog()
	if err != nil {
		return err
	}

	if err = os.WriteFile(output, safetydoc.Render(catalog.All()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}

	return nil
}
