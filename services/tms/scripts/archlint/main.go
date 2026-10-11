package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"

	"github.com/bytedance/sonic"
)

const (
	modulePath    = "github.com/emoss08/trenova"
	sharedModule  = modulePath + "/shared/"
	maxWarnings   = "32768"
	archLintTool  = "go-arch-lint"
	exitViolation = 1
)

var errViolations = errors.New("architecture rules violated")

type reference struct {
	Line int `json:"Line"`
}

type depWarning struct {
	ComponentName      string    `json:"ComponentName"`
	FileRelativePath   string    `json:"FileRelativePath"`
	ResolvedImportName string    `json:"ResolvedImportName"`
	Reference          reference `json:"Reference"`
}

type textWarning struct {
	Text string `json:"Text"`
}

type payload struct {
	ExecutionWarnings      []textWarning `json:"ExecutionWarnings"`
	ArchWarningsDeps       []depWarning  `json:"ArchWarningsDeps"`
	ArchWarningsNotMatched []textWarning `json:"ArchWarningsNotMatched"`
	ArchWarningsDeepScan   []textWarning `json:"ArchWarningsDeepScan"`
}

type report struct {
	Payload payload `json:"Payload"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	err := run(ctx)
	stop()
	if err != nil {
		fmt.Fprintln(os.Stderr, "archlint:", err)
		os.Exit(exitViolation)
	}
}

func run(ctx context.Context) error {
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(
		ctx,
		"go",
		"tool",
		archLintTool,
		"check",
		"--json",
		"--max-warnings",
		maxWarnings,
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	runErr := cmd.Run()

	var rep report
	if err := sonic.Unmarshal(stdout.Bytes(), &rep); err != nil {
		return fmt.Errorf(
			"%s produced no report: %w: %s",
			archLintTool,
			errors.Join(err, runErr),
			stderr.String(),
		)
	}

	problems := make([]string, 0)
	for _, w := range rep.Payload.ExecutionWarnings {
		problems = append(problems, "config: "+w.Text)
	}
	for _, w := range rep.Payload.ArchWarningsNotMatched {
		problems = append(problems, "not matched: "+w.Text)
	}
	for _, w := range rep.Payload.ArchWarningsDeepScan {
		problems = append(problems, "deep scan: "+w.Text)
	}
	for _, w := range rep.Payload.ArchWarningsDeps {
		if strings.HasPrefix(w.ResolvedImportName, sharedModule) {
			continue
		}
		problems = append(problems, fmt.Sprintf(
			"%s: %s:%d imports %s",
			w.ComponentName,
			strings.TrimPrefix(w.FileRelativePath, "/"),
			w.Reference.Line,
			w.ResolvedImportName,
		))
	}

	if len(problems) == 0 {
		return nil
	}
	for _, p := range problems {
		fmt.Println(p)
	}

	return fmt.Errorf("%w: %d", errViolations, len(problems))
}
