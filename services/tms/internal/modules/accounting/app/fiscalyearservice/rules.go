package fiscalyearservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func createDateValidationRule() validationframework.Rule[*fiscalyear.FiscalYear] {
	return validationframework.NewRule[*fiscalyear.FiscalYear]("date_validation").
		OnBoth().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *fiscalyear.FiscalYear,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if entity.EndDate <= entity.StartDate {
				multiErr.Add("endDate", errortypes.ErrInvalid, "End date must be after start date")
				return nil
			}

			startTime := time.Unix(entity.StartDate, 0).UTC()
			endTime := time.Unix(entity.EndDate, 0).UTC()
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

func createStatusConsistencyRule() validationframework.Rule[*fiscalyear.FiscalYear] {
	return validationframework.NewRule[*fiscalyear.FiscalYear]("status_consistency").
		OnBoth().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *fiscalyear.FiscalYear,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			//nolint:exhaustive // only actionable enum states require explicit handling here
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
) validationframework.Rule[*fiscalyear.FiscalYear] {
	return validationframework.NewRule[*fiscalyear.FiscalYear]("current_year_validation").
		OnBoth().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			ctx context.Context,
			entity *fiscalyear.FiscalYear,
			valCtx *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			return dbtx.ReadErr(ctx, db, func(ctx context.Context) error {
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

				fy := fiscalyeardb.FiscalYears
				exists, err := fy.Where(
					fy.IsCurrent.IsTrue(),
					keel.When(valCtx.IsUpdate(), fy.ID.Ne(entity.ID)),
				).Exists(ctx, db.Keel(ctx))
				if err != nil {
					return fmt.Errorf("failed to check current year uniqueness: %w", err)
				}

				if exists {
					multiErr.Add(
						"isCurrent",
						errortypes.ErrInvalid,
						"Another fiscal year is already marked as current. Only one fiscal year can be current at a time.",
					)
				}

				return nil
			})
		})
}

func createOverlappingYearsRule(
	db *postgres.Connection,
) validationframework.Rule[*fiscalyear.FiscalYear] {
	return validationframework.NewRule[*fiscalyear.FiscalYear]("no_overlapping_years").
		OnBoth().
		WithStage(validationframework.StageDataIntegrity).
		WithValidation(func(
			ctx context.Context,
			entity *fiscalyear.FiscalYear,
			valCtx *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			return dbtx.ReadErr(ctx, db, func(ctx context.Context) error {
				fy := fiscalyeardb.FiscalYears
				exists, err := fy.Where(
					fy.StartDate.Lte(entity.EndDate),
					fy.EndDate.Gte(entity.StartDate),
					keel.When(valCtx.IsUpdate(), fy.ID.Ne(entity.ID)),
				).Exists(ctx, db.Keel(ctx))
				if err != nil {
					return fmt.Errorf("failed to check for overlapping fiscal years: %w", err)
				}

				if exists {
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
		})
}
