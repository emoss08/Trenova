package fiscalperiodvalidator

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	factory *framework.TenantedValidatorFactory[*accounting.FiscalPeriod]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewValidator(p Params) *Validator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*accounting.FiscalPeriod](
		getDB,
	).
		WithModelName("FiscalPeriod").
		WithCustomRules(func(entity *accounting.FiscalPeriod, vc *validator.ValidationContext) []framework.ValidationRule {
			var rules []framework.ValidationRule

			if vc.IsCreate {
				rules = append(rules, framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if entity.ID.IsNotNil() {
							multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
						}
						return nil
					}),
				)
			}

			rules = append(rules,
				framework.NewBusinessRule("fiscal_period_business_rules").
					WithStage(framework.ValidationStageCompliance).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateDateRules(entity, me)
						validateStatusRules(entity, me)
						validatePeriodNumberRules(ctx, entity, me, getDB, vc)
						return nil
					}),

				framework.NewBusinessRule("no_overlapping_periods").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateNoOverlappingPeriods(ctx, entity, me, getDB, vc)
						return nil
					}),

				framework.NewBusinessRule("period_within_fiscal_year").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validatePeriodWithinFiscalYear(ctx, entity, me, getDB)
						return nil
					}),

				framework.NewBusinessRule("sequential_period_closing").
					WithStage(framework.ValidationStageCompliance).
					WithPriority(framework.ValidationPriorityMedium).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						if vc.IsUpdate && entity.Status == accounting.PeriodStatusClosed {
							validateSequentialClosing(ctx, entity, me, getDB)
						}
						return nil
					}),
			)

			return rules
		})

	return &Validator{
		factory: factory,
		getDB:   getDB,
	}
}

func validateDateRules(entity *accounting.FiscalPeriod, me *errortypes.MultiError) {
	if entity.EndDate <= entity.StartDate {
		me.Add("endDate", errortypes.ErrInvalid, "End date must be after start date")
		return
	}

	startTime := time.Unix(entity.StartDate, 0)
	endTime := time.Unix(entity.EndDate, 0)
	durationDays := endTime.Sub(startTime).Hours() / 24

	switch entity.PeriodType {
	case accounting.PeriodTypeMonth:
		if durationDays < 28 {
			me.Add("endDate", errortypes.ErrInvalid, "Monthly period must be at least 28 days")
		}
		if durationDays > 31 {
			me.Add("endDate", errortypes.ErrInvalid, "Monthly period cannot exceed 31 days")
		}

	case accounting.PeriodTypeQuarter:
		if durationDays < 89 {
			me.Add("endDate", errortypes.ErrInvalid, "Quarterly period must be at least 89 days")
		}
		if durationDays > 92 {
			me.Add("endDate", errortypes.ErrInvalid, "Quarterly period cannot exceed 92 days")
		}

	case accounting.PeriodTypeYear:
		if durationDays < 350 {
			me.Add("endDate", errortypes.ErrInvalid, "Yearly period must be at least 350 days")
		}
		if durationDays > 380 {
			me.Add("endDate", errortypes.ErrInvalid, "Yearly period cannot exceed 380 days")
		}
	}
}

func validateStatusRules(entity *accounting.FiscalPeriod, me *errortypes.MultiError) {
	switch entity.Status { //nolint:exhaustive // We only validate specific statuses
	case accounting.PeriodStatusClosed:
		if entity.ClosedAt == nil {
			me.Add(
				"closedAt",
				errortypes.ErrInvalid,
				"Closed date is required when status is Closed",
			)
		}
		if entity.ClosedByID == nil || entity.ClosedByID.IsNil() {
			me.Add(
				"closedById",
				errortypes.ErrInvalid,
				"Closed by user is required when status is Closed",
			)
		}

	case accounting.PeriodStatusLocked:
		if entity.ClosedAt == nil {
			me.Add(
				"status",
				errortypes.ErrInvalid,
				"Period must be closed before it can be locked",
			)
		}
	}
}

