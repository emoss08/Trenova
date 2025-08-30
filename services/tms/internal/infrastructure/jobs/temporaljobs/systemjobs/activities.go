package systemjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	AuditRepository         repositories.AuditRepository
	DataRetentionRepository repositories.DataRetentionRepository
}

type Activities struct {
	ar repositories.AuditRepository
	dr repositories.DataRetentionRepository
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		ar: p.AuditRepository,
		dr: p.DataRetentionRepository,
	}
}

func (a *Activities) DeleteAuditEntriesActivity(
	ctx context.Context,
) (*DeleteAuditEntriesResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting audit deletion activity")

	activity.RecordHeartbeat(ctx, "fetching data retention entities")

	drEntities, err := a.dr.List(ctx)
	if err != nil {
		logger.Error("Failed to get data retention entities", "error", err)
		return nil, temporaltype.NewRetryableError(
			"Failed to fetch data retention configuration",
			err,
		).ToTemporalError()
	}

	if drEntities.Total == 0 {
		logger.Info("No data retention entities found, skipping deletion")
		return &DeleteAuditEntriesResult{
			TotalDeleted: 0,
			Result:       "No data retention entities configured",
		}, nil
	}

	totalDeleted := 0
	deletedOrgIDs := make([]pulid.ID, 0, drEntities.Total)
	failedOrgs := make([]pulid.ID, 0)

	for _, drEntity := range drEntities.Items {
		if drEntity.AuditRetentionPeriod <= 0 {
			logger.Warn("Invalid retention period for organization",
				"orgID", drEntity.OrganizationID,
				"retentionPeriod", drEntity.AuditRetentionPeriod,
			)
			continue
		}

		timestamp := time.Now().AddDate(0, 0, -drEntity.AuditRetentionPeriod).Unix()

		activity.RecordHeartbeat(
			ctx,
			fmt.Sprintf("deleting audit entries for org %s", drEntity.OrganizationID),
		)

		deletedRows, err := a.ar.DeleteAuditEntries(ctx, timestamp)
		if err != nil {
			logger.Error("Failed to delete audit entries for organization",
				"error", err,
				"orgID", drEntity.OrganizationID,
				"retentionDays", drEntity.AuditRetentionPeriod,
			)

			failedOrgs = append(failedOrgs, drEntity.OrganizationID)
			continue
		}

		deletedOrgIDs = append(deletedOrgIDs, drEntity.OrganizationID)
		totalDeleted += int(deletedRows)

		logger.Info("Deleted audit entries for organization",
			"orgID", drEntity.OrganizationID,
			"deletedCount", deletedRows,
			"retentionDays", drEntity.AuditRetentionPeriod,
		)
	}

	if len(failedOrgs) > 0 && len(deletedOrgIDs) == 0 {
		return nil, temporaltype.NewRetryableError(
			fmt.Sprintf("Failed to delete audit entries for all organizations: %v", failedOrgs),
			nil,
		).ToTemporalError()
	}

	result := fmt.Sprintf(
		"Deleted %d audit entries for %d organizations. Successful: %v",
		totalDeleted,
		len(deletedOrgIDs),
		deletedOrgIDs,
	)

	if len(failedOrgs) > 0 {
		result += fmt.Sprintf(". Failed: %v", failedOrgs)
	}

	logger.Info("Audit deletion completed",
		"totalDeleted", totalDeleted,
		"successfulOrgs", len(deletedOrgIDs),
		"failedOrgs", len(failedOrgs),
	)

	return &DeleteAuditEntriesResult{
		TotalDeleted: totalDeleted,
		Result:       result,
	}, nil
}
