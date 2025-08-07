/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetUserPreferencesRequest struct {
	UserID         pulid.ID
	OrganizationID pulid.ID
	Resource       permission.Resource
	IsActive       bool
}

type ListNotificationPreferencesRequest struct {
	Filter *ports.LimitOffsetQueryOptions
}

type NotificationPreferenceRepository interface {
	// Create creates a new notification preference
	Create(
		ctx context.Context,
		pref *notification.NotificationPreference,
	) (*notification.NotificationPreference, error)

	// Update updates an existing notification preference
	Update(
		ctx context.Context,
		pref *notification.NotificationPreference,
	) (*notification.NotificationPreference, error)

	// Delete deletes a notification preference
	Delete(ctx context.Context, id pulid.ID) error

	// GetByID retrieves a notification preference by ID
	GetByID(ctx context.Context, id pulid.ID) (*notification.NotificationPreference, error)

	// GetUserPreferences retrieves notification preferences for a user
	GetUserPreferences(
		ctx context.Context,
		req *GetUserPreferencesRequest,
	) ([]*notification.NotificationPreference, error)

	// List retrieves all notification preferences with filtering
	List(
		ctx context.Context,
		req ListNotificationPreferencesRequest,
	) (*ports.ListResult[*notification.NotificationPreference], error)
}
