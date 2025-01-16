package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/trenova-app/transport/internal/pkg/rptmeta"
)

func main() {
	dir, err := rptmeta.GetCurrentWorkingDirectory()
	if err != nil {
		log.Fatalf("Failed to get current working directory: %v", err)
	}

	// Go back two directories
	dir = filepath.Join(dir, "..", "..")

	parsed, err := rptmeta.Parse(filepath.Join(dir, "internal/pkg/rptmeta/canned", "workers.rptmeta.yaml"))
	if err != nil {
		log.Fatalf("Failed to parse report: %v", err)
	}
	fmt.Printf("Report: %+v\n\n", parsed)
}
