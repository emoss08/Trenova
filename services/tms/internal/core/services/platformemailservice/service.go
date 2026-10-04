package platformemailservice

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/emailservice"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	productName       = "Trenova"
	defaultTimezone   = "UTC"
	signupVerifyPath  = "/signup/verify"
	signupPath        = "/signup"
	loginPath         = "/login"
	idempotencyPrefix = "platform-"
)

var ErrRecipientRequired = errors.New("platform email: recipient is required")

type Sender interface {
	Send(ctx context.Context, req emailservice.SendProviderRequest) (*emailservice.SendProviderResponse, error)
}

type Params struct {
	fx.In

	Config *config.Config
	Logger *zap.Logger
}

type Service struct {
	cfg        config.CloudSystemEmailConfig
	baseURL    string
	production bool
	sender     Sender
	renderer   *renderer
	l          *zap.Logger
}

func New(p Params) (services.PlatformEmailService, error) {
	systemEmail := p.Config.Platform.Cloud.SystemEmail
	sender := emailservice.NewResendSenderWithClient(
		&http.Client{Timeout: systemEmail.GetTimeout()},
	)

	return NewService(&Options{
		Config:     systemEmail,
		BaseURL:    p.Config.App.GetWebBaseURL(),
		Production: p.Config.App.IsProduction(),
		Sender:     sender,
		Logger:     p.Logger,
	})
}

type Options struct {
	Config     config.CloudSystemEmailConfig
	BaseURL    string
	Production bool
	Sender     Sender
	Logger     *zap.Logger
}

func NewService(opts *Options) (*Service, error) {
	r, err := newRenderer()
	if err != nil {
		return nil, err
	}

	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &Service{
		cfg:        opts.Config,
		baseURL:    strings.TrimRight(strings.TrimSpace(opts.BaseURL), "/"),
		production: opts.Production,
		sender:     opts.Sender,
		renderer:   r,
		l:          logger.Named("service.platform-email"),
	}, nil
}

func (s *Service) SendSignupVerification(
	ctx context.Context,
	msg *services.SignupVerificationEmail,
) error {
	verifyURL := s.link(signupVerifyPath, url.Values{"token": {msg.Token}})

	return s.sendKind(ctx, KindSignupVerification, msg.To, &templateData{
		FirstName:   firstNameOr(msg.Name, "there"),
		CompanyName: msg.CompanyName,
		VerifyURL:   verifyURL,
		ExpiresAt:   timeutils.FormatStampIn(msg.ExpiresAt, defaultTimezone),
	}, "signup-verification-"+tokenutils.Hash(msg.Token))
}

func (s *Service) SendSignupExistingAccount(
	ctx context.Context,
	msg *services.SignupExistingAccountEmail,
) error {
	return s.sendKind(ctx, KindSignupExistingAccount, msg.To, &templateData{
		FirstName:         stringutils.FirstName(msg.Name),
		LoginURL:          s.link(loginPath, nil),
		ForgotPasswordURL: s.link(loginPath, nil),
	}, "")
}

func (s *Service) SendWelcome(ctx context.Context, msg *services.WelcomeEmail) error {
	return s.sendKind(ctx, KindWelcome, msg.To, &templateData{
		FirstName:   firstNameOr(msg.Name, "there"),
		CompanyName: msg.CompanyName,
		AppURL:      s.link("/", nil),
		TrialEndsAt: timeutils.FormatStampIn(msg.TrialEndsAt, timezoneOr(msg.Timezone)),
	}, "")
}

func (s *Service) SendTrialEnded(ctx context.Context, msg *services.TrialEndedEmail) error {
	return s.sendKind(ctx, KindTrialEnded, msg.To, &templateData{
		FirstName:     firstNameOr(msg.Name, "there"),
		CompanyName:   msg.CompanyName,
		AppURL:        s.link("/", nil),
		ReadOnlyUntil: timeutils.FormatStampIn(msg.ReadOnlyUntil, timezoneOr(msg.Timezone)),
	}, "")
}

func (s *Service) SendAccountPurged(ctx context.Context, msg *services.AccountPurgedEmail) error {
	return s.sendKind(ctx, KindAccountPurged, msg.To, &templateData{
		FirstName:   stringutils.FirstName(msg.Name),
		CompanyName: msg.CompanyName,
		SignupURL:   s.link(signupPath, nil),
	}, "")
}

func (s *Service) SendRendered(ctx context.Context, msg *services.PlatformEmailMessage) error {
	return s.deliver(ctx, msg)
}

func (s *Service) sendKind(
	ctx context.Context,
	kind Kind,
	to string,
	data *templateData,
	idempotencyKey string,
) error {
	data.ProductName = productName
	if data.CompanyName == "" {
		data.CompanyName = "your company"
	}

	rendered, err := s.renderer.render(kind, data)
	if err != nil {
		return err
	}

	return s.deliver(ctx, &services.PlatformEmailMessage{
		Kind:           string(kind),
		To:             to,
		Subject:        rendered.Subject,
		HTML:           rendered.HTML,
		Text:           rendered.Text,
		IdempotencyKey: idempotencyKey,
	})
}

func (s *Service) deliver(ctx context.Context, msg *services.PlatformEmailMessage) error {
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return ErrRecipientRequired
	}

	if !s.cfg.HasAPIKey() {
		if s.production {
			return services.ErrPlatformEmailNotConfigured
		}

		s.l.Info("platform email not sent: platform.cloud.systemEmail.apiKey is empty",
			zap.String("kind", msg.Kind),
			zap.String("to", to),
			zap.String("subject", msg.Subject),
			zap.String("text", msg.Text),
		)
		return nil
	}

	if s.sender == nil {
		return services.ErrPlatformEmailNotConfigured
	}

	sendCtx, cancel := context.WithTimeout(ctx, s.cfg.GetTimeout())
	defer cancel()

	key := strings.TrimSpace(msg.IdempotencyKey)
	if key != "" {
		key = idempotencyPrefix + key
	}

	resp, err := s.sender.Send(sendCtx, emailservice.SendProviderRequest{
		Message: emailservice.SendProviderMessage{
			IdempotencyKey: key,
			From:           stringutils.FormatEmailAddress(s.cfg.GetFromName(), s.cfg.GetFromAddress()),
			ReplyTo:        s.cfg.GetReplyTo(),
			To:             []string{to},
			Subject:        msg.Subject,
			HTML:           msg.HTML,
			Text:           msg.Text,
		},
		Config: map[string]string{"apiKey": strings.TrimSpace(s.cfg.APIKey)},
	})
	if err != nil {
		s.l.Error("failed to send platform email",
			zap.String("kind", msg.Kind),
			zap.Error(err),
		)
		return fmt.Errorf("send platform email %s: %w", msg.Kind, err)
	}

	s.l.Info("platform email sent",
		zap.String("kind", msg.Kind),
		zap.String("providerMessageId", resp.ProviderMessageID),
	)

	return nil
}

func (s *Service) link(path string, query url.Values) string {
	target := s.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}

	return target
}

func firstNameOr(name, fallback string) string {
	if first := stringutils.FirstName(name); first != "" {
		return first
	}

	return fallback
}

func timezoneOr(timezone string) string {
	if strings.TrimSpace(timezone) == "" {
		return defaultTimezone
	}

	return timezone
}
