package briefingjobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// retentionDays is how far back a reader can page. A fortnight covers
	// coming back from leave; past that the records themselves are the
	// record.
	retentionDays  = 60
	retentionBatch = 1000
	// retentionPasses bounds one run's deletes so a first sweep over a
	// long-running deployment ends in a predictable time.
	retentionPasses = 10
)

type ActivitiesParams struct {
	fx.In

	Logger    *zap.Logger
	Briefings services.BriefingService
	Repo      repositories.BriefingRepository
	Controls  repositories.AgentControlRepository
	Tenants   repositories.TenantSyncRepository
	Cache     repositories.OrganizationCacheRepository
}

type Activities struct {
	l         *zap.Logger
	briefings services.BriefingService
	repo      repositories.BriefingRepository
	controls  repositories.AgentControlRepository
	tenants   repositories.TenantSyncRepository
	cache     repositories.OrganizationCacheRepository
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		l:         p.Logger.Named("job.daily-briefing"),
		briefings: p.Briefings,
		repo:      p.Repo,
		controls:  p.Controls,
		tenants:   p.Tenants,
		cache:     p.Cache,
	}
}

// WriteDueBriefingsActivity writes the morning for every organization whose
// local clock has just reached its briefing hour.
//
// The whole sweep is anchored to one instant so every organization written
// in it describes the same moment; the hour each is compared against is
// still its own. An organization that fails is recorded and stepped over,
// because one tenant's unreadable table must not cost every other tenant
// its morning.
func (a *Activities) WriteDueBriefingsActivity(
	ctx context.Context,
) (*DailyBriefingResult, error) {
	organizations, err := a.tenants.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("list organizations: %w", err)
	}

	now := timeutils.NowUnix()
	result := &DailyBriefingResult{FailedOrganizations: make([]string, 0)}

	for _, org := range organizations {
		activity.RecordHeartbeat(ctx, org.ID.String())

		tenantInfo := pagination.TenantInfo{OrgID: org.ID, BuID: org.BusinessUnitID}
		due, dErr := a.isDue(ctx, tenantInfo, now)
		if dErr != nil {
			a.l.Warn("could not decide whether a briefing is due",
				zap.String("organization", org.ID.String()),
				zap.Error(dErr),
			)
			result.FailedOrganizations = append(result.FailedOrganizations, org.ID.String())

			continue
		}
		if !due {
			continue
		}

		result.OrganizationsDue++
		written, wErr := a.briefings.WriteForDay(ctx, services.WriteBriefingRequest{
			TenantInfo: tenantInfo,
			Now:        now,
		})
		if wErr != nil {
			a.l.Error("failed to write the morning briefing",
				zap.String("organization", org.ID.String()),
				zap.Error(wErr),
			)
			result.FailedOrganizations = append(result.FailedOrganizations, org.ID.String())

			continue
		}

		result.BriefingsWritten += written.Written
		result.BriefingsNarrated += written.Narrated
	}

	a.l.Info("daily briefing sweep complete",
		zap.Int("due", result.OrganizationsDue),
		zap.Int("written", result.BriefingsWritten),
		zap.Int("narrated", result.BriefingsNarrated),
	)

	return result, nil
}

// BriefingRetentionActivity removes briefings older than the window a reader can
// page back through.
func (a *Activities) BriefingRetentionActivity(
	ctx context.Context,
) (*BriefingRetentionResult, error) {
	cutoff := time.Unix(timeutils.NowUnix(), 0).
		UTC().
		AddDate(0, 0, -retentionDays).
		Format(timeutils.ISODateLayout)
	result := &BriefingRetentionResult{}

	for pass := range retentionPasses {
		activity.RecordHeartbeat(ctx, pass)

		deleted, err := a.repo.DeleteBefore(ctx, repositories.DeleteBriefingsBeforeRequest{
			BeforeDate: cutoff,
			Limit:      retentionBatch,
		})
		if err != nil {
			return nil, fmt.Errorf("delete briefings: %w", err)
		}
		result.Deleted += deleted
		if deleted < retentionBatch {
			break
		}
	}

	a.l.Info("briefing retention sweep complete", zap.Int("deleted", result.Deleted))

	return result, nil
}

// isDue reports whether this organization's local clock has just reached
// its briefing hour. The comparison is on the hour rather than the minute
// because the sweep runs hourly: a tenant is written in the firing whose
// local hour matches, and in no other.
func (a *Activities) isDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	now int64,
) (bool, error) {
	control, err := a.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return false, err
	}
	if !control.BriefingEnabled {
		return false, nil
	}

	organization, err := a.cache.GetByID(ctx, tenantInfo.OrgID)
	if err != nil {
		return false, err
	}

	location, err := time.LoadLocation(timeutils.NormalizeTimezone(organization.Timezone))
	if err != nil {
		return false, fmt.Errorf("load timezone %q: %w", organization.Timezone, err)
	}

	return time.Unix(now, 0).In(location).Hour() == briefingHour(control), nil
}

func briefingHour(control *tenant.AgentControl) int {
	if control.BriefingHourLocal < 0 || control.BriefingHourLocal > 23 {
		return tenant.DefaultBriefingHourLocal
	}

	return control.BriefingHourLocal
}
