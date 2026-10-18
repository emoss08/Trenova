package exa

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	endpointCreateMonitor = "create-monitor"
	endpointGetMonitor    = "get-monitor"
	endpointDeleteMonitor = "delete-monitor"
	endpointListRuns      = "list-monitor-runs"
	endpointGetRun        = "get-monitor-run"
	pathMonitors          = "/monitors"

	EventMonitorRunCompleted = "monitor.run.completed"

	TriggerInterval = "interval"

	MonitorStatusActive   = "active"
	MonitorStatusPaused   = "paused"
	MonitorStatusDisabled = "disabled"

	RunStatusPending   = "pending"
	RunStatusRunning   = "running"
	RunStatusCompleted = "completed"
	RunStatusFailed    = "failed"
	RunStatusCancelled = "cancelled"

	MaxMonitorNameLength = 250
	MaxMonitorRunsPage   = 100

	// SignatureHeader carries a webhook delivery's signature as
	// "t=<unix seconds>,v1=<hex HMAC-SHA256 of t.body>".
	SignatureHeader = "Exa-Signature"

	// SignatureTolerance is how far a delivery's timestamp may be from now
	// before it is refused as a replay.
	SignatureTolerance = 5 * time.Minute
)

var (
	ErrSignatureInvalid = errors.New("exa: webhook signature is invalid")

	monitorPeriods = []string{"1h", "6h", "12h", "1d", "7d"}
)

