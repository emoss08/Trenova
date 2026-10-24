package e2b

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/restx"
)

const (
	DefaultAPIURL   = "https://api.e2b.app"
	DefaultDomain   = "e2b.app"
	DefaultTemplate = "code-interpreter-v1"

	defaultTimeout        = 30 * time.Second
	defaultUserAgent      = "trenova-e2b/1"
	defaultMaxAttempts    = 2
	defaultInitialBackoff = 500 * time.Millisecond
	defaultMaxBackoff     = 5 * time.Second
	maxAPIResponseBytes   = 4 << 20
	maxRunResponseBytes   = 48 << 20
	maxLogBytes           = 1 << 20
	runTimeoutMargin      = 15 * time.Second

	envdPort    = 49983
	jupyterPort = 49999

	headerAPIKey       = "X-API-Key"
	headerAccessToken  = "X-Access-Token"
	headerTrafficToken = "E2B-Traffic-Access-Token"
	headerSandboxID    = "E2b-Sandbox-Id"
	headerSandboxPort  = "E2b-Sandbox-Port"
	headerTotalRunning = "X-Total-Running"
	sandboxUser        = "user"
	languagePython     = "python"

	endpointCreate  = "create-sandbox"
	endpointDetail  = "get-sandbox"
	endpointKill    = "kill-sandbox"
	endpointList    = "list-sandboxes"
	endpointUpload  = "upload-file"
	endpointExecute = "execute"

	MaxTimeoutSeconds = 3600
	MaxMetadata       = 10
)

var (
	ErrAPIKeyRequired    = errors.New("e2b: API key is required")
	ErrInvalidRequest    = errors.New("e2b: invalid request")
	ErrUnexpectedPayload = errors.New("e2b: unexpected response payload")
	ErrIncompleteRun     = errors.New("e2b: the code stopped without finishing")
)

type Client struct {
	transport *restx.Client
	once      *restx.Client
	opts      options
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

	apiConfig := restx.Config{
		BaseURL:          cfg.baseURL,
		Timeout:          cfg.timeout,
		UserAgent:        cfg.userAgent,
		HTTPClient:       cfg.httpClient,
		Headers:          map[string]string{headerAPIKey: key},
		Retry:            cfg.retry,
		Observer:         cfg.observer,
		MaxResponseBytes: maxAPIResponseBytes,
	}
	transport, err := restx.New(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("e2b: configure transport: %w", err)
	}
	apiConfig.Retry = restx.RetryConfig{}
	once, err := restx.New(apiConfig)
	if err != nil {
		return nil, fmt.Errorf("e2b: configure transport: %w", err)
	}

	return &Client{transport: transport, once: once, opts: cfg}, nil
}

// NetworkConfig closes a sandbox's own URLs to anyone without its traffic
// token. Egress is closed separately, by AllowInternetAccess.
type NetworkConfig struct {
	AllowPublicTraffic bool `json:"allowPublicTraffic"`
}

type CreateSandboxRequest struct {
	TemplateID          string            `json:"templateID"`
	TimeoutSeconds      int               `json:"timeout"`
	AllowInternetAccess bool              `json:"allow_internet_access"`
	Network             *NetworkConfig    `json:"network,omitempty"`
	Metadata            map[string]string `json:"metadata,omitempty"`
}

func (r *CreateSandboxRequest) Validate() error {
	switch {
	case strings.TrimSpace(r.TemplateID) == "":
		return fmt.Errorf("%w: templateID is required", ErrInvalidRequest)
	case r.TimeoutSeconds < 1 || r.TimeoutSeconds > MaxTimeoutSeconds:
		return fmt.Errorf("%w: timeout is out of range", ErrInvalidRequest)
	case len(r.Metadata) > MaxMetadata:
		return fmt.Errorf("%w: too many metadata entries", ErrInvalidRequest)
	}

	return nil
}

type Sandbox struct {
	TemplateID         string  `json:"templateID"`
	SandboxID          string  `json:"sandboxID"`
	EnvdVersion        string  `json:"envdVersion"`
	EnvdAccessToken    string  `json:"envdAccessToken"`
	TrafficAccessToken *string `json:"trafficAccessToken"`
	Domain             *string `json:"domain"`
}

