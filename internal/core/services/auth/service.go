package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/cache/redis"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

const (
	loginRateLimitWindow = 15 * time.Minute
	maxLoginAttempts     = 5
)

type ServiceParams struct {
	fx.In

	Cache       *redis.Client
	Logger      *logger.Logger
	UserRepo    repositories.UserRepository
	SessionRepo repositories.SessionRepository
}

type Service struct {
	userRepo    repositories.UserRepository
	sessionRepo repositories.SessionRepository
	cache       *redis.Client
	l           *zerolog.Logger
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "auth").
		Logger()

	return &Service{
		userRepo:    p.UserRepo,
		sessionRepo: p.SessionRepo,
		cache:       p.Cache,
		l:           &log,
	}
}

// Login logs a user in and returns a session ID.
func (s *Service) Login(ctx context.Context, ip, userAgent string, req *services.LoginRequest) (*services.LoginResponse, error) {
	// validate the request
	if err := req.Validate(); err != nil {
		return nil, eris.Wrap(err, "invalid login request")
	}

	usr, err := s.userRepo.FindByEmail(ctx, req.EmailAddress)
	if err != nil {
		return nil, eris.Wrap(err, "failed to find user by email")
	}

	// check the rate limit for the user
	if err = s.checkLoginRateLimit(ctx, ip, usr.ID); err != nil {
		s.l.Warn().Err(err).Msg("login rate limit exceeded")
		return nil, err
	}

	// Verify the user credentials
	if err = usr.VerifyCredentials(req.Password); err != nil {
		return nil, err
	}

	sess, err := s.createSession(ctx, createSessionParams{
		User:      usr,
		IP:        ip,
		UserAgent: userAgent,
	})
	if err != nil {
		return nil, eris.Wrap(err, "failed to create session")
	}

	usr.UpdateLastLogin()

	// Reset the login attempts for the user
	if err = s.resetLoginAttempts(ctx, ip, usr.ID); err != nil {
		s.l.Error().
			Str("ip", ip).
			Str("userID", usr.ID.String()).
			Msg("failed to reset login attempts")
	}

	s.l.Debug().
		Str("session_id", sess.ID.String()).
		Msg("successful login")

	return &services.LoginResponse{
		User:      usr,
		SessionID: sess.ID.String(),
		ExpiresAt: sess.ExpiresAt,
	}, nil
}

func (s *Service) ValidateSession(ctx context.Context, sessionID pulid.ID, clientIP string) (bool, error) {
	_, err := s.sessionRepo.GetValidSession(ctx, sessionID, clientIP)
	if err != nil {
		s.l.Error().
			Str("sessionID", sessionID.String()).
			Str("clientIP", clientIP).
			Err(err).
			Msg("failed to validate session")
		return false, eris.Wrap(err, "failed to validate session")
	}

	return true, nil
}

// checkLoginRateLimit checks if a user has exceeded the login rate limit.
func (s *Service) checkLoginRateLimit(ctx context.Context, ip string, userID pulid.ID) error {
	// First increment, then check if the count is greater than the limit
	count, err := s.incrementLoginAttempts(ctx, ip, userID)
	if err != nil {
		// On redis errors, allow the request ,but log the error
		s.l.Error().
			Str("ip", ip).
			Str("userID", userID.String()).
			Msg("failed to increment login attempts")
	}

	if count > maxLoginAttempts {
		// * Ensure we include the email address in the error message
		// * because this will be shown to the user on the frontend
		return errors.NewRateLimitError("emailAddress", "Too many login attempts, please try again later")
	}

	return nil
}

// incrementLoginAttempts increments the login attempts for a user.
func (s *Service) incrementLoginAttempts(ctx context.Context, ip string, userID pulid.ID) (int64, error) {
	key := fmt.Sprintf("login_attempts:%s:%s", ip, userID.String())
	count, err := s.cache.IncreaseWithExpiry(ctx, key, loginRateLimitWindow)
	if err != nil {
		s.l.Error().
			Str("ip", ip).
			Str("userID", userID.String()).
			Msg("failed to increment login attempts")
		return 0, eris.Wrap(err, "failed to increment login attempts")
	}

	return count, nil
}

// resetLoginAttempts resets the login attempts for a user.
func (s *Service) resetLoginAttempts(ctx context.Context, ip string, userID pulid.ID) error {
	key := fmt.Sprintf("login_attempts:%s:%s", ip, userID.String())
	err := s.cache.Del(ctx, key)
	if err != nil {
		s.l.Error().
			Str("ip", ip).
			Str("userID", userID.String()).
			Msg("failed to reset login attempts")
		return eris.Wrap(err, "failed to reset login attempts")
	}

	return nil
}

// CheckEmail checks if an email address is valid and returns a message.
func (s *Service) CheckEmail(ctx context.Context, req *services.CheckEmailRequest) (bool, error) {
	usr, err := s.userRepo.FindByEmail(ctx, req.EmailAddress)
	if err != nil {
		return false, eris.Wrap(err, "failed to find user by email")
	}

	// Verify the user status
	if err = usr.ValidateStatus(); err != nil {
		return false, err
	}

	return true, nil
}

// RefreshSession updates the session activity and extends the session expiration time.
func (s *Service) RefreshSession(ctx context.Context, sessionID pulid.ID, ip, userAgent string) (*session.Session, error) {
	// First get and validate the session
	sess, err := s.sessionRepo.GetValidSession(ctx, sessionID, ip)
	if err != nil {
		return nil, err
	}

	// Update session activity
	if err = s.sessionRepo.UpdateSessionActivity(
		ctx,
		sessionID,
		ip,
		userAgent,
		session.EventTypeAccessed,
		nil,
	); err != nil {
		s.l.Warn().Err(err).Msg("failed to update session activity")
	}

	return sess, nil
}

// Logout revokes a session and logs a user out.
func (s *Service) Logout(ctx context.Context, sessionID pulid.ID, ip, userAgent string) error {
	// First verify the session is valid for this IP
	_, err := s.sessionRepo.GetValidSession(ctx, sessionID, ip)
	if err != nil {
		s.l.Error().
			Str("sessionId", sessionID.String()).
			Str("ip", ip).
			Err(err).
			Msg("invalid session during logout")
		return eris.Wrap(err, "invalid session")
	}

	err = s.sessionRepo.RevokeSession(ctx, sessionID, ip, userAgent, "User logged out")
	if err != nil {
		s.l.Error().
			Str("sessionId", sessionID.String()).
			Err(err).
			Msg("failed to revoke session")
		return eris.Wrap(err, "failed to revoke session")
	}

	return nil
}

type createSessionParams struct {
	User      *user.User
	IP        string
	UserAgent string
}

// createSession creates a session for a user.
func (s *Service) createSession(ctx context.Context, p createSessionParams) (*session.Session, error) {
	// Use the constructor instead of manual creation
	expiresAt := timeutils.NowUnix() + 30*24*60*60 // 30 days
	sess := session.NewSession(p.User.ID, p.User.BusinessUnitID, p.User.CurrentOrganizationID, p.IP, p.UserAgent, expiresAt)

	// The session is already validated in the constructor, but we can double-check
	if err := sess.Validate(p.IP); err != nil {
		return nil, eris.Wrap(err, "failed to validate session")
	}

	// Initial login event should be added here
	sess.AddEvent(session.EventTypeLogin, p.IP, p.UserAgent, nil)

	if err := s.sessionRepo.Create(ctx, sess); err != nil {
		return nil, eris.Wrap(err, "failed to create session")
	}

	return sess, nil
}
