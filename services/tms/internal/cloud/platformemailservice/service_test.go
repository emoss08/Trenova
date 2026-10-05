package platformemailservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/cloudconfig"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type recordingSender struct {
	mu       sync.Mutex
	requests []emailservice.SendProviderRequest
	err      error
}

func (r *recordingSender) Send(
	ctx context.Context,
	req emailservice.SendProviderRequest,
) (*emailservice.SendProviderResponse, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := ctx.Deadline(); !ok {
		return nil, errors.New("send must run under a deadline")
	}
	r.requests = append(r.requests, req)
	if r.err != nil {
		return nil, r.err
	}

	return &emailservice.SendProviderResponse{ProviderMessageID: "msg_1"}, nil
}

func newTestService(
	t *testing.T,
	cfg cloudconfig.CloudSystemEmailConfig,
	production bool,
	sender Sender,
	logger *zap.Logger,
) *Service {
	t.Helper()

	svc, err := NewService(&Options{
		Config:     cfg,
		BaseURL:    "https://app.trenova.test/",
		Production: production,
		Sender:     sender,
		Logger:     logger,
	})
	require.NoError(t, err)

	return svc
}

func TestEveryKindRenders(t *testing.T) {
	t.Parallel()

	r, err := newRenderer()
	require.NoError(t, err)

	for _, kind := range AllKinds() {
		rendered, renderErr := r.render(kind, &templateData{
			ProductName:       productName,
			FirstName:         "Dana",
			CompanyName:       "Acme Freight",
			VerifyURL:         "https://app.trenova.test/signup/verify?token=abc",
			ExpiresAt:         "2026-10-05 14:00 UTC",
			LoginURL:          "https://app.trenova.test/login",
			ForgotPasswordURL: "https://app.trenova.test/login",
			AppURL:            "https://app.trenova.test/",
			TrialEndsAt:       "2026-11-03 14:00 UTC",
			ReadOnlyUntil:     "2026-11-17 14:00 UTC",
			SignupURL:         "https://app.trenova.test/signup",
		})
		require.NoError(t, renderErr, kind)
		assert.NotEmpty(t, rendered.Subject, kind)
		assert.NotContains(t, rendered.Subject, "\n", kind)
		assert.Contains(t, rendered.HTML, "<!doctype html>", kind)
		assert.Contains(t, rendered.HTML, rendered.Subject, kind)
		assert.NotEmpty(t, strings.TrimSpace(rendered.Text), kind)
	}
}

func TestHTMLEscapesUserText(t *testing.T) {
	t.Parallel()

	r, err := newRenderer()
	require.NoError(t, err)

	rendered, err := r.render(KindWelcome, &templateData{
		ProductName: productName,
		FirstName:   "<script>alert(1)</script>",
		CompanyName: "Acme & Sons",
		AppURL:      "javascript:alert(1)",
		TrialEndsAt: "soon",
	})
	require.NoError(t, err)
	assert.NotContains(t, rendered.HTML, "<script>alert(1)</script>")
	assert.Contains(t, rendered.HTML, "Acme &amp; Sons")
	assert.NotContains(t, rendered.HTML, `href="javascript:`)
}

func TestSendSignupVerificationThroughResend(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := newTestService(t, cloudconfig.CloudSystemEmailConfig{
		APIKey:      "re_test",
		FromAddress: "noreply@trenova.test",
		FromName:    "Trenova",
		ReplyTo:     "help@trenova.test",
		Timeout:     time.Second,
	}, true, sender, nil)

	err := svc.SendSignupVerification(t.Context(), &services.SignupVerificationEmail{
		To:          "dana@example.com",
		Name:        "Dana Whitfield",
		CompanyName: "Acme Freight",
		Token:       "tok_123",
		ExpiresAt:   1_790_000_000,
	})
	require.NoError(t, err)

	require.Len(t, sender.requests, 1)
	req := sender.requests[0]
	assert.Equal(t, "re_test", req.Config["apiKey"])
	assert.Equal(t, "Trenova <noreply@trenova.test>", req.Message.From)
	assert.Equal(t, "help@trenova.test", req.Message.ReplyTo)
	assert.Equal(t, []string{"dana@example.com"}, req.Message.To)
	assert.Contains(t, req.Message.Text, "https://app.trenova.test/signup/verify?token=tok_123")
	assert.Contains(t, req.Message.HTML, "https://app.trenova.test/signup/verify?token=tok_123")
	assert.Contains(t, req.Message.Text, "Hi Dana,")
	assert.True(t, strings.HasPrefix(req.Message.IdempotencyKey, "platform-signup-verification-"))
}

func TestSendLogsInsteadOfSendingOutsideProduction(t *testing.T) {
	t.Parallel()

	core, logs := observer.New(zap.InfoLevel)
	sender := &recordingSender{}
	svc := newTestService(t, cloudconfig.CloudSystemEmailConfig{}, false, sender, zap.New(core))

	err := svc.SendSignupVerification(t.Context(), &services.SignupVerificationEmail{
		To:    "dana@example.com",
		Name:  "Dana",
		Token: "tok_dev",
	})
	require.NoError(t, err)
	assert.Empty(t, sender.requests)

	entries := logs.FilterMessageSnippet("platform email not sent").All()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].ContextMap()["text"], "signup/verify?token=tok_dev")
}

func TestSendRefusesInProductionWithoutAPIKey(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, cloudconfig.CloudSystemEmailConfig{}, true, &recordingSender{}, nil)

	err := svc.SendWelcome(t.Context(), &services.WelcomeEmail{To: "dana@example.com"})
	require.ErrorIs(t, err, services.ErrPlatformEmailNotConfigured)
}

func TestSendRequiresRecipient(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, cloudconfig.CloudSystemEmailConfig{APIKey: "k"}, true, &recordingSender{}, nil)

	err := svc.SendRendered(t.Context(), &services.PlatformEmailMessage{Subject: "x"})
	require.ErrorIs(t, err, ErrRecipientRequired)
}

func TestSendWrapsProviderFailures(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{err: emailservice.ErrRetryableSend}
	svc := newTestService(t, cloudconfig.CloudSystemEmailConfig{APIKey: "k"}, true, sender, nil)

	err := svc.SendAccountPurged(t.Context(), &services.AccountPurgedEmail{
		To:          "dana@example.com",
		CompanyName: "Acme",
	})
	require.ErrorIs(t, err, emailservice.ErrRetryableSend)
}

func TestTrialEndedUsesTimezoneAndLinks(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := newTestService(t, cloudconfig.CloudSystemEmailConfig{APIKey: "k"}, true, sender, nil)

	require.NoError(t, svc.SendTrialEnded(t.Context(), &services.TrialEndedEmail{
		To:            "dana@example.com",
		Name:          "Dana",
		CompanyName:   "Acme",
		ReadOnlyUntil: 1_790_000_000,
		Timezone:      "America/Chicago",
	}))
	require.NoError(t, svc.SendSignupExistingAccount(t.Context(), &services.SignupExistingAccountEmail{
		To: "dana@example.com",
	}))

	require.Len(t, sender.requests, 2)
	assert.Contains(t, sender.requests[0].Message.Text, "https://app.trenova.test/")
	assert.Contains(t, sender.requests[0].Message.Text, "CDT")
	assert.Contains(t, sender.requests[1].Message.Text, "https://app.trenova.test/login")
	assert.Contains(t, sender.requests[1].Message.Text, "Hi,")
}
