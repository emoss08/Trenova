package platformemailservice

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
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
	cfg config.CloudSystemEmailConfig,
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
			ResetURL:          "https://app.trenova.test/auth/reset?token=abc",
			ExpiresIn:         "This link works once and expires in 30 minutes.",
		})
		require.NoError(t, renderErr, kind)
		assert.NotEmpty(t, rendered.Subject, kind)
		assert.NotContains(t, rendered.Subject, "\n", kind)
		assert.Contains(t, rendered.HTML, "<!DOCTYPE html", kind)
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
	svc := newTestService(t, config.CloudSystemEmailConfig{
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
	svc := newTestService(t, config.CloudSystemEmailConfig{}, false, sender, zap.New(core))

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

	svc := newTestService(t, config.CloudSystemEmailConfig{}, true, &recordingSender{}, nil)

	err := svc.SendWelcome(t.Context(), &services.WelcomeEmail{To: "dana@example.com"})
	require.ErrorIs(t, err, services.ErrPlatformEmailNotConfigured)
}

func TestSendRequiresRecipient(t *testing.T) {
	t.Parallel()

	svc := newTestService(t, config.CloudSystemEmailConfig{APIKey: "k"}, true, &recordingSender{}, nil)

	err := svc.SendWelcome(t.Context(), &services.WelcomeEmail{To: "  "})
	require.ErrorIs(t, err, ErrRecipientRequired)
}

func TestSendWrapsProviderFailures(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{err: emailservice.ErrRetryableSend}
	svc := newTestService(t, config.CloudSystemEmailConfig{APIKey: "k"}, true, sender, nil)

	err := svc.SendAccountPurged(t.Context(), &services.AccountPurgedEmail{
		To:          "dana@example.com",
		CompanyName: "Acme",
	})
	require.ErrorIs(t, err, emailservice.ErrRetryableSend)
}

func TestTrialEndedUsesTimezoneAndLinks(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := newTestService(t, config.CloudSystemEmailConfig{APIKey: "k"}, true, sender, nil)

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

func TestEveryKindCarriesTheTrenovaCloudLayout(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := newTestService(t, config.CloudSystemEmailConfig{
		APIKey:  "k",
		LogoURL: "https://cdn.trenova.test/mark.png",
	}, true, sender, nil)

	ctx := t.Context()
	require.NoError(t, svc.SendSignupVerification(ctx, &services.SignupVerificationEmail{
		To: "dana@example.com", Name: "Dana", CompanyName: "Acme", Token: "tok", ExpiresAt: 1_790_000_000,
	}))
	require.NoError(t, svc.SendSignupExistingAccount(ctx, &services.SignupExistingAccountEmail{
		To: "dana@example.com",
	}))
	require.NoError(t, svc.SendWelcome(ctx, &services.WelcomeEmail{
		To: "dana@example.com", Name: "Dana", CompanyName: "Acme", TrialEndsAt: 1_790_000_000,
	}))
	require.NoError(t, svc.SendTrialEnded(ctx, &services.TrialEndedEmail{
		To: "dana@example.com", Name: "Dana", CompanyName: "Acme", ReadOnlyUntil: 1_790_000_000,
	}))
	require.NoError(t, svc.SendAccountPurged(ctx, &services.AccountPurgedEmail{
		To: "dana@example.com", CompanyName: "Acme",
	}))
	require.NoError(t, svc.SendPasswordReset(ctx, &services.PasswordResetEmail{
		To: "dana@example.com", Name: "Dana", CompanyName: "Acme",
		ResetURL: "https://app.trenova.test/auth/reset?token=x", ExpiresInMinutes: 30,
		ExpiresAt: 1_790_000_000,
	}))

	require.Len(t, sender.requests, len(AllKinds()))
	for _, req := range sender.requests {
		html := req.Message.HTML
		assert.Contains(t, html, `src="https://cdn.trenova.test/mark.png"`, req.Message.Subject)
		assert.Contains(t, html, "[ Trenova Cloud ]", req.Message.Subject)
		assert.Contains(t, html, "background-color:#ffa31a", req.Message.Subject)
		assert.Contains(t, html, "border:1px solid #121210", req.Message.Subject)
		assert.Contains(t, html, `href="https://app.trenova.test/"`, req.Message.Subject)
		assert.Contains(t, html, ">app.trenova.test</a>", req.Message.Subject)
		assert.Contains(t, html, "color-scheme: light only", req.Message.Subject)
	}
}

func TestLogoFallsBackToTheDefaultMark(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := newTestService(t, config.CloudSystemEmailConfig{APIKey: "k"}, true, sender, nil)

	require.NoError(t, svc.SendWelcome(t.Context(), &services.WelcomeEmail{To: "dana@example.com"}))
	require.Len(t, sender.requests, 1)
	assert.Contains(t, sender.requests[0].Message.HTML, config.DefaultCloudSystemEmailLogoURL)
}

func TestPasswordResetRendersTheLinkAndExpiry(t *testing.T) {
	t.Parallel()

	sender := &recordingSender{}
	svc := newTestService(t, config.CloudSystemEmailConfig{APIKey: "k"}, true, sender, nil)

	require.NoError(t, svc.SendPasswordReset(t.Context(), &services.PasswordResetEmail{
		To:               "dana@example.com",
		Name:             "Dana Whitfield",
		CompanyName:      "Acme Freight",
		ResetURL:         "https://app.trenova.test/auth/reset?token=tok_1",
		ExpiresInMinutes: 30,
		ExpiresAt:        1_790_000_000,
		Timezone:         "America/Chicago",
		IdempotencyKey:   "password-reset-abc",
	}))

	require.Len(t, sender.requests, 1)
	msg := sender.requests[0].Message
	assert.Equal(t, "Reset your Trenova password", msg.Subject)
	assert.Equal(t, "platform-password-reset-abc", msg.IdempotencyKey)
	assert.Contains(t, msg.HTML, `href="https://app.trenova.test/auth/reset?token=tok_1"`)
	assert.Contains(t, msg.HTML, "expires in 30 minutes")
	assert.Contains(t, msg.HTML, "CDT")
	assert.Contains(t, msg.Text, "Hi Dana,")
	assert.Contains(t, msg.Text, "https://app.trenova.test/auth/reset?token=tok_1")
	assert.Contains(t, msg.Text, "Acme Freight")
}

func TestExpiresInSentence(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "This link works once.", expiresInSentence(0))
	assert.Equal(t, "This link works once and expires in 1 minute.", expiresInSentence(1))
	assert.Equal(t, "This link works once and expires in 45 minutes.", expiresInSentence(45))
}

func TestStepsNumberAndMarkTheLastRow(t *testing.T) {
	t.Parallel()

	steps := newSteps("Where to start", "", "One", "Two", "Three")
	require.Len(t, steps.Items, 3)
	assert.Equal(t, "01", steps.Items[0].Number)
	assert.Equal(t, "03", steps.Items[2].Number)
	assert.False(t, steps.Items[1].Last)
	assert.True(t, steps.Items[2].Last)
}
