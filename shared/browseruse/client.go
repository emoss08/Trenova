package browseruse

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/shopspring/decimal"
)

const (
	DefaultBaseURL = "https://api.browser-use.com/api/v2"

	defaultTimeout        = 30 * time.Second
	defaultUserAgent      = "trenova-browseruse/1"
	defaultMaxAttempts    = 2
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	maxResponseBytes      = 8 << 20

	endpointCreateTask = "create-task"
	endpointGetTask    = "get-task"
	endpointStopTask   = "stop-task"
	endpointGetSession = "get-session"
	endpointAccount    = "get-account"
	pathTasks          = "/tasks"
	pathSessions       = "/sessions"
	pathAccount        = "/billing/account"

	StatusCreated  = "created"
	StatusStarted  = "started"
	StatusFinished = "finished"
	StatusFailed   = "failed"
	StatusStopped  = "stopped"

	actionStopTaskAndSession = "stop_task_and_session"

	MaxTaskLength     = 20000
	MaxSteps          = 200
	MaxAllowedDomains = 50
	MaxSecrets        = 20
	MaxMetadata       = 10
)

var (
	ErrAPIKeyRequired    = errors.New("browseruse: API key is required")
	ErrInvalidRequest    = errors.New("browseruse: invalid request")
	ErrUnexpectedPayload = errors.New("browseruse: unexpected response payload")
)

type Client struct {
	transport *restx.Client
}

func New(apiKey string, opts ...Option) (*Client, error) {
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return nil, ErrAPIKeyRequired
	}

	cfg := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	transport, err := restx.New(restx.Config{
		BaseURL:          cfg.baseURL,
		Timeout:          cfg.timeout,
		UserAgent:        cfg.userAgent,
		HTTPClient:       cfg.httpClient,
		Headers:          map[string]string{"X-Browser-Use-API-Key": key},
		Retry:            cfg.retry,
		Observer:         cfg.observer,
		MaxResponseBytes: maxResponseBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("browseruse: configure transport: %w", err)
	}

	return &Client{transport: transport}, nil
}

type SessionSettings struct {
	ProfileID        string  `json:"profileId,omitempty"`
	ProxyCountryCode *string `json:"proxyCountryCode"`
	EnableRecording  bool    `json:"enableRecording"`
}

type CreateTaskRequest struct {
	Task                  string            `json:"task"`
	LLM                   string            `json:"llm,omitempty"`
	StartURL              string            `json:"startUrl,omitempty"`
	MaxSteps              int               `json:"maxSteps,omitempty"`
	StructuredOutput      string            `json:"structuredOutput,omitempty"`
	Metadata              map[string]string `json:"metadata,omitempty"`
	Secrets               map[string]string `json:"secrets,omitempty"`
	AllowedDomains        []string          `json:"allowedDomains"`
	SessionSettings       *SessionSettings  `json:"sessionSettings,omitempty"`
	Vision                bool              `json:"vision"`
	SystemPromptExtension string            `json:"systemPromptExtension,omitempty"`
}

type TaskCreated struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
}

type TaskStep struct {
	Number int    `json:"number"`
	URL    string `json:"url"`
}

