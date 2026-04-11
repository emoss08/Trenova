package fiscaljobs

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
)

type ActivitiesParams struct {
	fx.In

	AccountingControlRepo repositories.AccountingControlRepository
	FiscalYearRepo        repositories.FiscalYearRepository
	FiscalPeriodRepo      repositories.FiscalPeriodRepository
	FiscalPeriodService   *fiscalperiodservice.Service
	UserRepo              repositories.UserRepository
}

type Activities struct {
	acRepo   repositories.AccountingControlRepository
	fyRepo   repositories.FiscalYearRepository
	fpRepo   repositories.FiscalPeriodRepository
	fpSvc    *fiscalperiodservice.Service
	userRepo repositories.UserRepository
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		acRepo:   p.AccountingControlRepo,
		fyRepo:   p.FiscalYearRepo,
		fpRepo:   p.FiscalPeriodRepo,
		fpSvc:    p.FiscalPeriodService,
		userRepo: p.UserRepo,
	}
}

func (a *Activities) GetAutoCloseTenantsActivity(
	ctx context.Context,
) (*GetAutoCloseTenantsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Fetching tenants with scheduled period close enabled")

	controls, err := a.acRepo.ListWithScheduledPeriodClose(ctx)
	if err != nil {
		logger.Error("Failed to list accounting controls", "error", err)
		return nil, temporaltype.NewRetryableError("Failed to list accounting controls", err).
			ToTemporalError()
	}

	tenants := make([]OrgTenant, 0, len(controls))
	for _, ac := range controls {
		tenants = append(tenants, OrgTenant{
			OrganizationID: ac.OrganizationID,
			BusinessUnitID: ac.BusinessUnitID,
		})
	}

	logger.Info("Found tenants with scheduled period close enabled", "count", len(tenants))

	return &GetAutoCloseTenantsResult{Tenants: tenants}, nil
}

func (a *Activities) CloseExpiredPeriodsActivity(
	ctx context.Context,
	payload *AutoClosePeriodsPayload,
) (*AutoClosePeriodsResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Closing expired periods",
		"orgId", payload.OrganizationID.String(),
		"buId", payload.BusinessUnitID.String(),
	)

	now := timeutils.NowUnix()
	result := &AutoClosePeriodsResult{}

	periods, err := a.fpRepo.GetExpiredOpenPeriods(ctx, repositories.GetExpiredOpenPeriodsRequest{
		OrgID:      payload.OrganizationID,
		BuID:       payload.BusinessUnitID,
		BeforeDate: now,
	})
	if err != nil {
		logger.Error("Failed to get expired open periods", "error", err)
		return nil, temporaltype.NewRetryableError("Failed to get expired open periods", err).
			ToTemporalError()
	}

	if len(periods) == 0 {
		logger.Info("No expired open periods found")
		return result, nil
	}

	activity.RecordHeartbeat(ctx, fmt.Sprintf("Closing %d expired periods", len(periods)))

	systemUser, err := a.userRepo.GetSystemUser(ctx, "id")
	if err != nil {
		logger.Error("Failed to get system user", "error", err)
		return nil, temporaltype.NewRetryableError("Failed to get system user", err).
			ToTemporalError()
	}

	for _, period := range periods {
		_, closeErr := a.fpSvc.Close(ctx, repositories.CloseFiscalPeriodRequest{
			ID: period.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: payload.OrganizationID,
				BuID:  payload.BusinessUnitID,
			},
		}, systemUser.ID)

		if closeErr != nil {
			logger.Error("Failed to close period",
				"periodId", period.ID.String(),
				"periodNumber", period.PeriodNumber,
				"error", closeErr,
			)
			result.Errors = append(
				result.Errors,
				fmt.Sprintf("period %d: %s", period.PeriodNumber, closeErr.Error()),
			)
			continue
		}

		result.ClosedCount++
		logger.Info("Closed expired period",
			"periodId", period.ID.String(),
			"periodNumber", period.PeriodNumber,
		)
	}

	return result, nil
}

func (a *Activities) CheckAndCreateNextFiscalYearActivity(
	ctx context.Context,
	payload *AutoCreateFiscalYearPayload,
) (*AutoCreateFiscalYearResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Checking if next fiscal year needs creation",
		"orgId", payload.OrganizationID.String(),
	)

	result := &AutoCreateFiscalYearResult{}

	currentFY, err := a.fyRepo.GetCurrentFiscalYear(ctx, repositories.GetCurrentFiscalYearRequest{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	})
	if err != nil {
		result.SkipReason = "No current fiscal year found"
		return result, nil //nolint:nilerr // we want to skip the creation if no current fiscal year is found
	}

	endTime := time.Unix(currentFY.EndDate, 0).UTC()
	now := time.Now().UTC()
	daysUntilEnd := endTime.Sub(now).Hours() / 24

	if daysUntilEnd > float64(DefaultDaysBeforeYearEnd) {
		result.SkipReason = fmt.Sprintf(
			"Current fiscal year ends in %.0f days (threshold: %d)",
			daysUntilEnd,
			DefaultDaysBeforeYearEnd,
		)
		return result, nil
	}

	nextYear := currentFY.Year + 1
	nextStartDate := endTime.Add(time.Second)
	nextEndDate := nextStartDate.AddDate(1, 0, 0).Add(-time.Second)

	activity.RecordHeartbeat(ctx, fmt.Sprintf("Creating fiscal year %d", nextYear))

	newFY := &fiscalyear.FiscalYear{
		BusinessUnitID: currentFY.BusinessUnitID,
		OrganizationID: currentFY.OrganizationID,
		Status:         fiscalyear.StatusDraft,
		Year:           nextYear,
		Name:           fmt.Sprintf("Fiscal Year %d", nextYear),
		StartDate:      nextStartDate.Unix(),
		EndDate:        nextEndDate.Unix(),
		IsCalendarYear: currentFY.IsCalendarYear,
	}

	createdFY, createErr := a.fyRepo.Create(ctx, newFY)
	if createErr != nil {
		logger.Error("Failed to create next fiscal year", "error", createErr)
		return nil, temporaltype.NewRetryableError("Failed to create next fiscal year", createErr).
			ToTemporalError()
	}

	periods := fiscalyearservice.GenerateMonthlyPeriods(createdFY)
	bulkErr := a.fpRepo.BulkCreate(ctx, &repositories.BulkCreateFiscalPeriodsRequest{
		Periods: periods,
	})
	if bulkErr != nil {
		logger.Error("Failed to create periods for next fiscal year", "error", bulkErr)
		return nil, temporaltype.NewRetryableError("Failed to create periods for fiscal year", bulkErr).
			ToTemporalError()
	}

	result.Created = true
	result.FiscalYear = nextYear

	logger.Info("Created next fiscal year",
		"year", nextYear,
		"periods", len(periods),
	)

	return result, nil
}
