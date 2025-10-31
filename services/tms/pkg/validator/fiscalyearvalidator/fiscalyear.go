package fiscalyearvalidator

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accounting"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	factory *framework.TenantedValidatorFactory[*accounting.FiscalYear]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewValidator(p ValidatorParams) *Validator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*accounting.FiscalYear](
		getDB,
	).
		WithModelName("FiscalYear").
		WithCustomRules(func(entity *accounting.FiscalYear, vc *validator.ValidationContext) []framework.ValidationRule {
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
				// Date and business rules validation
				framework.NewBusinessRule("fiscal_year_business_rules").
					WithStage(framework.ValidationStageCompliance).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateDateRules(entity, me)
						validateStatusRules(entity, me)
						validateCurrentYearRules(ctx, entity, me, getDB, vc)
						return nil
					}),

				// Overlapping Years Check (database level)
				framework.NewBusinessRule("no_overlapping_fiscal_years").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateNoOverlappingYears(ctx, entity, me, getDB, vc)
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

func validateDateRules(entity *accounting.FiscalYear, me *errortypes.MultiError) {
	if entity.EndDate <= entity.StartDate {
		me.Add("endDate", errortypes.ErrInvalid, "End date must be after start date")
		return
	}

	startTime := time.Unix(entity.StartDate, 0)
	endTime := time.Unix(entity.EndDate, 0)
	durationDays := endTime.Sub(startTime).Hours() / 24

	// Fiscal year must be between 350-380 days (accounting for leap years and week-based years)
	if durationDays < 350 {
		me.Add("endDate", errortypes.ErrInvalid, "Fiscal year must be at least 350 days")
	}

	if durationDays > 380 {
		me.Add("endDate", errortypes.ErrInvalid, "Fiscal year must be less than 380 days")
	}

	if entity.IsCalendarYear {
		if startTime.Month() != time.January || startTime.Day() != 1 {
			me.Add("startDate", errortypes.ErrInvalid, "Start date must be January 1st")
		}
		if endTime.Month() != time.December || endTime.Day() != 31 {
			me.Add("endDate", errortypes.ErrInvalid, "Calendar year must end on December 31st")
		}
	}

	// Prevent creating fiscal years too far in the future
	maxFutureYears := 5
	currentYear := utils.GetCurrentYear()
	if entity.Year > currentYear+maxFutureYears {
		me.Add(
			"year",
			errortypes.ErrInvalid,
			"Cannot create fiscal years more than 5 years in the future",
		)
	}

	if entity.TaxYear != 0 && entity.TaxYear != entity.Year {
		me.Add(
			"taxYear",
			errortypes.ErrInvalid,
			"Taxs year must match fiscal year. Per IRS rules, a business's tax year if defined by its fiscal year. Form 1128 required for changes.",
		)
	}
}

func validateStatusRules(entity *accounting.FiscalYear, me *errortypes.MultiError) {
	switch entity.Status { //nolint:exhaustive // We only support these statuses
	case accounting.FiscalYearStatusClosed:
		if entity.ClosedAt == nil {
			me.Add(
				"closedAt",
				errortypes.ErrInvalid,
				"Closed date is required when status is Closed",
			)
		}
		if entity.ClosedByID.IsNil() {
			me.Add(
				"closedById",
				errortypes.ErrInvalid,
				"Closed by user is required when status is Closed",
			)
		}

		// Adjustment deadline validation
		if entity.AllowAdjustingEntries {
			if entity.AdjustmentDeadline == 0 {
				me.Add(
					"adjustmentDeadline",
					errortypes.ErrInvalid,
					"Adjustment deadline is required when adjusting entries are allowed",
				)
			} else if entity.AdjustmentDeadline <= entity.EndDate {
				me.Add(
					"adjustmentDeadline",
					errortypes.ErrInvalid,
					"Adjustment deadline must be after fiscal year end date",
				)
			}
		}

	case accounting.FiscalYearStatusLocked:
		if entity.LockedAt == nil {
			me.Add(
				"lockedAt",
				errortypes.ErrInvalid,
				"Locked date is required when status is Locked",
			)
		}
		if entity.LockedByID.IsNil() {
			me.Add(
				"lockedById",
				errortypes.ErrInvalid,
				"Locked by user is required when status is Locked",
			)
		}
		if entity.ClosedAt == nil {
			me.Add(
				"status",
				errortypes.ErrInvalid,
				"Fiscal year must be closed before it can be locked",
			)
		}
	}
}

func validateCurrentYearRules(
	ctx context.Context,
	entity *accounting.FiscalYear,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
	vCtx *validator.ValidationContext,
) {
	if entity.IsCurrent && entity.Status != accounting.FiscalYearStatusOpen {
		me.Add("status", errortypes.ErrInvalid, "Current fiscal year must have status 'Open'")
	}

	if entity.IsCurrent {
		db, err := getDB(ctx)
		if err != nil {
			me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
			return
		}

		q := db.NewSelect().
			Model((*accounting.FiscalYear)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where("fy.organization_id = ?", entity.OrganizationID).
					Where("fy.business_unit_id = ?", entity.BusinessUnitID).
					Where("fy.is_current = ?", true)
			})

		if vCtx.IsUpdate {
			q = q.Where("fy.id != ?", entity.ID)
		}

		count, err := q.Count(ctx)
		if err != nil {
			me.Add("__all__", errortypes.ErrSystemError, "Failed to check current year uniqueness")
			return
		}

		if count > 0 {
			me.Add(
				"isCurrent",
				errortypes.ErrInvalid,
				"Another fiscal year is already marked as current. Only one fiscal year can be current at a time.",
			)
		}
	}
}

func validateNoOverlappingYears(
	ctx context.Context,
	entity *accounting.FiscalYear,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
	vCtx *validator.ValidationContext,
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Check for overlapping date ranges
	// A overlap B if (A.start <= B.end) AND (A.end >= B.start)
	q := db.NewSelect().
		Model((*accounting.FiscalYear)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where("fy.organization_id = ?", entity.OrganizationID).
				Where("fy.business_unit_id = ?", entity.BusinessUnitID).
				Where("start_date <= ?", entity.EndDate).
				Where("end_date >= ?", entity.StartDate)
		})

	if vCtx.IsUpdate {
		q = q.Where("fy.id != ?", entity.ID)
	}

	count, err := q.Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for overlapping fiscal years")
	}

	if count > 0 {
		me.Add(
			"startDate",
			errortypes.ErrInvalid,
			"This fiscal year's date range overlaps with an existing fiscal year",
		)
		me.Add(
			"endDate",
			errortypes.ErrInvalid,
			"This fiscal year's date range overlaps with an existing fiscal year",
		)
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *accounting.FiscalYear,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
