/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/session"
	"github.com/emoss08/trenova/shared/pulid"
)

type SessionRepository interface {
	Create(ctx context.Context, session *session.Session) error
	GetValidSession(
		ctx context.Context,
		sessionID pulid.ID,
		clientIP string,
	) (*session.Session, error)
	GetUserActiveSessions(ctx context.Context, userID pulid.ID) ([]*session.Session, error)
	UpdateSessionActivity(
		ctx context.Context,
		sessionID pulid.ID,
		clientIP, userAgent string,
		eventType session.EventType,
		metadata map[string]any,
	) error
	UpdateSessionOrganization(
		ctx context.Context,
		sessionID pulid.ID,
		newOrgID pulid.ID,
	) error
	RevokeSession(ctx context.Context, sessionID pulid.ID, ip, userAgent, reason string) error
}
