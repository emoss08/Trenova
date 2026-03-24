package fiscalyearservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
)

func createDateValidationRule() validationframework.TenantedRule[*fiscalyear.FiscalYear] {
	return validationframework.NewTenantedRule[*fiscalyear.FiscalYear]("date_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *fiscalyear.FiscalYear,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.EndDate <= entity.StartDate {
				multiErr.Add("endDate", errortypes.ErrInvalid, "End date must be after start date")
				return nil
			}

			startTime := time.Unix(entity.StartDate, 0)
			endTime := time.Unix(entity.EndDate, 0)
			durationDays := endTime.Sub(startTime).Hours() / 24

			if durationDays < 350 {
				multiErr.Add(
					"endDate",
					errortypes.ErrInvalid,
					"Fiscal year must be at least 350 days",
				)
			}

			if durationDays > 380 {
				multiErr.Add(
					"endDate",
					errortypes.ErrInvalid,
					"Fiscal year must be less than 380 days",
				)
			}

			if entity.IsCalendarYear {
				if startTime.Month() != time.January || startTime.Day() != 1 {
					multiErr.Add(
						"startDate",
						errortypes.ErrInvalid,
						"Calendar year must start on January 1st",
					)
				}
				if endTime.Month() != time.December || endTime.Day() != 31 {
					multiErr.Add(
						"endDate",
						errortypes.ErrInvalid,
						"Calendar year must end on December 31st",
					)
				}
			}

			maxFutureYears := 5
			currentYear := time.Now().Year()
			if entity.Year > currentYear+maxFutureYears {
				multiErr.Add(
					"year",
					errortypes.ErrInvalid,
					"Cannot create fiscal years more than 5 years in the future",
				)
			}

			return nil
		})
}

func createStatusConsistencyRule() validationframework.TenantedRule[*fiscalyear.FiscalYear] {
	return validationframework.NewTenantedRule[*fiscalyear.FiscalYear]("status_consistency").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *fiscalyear.FiscalYear,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			switch entity.Status {
			case fiscalyear.StatusClosed:
				if entity.ClosedAt == nil {
					multiErr.Add(
						"closedAt",
						errortypes.ErrInvalid,
						"Closed date is required when status is Closed",
					)
				}
				if entity.ClosedByID.IsNil() {
					multiErr.Add(
						"closedById",
						errortypes.ErrInvalid,
						"Closed by user is required when status is Closed",
					)
				}

			default:
			}

			return nil
		})
}

func createCurrentYearRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*fiscalyear.FiscalYear] {
	return validationframework.NewTenantedRule[*fiscalyear.FiscalYear]("current_year_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *fiscalyear.FiscalYear,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.IsCurrent && entity.Status != fiscalyear.StatusOpen {
				multiErr.Add(
					"status",
					errortypes.ErrInvalid,
					"Current fiscal year must have status 'Open'",
				)
			}

			if !entity.IsCurrent {
				return nil
			}

			q := db.DB().NewSelect().
				Model((*fiscalyear.FiscalYear)(nil)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where("fy.organization_id = ?", valCtx.OrganizationID).
						Where("fy.business_unit_id = ?", valCtx.BusinessUnitID).
						Where("fy.is_current = ?", true)
				})

			if valCtx.IsUpdate() {
				q = q.Where("fy.id != ?", entity.ID)
			}

			count, err := q.Count(ctx)
			if err != nil {
				multiErr.Add(
					"__all__",
					errortypes.ErrSystemError,
					"Failed to check current year uniqueness",
				)
				return nil
			}

			if count > 0 {
				multiErr.Add(
					"isCurrent",
					errortypes.ErrInvalid,
					"Another fiscal year is already marked as current. Only one fiscal year can be current at a time.",
				)
			}

			return nil
		})
}

func createOverlappingYearsRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*fiscalyear.FiscalYear] {
	return validationframework.NewTenantedRule[*fiscalyear.FiscalYear]("no_overlapping_years").
		OnBoth().
		WithStage(validationframework.ValidationStageDataIntegrity).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *fiscalyear.FiscalYear,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			q := db.DB().NewSelect().
				Model((*fiscalyear.FiscalYear)(nil)).
				WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
					return sq.Where("fy.organization_id = ?", valCtx.OrganizationID).
						Where("fy.business_unit_id = ?", valCtx.BusinessUnitID).
						Where("start_date <= ?", entity.EndDate).
						Where("end_date >= ?", entity.StartDate)
				})

			if valCtx.IsUpdate() {
				q = q.Where("fy.id != ?", entity.ID)
			}

			count, err := q.Count(ctx)
			if err != nil {
				multiErr.Add(
					"__all__",
					errortypes.ErrSystemError,
					"Failed to check for overlapping fiscal years",
				)
				return nil
			}

			if count > 0 {
				multiErr.Add(
					"startDate",
					errortypes.ErrInvalid,
					"This fiscal year's date range overlaps with an existing fiscal year",
				)
				multiErr.Add(
					"endDate",
					errortypes.ErrInvalid,
					"This fiscal year's date range overlaps with an existing fiscal year",
				)
			}

			return nil
		})
}
