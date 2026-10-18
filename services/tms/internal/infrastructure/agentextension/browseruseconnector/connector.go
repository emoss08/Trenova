package browseruseconnector

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/agentextension/connectorkit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentextensionports"
	"github.com/emoss08/trenova/shared/browseruse"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/restx"
)

const requestTimeout = 30 * time.Second

var (
	_ agentextensionports.ExtensionConnector = (*Connector)(nil)
	_ agentextensionports.BrowserProvider    = (*Connector)(nil)
)

type Connector struct {
	httpClient *http.Client
	baseURL    string
}

func New() *Connector {
	return newConnector(httpsafe.NewClient(requestTimeout), browseruse.DefaultBaseURL)
}

func newConnector(httpClient *http.Client, baseURL string) *Connector {
	return &Connector{httpClient: httpClient, baseURL: baseURL}
}

func (c *Connector) Type() agentextension.Type { return agentextension.TypeBrowserUse }

func (c *Connector) Classify(err error) agentextensionports.ProviderFailure {
	switch {
	case errors.Is(err, browseruse.ErrInvalidRequest):
		return connectorkit.InvalidRequest(err)
	case restx.IsStatus(err, http.StatusTooManyRequests):
		return agentextensionports.ProviderFailure{
			Kind:   agentextensionports.ProviderFailureRateLimited,
			Detail: "every browser the account may run at once is busy",
		}
	default:
		return connectorkit.ClassifyHTTP(err)
	}
}

func (c *Connector) Test(
	ctx context.Context,
	settings map[string]string,
) (*agentextensionports.ConnectorTestResult, error) {
	client, _, err := c.client(settings, true)
	if err != nil {
		return nil, err
	}

	account, err := client.GetAccount(ctx)
	if err != nil {
		return nil, err
	}

	note := "The account has $" + account.TotalCreditsBalanceUSD.StringFixed(2) +
		" of credit and runs up to " + strconv.Itoa(account.ConcurrentSessionLimit) +
		" browsers at once."
	if account.IsFreeTier {
		note += " It is on the free tier, which limits how long a task may run."
	}

	return &agentextensionports.ConnectorTestResult{Note: note}, nil
}

// client is the vendor's client. Starting a task is never retried: a request
// that failed after the vendor began it would start a second browser doing
// the same thing on the portal.
func (c *Connector) client(
	settings map[string]string,
	retry bool,
) (*browseruse.Client, agentextension.BrowserUseSettings, error) {
	parsed, err := agentextension.ParseBrowserUseSettings(settings)
	if err != nil {
		return nil, agentextension.BrowserUseSettings{}, err
	}

	opts := []browseruse.Option{
		browseruse.WithHTTPClient(c.httpClient),
		browseruse.WithBaseURL(c.baseURL),
	}
	if !retry {
		opts = append(opts, browseruse.WithRetry(restx.RetryConfig{}))
	}
	client, err := browseruse.New(parsed.APIKey, opts...)

	return client, parsed, err
}