// CreateMonitorRequest runs Search on a schedule and sends each run's new
// results to the webhook; each run drops what earlier runs returned.
type CreateMonitorRequest struct {
	Name     string            `json:"name,omitempty"`
	Search   MonitorSearch     `json:"search"`
	Trigger  *MonitorTrigger   `json:"trigger,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
	Webhook  MonitorWebhook    `json:"webhook"`
}

type MonitorSearch struct {
	Query          string          `json:"query"`
	NumResults     int             `json:"numResults,omitempty"`
	IncludeDomains []string        `json:"includeDomains,omitempty"`
	ExcludeDomains []string        `json:"excludeDomains,omitempty"`
	Contents       *MonitorContent `json:"contents,omitempty"`
}

type MonitorContent struct {
	Highlights *HighlightOptions `json:"highlights,omitempty"`
}

type MonitorTrigger struct {
	Type   string `json:"type"`
	Period string `json:"period"`
}

type MonitorWebhook struct {
	URL    string   `json:"url"`
	Events []string `json:"events,omitempty"`
}

type Monitor struct {
	ID            string     `json:"id"`
	Status        string     `json:"status"`
	NextRunAt     *time.Time `json:"nextRunAt"`
	WebhookSecret string     `json:"webhookSecret,omitempty"`
}

type MonitorRun struct {
	ID          string            `json:"id"`
	MonitorID   string            `json:"monitorId"`
	Status      string            `json:"status"`
	Output      *MonitorRunOutput `json:"output"`
	FailReason  *string           `json:"failReason"`
	CompletedAt *time.Time        `json:"completedAt"`
	CreatedAt   time.Time         `json:"createdAt"`
}

type MonitorRunOutput struct {
	Results []MonitorRunResult `json:"results"`
	Content any                `json:"content"`
}

type MonitorRunResult struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	Title         string   `json:"title"`
	PublishedDate string   `json:"publishedDate"`
	Author        string   `json:"author"`
	Highlights    []string `json:"highlights"`
}

type MonitorRunsPage struct {
	Data       []MonitorRun `json:"data"`
	HasMore    bool         `json:"hasMore"`
	NextCursor *string      `json:"nextCursor"`
}

// Summary is the run's synthesized findings as text. Exa writes a string for
// a text output schema; anything else is rendered as JSON so nothing is lost.
func (o *MonitorRunOutput) Summary() string {
	if o == nil || o.Content == nil {
		return ""
	}
	if text, ok := o.Content.(string); ok {
		return strings.TrimSpace(text)
	}
	encoded, err := sonic.MarshalString(o.Content)
	if err != nil {
		return ""
	}

	return encoded
}

// FailureReason is why a failed run failed, empty for one that did not.
func (r *MonitorRun) FailureReason() string {
	if r == nil || r.FailReason == nil {
		return ""
	}

	return strings.TrimSpace(*r.FailReason)
}

func ValidMonitorPeriod(period string) bool {
	for _, allowed := range monitorPeriods {
		if period == allowed {
			return true
		}
	}

	return false
}

func (r *CreateMonitorRequest) Validate() error {
	if strings.TrimSpace(r.Search.Query) == "" {
		return fmt.Errorf("%w: query is required", ErrInvalidRequest)
	}
	if len(r.Search.Query) > MaxQueryLength {
		return fmt.Errorf("%w: query is longer than %d characters", ErrInvalidRequest,
			MaxQueryLength)
	}
	if len(r.Name) > MaxMonitorNameLength {
		return fmt.Errorf("%w: name is longer than %d characters", ErrInvalidRequest,
			MaxMonitorNameLength)
	}
	if r.Search.NumResults < 0 || r.Search.NumResults > MaxNumResults {
		return fmt.Errorf("%w: numResults is out of range", ErrInvalidRequest)
	}
	if len(r.Search.IncludeDomains)+len(r.Search.ExcludeDomains) > MaxDomainFilters {
		return fmt.Errorf("%w: too many domain filters", ErrInvalidRequest)
	}
	if r.Trigger != nil && (r.Trigger.Type != TriggerInterval ||
		!ValidMonitorPeriod(r.Trigger.Period)) {
		return fmt.Errorf("%w: the trigger must be an interval of 1h, 6h, 12h, 1d or 7d",
			ErrInvalidRequest)
	}
	parsed, err := url.Parse(r.Webhook.URL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("%w: the webhook must be an https URL", ErrInvalidRequest)
	}

	return nil
}

// CreateMonitor creates a monitor and returns it with its webhook secret,
// which Exa returns only here. Call it on a client built without retries: a
// create that failed after Exa made the monitor would leave a second one
// running and billed.
func (c *Client) CreateMonitor(ctx context.Context, req *CreateMonitorRequest) (*Monitor, error) {
	req.Search.Query = strings.TrimSpace(req.Search.Query)
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint:       endpointCreateMonitor,
		Method:         http.MethodPost,
		Path:           pathMonitors,
		Body:           req,
		ExpectedStatus: []int{http.StatusOK, http.StatusCreated},
	})
	if err != nil {
		return nil, err
	}

	out := new(Monitor)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointCreateMonitor, err)
	}
	if out.ID == "" || out.WebhookSecret == "" {
		return nil, fmt.Errorf("%w: %s: no monitor id or webhook secret", ErrUnexpectedPayload,
			endpointCreateMonitor)
	}

	return out, nil
}

func (c *Client) GetMonitor(ctx context.Context, monitorID string) (*Monitor, error) {
	id, err := monitorPathID(monitorID)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointGetMonitor,
		Method:   http.MethodGet,
		Path:     pathMonitors + "/" + id,
	})
	if err != nil {
		return nil, err
	}

	out := new(Monitor)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointGetMonitor, err)
	}

	return out, nil
}

// DeleteMonitor removes a monitor so it runs and bills no more. One already
// gone is not an error.
func (c *Client) DeleteMonitor(ctx context.Context, monitorID string) error {
	id, err := monitorPathID(monitorID)
	if err != nil {
		return err
	}

	_, err = c.transport.Do(ctx, &restx.Request{
		Endpoint:       endpointDeleteMonitor,
		Method:         http.MethodDelete,
		Path:           pathMonitors + "/" + id,
		ExpectedStatus: []int{http.StatusOK, http.StatusAccepted, http.StatusNoContent},
	})
	if restx.IsStatus(err, http.StatusNotFound) {
		return nil
	}

	return err
}

// ListMonitorRuns returns a monitor's runs, newest first.
func (c *Client) ListMonitorRuns(
	ctx context.Context,
	monitorID string,
	limit int,
) (*MonitorRunsPage, error) {
	id, err := monitorPathID(monitorID)
	if err != nil {
		return nil, err
	}
	if limit < 1 || limit > MaxMonitorRunsPage {
		return nil, fmt.Errorf("%w: limit must be from 1 to %d", ErrInvalidRequest,
			MaxMonitorRunsPage)
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointListRuns,
		Method:   http.MethodGet,
		Path:     pathMonitors + "/" + id + "/runs",
		Query:    url.Values{"limit": []string{strconv.Itoa(limit)}},
	})
	if err != nil {
		return nil, err
	}

	out := new(MonitorRunsPage)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointListRuns, err)
	}

	return out, nil
}

// GetMonitorRun returns one run with its full output once it has completed.
func (c *Client) GetMonitorRun(ctx context.Context, monitorID, runID string) (*MonitorRun, error) {
	id, err := monitorPathID(monitorID)
	if err != nil {
		return nil, err
	}
	run, err := monitorPathID(runID)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointGetRun,
		Method:   http.MethodGet,
		Path:     pathMonitors + "/" + id + "/runs/" + run,
	})
	if err != nil {
		return nil, err
	}

	out := new(MonitorRun)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointGetRun, err)
	}

	return out, nil
}

// VerifySignature checks a webhook delivery against the monitor's secret:
// the HMAC-SHA256 of "<t>.<raw body>" must equal v1, compared in constant
// time, and t must be within SignatureTolerance of now so a captured delivery
// cannot be replayed later.
func VerifySignature(header string, body []byte, secret string, now time.Time) error {
	if strings.TrimSpace(secret) == "" {
		return ErrSignatureInvalid
	}

	var timestamp, signature string
	for part := range strings.SplitSeq(header, ",") {
		key, value, found := strings.Cut(strings.TrimSpace(part), "=")
		if !found {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			if signature == "" {
				signature = value
			}
		}
	}
	if timestamp == "" || signature == "" {
		return ErrSignatureInvalid
	}

	seconds, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return ErrSignatureInvalid
	}
	sent := time.Unix(seconds, 0)
	if sent.Before(now.Add(-SignatureTolerance)) || sent.After(now.Add(SignatureTolerance)) {
		return ErrSignatureInvalid
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(timestamp))
	mac.Write([]byte("."))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(signature))) {
		return ErrSignatureInvalid
	}

	return nil
}

// monitorPathID refuses an id that would change the request's path.
func monitorPathID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" || strings.ContainsAny(id, "/?#%\\ ") || id == "." || id == ".." {
		return "", fmt.Errorf("%w: the id is not valid", ErrInvalidRequest)
	}

	return id, nil
}