type Task struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"sessionId"`
	Status     string     `json:"status"`
	Output     *string    `json:"output"`
	IsSuccess  *bool      `json:"isSuccess"`
	Cost       *string    `json:"cost"`
	Steps      []TaskStep `json:"steps"`
	StartedAt  *string    `json:"startedAt"`
	FinishedAt *string    `json:"finishedAt"`
}

type Session struct {
	ID             string  `json:"id"`
	Status         string  `json:"status"`
	RecordingURL   *string `json:"recordingUrl"`
	PublicShareURL *string `json:"publicShareUrl"`
	ProxyCost      string  `json:"proxyCost"`
}

type Account struct {
	Name                   *string         `json:"name"`
	TotalCreditsBalanceUSD decimal.Decimal `json:"totalCreditsBalanceUsd"`
	ConcurrentSessionLimit int             `json:"concurrentSessionLimit"`
	ActiveSessionCount     int             `json:"activeSessionCount"`
	IsFreeTier             bool            `json:"isFreeTier"`
}

func (c *Client) GetAccount(ctx context.Context) (*Account, error) {
	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointAccount,
		Method:   http.MethodGet,
		Path:     pathAccount,
	})
	if err != nil {
		return nil, err
	}

	out := new(Account)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointAccount, err)
	}

	return out, nil
}

func (r *CreateTaskRequest) Validate() error {
	task := strings.TrimSpace(r.Task)
	switch {
	case task == "":
		return fmt.Errorf("%w: task is required", ErrInvalidRequest)
	case len(task) > MaxTaskLength:
		return fmt.Errorf("%w: task is too long", ErrInvalidRequest)
	case r.MaxSteps < 0 || r.MaxSteps > MaxSteps:
		return fmt.Errorf("%w: maxSteps is out of range", ErrInvalidRequest)
	case len(r.AllowedDomains) == 0:
		return fmt.Errorf("%w: allowedDomains is required", ErrInvalidRequest)
	case len(r.AllowedDomains) > MaxAllowedDomains:
		return fmt.Errorf("%w: too many allowed domains", ErrInvalidRequest)
	case len(r.Secrets) > MaxSecrets:
		return fmt.Errorf("%w: too many secrets", ErrInvalidRequest)
	case len(r.Metadata) > MaxMetadata:
		return fmt.Errorf("%w: too many metadata entries", ErrInvalidRequest)
	}
	for _, domain := range r.AllowedDomains {
		if strings.TrimSpace(domain) == "" {
			return fmt.Errorf("%w: allowed domains cannot be blank", ErrInvalidRequest)
		}
	}
	if r.StartURL != "" {
		parsed, err := url.Parse(r.StartURL)
		if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
			return fmt.Errorf("%w: startUrl must be an https URL", ErrInvalidRequest)
		}
	}

	return nil
}

func (c *Client) CreateTask(ctx context.Context, req *CreateTaskRequest) (*TaskCreated, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint:       endpointCreateTask,
		Method:         http.MethodPost,
		Path:           pathTasks,
		Body:           req,
		ExpectedStatus: []int{http.StatusOK, http.StatusCreated, http.StatusAccepted},
	})
	if err != nil {
		return nil, err
	}

	out := new(TaskCreated)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointCreateTask, err)
	}
	if out.ID == "" {
		return nil, fmt.Errorf("%w: %s: no task id", ErrUnexpectedPayload, endpointCreateTask)
	}

	return out, nil
}

func (c *Client) GetTask(ctx context.Context, taskID string) (*Task, error) {
	id, err := pathID(taskID)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointGetTask,
		Method:   http.MethodGet,
		Path:     pathTasks + "/" + id,
	})
	if err != nil {
		return nil, err
	}

	out := new(Task)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointGetTask, err)
	}

	return out, nil
}

func (c *Client) StopTask(ctx context.Context, taskID string) error {
	id, err := pathID(taskID)
	if err != nil {
		return err
	}

	_, err = c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointStopTask,
		Method:   http.MethodPatch,
		Path:     pathTasks + "/" + id,
		Body:     map[string]string{"action": actionStopTaskAndSession},
	})

	return err
}

func (c *Client) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	id, err := pathID(sessionID)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointGetSession,
		Method:   http.MethodGet,
		Path:     pathSessions + "/" + id,
	})
	if err != nil {
		return nil, err
	}

	out := new(Session)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointGetSession, err)
	}

	return out, nil
}

func pathID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" || strings.ContainsAny(id, "/?#%") {
		return "", fmt.Errorf("%w: id is invalid", ErrInvalidRequest)
	}

	return url.PathEscape(id), nil
}

func (t *Task) Finished() bool {
	return t != nil && (t.Status == StatusFinished || t.Status == StatusFailed ||
		t.Status == StatusStopped)
}

func (t *Task) CostUSD() decimal.Decimal {
	if t == nil || t.Cost == nil {
		return decimal.Zero
	}
	cost, err := decimal.NewFromString(strings.TrimSpace(*t.Cost))
	if err != nil || cost.IsNegative() {
		return decimal.Zero
	}

	return cost
}

func (t *Task) Answer() string {
	if t == nil || t.Output == nil {
		return ""
	}

	return strings.TrimSpace(*t.Output)
}

func (t *Task) VisitedURLs() []string {
	if t == nil {
		return nil
	}
	urls := make([]string, 0, len(t.Steps))
	for _, step := range t.Steps {
		target := strings.TrimSpace(step.URL)
		if target != "" {
			urls = append(urls, target)
		}
	}

	return urls
}
