package exaconnector

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/domain/extensionwebwatch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/exa"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const (
	watchRunsPage        = 20
	watchExcerptCharsMax = 600
)

var _ agentextensionports.WebWatchProvider = (*Connector)(nil)

func vendorPeriod(cadence extensionwebwatch.WebWatchCadence) string {
	switch cadence {
	case extensionwebwatch.WebWatchCadenceSixHours:
		return "6h"
	case extensionwebwatch.WebWatchCadenceDaily:
		return "1d"
	case extensionwebwatch.WebWatchCadenceWeekly:
		return "7d"
	default:
		return ""
	}
}

// CreateWatch makes an Exa monitor that runs the watch's search on its
// cadence and reports each run to Trenova. It is sent once, never retried.
func (c *Connector) CreateWatch(
	ctx context.Context,
	req *agentextensionports.WebWatchCreateRequest,
) (*agentextensionports.WebWatchCreated, error) {
	parsed, err := agentextension.ParseExaSettings(req.Settings)
	if err != nil {
		return nil, err
	}
	client, err := exa.New(
		parsed.APIKey,
		exa.WithHTTPClient(c.httpClient),
		exa.WithBaseURL(c.baseURL),
		exa.WithRetry(restx.RetryConfig{}),
	)
	if err != nil {
		return nil, err
	}

	monitor, err := client.CreateMonitor(ctx, &exa.CreateMonitorRequest{
		Name: req.Name,
		Search: exa.MonitorSearch{
			Query:          req.Query,
			NumResults:     extensionwebwatch.WebWatchResultsPerRun,
			IncludeDomains: req.IncludeDomains,
			ExcludeDomains: req.ExcludeDomains,
			Contents: &exa.MonitorContent{
				Highlights: &exa.HighlightOptions{
					Query:         req.Query,
					MaxCharacters: watchExcerptCharsMax,
				},
			},
		},
		Trigger:  &exa.MonitorTrigger{Type: exa.TriggerInterval, Period: vendorPeriod(req.Cadence)},
		Metadata: req.Metadata,
		Webhook: exa.MonitorWebhook{
			URL:    req.WebhookURL,
			Events: []string{exa.EventMonitorRunCompleted},
		},
	})
	if err != nil {
		return nil, err
	}

	return &agentextensionports.WebWatchCreated{
		VendorID: monitor.ID,
		Secret:   monitor.WebhookSecret,
	}, nil
}

func (c *Connector) DeleteWatch(
	ctx context.Context,
	settings map[string]string,
	vendorID string,
) error {
	client, _, err := c.client(settings)
	if err != nil {
		return err
	}

	return client.DeleteMonitor(ctx, vendorID)
}

func (c *Connector) WatchState(
	ctx context.Context,
	settings map[string]string,
	vendorID string,
) (*agentextensionports.WebWatchVendorState, error) {
	client, _, err := c.client(settings)
	if err != nil {
		return nil, err
	}

	monitor, err := client.GetMonitor(ctx, vendorID)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.WebWatchVendorState{
		Disabled: monitor.Status == exa.MonitorStatusDisabled,
	}, nil
}

// WatchRuns lists the monitor's latest runs and reads each completed one in
// full, since the list leaves a run's output out. A run already counted is
// not read again: the watch has kept what it found.
func (c *Connector) WatchRuns(
	ctx context.Context,
	req *agentextensionports.WebWatchRunsRequest,
) ([]agentextensionports.WebWatchRun, error) {
	client, _, err := c.client(req.Settings)
	if err != nil {
		return nil, err
	}

	page, err := client.ListMonitorRuns(ctx, req.VendorID, watchRunsPage)
	if err != nil {
		return nil, err
	}

	runs := make([]agentextensionports.WebWatchRun, 0, len(page.Data))
	for idx := range page.Data {
		run := &page.Data[idx]
		counted := req.After > 0 && run.CreatedAt.Unix() <= req.After
		if run.Status == exa.RunStatusCompleted && run.Output == nil && !counted {
			full, getErr := client.GetMonitorRun(ctx, req.VendorID, run.ID)
			if getErr != nil {
				return nil, getErr
			}
			run = full
		}
		runs = append(runs, watchRunOf(run))
	}

	return runs, nil
}

func watchRunOf(run *exa.MonitorRun) agentextensionports.WebWatchRun {
	out := agentextensionports.WebWatchRun{
		ID:        run.ID,
		Phase:     phaseOfRun(run.Status),
		Failure:   run.FailureReason(),
		CreatedAt: run.CreatedAt.Unix(),
		CostUSD:   decimal.Zero,
	}
	out.SettledAt = out.CreatedAt
	if run.CompletedAt != nil {
		out.SettledAt = run.CompletedAt.Unix()
	}
	if out.Phase == agentextensionports.WebWatchRunCompleted {
		out.CostUSD = extensionwebwatch.ExaMonitorRunCostUSD
	}
	if run.Output == nil {
		return out
	}

	out.Summary = run.Output.Summary()
	out.Results = make([]extensionwebwatch.WebWatchResult, 0, len(run.Output.Results))
	for idx := range run.Output.Results {
		result := &run.Output.Results[idx]
		out.Results = append(out.Results, extensionwebwatch.WebWatchResult{
			Title:         strings.TrimSpace(result.Title),
			URL:           strings.TrimSpace(result.URL),
			PublishedDate: strings.TrimSpace(result.PublishedDate),
			Excerpt:       strings.Join(result.Highlights, " … "),
		})
	}

	return out
}

func phaseOfRun(status string) agentextensionports.WebWatchRunPhase {
	switch status {
	case exa.RunStatusCompleted:
		return agentextensionports.WebWatchRunCompleted
	case exa.RunStatusFailed:
		return agentextensionports.WebWatchRunFailed
	case exa.RunStatusCancelled:
		return agentextensionports.WebWatchRunCancelled
	default:
		return agentextensionports.WebWatchRunPending
	}
}

func (c *Connector) VerifyDelivery(
	delivery *agentextensionports.WebWatchDelivery,
	secret string,
) error {
	at := delivery.At
	if at.IsZero() {
		at = time.Now()
	}

	return exa.VerifySignature(delivery.Header.Get(exa.SignatureHeader), delivery.Body, secret, at)
}
