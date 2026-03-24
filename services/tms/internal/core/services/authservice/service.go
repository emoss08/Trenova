package authservice

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	UserRepository    repositories.UserRepository
	SessionRepository repositories.SessionRepository
	APIKeyRepository  repositories.APIKeyRepository
	UsageRecorder     services.UsageRecorder
	Config            *config.Config
	Logger            *zap.Logger
}

type Service struct {
	ur       repositories.UserRepository
	sr       repositories.SessionRepository
	akr      repositories.APIKeyRepository
	usageBuf services.UsageRecorder
	cfg      *config.Config
	l        *zap.Logger
}

func New(p Params) services.AuthService {
	return &Service{
		ur:       p.UserRepository,
		sr:       p.SessionRepository,
		akr:      p.APIKeyRepository,
		usageBuf: p.UsageRecorder,
		cfg:      p.Config,
		l:        p.Logger.Named("service.auth"),
	}
}

var errInvalidCredentials = errortypes.NewAuthenticationError("Invalid email or password")

func (s *Service) Login(
	ctx context.Context,
	req services.LoginRequest,
) (*services.LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	usr, err := s.ur.FindByEmail(ctx, req.EmailAddress)
	if err != nil {
		return nil, errInvalidCredentials
	}

	if err = usr.VerifyCredentials(req.Password); err != nil {
		if errortypes.IsAuthorizationError(err) {
			return nil, err
		}
		return nil, errInvalidCredentials
	}

	sess, err := s.createSession(ctx, usr)
	if err != nil {
		return nil, err
	}

	if err = s.ur.UpdateLastLoginAt(ctx, usr.ID); err != nil {
		s.l.Error("failed to update last login at", zap.Error(err))
	}

	return &services.LoginResponse{
		User:      usr,
		ExpiresAt: sess.ExpiresAt,
		SessionID: sess.ID.String(),
	}, nil
}

func (s *Service) ValidateSession(
	ctx context.Context,
	sessionID pulid.ID,
) (*session.Session, error) {
	sess, err := s.sr.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	if err = sess.Validate(); err != nil {
		if delErr := s.sr.Delete(ctx, sessionID); delErr != nil {
			s.l.Error("failed to delete expired session", zap.Error(delErr))
		}
		return nil, errortypes.NewAuthenticationError("Session has expired. Please login again.")
	}

	return sess, nil
}

func (s *Service) Logout(ctx context.Context, sessionID pulid.ID) error {
	return s.sr.Delete(ctx, sessionID)
}

func (s *Service) AuthenticateAPIKey(
	ctx context.Context,
	token string,
	ipAddress, userAgent string,
) (*services.AuthenticatedPrincipal, error) {
	if !s.cfg.Security.APIToken.Enabled {
		return nil, errortypes.NewAuthenticationError("API token authentication is disabled")
	}

	prefix, secret, err := apikey.SplitToken(token)
	if err != nil {
		return nil, errortypes.NewAuthenticationError("Invalid bearer token")
	}

	key, err := s.akr.GetByPrefix(ctx, prefix)
	if err != nil {
		return nil, errortypes.NewAuthenticationError("Invalid bearer token")
	}

	if key.Status != apikey.StatusActive {
		return nil, errortypes.NewAuthenticationError("API key is inactive")
	}

	if key.IsExpired(timeutils.NowUnix()) {
		return nil, errortypes.NewAuthenticationError("API key has expired")
	}

	computed := apikey.HashSecret(key.SecretSalt, secret)
	if subtle.ConstantTimeCompare([]byte(computed), []byte(key.SecretHash)) != 1 {
		return nil, errortypes.NewAuthenticationError("Invalid bearer token")
	}

	s.usageBuf.RecordUsage(services.APIKeyUsageEvent{
		APIKeyID:       key.ID,
		OrganizationID: key.OrganizationID,
		BusinessUnitID: key.BusinessUnitID,
		OccurredAt:     time.Now().UTC(),
		IPAddress:      ipAddress,
		UserAgent:      userAgent,
	})

	return &services.AuthenticatedPrincipal{
		Type:           services.PrincipalTypeAPIKey,
		PrincipalID:    key.ID,
		APIKeyID:       key.ID,
		BusinessUnitID: key.BusinessUnitID,
		OrganizationID: key.OrganizationID,
		APIKey:         key,
	}, nil
}

func (s *Service) createSession(ctx context.Context, user *tenant.User) (*session.Session, error) {
	expiresAt := timeutils.NowUnix() + int64(session.DefaultTTL.Seconds())

	sess := session.NewSession(&session.NewSessionRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID:  user.CurrentOrganizationID,
			BuID:   user.BusinessUnitID,
			UserID: user.ID,
		},
		ExpiresAt: expiresAt,
	})

	if err := s.sr.Create(ctx, sess); err != nil {
		return nil, err
	}

	return sess, nil
}
