package agentextensionservice

import (
	"bufio"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/codemode"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sandboxHome = "/home/user"

type localSandbox struct {
	*fakeSandbox
	root string
}

func newLocalSandbox(t *testing.T) *localSandbox {
	t.Helper()

	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 is not installed; the script runner's own code is not exercised")
	}

	return &localSandbox{fakeSandbox: newFakeSandbox(), root: t.TempDir()}
}

func (l *localSandbox) local(path string) string {
	return filepath.Join(l.root, strings.TrimPrefix(path, sandboxHome))
}

func (l *localSandbox) rewrite(content []byte) []byte {
	return []byte(strings.ReplaceAll(string(content), sandboxHome, l.root))
}

func (l *localSandbox) WriteFiles(
	_ context.Context,
	_ *agentextensionports.Sandbox,
	files []agentextensionports.SandboxFile,
) error {
	for _, file := range files {
		target := l.local(file.Path)
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		content := file.Content
		if strings.HasSuffix(file.Path, ".py") || strings.HasSuffix(file.Path, "config.json") {
			content = l.rewrite(content)
		}
		temp := target + ".part"
		if err := os.WriteFile(temp, content, 0o600); err != nil {
			return err
		}
		if err := os.Rename(temp, target); err != nil {
			return err
		}
	}

	return nil
}

func (l *localSandbox) RunCode(
	ctx context.Context,
	req *agentextensionports.RunCodeRequest,
) (*agentextensionports.SandboxExecution, error) {
	runCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	cmd := exec.CommandContext(runCtx, "python3", "-c", string(l.rewrite([]byte(req.Code))))
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		return nil, err
	}

	reader := bufio.NewReader(stdout)
	for {
		line, readErr := reader.ReadString('\n')
		if line != "" && req.Output != nil {
			if _, filterErr := req.Output("stdout", line); filterErr != nil {
				return nil, filterErr
			}
		}
		if readErr != nil {
			break
		}
	}
	waitErr := cmd.Wait()
	if errors.Is(runCtx.Err(), context.DeadlineExceeded) {
		return &agentextensionports.SandboxExecution{TimedOut: true}, nil
	}
	var exit *exec.ExitError
	if waitErr != nil && !errors.As(waitErr, &exit) {
		return nil, waitErr
	}

	return &agentextensionports.SandboxExecution{Stderr: stderr.String()}, nil
}

func runLocalScript(
	t *testing.T,
	code string,
	settings map[string]string,
) (*agentextensionports.ScriptOutcome, *fakeBroker) {
	t.Helper()

	sandbox := newLocalSandbox(t)
	f := newScriptFixture(t, sandbox)
	f.enableScripts(t, settings)
	broker := newFakeBroker()
	broker.index = append(broker.index, codemode.Read{
		Name:        "customer_get",
		Description: "Get one customer.",
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{"customerId": map[string]any{"type": "string"}},
		},
	})
	broker.answers["customer_get"] = codemode.Reply{
		Status: codemode.CallOK, Data: []byte(`{"name":"Acme"}`),
	}
	req := f.scriptRequest(broker)
	req.Code = code

	outcome, err := f.svc.RunScript(t.Context(), req)
	require.NoError(t, err)

	return outcome, broker
}

func TestScriptRunsThroughTheRealShimAndRunner(t *testing.T) {
	t.Parallel()

	outcome, broker := runLocalScript(t, strings.Join([]string{
		"rows = trenova.shipment.list(status=['Delayed'])['items']",
		"names = trenova.gather(*[trenova.customer.get.defer(customerId='c%d' % i) for i in range(3)])",
		"print('late', len(rows), [n['name'] for n in names])",
		"print(trenova.inputs['day'])",
		"trenova.result({'late': len(rows)})",
	}, "\n"), nil)

	assert.Equal(t, agentextensionports.ScriptSucceeded, outcome.Status, outcome.Output)
	assert.Contains(t, outcome.Output, "late 1 ['Acme', 'Acme', 'Acme']")
	assert.Contains(t, outcome.Output, "2026-10-20")
	assert.JSONEq(t, `{"late":1}`, string(outcome.Result))
	assert.Len(t, broker.seen(), 4)
	assert.Len(t, outcome.RoundTripsMs, 4, "the shim reports each round trip it waited")
}

func TestScriptRefusalRaisesAndIsReported(t *testing.T) {
	t.Parallel()

	outcome, broker := runLocalScript(t, strings.Join([]string{
		"try:",
		"    trenova.invoice.void(invoiceIds=['inv_1'])",
		"except AttributeError as missing:",
		"    print('no such read:', missing)",
		"try:",
		"    trenova._wait('x', trenova._send('invoice_void', {})[0], 0)",
		"except trenova.Refused as refused:",
		"    print('refused', refused.status)",
	}, "\n"), nil)

	assert.Equal(t, agentextensionports.ScriptSucceeded, outcome.Status, outcome.Output)
	assert.Contains(t, outcome.Output, "no such read")
	assert.Contains(t, outcome.Output, "refused refused")
	require.Len(t, broker.seen(), 1, "a frame the script forges still reaches only the broker")
	assert.Equal(t, "invoice_void", broker.seen()[0].Tool)
}

func TestScriptThatRaisesIsReportedWithItsError(t *testing.T) {
	t.Parallel()

	outcome, _ := runLocalScript(t, "rows = {}\nprint(rows['missing'])", nil)

	assert.Equal(t, agentextensionports.ScriptErrored, outcome.Status)
	require.NotNil(t, outcome.Error)
	assert.Equal(t, "KeyError", outcome.Error.Name)
	assert.Contains(t, outcome.Error.Traceback, "script.py")
}

func TestScriptThatRunsTooLongIsStopped(t *testing.T) {
	t.Parallel()

	started := time.Now()
	outcome, _ := runLocalScript(t, "while True:\n    pass", map[string]string{
		"scriptSeconds": "15",
	})

	assert.Equal(t, agentextensionports.ScriptTimedOut, outcome.Status)
	assert.Contains(t,
		[]codemode.Limit{codemode.LimitWall, codemode.LimitCPU}, outcome.Limit)
	assert.Less(t, time.Since(started), 40*time.Second)
}

func TestScriptThatAllocatesTooMuchIsStopped(t *testing.T) {
	t.Parallel()

	outcome, _ := runLocalScript(t, "blob = bytearray(4 << 30)\nprint(len(blob))", nil)

	assert.Equal(t, agentextensionports.ScriptStopped, outcome.Status)
	assert.Equal(t, codemode.LimitMemory, outcome.Limit)
}

func TestScriptCannotRaiseItsOwnLimits(t *testing.T) {
	t.Parallel()

	outcome, _ := runLocalScript(t, strings.Join([]string{
		"import resource",
		"try:",
		"    resource.setrlimit(resource.RLIMIT_AS, (resource.RLIM_INFINITY, resource.RLIM_INFINITY))",
		"    print('raised')",
		"except (ValueError, OSError) as refused:",
		"    print('kept', type(refused).__name__)",
	}, "\n"), nil)

	if os.Geteuid() == 0 {
		t.Skip("root may raise a hard limit; the sandbox runs scripts as an unprivileged user")
	}
	assert.Contains(t, outcome.Output, "kept")
}
