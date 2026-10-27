package fiscalperiodservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalyear/fiscalyeardb"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func createDateValidationRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*fiscalperiod.FiscalPeriod] {
	return validationframework.NewTenantedRule[*fiscalperiod.FiscalPeriod]("date_validation").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *fiscalperiod.FiscalPeriod,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			return dbtx.ReadErr(ctx, db, func(ctx context.Context) error {
				if entity.EndDate <= entity.StartDate {
					multiErr.Add(
						"endDate",
						errortypes.ErrInvalid,
						"End date must be after start date",
					)
					return nil
				}

				if entity.FiscalYearID.IsNil() {
					return nil
				}

				fyt := fiscalyeardb.FiscalYears
				fy, err := fyt.Pick(fyt.StartDate, fyt.EndDate).
					Where(fyt.ID.Eq(entity.FiscalYearID)).
					One(ctx, db.Keel(ctx))
				if err != nil {
					return nil //nolint:nilerr // validation callbacks collect field errors and intentionally continue
				}

				if entity.StartDate < fy.StartDate {
					multiErr.Add(
						"startDate",
						errortypes.ErrInvalid,
						"Period start date cannot be before fiscal year start date",
					)
				}

				if entity.EndDate > fy.EndDate {
					multiErr.Add(
						"endDate",
						errortypes.ErrInvalid,
						"Period end date cannot be after fiscal year end date",
					)
				}

				return nil
			})
		})
}

func createStatusConsistencyRule() validationframework.TenantedRule[*fiscalperiod.FiscalPeriod] {
	return validationframework.NewTenantedRule[*fiscalperiod.FiscalPeriod]("status_consistency").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *fiscalperiod.FiscalPeriod,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Status == fiscalperiod.StatusClosed {
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
			}

			return nil
		})
}
