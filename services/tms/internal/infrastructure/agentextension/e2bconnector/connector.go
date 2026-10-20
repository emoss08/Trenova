package e2bconnector

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentextension/connectorkit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/e2b"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const (
	requestTimeout = 30 * time.Second
	// sandboxLifeMargin keeps the sandbox alive past the code's own limit for
	// the files written before it, the preparation that loads them and the
	// stop after it; a sandbox Trenova fails to stop ends on its own at this
	// lifetime.
	sandboxLifeMargin = 120 * time.Second
	defaultCPUCount   = 2
	defaultMemoryMB   = 512
)

// E2B's published per-second prices for what a sandbox holds, whether it uses
// it or not.
var (
	pricePerCPUSecond = decimal.RequireFromString("0.000014")
	pricePerGiBSecond = decimal.RequireFromString("0.0000045")
	mebibytesPerGiB   = decimal.NewFromInt(1024)
)

var (
	_ agentextensionports.ExtensionConnector = (*Connector)(nil)
	_ agentextensionports.SandboxProvider    = (*Connector)(nil)
)

type Connector struct {
	httpClient *http.Client
	baseURL    string
	sandboxURL string
}

func New() *Connector {
	return newConnector(httpsafe.NewClient(requestTimeout), e2b.DefaultAPIURL, "")
}

func newConnector(httpClient *http.Client, baseURL, sandboxURL string) *Connector {
	return &Connector{httpClient: httpClient, baseURL: baseURL, sandboxURL: sandboxURL}
}

func (c *Connector) Type() agentextension.Type { return agentextension.TypeE2B }

func (c *Connector) Classify(err error) agentextensionports.ProviderFailure {
	switch {
	case errors.Is(err, e2b.ErrInvalidRequest):
		return connectorkit.InvalidRequest(err)
	case restx.IsStatus(err, http.StatusTooManyRequests):
		return agentextensionports.ProviderFailure{
			Kind:   agentextensionports.ProviderFailureRateLimited,
			Detail: "every sandbox the account may run at once is in use",
		}
	default:
		return connectorkit.ClassifyHTTP(err)
	}
}

func (c *Connector) options() []e2b.Option {
	opts := []e2b.Option{e2b.WithHTTPClient(c.httpClient), e2b.WithBaseURL(c.baseURL)}
	if c.sandboxURL != "" {
		opts = append(opts, e2b.WithSandboxURL(c.sandboxURL))
	}

	return opts
}

func (c *Connector) client(
	settings map[string]string,
) (*e2b.Client, agentextension.E2BSettings, error) {
	parsed, err := agentextension.ParseE2BSettings(settings)
	if err != nil {
		return nil, agentextension.E2BSettings{}, err
	}

	client, err := e2b.New(parsed.APIKey, c.options()...)

	return client, parsed, err
}

func (c *Connector) Test(
	ctx context.Context,
	settings map[string]string,
) (*agentextensionports.ConnectorTestResult, error) {
	client, _, err := c.client(settings)
	if err != nil {
		return nil, err
	}

	running, err := client.RunningSandboxes(ctx)
	if err != nil {
		return nil, err
	}

	note := "The key works. No sandboxes are running on this account now."
	if running == 1 {
		note = "The key works. 1 sandbox is running on this account now."
	} else if running > 1 {
		note = "The key works. " + strconv.Itoa(
			running,
		) + " sandboxes are running on this account now."
	}

	return &agentextensionports.ConnectorTestResult{Note: note}, nil
}

// StartSandbox starts a sandbox with no way out to the network and its own
// URLs closed to anyone without its token.
func (c *Connector) StartSandbox(
	ctx context.Context,
	req *agentextensionports.SandboxRequest,
) (*agentextensionports.Sandbox, error) {
	client, settings, err := c.client(req.Settings)
	if err != nil {
		return nil, err
	}

	lifetime := time.Duration(settings.RunSeconds)*time.Second + sandboxLifeMargin
	if req.Lifetime > 0 {
		lifetime = req.Lifetime + sandboxLifeMargin
	}
	created, err := client.CreateSandbox(ctx, &e2b.CreateSandboxRequest{
		TemplateID:          settings.Template,
		TimeoutSeconds:      int(lifetime / time.Second),
		AllowInternetAccess: false,
		Network:             &e2b.NetworkConfig{AllowPublicTraffic: false},
		Metadata:            req.Metadata,
	})
	if err != nil {
		return nil, err
	}

	sandbox := &agentextensionports.Sandbox{
		ID:           created.SandboxID,
		Domain:       created.Host(),
		AccessToken:  created.EnvdAccessToken,
		TrafficToken: created.TrafficToken(),
		CPUCount:     defaultCPUCount,
		MemoryMB:     defaultMemoryMB,
	}
	if detail, detailErr := client.GetSandbox(ctx, created.SandboxID); detailErr == nil {
		if detail.CPUCount > 0 {
			sandbox.CPUCount = detail.CPUCount
		}
		if detail.MemoryMB > 0 {
			sandbox.MemoryMB = detail.MemoryMB
		}
	}

	return sandbox, nil
}