// Host is the domain the sandbox's ports are reached under.
func (s *Sandbox) Host() string {
	if s == nil || s.Domain == nil || strings.TrimSpace(*s.Domain) == "" {
		return DefaultDomain
	}

	return strings.TrimSpace(*s.Domain)
}

// TrafficToken is the token the sandbox's own URLs ask for, if any.
func (s *Sandbox) TrafficToken() string {
	if s == nil || s.TrafficAccessToken == nil {
		return ""
	}

	return *s.TrafficAccessToken
}

type SandboxDetail struct {
	SandboxID string `json:"sandboxID"`
	State     string `json:"state"`
	CPUCount  int    `json:"cpuCount"`
	MemoryMB  int    `json:"memoryMB"`
}

// CreateSandbox starts a sandbox. It is never retried, whatever the client's
// retry setting: a request that failed after E2B began the sandbox would
// leave a second one running and billed.
func (c *Client) CreateSandbox(ctx context.Context, req *CreateSandboxRequest) (*Sandbox, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	resp, err := c.once.Do(ctx, &restx.Request{
		Endpoint:       endpointCreate,
		Method:         http.MethodPost,
		Path:           "/v2/sandboxes",
		Body:           req,
		ExpectedStatus: []int{http.StatusCreated, http.StatusOK},
	})
	if err != nil {
		return nil, err
	}

	out := new(Sandbox)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointCreate, err)
	}
	if out.SandboxID == "" || out.EnvdAccessToken == "" {
		return nil, fmt.Errorf("%w: %s: no sandbox id or access token", ErrUnexpectedPayload,
			endpointCreate)
	}

	return out, nil
}

func (c *Client) GetSandbox(ctx context.Context, sandboxID string) (*SandboxDetail, error) {
	id, err := pathID(sandboxID)
	if err != nil {
		return nil, err
	}

	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointDetail,
		Method:   http.MethodGet,
		Path:     "/sandboxes/" + id,
	})
	if err != nil {
		return nil, err
	}

	out := new(SandboxDetail)
	if err = sonic.Unmarshal(resp.Body, out); err != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointDetail, err)
	}

	return out, nil
}

// KillSandbox stops a sandbox. One already gone is not an error.
func (c *Client) KillSandbox(ctx context.Context, sandboxID string) error {
	id, err := pathID(sandboxID)
	if err != nil {
		return err
	}

	_, err = c.transport.Do(ctx, &restx.Request{
		Endpoint:       endpointKill,
		Method:         http.MethodDelete,
		Path:           "/sandboxes/" + id,
		ExpectedStatus: []int{http.StatusNoContent, http.StatusOK},
	})
	if restx.IsStatus(err, http.StatusNotFound) {
		return nil
	}

	return err
}

// RunningSandboxes reads how many of the team's sandboxes are running. It
// starts nothing, so it is the connection test.
func (c *Client) RunningSandboxes(ctx context.Context) (int, error) {
	resp, err := c.transport.Do(ctx, &restx.Request{
		Endpoint: endpointList,
		Method:   http.MethodGet,
		Path:     "/v2/sandboxes",
		Query:    url.Values{"state": {"running"}, "limit": {"1"}},
	})
	if err != nil {
		return 0, err
	}

	var listed []map[string]any
	if err = sonic.Unmarshal(resp.Body, &listed); err != nil {
		return 0, fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointList, err)
	}
	header := strings.TrimSpace(resp.Header.Get(headerTotalRunning))
	if total, convErr := strconv.Atoi(header); convErr == nil {
		return total, nil
	}

	return len(listed), nil
}

// Session reaches one running sandbox's own services: its file system and
// its code interpreter.
type Session struct {
	sandbox *Sandbox
	files   *restx.Client
	runner  *restx.Client
	timeout time.Duration
}

func (c *Client) Session(sandbox *Sandbox, runTimeout time.Duration) (*Session, error) {
	return openSession(sandbox, runTimeout, &c.opts)
}

// OpenSession reaches a sandbox started earlier. It needs only the sandbox's
// own tokens, not the API key.
func OpenSession(sandbox *Sandbox, runTimeout time.Duration, opts ...Option) (*Session, error) {
	cfg := defaultOptions()
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	return openSession(sandbox, runTimeout, &cfg)
}

