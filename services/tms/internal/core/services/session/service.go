/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package session

import (
	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"golang.org/x/net/context"
)

type ServiceParams struct {
	fx.In

	Logger *logger.Logger
	Repo   repositories.SessionRepository
}

type Service struct {
	repo repositories.SessionRepository
	l    *zerolog.Logger
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().Str("service", "session").Logger()

	return &Service{
		repo: p.Repo,
		l:    &log,
	}
}

func (s *Service) GetSessions(ctx context.Context, userID pulid.ID) ([]*session.Session, error) {
	log := s.l.With().Str("operation", "GetSessions").Logger()

	sessions, err := s.repo.GetUserActiveSessions(ctx, userID)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user active sessions")
		return nil, eris.Wrap(err, "failed to get user active sessions")
	}

	return sessions, nil
}

func (s *Service) RevokeSession(
	ctx context.Context,
	sessionID pulid.ID,
	clientIP, userAgent, reason string,
) error {
	log := s.l.With().Str("operation", "RevokeSession").Logger()

	// TODO(Wolfred): We may want to check if the userID to see if the session is being revoked by the user
	// or another user. We could always use the permission check to see if the user has the permission to revoke
	// the session.

	err := s.repo.RevokeSession(ctx, sessionID, clientIP, userAgent, reason)
	if err != nil {
		log.Error().Err(err).Msg("failed to revoke session")
		return eris.Wrap(err, "failed to revoke session")
	}

	return nil
}
