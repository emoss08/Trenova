package trenova

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/bytedance/sonic"
)

var ErrNoCLI = errors.New("no Trenova CLI binary")

type CLI struct {
	Binary string
	Dir    string
}

type AdvanceAction struct {
	Kind    string `json:"kind"`
	Set     string `json:"set,omitempty"`
	Message string `json:"message,omitempty"`
}

type AdvanceSent struct {
	ID             string `json:"id"`
	TransactionSet string `json:"transactionSet"`
	ControlNumber  string `json:"controlNumber"`
}

type Advance struct {
	InstanceID string          `json:"instanceId"`
	Flow       string          `json:"flow"`
	Key        string          `json:"key"`
	AdvancedBy int64           `json:"advancedBy"`
	Woke       bool            `json:"woke"`
	Status     string          `json:"status"`
	Reason     string          `json:"reason,omitempty"`
	NextDue    *int64          `json:"nextDue"`
	Actions    []AdvanceAction `json:"actions"`
	Sent       []AdvanceSent   `json:"sent"`
	Error      string          `json:"error,omitempty"`
}

func (c *CLI) Check() error {
	if strings.TrimSpace(c.Binary) == "" {
		return fmt.Errorf("%w: pass --trenova-cli with a built binary "+
			"(go build -o ./build/trenova ./cmd/cli in services/tms)", ErrNoCLI)
	}
	info, err := os.Stat(c.Binary)
	if err != nil {
		return fmt.Errorf("%w: %s: %w", ErrNoCLI, c.Binary, err)
	}
	if info.IsDir() || info.Mode().Perm()&0o111 == 0 {
		return fmt.Errorf("%w: %s is not an executable file", ErrNoCLI, c.Binary)
	}
	return nil
}

func (c *CLI) Advance(ctx context.Context, instanceID string, by time.Duration) (*Advance, error) {
	if err := c.Check(); err != nil {
		return nil, err
	}
	command := exec.CommandContext(ctx, c.Binary, "edi", "flows", "advance",
		"--instance", instanceID, "--by", by.String(), "--json")
	command.Dir = c.Dir
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	runErr := command.Run()
	report := new(Advance)
	start := bytes.IndexByte(stdout.Bytes(), '{')
	if start >= 0 {
		if err := sonic.Unmarshal(stdout.Bytes()[start:], report); err == nil {
			if report.Error != "" {
				return report, fmt.Errorf("edi flows advance: %s", report.Error)
			}
			return report, nil
		}
	}
	detail := strings.TrimSpace(stderr.String())
	if detail == "" {
		detail = strings.TrimSpace(stdout.String())
	}
	if runErr != nil {
		return nil, fmt.Errorf("edi flows advance --instance %s --by %s: %w: %s",
			instanceID, by, runErr, detail)
	}
	return nil, fmt.Errorf("edi flows advance printed no report: %s", detail)
}
