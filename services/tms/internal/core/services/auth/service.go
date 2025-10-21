package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	UserRepository     repositories.UserRepository
	SessionRepository  repositories.SessionRepository
	APITokenRepository repositories.APITokenRepository
	Cache              *redis.Connection
	Logger             *zap.Logger
}

type Service struct {
	userRepository     repositories.UserRepository
	sessionRepository  repositories.SessionRepository
	apiTokenRepository repositories.APITokenRepository
	cache              *redis.Connection
	l                  *zap.Logger
}

func NewService(p ServiceParams) services.AuthService {
	return &Service{
		userRepository:     p.UserRepository,
		sessionRepository:  p.SessionRepository,
		apiTokenRepository: p.APITokenRepository,
		cache:              p.Cache,
		l:                  p.Logger.Named("service.auth"),
	}
}

func (s *Service) Login(
	ctx context.Context,
	req services.LoginRequest,
) (*services.LoginResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	usr, err := s.userRepository.FindByEmail(ctx, req.EmailAddress)
	if err != nil {
		return nil, err
	}

	if err = usr.VerifyCredentials(req.Password); err != nil {
		return nil, err
	}

	sess, err := s.createSession(ctx, &createSessionRequest{
		User:      usr,
		ClientIP:  req.ClientIP,
		UserAgent: req.UserAgent,
	})
	if err != nil {
		return nil, err
	}

	if err = s.userRepository.UpdateLastLogin(ctx, usr.ID); err != nil {
		// ! we're not going to return an error here because we want to return the session to the user
		s.l.Error("failed to update last login", zap.Error(err))
	}

	if err = s.resetLoginAttempts(ctx, req.ClientIP, usr.ID); err != nil {
		// ! we're not going to return an error here because we want to return the session to the user
		s.l.Error("failed to reset login attempts", zap.Error(err))
	}

	return &services.LoginResponse{
		User:      usr,
		ExpiresAt: sess.ExpiresAt,
		SessionID: sess.ID.String(),
	}, nil
}

func (s *Service) ValidateSession(
	ctx context.Context,
	req services.ValidateSessionRequest,
) (bool, error) {
	_, err := s.sessionRepository.GetValidSession(ctx, repositories.GetValidSessionRequest{
		SessionID: req.SessionID,
		ClientIP:  req.ClientIP,
	})
	if err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) RefreshSession(
	ctx context.Context,
	req services.RefreshSessionRequest,
) (*session.Session, error) {
	sess, err := s.sessionRepository.GetValidSession(ctx, repositories.GetValidSessionRequest{
		SessionID: req.SessionID,
		ClientIP:  req.ClientIP,
	})
	if err != nil {
		return nil, err
	}

	if err = s.sessionRepository.UpdateSessionActivity(
		ctx,
		&repositories.UpdateSessionActivityRequest{
			SessionID: req.SessionID,
			ClientIP:  req.ClientIP,
			UserAgent: req.UserAgent,
			EventType: session.EventTypeAccessed,
			Metadata:  req.Metadata,
		},
	); err != nil {
		s.l.Warn("failed to update session activity", zap.Error(err))
	}

	return sess, nil
}

func (s *Service) CheckEmail(ctx context.Context, req services.CheckEmailRequest) (bool, error) {
	usr, err := s.userRepository.FindByEmail(ctx, req.EmailAddress)
	if err != nil {
		return false, err
	}

	if err = usr.ValidateStatus(); err != nil {
		return false, err
	}

	return true, nil
}

func (s *Service) Logout(ctx context.Context, req services.LogoutRequest) error {
	_, err := s.sessionRepository.GetValidSession(ctx, repositories.GetValidSessionRequest{
		SessionID: req.SessionID,
		ClientIP:  req.ClientIP,
	})
	if err != nil {
		s.l.Error(
			"invalid session during logout",
			zap.Error(err),
			zap.String("sessionId", req.SessionID.String()),
			zap.String("ip", req.ClientIP),
		)
		return err
	}

	err = s.sessionRepository.RevokeSession(ctx, repositories.RevokeSessionRequest{
		SessionID: req.SessionID,
		ClientIP:  req.ClientIP,
		UserAgent: req.UserAgent,
		Reason:    req.Reason,
	})
	if err != nil {
		s.l.Error(
			"failed to revoke session",
			zap.Error(err),
			zap.String("sessionId", req.SessionID.String()),
			zap.String("ip", req.ClientIP),
		)
		return err
	}

	return nil
}

func (s *Service) UpdateSessionOrganization(
	ctx context.Context,
	sessionID pulid.ID,
	newOrgID pulid.ID,
) error {
	log := s.l.With(
		zap.String("operation", "UpdateSessionOrganization"),
		zap.String("sessionID", sessionID.String()),
		zap.String("newOrgID", newOrgID.String()),
	)

	err := s.sessionRepository.UpdateSessionOrganization(
		ctx,
		repositories.UpdateSessionOrganizationRequest{
			SessionID: sessionID,
			NewOrgID:  newOrgID,
		},
	)
	if err != nil {
		log.Error("failed to update session organization", zap.Error(err))
		return err
	}

	log.Info("session organization updated successfully")
	return nil
}

type createSessionRequest struct {
	User      *tenant.User
	ClientIP  string
	UserAgent string
}