func openSession(sandbox *Sandbox, runTimeout time.Duration, cfg *options) (*Session, error) {
	if sandbox == nil || sandbox.SandboxID == "" || sandbox.EnvdAccessToken == "" {
		return nil, fmt.Errorf("%w: the sandbox has no id or access token", ErrInvalidRequest)
	}
	if _, err := pathID(sandbox.SandboxID); err != nil {
		return nil, err
	}
	if runTimeout <= 0 {
		return nil, fmt.Errorf("%w: the run timeout must be positive", ErrInvalidRequest)
	}

	files, err := portClient(sandbox, envdPort, cfg.timeout, cfg)
	if err != nil {
		return nil, err
	}
	runner, err := portClient(sandbox, jupyterPort, runTimeout+runTimeoutMargin, cfg)
	if err != nil {
		return nil, err
	}

	return &Session{
		sandbox: sandbox,
		files:   files,
		runner:  runner,
		timeout: runTimeout,
	}, nil
}

func portClient(
	sandbox *Sandbox,
	port int,
	timeout time.Duration,
	cfg *options,
) (*restx.Client, error) {
	headers := map[string]string{
		headerAccessToken: sandbox.EnvdAccessToken,
		headerSandboxID:   sandbox.SandboxID,
		headerSandboxPort: strconv.Itoa(port),
	}
	if token := sandbox.TrafficToken(); token != "" {
		headers[headerTrafficToken] = token
	}

	base := cfg.sandboxURL
	if base == "" {
		base = "https://" + strconv.Itoa(port) + "-" + sandbox.SandboxID + "." + sandbox.Host()
	}

	client, err := restx.New(restx.Config{
		BaseURL:          base,
		Timeout:          timeout,
		UserAgent:        cfg.userAgent,
		HTTPClient:       cfg.httpClient,
		Headers:          headers,
		Retry:            restx.RetryConfig{},
		Observer:         cfg.observer,
		MaxResponseBytes: maxRunResponseBytes,
	})
	if err != nil {
		return nil, fmt.Errorf("e2b: configure sandbox transport: %w", err)
	}

	return client, nil
}

// WriteFile writes a file into the sandbox, creating its folders.
func (s *Session) WriteFile(ctx context.Context, path string, content []byte) error {
	if !strings.HasPrefix(path, "/") || strings.Contains(path, "..") {
		return fmt.Errorf(
			"%w: a file path must be absolute and stay where it points",
			ErrInvalidRequest,
		)
	}

	var body bytes.Buffer
	form := multipart.NewWriter(&body)
	part, err := form.CreateFormFile("file", path[strings.LastIndex(path, "/")+1:])
	if err != nil {
		return fmt.Errorf("e2b: build upload: %w", err)
	}
	if _, err = part.Write(content); err != nil {
		return fmt.Errorf("e2b: build upload: %w", err)
	}
	if err = form.Close(); err != nil {
		return fmt.Errorf("e2b: build upload: %w", err)
	}

	_, err = s.files.Do(ctx, &restx.Request{
		Endpoint: endpointUpload,
		Method:   http.MethodPost,
		Path:     "/files",
		Query:    url.Values{"path": {path}, "username": {sandboxUser}},
		Raw:      &restx.RawBody{Data: body.Bytes(), ContentType: form.FormDataContentType()},
	})

	return err
}

// Result is one value the code displayed. JSON is what it displayed as
// application/json, such as a table handed back as values.
type Result struct {
	Text       string
	PNG        []byte
	JSON       map[string]any
	ChartTitle string
	IsMain     bool
}

type ExecutionError struct {
	Name      string
	Value     string
	Traceback string
}

type Execution struct {
	Stdout    string
	Stderr    string
	Results   []Result
	Error     *ExecutionError
	Truncated bool
}

type executeRequest struct {
	Code     string `json:"code"`
	Language string `json:"language"`
}

