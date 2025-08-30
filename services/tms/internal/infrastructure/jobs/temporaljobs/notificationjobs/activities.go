/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package notificationjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	NotificationService services.NotificationService
}

type Activities struct {
	ns services.NotificationService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ns: p.NotificationService,
	}
}

func (a *Activities) SendNotificationActivity(
	ctx context.Context,
	payload *SendNotificationPayload,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting notification activity",
		"userID", payload.UserID,
		"jobType", payload.JobType,
		"jobID", payload.JobID,
	)

	// Validate input
	if payload.UserID.IsNil() {
		return temporaltype.NewInvalidInputError(
			"User ID is required for notification",
			map[string]any{
				"userID": payload.UserID,
			},
		).ToTemporalError()
	}

	if payload.JobType == "" {
		return temporaltype.NewInvalidInputError(
			"Job type is required for notification",
			map[string]any{
				"jobType": payload.JobType,
			},
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "sending notification")

	err := a.ns.SendJobCompletionNotification(ctx, payload)
	if err != nil {
		logger.Error("Failed to send notification",
			"error", err,
			"userID", payload.UserID,
			"jobType", payload.JobType,
		)

		// Classify the error for proper retry behavior
		appErr := temporaltype.ClassifyError(err)

		// If it's a user not found or invalid recipient, don't retry
		if appErr.Type == temporaltype.ErrorTypeResourceNotFound {
			return temporaltype.NewNonRetryableError(
				fmt.Sprintf("Recipient user not found: %s", payload.UserID),
				err,
			).ToTemporalError()
		}

		return appErr.ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "notification sent successfully")
	logger.Info("Notification sent successfully",
		"userID", payload.UserID,
		"jobType", payload.JobType,
	)

	return nil
}

func (a *Activities) SendConfigurationCopiedNotificationActivity(
	ctx context.Context,
	payload *SendConfigurationCopiedNotificationPayload,
) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting configuration copied notification activity",
		"userID", payload.UserID,
		"configID", payload.ConfigID,
		"configName", payload.ConfigName,
	)

	// Validate input
	if payload.UserID.IsNil() {
		return temporaltype.NewInvalidInputError(
			"User ID is required for notification",
			map[string]any{
				"userID": payload.UserID,
			},
		).ToTemporalError()
	}

	if payload.ConfigID.IsNil() {
		return temporaltype.NewInvalidInputError(
			"Configuration ID is required",
			map[string]any{
				"configID": payload.ConfigID,
			},
		).ToTemporalError()
	}

	if payload.ConfigName == "" {
		return temporaltype.NewInvalidInputError(
			"Configuration name is required",
			map[string]any{
				"configName": payload.ConfigName,
			},
		).ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "sending configuration copied notification")

	err := a.ns.SendConfigurationCopiedNotification(ctx, payload)
	if err != nil {
		logger.Error("Failed to send configuration copied notification",
			"error", err,
			"userID", payload.UserID,
			"configID", payload.ConfigID,
			"configName", payload.ConfigName,
		)

		// Classify the error for proper retry behavior
		appErr := temporaltype.ClassifyError(err)

		// Special handling for configuration-related errors
		if appErr.Type == temporaltype.ErrorTypeResourceNotFound {
			return temporaltype.NewNonRetryableError(
				fmt.Sprintf("Configuration or user not found: user=%s, config=%s",
					payload.UserID, payload.ConfigID),
				err,
			).ToTemporalError()
		}

		return appErr.ToTemporalError()
	}

	activity.RecordHeartbeat(ctx, "configuration copied notification sent successfully")
	logger.Info("Configuration copied notification sent successfully",
		"userID", payload.UserID,
		"configID", payload.ConfigID,
		"configName", payload.ConfigName,
	)

	return nil
}