// StartTask runs a task in a browser that stays open after it ends, so a
// follow-up can carry on from the page it stopped at: a portal waiting for a
// sign-in code a person relays. A request with a session continues in that
// browser; one without opens a new one first. Nothing here is retried, and a
// browser opened for a task that could not start is stopped.
func (c *Connector) StartTask(
	ctx context.Context,
	req *agentextensionports.PortalTaskRequest,
) (*agentextensionports.PortalTaskStarted, error) {
	client, settings, err := c.client(req.Settings, false)
	if err != nil {
		return nil, err
	}

	sessionID := req.SessionID
	opened := false
	if sessionID == "" {
		session := &browseruse.CreateSessionRequest{
			ProfileID:       req.ProfileID,
			EnableRecording: settings.RecordSessions,
			KeepAlive:       true,
			PersistMemory:   true,
		}
		if settings.ProxyCountry != agentextension.ProxyCountryNone {
			country := settings.ProxyCountry
			session.ProxyCountryCode = &country
		}
		created, createErr := client.CreateSession(ctx, session)
		if createErr != nil {
			return nil, createErr
		}
		sessionID = created.ID
		opened = true
	}

	task, err := client.CreateTask(ctx, &browseruse.CreateTaskRequest{
		Task:                  req.Instruction,
		LLM:                   settings.Model,
		StartURL:              req.StartURL,
		MaxSteps:              settings.MaxSteps,
		Metadata:              req.Metadata,
		Secrets:               req.Secrets,
		AllowedDomains:        req.AllowedDomains,
		SessionID:             sessionID,
		Vision:                settings.AllowScreenshots,
		SystemPromptExtension: req.Guidance,
	})
	if err != nil {
		if opened {
			stopCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), requestTimeout)
			defer cancel()
			if stopErr := client.StopSession(stopCtx, sessionID); stopErr != nil {
				stopped := fmt.Errorf("stop the browser opened for it: %w", stopErr)
				return nil, errors.Join(err, stopped)
			}
		}
		return nil, err
	}

	return &agentextensionports.PortalTaskStarted{TaskID: task.ID, SessionID: sessionID}, nil
}

func (c *Connector) TaskState(
	ctx context.Context,
	settings map[string]string,
	taskID string,
) (*agentextensionports.PortalTaskState, error) {
	client, _, err := c.client(settings, true)
	if err != nil {
		return nil, err
	}

	task, err := client.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}

	return &agentextensionports.PortalTaskState{
		Phase:       phaseOf(task.Status),
		Answer:      task.Answer(),
		Succeeded:   task.IsSuccess,
		VisitedURLs: task.VisitedURLs(),
		Steps:       len(task.Steps),
		CostUSD:     task.CostUSD(),
		SessionID:   strings.TrimSpace(task.SessionID),
	}, nil
}

func phaseOf(status string) agentextensionports.PortalTaskPhase {
	switch status {
	case browseruse.StatusFinished:
		return agentextensionports.PortalTaskFinished
	case browseruse.StatusFailed:
		return agentextensionports.PortalTaskFailed
	case browseruse.StatusStopped:
		return agentextensionports.PortalTaskStopped
	default:
		return agentextensionports.PortalTaskRunning
	}
}

func (c *Connector) StopTask(ctx context.Context, settings map[string]string, taskID string) error {
	client, _, err := c.client(settings, true)
	if err != nil {
		return err
	}

	return client.StopTask(ctx, taskID)
}

// StopSession closes the browser a task ran in, once its run has ended.
func (c *Connector) StopSession(
	ctx context.Context,
	settings map[string]string,
	sessionID string,
) error {
	client, _, err := c.client(settings, true)
	if err != nil {
		return err
	}

	return client.StopSession(ctx, sessionID)
}

func (c *Connector) Recording(
	ctx context.Context,
	settings map[string]string,
	sessionID string,
) (*agentextensionports.PortalRecording, error) {
	client, _, err := c.client(settings, true)
	if err != nil {
		return nil, err
	}

	session, err := client.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	recording := &agentextensionports.PortalRecording{}
	if session.RecordingURL != nil {
		recording.RecordingURL = strings.TrimSpace(*session.RecordingURL)
	}
	if session.PublicShareURL != nil {
		recording.ShareURL = strings.TrimSpace(*session.PublicShareURL)
	}

	return recording, nil
}

// LiveView is the address of the open browser, where a person watches the
// task or takes over to sign in.
func (c *Connector) LiveView(
	ctx context.Context,
	settings map[string]string,
	sessionID string,
) (string, error) {
	client, _, err := c.client(settings, true)
	if err != nil {
		return "", err
	}

	session, err := client.GetSession(ctx, sessionID)
	if err != nil {
		return "", err
	}
	if session.LiveURL == nil {
		return "", nil
	}

	return strings.TrimSpace(*session.LiveURL), nil
}
