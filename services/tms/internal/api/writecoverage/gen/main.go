package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/emoss08/trenova/internal/api/writecoverage"
	"github.com/gin-gonic/gin"
)

var (
	errProblems = errors.New("the write coverage mapping needs attention")
	errStale    = errors.New("the write coverage document is stale")
)

func main() {
	output := flag.String("output", "", "where to write the write coverage document")
	dir := flag.String("dir", ".", "the writecoverage package directory")
	check := flag.Bool("check", false, "fail instead of writing when the document is stale")
	flag.Parse()

	if err := run(*dir, *output, *check); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating the agent write coverage document: %v\n", err)
		os.Exit(1)
	}
}

func run(dir, output string, check bool) error {
	if output == "" {
		return errors.New("-output is required")
	}

	gin.SetMode(gin.ReleaseMode)
	report, err := writecoverage.Load(dir)
	if err != nil {
		return err
	}

	if problems := report.Problems(); len(problems) > 0 {
		fmt.Fprintf(os.Stderr, "%d problems:\n  - %s\n",
			len(problems), strings.Join(problems, "\n  - "))
		return errProblems
	}

	rendered := writecoverage.Render(&report)
	fmt.Fprintf(os.Stdout, "agent write coverage: %d pending writes\n", report.Pending())

	if check {
		current, readErr := os.ReadFile(output)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", output, readErr)
		}
		if !bytes.Equal(current, rendered) {
			return fmt.Errorf("%w: run '%s' in services/tms and commit %s",
				errStale, writecoverage.GenerateCommand, writecoverage.DocumentDisplayPath)
		}
		return nil
	}

	if err = os.WriteFile(output, rendered, 0o600); err != nil {
		return fmt.Errorf("write %s: %w", output, err)
	}

	return nil
}