// RunCode runs Python as one cell and reads everything it printed, displayed
// and raised. It is never retried: running it twice would do its work twice.
func (s *Session) RunCode(ctx context.Context, code string) (*Execution, error) {
	if strings.TrimSpace(code) == "" {
		return nil, fmt.Errorf("%w: code is required", ErrInvalidRequest)
	}

	runCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	resp, err := s.runner.Do(runCtx, &restx.Request{
		Endpoint: endpointExecute,
		Method:   http.MethodPost,
		Path:     "/execute",
		Body:     executeRequest{Code: code, Language: languagePython},
	})
	if err != nil {
		if errors.Is(runCtx.Err(), context.DeadlineExceeded) && ctx.Err() == nil {
			return nil, context.DeadlineExceeded
		}
		return nil, err
	}

	return ParseExecution(resp.Body)
}

type outputLine struct {
	Type         string         `json:"type"`
	Text         *string        `json:"text"`
	PNG          *string        `json:"png"`
	JSON         map[string]any `json:"json"`
	Chart        map[string]any `json:"chart"`
	IsMainResult *bool          `json:"is_main_result"`
	Name         *string        `json:"name"`
	Value        *string        `json:"value"`
	Traceback    *string        `json:"traceback"`
}

// ParseExecution reads the interpreter's stream, one JSON object a line. A
// stream that ends before the interpreter says the cell finished is
// ErrIncompleteRun, with what it did read.
func ParseExecution(body []byte) (*Execution, error) {
	reader := &executionReader{}
	if err := reader.read(bytes.NewReader(body)); err != nil {
		return nil, err
	}

	return reader.finish()
}

type executionReader struct {
	execution Execution
	stdout    strings.Builder
	stderr    strings.Builder
	finished  bool
}

func (r *executionReader) read(body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64<<10), maxRunResponseBytes)
	for scanner.Scan() {
		if err := r.line(scanner.Bytes()); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointExecute, err)
	}

	return nil
}

func (r *executionReader) line(raw []byte) error {
	line := bytes.TrimSpace(raw)
	if len(line) == 0 {
		return nil
	}
	var out outputLine
	if err := sonic.Unmarshal(line, &out); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrUnexpectedPayload, endpointExecute, err)
	}

	switch out.Type {
	case "stdout", "stderr":
		log := &r.stdout
		if out.Type == "stderr" {
			log = &r.stderr
		}
		r.execution.Truncated = appendLog(log, out.Text) || r.execution.Truncated
	case "result":
		result, err := resultOf(&out)
		if err != nil {
			return err
		}
		r.execution.Results = append(r.execution.Results, result)
	case "error":
		r.execution.Error = &ExecutionError{
			Name:      deref(out.Name),
			Value:     deref(out.Value),
			Traceback: deref(out.Traceback),
		}
	case "end_of_execution":
		r.finished = true
	}

	return nil
}

func (r *executionReader) finish() (*Execution, error) {
	execution := r.execution
	execution.Stdout = r.stdout.String()
	execution.Stderr = r.stderr.String()
	if !r.finished && execution.Error == nil {
		return &execution, ErrIncompleteRun
	}

	return &execution, nil
}

func resultOf(out *outputLine) (Result, error) {
	result := Result{
		Text:   deref(out.Text),
		JSON:   out.JSON,
		IsMain: out.IsMainResult != nil && *out.IsMainResult,
	}
	if png := strings.TrimSpace(deref(out.PNG)); png != "" {
		decoded, err := base64.StdEncoding.DecodeString(png)
		if err != nil {
			return Result{}, fmt.Errorf("%w: %s: an image is not base64: %w", ErrUnexpectedPayload,
				endpointExecute, err)
		}
		result.PNG = decoded
	}
	if title, ok := out.Chart["title"].(string); ok {
		result.ChartTitle = strings.TrimSpace(title)
	}

	return result, nil
}

func appendLog(log *strings.Builder, text *string) bool {
	if text == nil {
		return false
	}
	room := maxLogBytes - log.Len()
	if room <= 0 {
		return true
	}
	if len(*text) > room {
		cut := room
		for cut > 0 && !utf8.RuneStart((*text)[cut]) {
			cut--
		}
		log.WriteString((*text)[:cut])
		return true
	}
	log.WriteString(*text)

	return false
}

func deref(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func pathID(raw string) (string, error) {
	id := strings.TrimSpace(raw)
	if id == "" || strings.ContainsAny(id, "/?#%.") {
		return "", fmt.Errorf("%w: id is invalid", ErrInvalidRequest)
	}

	return url.PathEscape(id), nil
}