func validatePeriodNumberRules(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
	vCtx *validator.ValidationContext,
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	q := db.NewSelect().
		Model((*accounting.FiscalPeriod)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.fiscal_year_id = ?", entity.FiscalYearID).
				Where("fp.organization_id = ?", entity.OrganizationID).
				Where("fp.business_unit_id = ?", entity.BusinessUnitID).
				Where("fp.period_number = ?", entity.PeriodNumber)
		})

	if vCtx.IsUpdate {
		q = q.Where("fp.id != ?", entity.ID)
	}

	count, err := q.Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check period number uniqueness")
		return
	}

	if count > 0 {
		me.Add(
			"periodNumber",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Period number %d already exists for this fiscal year",
				entity.PeriodNumber,
			),
		)
	}

	switch entity.PeriodType {
	case accounting.PeriodTypeMonth:
		if entity.PeriodNumber < 1 || entity.PeriodNumber > 12 {
			me.Add(
				"periodNumber",
				errortypes.ErrInvalid,
				"Monthly period number must be between 1 and 12",
			)
		}
	case accounting.PeriodTypeQuarter:
		if entity.PeriodNumber < 1 || entity.PeriodNumber > 4 {
			me.Add(
				"periodNumber",
				errortypes.ErrInvalid,
				"Quarterly period number must be between 1 and 4",
			)
		}
	case accounting.PeriodTypeYear:
		if entity.PeriodNumber != 1 {
			me.Add(
				"periodNumber",
				errortypes.ErrInvalid,
				"Yearly period number must be 1",
			)
		}
	}
}

func validateNoOverlappingPeriods(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
	vCtx *validator.ValidationContext,
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Check for overlapping date ranges within the same fiscal year
	// A overlaps B if (A.start <= B.end) AND (A.end >= B.start)
	q := db.NewSelect().
		Model((*accounting.FiscalPeriod)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.fiscal_year_id = ?", entity.FiscalYearID).
				Where("fp.organization_id = ?", entity.OrganizationID).
				Where("fp.business_unit_id = ?", entity.BusinessUnitID).
				Where("fp.start_date <= ?", entity.EndDate).
				Where("fp.end_date >= ?", entity.StartDate)
		})

	if vCtx.IsUpdate {
		q = q.Where("fp.id != ?", entity.ID)
	}

	count, err := q.Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for overlapping periods")
		return
	}

	if count > 0 {
		me.Add(
			"startDate",
			errortypes.ErrInvalid,
			"This period's date range overlaps with an existing period",
		)
		me.Add(
			"endDate",
			errortypes.ErrInvalid,
			"This period's date range overlaps with an existing period",
		)
	}
}

func validatePeriodWithinFiscalYear(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	fiscalYear := new(accounting.FiscalYear)
	err = db.NewSelect().
		Model(fiscalYear).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fy.id = ?", entity.FiscalYearID).
				Where("fy.organization_id = ?", entity.OrganizationID).
				Where("fy.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("fiscalYearId", errortypes.ErrInvalid, "Fiscal year not found")
		return
	}

	if entity.StartDate < fiscalYear.StartDate {
		me.Add(
			"startDate",
			errortypes.ErrInvalid,
			"Period start date must be on or after fiscal year start date",
		)
	}

	if entity.EndDate > fiscalYear.EndDate {
		me.Add(
			"endDate",
			errortypes.ErrInvalid,
			"Period end date must be on or before fiscal year end date",
		)
	}
}

func validateSequentialClosing(
	ctx context.Context,
	entity *accounting.FiscalPeriod,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	// Periods should be closed in order (Period 1 before Period 2, etc.)
	if entity.PeriodNumber == 1 {
		return
	}

	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	previousPeriodNumber := entity.PeriodNumber - 1

	var previousPeriod accounting.FiscalPeriod
	err = db.NewSelect().
		Model(&previousPeriod).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("fp.fiscal_year_id = ?", entity.FiscalYearID).
				Where("fp.organization_id = ?", entity.OrganizationID).
				Where("fp.business_unit_id = ?", entity.BusinessUnitID).
				Where("fp.period_number = ?", previousPeriodNumber)
		}).
		Scan(ctx)
	if err != nil {
		return
	}

	if previousPeriod.Status != accounting.PeriodStatusClosed &&
		previousPeriod.Status != accounting.PeriodStatusLocked {
		me.Add(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf("Period %d must be closed before closing Period %d",
				previousPeriodNumber, entity.PeriodNumber),
		)
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *accounting.FiscalPeriod,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
