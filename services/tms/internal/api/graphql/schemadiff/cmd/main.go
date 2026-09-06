package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
)

const (
	exitOK       = 0
	exitBreaking = 1
	exitUsage    = 2
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	flags := flag.NewFlagSet("schemadiff", flag.ContinueOnError)
	flags.SetOutput(stderr)
	base := flags.String("base", "", "directory holding the base branch's *.graphqls files")
	head := flags.String("head", "", "directory holding the current *.graphqls files")
	failOnDangerous := flags.Bool("fail-on-dangerous", false,
		"also exit non-zero for dangerous changes such as added enum values")
	githubAnnotations := flags.Bool("github-annotations", false,
		"emit GitHub Actions workflow commands so changes surface as PR annotations")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if *base == "" || *head == "" {
		fmt.Fprintln(stderr, "schemadiff: -base and -head are both required")
		flags.Usage()
		return exitUsage
	}

	baseSchema, err := schemadiff.LoadDir(*base)
	if err != nil {
		fmt.Fprintf(stderr, "schemadiff: base: %v\n", err)
		return exitUsage
	}
	headSchema, err := schemadiff.LoadDir(*head)
	if err != nil {
		fmt.Fprintf(stderr, "schemadiff: head: %v\n", err)
		return exitUsage
	}

	report := schemadiff.Compare(baseSchema, headSchema)
	for _, change := range report.Changes {
		fmt.Fprintln(stdout, change)
		if *githubAnnotations {
			if annotation, ok := githubAnnotation(change); ok {
				fmt.Fprintln(stdout, annotation)
			}
		}
	}

	breaking := len(report.Breaking())
	dangerous := len(report.Dangerous())
	fmt.Fprintf(stdout, "\n%d change(s): %d breaking, %d dangerous, %d safe\n",
		len(report.Changes), breaking, dangerous,
		len(report.Changes)-breaking-dangerous)

	if breaking > 0 || (*failOnDangerous && dangerous > 0) {
		return exitBreaking
	}

	return exitOK
}

func githubAnnotation(change schemadiff.Change) (string, bool) {
	var level string
	switch change.Severity {
	case schemadiff.SeverityBreaking:
		level = "error"
	case schemadiff.SeverityDangerous:
		level = "warning"
	default:
		return "", false
	}

	return fmt.Sprintf(
		"::%s title=GraphQL schema %s change::%s: %s",
		level,
		change.Severity,
		change.Path,
		change.Message,
	), true
}
