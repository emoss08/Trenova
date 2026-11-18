package systememailservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/systememail"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	cacheTTL         = 30 * time.Second
	cacheKey         = systememail.CurrentID
	fieldAPIKey      = "apiKey"
	fingerprintShown = 12
)

type Params struct {
	fx.In

	Config *managedconfig.Config
	Repo   emailports.SystemSenderRepository
	Cipher kernelports.SecretCipher
	Clock  timeutils.Clock
	Logger *zap.Logger
}

type Service struct {
	enabled bool
	repo    emailports.SystemSenderRepository
	cipher  kernelports.SecretCipher
	clock   timeutils.Clock
	l       *zap.Logger
	cache   *otter.Cache[string, held]
}

type held struct {
	credential  *systememail.Credential
	fingerprint string
}

var _ emailports.SystemSenders = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		enabled: p.Config.Enabled,
		repo:    p.Repo,
		cipher:  p.Cipher,
		clock:   p.Clock,
		l:       p.Logger.Named("system-email"),
		cache: otter.Must(&otter.Options[string, held]{
			MaximumSize:      1,
			ExpiryCalculator: otter.ExpiryWriting[string, held](cacheTTL),
		}),
	}
}

func (s *Service) Current(ctx context.Context) (*systememail.Credential, error) {
	h, err := s.current(ctx)
	if err != nil {
		return nil, err
	}
	return h.credential, nil
}

func (s *Service) Status(ctx context.Context) (systememail.Status, error) {
	h, err := s.current(ctx)
	if err != nil || h.credential == nil {
		return systememail.Status{}, err
	}
	return systememail.Status{
		Configured:  true,
		Provider:    h.credential.Provider,
		FromAddress: h.credential.FromAddress,
		Fingerprint: h.fingerprint,
	}, nil
}

func (s *Service) Set(ctx context.Context, credential *systememail.Credential) error {
	if err := credential.Validate(); err != nil {
		return err
	}
	fingerprint := systememail.Fingerprint(credential)
	before, err := s.load(ctx)
	if err != nil {
		return err
	}
	if before.credential != nil && before.fingerprint == fingerprint {
		return nil
	}
	sender, err := s.seal(ctx, credential, fingerprint)
	if err != nil {
		return err
	}
	if err = s.repo.Save(ctx, sender); err != nil {
		return err
	}
	s.cache.Invalidate(cacheKey)
	s.l.Info(
		"the control plane set the system email sender",
		zap.String("provider", string(credential.Provider)),
		zap.String("fromAddress", credential.FromAddress),
		zap.String("fingerprint", fingerprint[:fingerprintShown]),
		zap.Bool("replaced", before.credential != nil),
	)
	return nil
}

func (s *Service) Remove(ctx context.Context) error {
	removed, err := s.repo.Delete(ctx)
	if err != nil {
		return err
	}
	s.cache.Invalidate(cacheKey)
	if removed {
		s.l.Info("the control plane removed the system email sender")
	}
	return nil
}

func (s *Service) current(ctx context.Context) (held, error) {
	if !s.enabled {
		return held{}, nil
	}
	return s.cache.Get(ctx, cacheKey, otter.LoaderFunc[string, held](
		func(ctx context.Context, _ string) (held, error) {
			return s.load(ctx)
		},
	))
}

func (s *Service) load(ctx context.Context) (held, error) {
	sender, err := s.repo.Get(ctx)
	if errors.Is(err, systememail.ErrNoSender) {
		return held{}, nil
	}
	if err != nil {
		return held{}, err
	}
	credential, err := s.open(ctx, sender)
	if err != nil {
		s.l.Error(
			"cannot open the stored system email sender; it counts as unset until the operator sends it again",
			zap.Error(err),
		)
		return held{}, nil
	}
	fingerprint := systememail.Fingerprint(credential)
	if fingerprint != sender.Fingerprint {
		s.l.Error(
			"the stored system email sender does not match its fingerprint; it counts as unset until the operator sends it again",
		)
		return held{}, nil
	}
	return held{credential: credential, fingerprint: fingerprint}, nil
}

func (s *Service) seal(
	ctx context.Context,
	credential *systememail.Credential,
	fingerprint string,
) (*systememail.Sender, error) {
	apiKey, err := s.cipher.Seal(ctx, credential.APIKey, scope(fieldAPIKey))
	if err != nil {
		return nil, fmt.Errorf("seal the system email API key: %w", err)
	}
	return &systememail.Sender{
		ID:           systememail.CurrentID,
		Provider:     credential.Provider,
		FromAddress:  credential.FromAddress,
		FromName:     credential.FromName,
		APIKeySealed: apiKey,
		Fingerprint:  fingerprint,
		ReceivedAt:   s.clock(),
	}, nil
}

func (s *Service) open(
	ctx context.Context,
	sender *systememail.Sender,
) (*systememail.Credential, error) {
	apiKey, err := s.cipher.Open(ctx, sender.APIKeySealed, scope(fieldAPIKey))
	if err != nil {
		return nil, fmt.Errorf("open the system email API key: %w", err)
	}
	return &systememail.Credential{
		Provider:    sender.Provider,
		APIKey:      apiKey,
		FromAddress: sender.FromAddress,
		FromName:    sender.FromName,
	}, nil
}

func scope(field string) *kernelports.SecretScope {
	return &kernelports.SecretScope{
		Purpose:    kernelports.SecretPurposeSystemEmail,
		ResourceID: systememail.CurrentID + ":" + field,
	}
}
