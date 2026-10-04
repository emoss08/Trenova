package turnstile

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxTokenLength   = 2048
	maxResponseBytes = 64 << 10
)

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Verifier struct {
	enabled          bool
	secret           string
	verifyURL        string
	expectedHostname string
	testKeys         bool
	client           *http.Client
	l                *zap.Logger
}

type siteVerifyResponse struct {
	Success     bool     `json:"success"`
	ChallengeTS string   `json:"challenge_ts"`
	Hostname    string   `json:"hostname"`
	ErrorCodes  []string `json:"error-codes"`
	Action      string   `json:"action"`
	CData       string   `json:"cdata"`
}

func New(p Params) services.TurnstileVerifier {
	cloud := p.Config.Platform.Cloud.Turnstile
	return NewVerifier(&Options{
		Enabled:          cloud.Enabled,
		SecretKey:        cloud.SecretKey,
		VerifyURL:        cloud.GetVerifyURL(),
		Timeout:          cloud.GetTimeout(),
		ExpectedHostname: hostnameOf(p.Config.App.GetWebBaseURL()),
		Logger:           p.Logger,
	})
}

type Options struct {
	Enabled          bool
	SecretKey        string
	VerifyURL        string
	Timeout          time.Duration
	ExpectedHostname string
	Client           *http.Client
	Logger           *zap.Logger
}

func NewVerifier(opts *Options) *Verifier {
	client := opts.Client
	if client == nil {
		client = &http.Client{Timeout: opts.Timeout}
	}
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	secret := strings.TrimSpace(opts.SecretKey)

	return &Verifier{
		enabled:          opts.Enabled,
		secret:           secret,
		verifyURL:        strings.TrimSpace(opts.VerifyURL),
		expectedHostname: strings.ToLower(strings.TrimSpace(opts.ExpectedHostname)),
		testKeys:         config.IsTurnstileTestSecret(secret),
		client:           client,
		l:                logger.Named("turnstile"),
	}
}

func (v *Verifier) Enabled() bool {
	return v.enabled
}

func (v *Verifier) Verify(ctx context.Context, req *services.TurnstileVerification) error {
	if !v.enabled {
		return nil
	}

	token := strings.TrimSpace(req.Token)
	if token == "" || len(token) > maxTokenLength {
		return services.ErrTurnstileRejected
	}

	result, err := v.siteVerify(ctx, token, req)
	if err != nil {
		return err
	}

	if !result.Success {
		v.l.Info("turnstile rejected a token", zap.Strings("errorCodes", result.ErrorCodes))
		return fmt.Errorf("%w: %s", services.ErrTurnstileRejected, strings.Join(result.ErrorCodes, ","))
	}

	if err = v.checkHostname(result.Hostname); err != nil {
		return err
	}

	return v.checkAction(result.Action, req.ExpectedAction)
}

func (v *Verifier) siteVerify(
	ctx context.Context,
	token string,
	req *services.TurnstileVerification,
) (*siteVerifyResponse, error) {
	form := url.Values{}
	form.Set("secret", v.secret)
	form.Set("response", token)
	if ip := strings.TrimSpace(req.RemoteIP); ip != "" {
		form.Set("remoteip", ip)
	}

	httpReq, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		v.verifyURL,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return nil, fmt.Errorf("%w: build request: %w", services.ErrTurnstileUnavailable, err)
	}
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := v.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", services.ErrTurnstileUnavailable, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %w", services.ErrTurnstileUnavailable, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: siteverify answered %d",
			services.ErrTurnstileUnavailable,
			resp.StatusCode,
		)
	}

	result := new(siteVerifyResponse)
	if err = sonic.Unmarshal(body, result); err != nil {
		return nil, fmt.Errorf("%w: decode response: %w", services.ErrTurnstileUnavailable, err)
	}

	return result, nil
}

func (v *Verifier) checkHostname(hostname string) error {
	if v.expectedHostname == "" || v.testKeys {
		return nil
	}

	if !strings.EqualFold(strings.TrimSpace(hostname), v.expectedHostname) {
		v.l.Warn("turnstile token was solved on another hostname",
			zap.String("hostname", hostname),
			zap.String("expected", v.expectedHostname))
		return fmt.Errorf("%w: hostname mismatch", services.ErrTurnstileRejected)
	}

	return nil
}

func (v *Verifier) checkAction(action, expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if !allowedAction(expected) {
		return fmt.Errorf("%w: unknown expected action %q", services.ErrTurnstileRejected, expected)
	}
	if v.testKeys && action == "" {
		return nil
	}

	if action != expected {
		v.l.Warn("turnstile token was issued for another action",
			zap.String("action", action),
			zap.String("expected", expected))
		return fmt.Errorf("%w: action mismatch", services.ErrTurnstileRejected)
	}

	return nil
}

func allowedAction(action string) bool {
	return action == services.TurnstileActionSignup ||
		action == services.TurnstileActionSignupResend
}

func hostnameOf(rawURL string) string {
	if strings.TrimSpace(rawURL) == "" {
		return ""
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}

	return parsed.Hostname()
}