func vendorSandbox(sandbox *agentextensionports.Sandbox) *e2b.Sandbox {
	domain := sandbox.Domain
	traffic := sandbox.TrafficToken

	return &e2b.Sandbox{
		SandboxID:          sandbox.ID,
		EnvdAccessToken:    sandbox.AccessToken,
		TrafficAccessToken: &traffic,
		Domain:             &domain,
	}
}

func (c *Connector) WriteFiles(
	ctx context.Context,
	sandbox *agentextensionports.Sandbox,
	files []agentextensionports.SandboxFile,
) error {
	session, err := e2b.OpenSession(vendorSandbox(sandbox), requestTimeout, c.options()...)
	if err != nil {
		return err
	}
	for idx := range files {
		if err = session.WriteFile(ctx, files[idx].Path, files[idx].Content); err != nil {
			return err
		}
	}

	return nil
}

func (c *Connector) RunCode(
	ctx context.Context,
	req *agentextensionports.RunCodeRequest,
) (*agentextensionports.SandboxExecution, error) {
	session, err := e2b.OpenSession(vendorSandbox(req.Sandbox), req.Timeout, c.options()...)
	if err != nil {
		return nil, err
	}

	var execution *e2b.Execution
	if req.Output != nil {
		execution, err = session.RunCodeStream(ctx, req.Code, e2b.OutputFilter(req.Output))
	} else {
		execution, err = session.RunCode(ctx, req.Code)
	}
	switch {
	case errors.Is(err, context.DeadlineExceeded) && ctx.Err() == nil:
		return &agentextensionports.SandboxExecution{TimedOut: true}, nil
	case errors.Is(err, e2b.ErrIncompleteRun):
		out := executionOf(execution)
		out.Incomplete = true
		return out, nil
	case err != nil:
		return nil, err
	}

	return executionOf(execution), nil
}

func executionOf(execution *e2b.Execution) *agentextensionports.SandboxExecution {
	out := &agentextensionports.SandboxExecution{
		Stdout:  execution.Stdout,
		Stderr:  execution.Stderr,
		Results: make([]agentextensionports.SandboxResult, 0, len(execution.Results)),
	}
	for idx := range execution.Results {
		result := &execution.Results[idx]
		out.Results = append(out.Results, agentextensionports.SandboxResult{
			Text:  result.Text,
			PNG:   result.PNG,
			JSON:  result.JSON,
			Title: result.ChartTitle,
		})
	}
	if execution.Error != nil {
		out.Error = &agentextension.AnalysisError{
			Name:      execution.Error.Name,
			Value:     execution.Error.Value,
			Traceback: execution.Error.Traceback,
		}
	}

	return out
}

func (c *Connector) StopSandbox(
	ctx context.Context,
	settings map[string]string,
	sandboxID string,
) error {
	client, _, err := c.client(settings)
	if err != nil {
		return err
	}

	return client.KillSandbox(ctx, sandboxID)
}

// CostOf prices the time a sandbox ran at E2B's per-second rates for the
// processors and memory it held.
func (c *Connector) CostOf(
	sandbox *agentextensionports.Sandbox,
	ran time.Duration,
) decimal.Decimal {
	if sandbox == nil || ran <= 0 {
		return decimal.Zero
	}

	seconds := decimal.NewFromInt(ran.Milliseconds()).Div(decimal.NewFromInt(1000))
	perSecond := pricePerCPUSecond.Mul(decimal.NewFromInt(int64(sandbox.CPUCount))).Add(
		pricePerGiBSecond.Mul(decimal.NewFromInt(int64(sandbox.MemoryMB)).Div(mebibytesPerGiB)),
	)

	return seconds.Mul(perSecond).Round(6)
}
