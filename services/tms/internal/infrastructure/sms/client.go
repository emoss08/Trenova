package sms

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultBaseURL   = "https://api.twilio.com/2010-04-01"
	requestTimeout   = 30 * time.Second
	maxResponseBytes = 1 << 20
)

var ErrNotConfigured = errors.New("sms: twilio is not enabled")

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Client struct {
	l       *zap.Logger
	http    *http.Client
	cfg     *config.TwilioConfig
	baseURL string
}

func New(p Params) *Client {
	return newClient(
		p.Config.GetTwilioConfig(),
		p.Logger,
		&http.Client{Timeout: requestTimeout},
		defaultBaseURL,
	)
}

func newClient(
	cfg *config.TwilioConfig,
	logger *zap.Logger,
	httpClient *http.Client,
	baseURL string,
) *Client {
	return &Client{
		l:       logger.Named("infrastructure.sms"),
		http:    httpClient,
		cfg:     cfg,
		baseURL: strings.TrimSuffix(baseURL, "/"),
	}
}

type SendRequest struct {
	To   string `json:"to"`
	Body string `json:"body"`
}

type APIError struct {
	StatusCode int    `json:"-"`
	RetryAfter int    `json:"-"`
	Code       int    `json:"code"`
	Message    string `json:"message"`
	MoreInfo   string `json:"more_info"`
}

func (e *APIError) Error() string {
	if e.Code != 0 {
		return fmt.Sprintf(
			"sms: twilio returned status %d (code %d): %s",
			e.StatusCode,
			e.Code,
			e.Message,
		)
	}

	return fmt.Sprintf("sms: twilio returned status %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) Retryable() bool {
	return e.StatusCode == http.StatusTooManyRequests ||
		e.StatusCode >= http.StatusInternalServerError
}

type messageResponse struct {
	SID    string `json:"sid"`
	Status string `json:"status"`
}

func (c *Client) Send(ctx context.Context, req SendRequest) error {
	if !c.cfg.Enabled {
		return ErrNotConfigured
	}

	log := c.l.With(zap.String("operation", "send_sms"))

	httpReq, err := c.messageRequest(ctx, req)
	if err != nil {
		return err
	}

	resp, err := c.http.Do(httpReq)
	if err != nil {
		log.Error("failed to send SMS", zap.Error(err))
		return fmt.Errorf("sms: send message: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return fmt.Errorf("sms: read response: %w", err)
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiErr := apiErrorFrom(resp, body)
		log.Error("twilio refused the SMS", zap.Error(apiErr))
		return apiErr
	}

	var msg messageResponse
	if err = sonic.Unmarshal(body, &msg); err != nil {
		return fmt.Errorf("sms: decode response: %w", err)
	}

	log.Debug("SMS sent", zap.String("sid", msg.SID), zap.String("status", msg.Status))
	return nil
}

func (c *Client) messageRequest(ctx context.Context, req SendRequest) (*http.Request, error) {
	form := url.Values{
		"To":   {req.To},
		"From": {c.cfg.FromNumber},
		"Body": {req.Body},
	}
	endpoint := c.baseURL + "/Accounts/" + url.PathEscape(c.cfg.AccountSID) + "/Messages.json"

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("sms: build request: %w", err)
	}

	httpReq.SetBasicAuth(c.cfg.AccountSID, c.cfg.AuthToken)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	return httpReq, nil
}

func apiErrorFrom(resp *http.Response, body []byte) *APIError {
	apiErr := &APIError{}
	if err := sonic.Unmarshal(body, apiErr); err != nil || apiErr.Message == "" {
		apiErr.Message = http.StatusText(resp.StatusCode)
	}
	apiErr.StatusCode = resp.StatusCode

	if seconds, err := strconv.Atoi(resp.Header.Get("Retry-After")); err == nil && seconds > 0 {
		apiErr.RetryAfter = seconds
	}

	return apiErr
}
