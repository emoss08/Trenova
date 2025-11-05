package glaccountvalidator

import (
	"context"
	"fmt"

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
	factory *framework.TenantedValidatorFactory[*accounting.GLAccount]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewValidator(p Params) *Validator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*accounting.GLAccount](
		getDB,
	).
		WithModelName("GLAccount").
		WithUniqueFields(func(g *accounting.GLAccount) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "account_code",
					GetValue: func() string { return g.AccountCode },
					Message:  "Account code ':value' already exists in the organization.",
				},
			}
		}).
		WithCustomRules(
			func(entity *accounting.GLAccount, vc *validator.ValidationContext) []framework.ValidationRule {
				var rules []framework.ValidationRule

				if vc.IsCreate {
					rules = append(rules, framework.NewBusinessRule("id_validation").
						WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
							if entity.ID.IsNotNil() {
								multiErr.Add(
									"id",
									errortypes.ErrInvalid,
									"ID cannot be set on create",
								)
							}
							return nil
						}),
					)
				}

				rules = append(rules,
					framework.NewBusinessRule("gl_account_business_rules").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validateSystemAccountProtection(entity, me, vc)
							return nil
						}),

					framework.NewBusinessRule("parent_account_validation").
						WithStage(framework.ValidationStageDataIntegrity).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							if entity.ParentID != nil && !entity.ParentID.IsNil() {
								validateParentAccount(ctx, entity, me, getDB)
								validateNoCircularReference(ctx, entity, me, getDB)
							}
							return nil
						}),

					framework.NewBusinessRule("account_type_validation").
						WithStage(framework.ValidationStageDataIntegrity).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							validateAccountTypeExists(ctx, entity, me, getDB)
							return nil
						}),

					framework.NewBusinessRule("balance_validation").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityMedium).
						WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
							validateBalanceConsistency(entity, me)
							return nil
						}),

					framework.NewBusinessRule("deletion_protection").
						WithStage(framework.ValidationStageCompliance).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							if vc.IsUpdate && !entity.IsActive {
								validateCanDeactivate(ctx, entity, me, getDB)
							}
							return nil
						}),
				)

				return rules
			},
		)

	return &Validator{
		factory: factory,
		getDB:   getDB,
	}
}

func validateSystemAccountProtection(
	entity *accounting.GLAccount,
	me *errortypes.MultiError,
	vCtx *validator.ValidationContext,
) {
	if vCtx.IsUpdate && entity.IsSystem {
		me.Add(
			"isSystem",
			errortypes.ErrInvalid,
			"System accounts cannot be modified. Please contact support if changes are needed.",
		)
	}
}

func validateParentAccount(
	ctx context.Context,
	entity *accounting.GLAccount,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	var parent accounting.GLAccount
	err = db.NewSelect().
		Model(&parent).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.id = ?", entity.ParentID).
				Where("gla.organization_id = ?", entity.OrganizationID).
				Where("gla.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("parentId", errortypes.ErrInvalid, "Parent account not found")
		return
	}

	if !parent.IsActive {
		me.Add("parentId", errortypes.ErrInvalid, "Parent account must be active")
	}
}

func validateNoCircularReference(
	ctx context.Context,
	entity *accounting.GLAccount,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	if entity.ParentID == nil || entity.ParentID.IsNil() {
		return
	}

	if entity.ParentID.String() == entity.ID.String() {
		me.Add("parentId", errortypes.ErrInvalid, "Account cannot be its own parent")
		return
	}

	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	visited := make(map[string]bool)
	currentParentID := entity.ParentID
	maxDepth := 10

	for i := 0; i < maxDepth; i++ {
		if currentParentID == nil || currentParentID.IsNil() {
			return
		}

		parentIDStr := currentParentID.String()
		if visited[parentIDStr] {
			me.Add(
				"parentId",
				errortypes.ErrInvalid,
				"Circular reference detected in parent hierarchy",
			)
			return
		}
		visited[parentIDStr] = true

		if parentIDStr == entity.ID.String() {
			me.Add(
				"parentId",
				errortypes.ErrInvalid,
				"Circular reference detected in parent hierarchy",
			)
			return
		}

		var parent accounting.GLAccount
		err = db.NewSelect().
			Model(&parent).
			Column("parent_id").
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					Where("gla.id = ?", currentParentID).
					Where("gla.organization_id = ?", entity.OrganizationID).
					Where("gla.business_unit_id = ?", entity.BusinessUnitID)
			}).
			Scan(ctx)
		if err != nil {
			return
		}

		currentParentID = parent.ParentID
	}

	if currentParentID != nil && !currentParentID.IsNil() {
		me.Add("parentId", errortypes.ErrInvalid, "Account hierarchy is too deep (max 10 levels)")
	}
}

func validateAccountTypeExists(
	ctx context.Context,
	entity *accounting.GLAccount,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	count, err := db.NewSelect().
		Model((*accounting.AccountType)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("at.id = ?", entity.AccountTypeID).
				Where("at.organization_id = ?", entity.OrganizationID).
				Where("at.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to validate account type")
		return
	}

	if count == 0 {
		me.Add("accountTypeId", errortypes.ErrInvalid, "Account type not found")
	}
}

func validateBalanceConsistency(entity *accounting.GLAccount, me *errortypes.MultiError) {
	if entity.DebitBalance < 0 {
		me.Add("debitBalance", errortypes.ErrInvalid, "Debit balance cannot be negative")
	}
	if entity.CreditBalance < 0 {
		me.Add("creditBalance", errortypes.ErrInvalid, "Credit balance cannot be negative")
	}
}

func validateCanDeactivate(
	ctx context.Context,
	entity *accounting.GLAccount,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	childCount, err := db.NewSelect().
		Model((*accounting.GLAccount)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("gla.parent_id = ?", entity.ID).
				Where("gla.organization_id = ?", entity.OrganizationID).
				Where("gla.business_unit_id = ?", entity.BusinessUnitID).
				Where("gla.is_active = ?", true)
		}).
		Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for child accounts")
		return
	}

	if childCount > 0 {
		me.Add(
			"isActive",
			errortypes.ErrInvalid,
			fmt.Sprintf("Cannot deactivate account with %d active child accounts", childCount),
		)
	}

	if entity.CurrentBalance != 0 {
		me.Add(
			"isActive",
			errortypes.ErrInvalid,
			"Cannot deactivate account with non-zero balance. Please transfer or clear the balance first.",
		)
	}

	// Check if account has posted journal entries in open periods
	jeCount, err := db.NewSelect().
		Model((*accounting.JournalEntryLine)(nil)).
		Join("INNER JOIN journal_entries AS je ON je.id = jel.journal_entry_id").
		Join("INNER JOIN fiscal_periods AS fp ON fp.id = je.fiscal_period_id").
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("jel.gl_account_id = ?", entity.ID).
				Where("jel.organization_id = ?", entity.OrganizationID).
				Where("jel.business_unit_id = ?", entity.BusinessUnitID).
				Where("je.is_posted = ?", true).
				Where("fp.status = ?", accounting.PeriodStatusOpen)
		}).
		Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for journal entries")
		return
	}

	if jeCount > 0 {
		me.Add(
			"isActive",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"Cannot deactivate account with %d posted transactions in open periods. Close the periods first.",
				jeCount,
			),
		)
	}
}

func (v *Validator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *accounting.GLAccount,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