func (s *Service) createSession(
	ctx context.Context,
	req *createSessionRequest,
) (*session.Session, error) {
	expiresAt := utils.NowUnix() + 30*24*60*60 // 30 days

	sess := session.NewSession(
		session.NewSessionRequest{
			UserID:                req.User.ID,
			BusinessUnitID:        req.User.BusinessUnitID,
			CurrentOrganizationID: req.User.CurrentOrganizationID,
			IP:                    req.ClientIP,
			UserAgent:             req.UserAgent,
			ExpiresAt:             expiresAt,
		},
	)

	if err := sess.Validate(req.ClientIP); err != nil {
		return nil, err
	}

	if err := s.sessionRepository.Create(ctx, sess); err != nil {
		return nil, err
	}

	return sess, nil
}

func (s *Service) resetLoginAttempts(ctx context.Context, clientIP string, userID pulid.ID) error {
	key := fmt.Sprintf("login_attempts:%s:%s", clientIP, userID.String())

	if err := s.cache.Delete(ctx, key); err != nil {
		return err
	}

	return nil
}

func (s *Service) CreateAPIToken(
	ctx context.Context,
	req *services.CreateAPITokenRequest,
) (*services.CreateAPITokenResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	token, err := tenant.NewAPIToken(tenant.NewAPITokenRequest{
		UserID:         req.UserID,
		BusinessUnitID: req.BusinessUnitID,
		OrganizationID: req.OrganizationID,
		Name:           req.Name,
		Description:    req.Description,
		Scopes:         req.Scopes,
		ExpiresAt:      req.ExpiresAt,
	})
	if err != nil {
		return nil, err
	}

	if err = s.apiTokenRepository.Create(ctx, repositories.CreateAPITokenRequest{
		Token: token,
	}); err != nil {
		return nil, err
	}

	if err = s.cacheAPIToken(ctx, token); err != nil {
		s.l.Warn("failed to cache API token", zap.Error(err))
	}

	return &services.CreateAPITokenResponse{
		Token:      token,
		PlainToken: token.PlainToken, // This is the only time we return the plain token
	}, nil
}

func (s *Service) ValidateAPIToken(
	ctx context.Context,
	req services.ValidateAPITokenRequest,
) (*tenant.APIToken, error) {
	tokenPrefix := req.Token
	if len(tokenPrefix) > tenant.TokenPrefixLength {
		tokenPrefix = tokenPrefix[:tenant.TokenPrefixLength]
	}

	token, err := s.getCachedAPIToken(ctx, tokenPrefix)
	if err == nil && token != nil {
		if err = token.VerifyToken(req.Token); err != nil {
			return nil, err
		}

		go func() {
			updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			if err = s.apiTokenRepository.UpdateLastUsed(updateCtx, repositories.UpdateAPITokenLastUsedRequest{
				TokenID: token.ID,
				IP:      req.ClientIP,
			}); err != nil {
				s.l.Warn("failed to update API token last used", zap.Error(err))
			}
		}()

		return token, nil
	}

	token, err = s.apiTokenRepository.FindByToken(ctx, repositories.FindAPITokenByTokenRequest{
		TokenPrefix: tokenPrefix,
		PlainToken:  req.Token,
	})
	if err != nil {
		return nil, err
	}

	if err = s.cacheAPIToken(ctx, token); err != nil {
		s.l.Warn("failed to cache API token", zap.Error(err))
	}

	go func() {
		updateCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err = s.apiTokenRepository.UpdateLastUsed(updateCtx, repositories.UpdateAPITokenLastUsedRequest{
			TokenID: token.ID,
			IP:      req.ClientIP,
		}); err != nil {
			s.l.Warn("failed to update API token last used", zap.Error(err))
		}
	}()

	return token, nil
}

func (s *Service) RevokeAPIToken(
	ctx context.Context,
	req services.RevokeAPITokenRequest,
) error {
	if err := s.apiTokenRepository.Revoke(ctx, req.TokenID); err != nil {
		return err
	}

	token, err := s.apiTokenRepository.FindByID(ctx, req.TokenID)
	if err == nil && token != nil {
		if err = s.clearCachedAPIToken(ctx, token.TokenPrefix); err != nil {
			s.l.Warn("failed to clear API token from cache", zap.Error(err))
		}
	}

	return nil
}

func (s *Service) ListUserAPITokens(
	ctx context.Context,
	userID pulid.ID,
) ([]*tenant.APIToken, error) {
	tokens, err := s.apiTokenRepository.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	for _, token := range tokens {
		token.SanitizeForResponse()
	}

	return tokens, nil
}

func (s *Service) cacheAPIToken(ctx context.Context, token *tenant.APIToken) error {
	key := fmt.Sprintf("api_token:%s", token.TokenPrefix)

	return s.cache.SetJSON(ctx, key, token, 5*time.Minute)
}

func (s *Service) getCachedAPIToken(
	ctx context.Context,
	tokenPrefix string,
) (*tenant.APIToken, error) {
	key := fmt.Sprintf("api_token:%s", tokenPrefix)

	var token tenant.APIToken
	if err := s.cache.GetJSON(ctx, key, &token); err != nil {
		return nil, err
	}

	return &token, nil
}

func (s *Service) clearCachedAPIToken(ctx context.Context, tokenPrefix string) error {
	key := fmt.Sprintf("api_token:%s", tokenPrefix)
	return s.cache.Delete(ctx, key)
}
