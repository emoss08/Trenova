package glaccountservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxHierarchyDepth = 10

func createSystemAccountProtectionRule() validationframework.TenantedRule[*glaccount.GLAccount] {
	return validationframework.NewTenantedRule[*glaccount.GLAccount]("system_account_protection").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *glaccount.GLAccount,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.IsSystem {
				multiErr.Add(
					"isSystem",
					errortypes.ErrInvalid,
					"System accounts cannot be modified",
				)
			}
			return nil
		})
}

func createParentAccountActiveRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*glaccount.GLAccount] {
	return validationframework.NewTenantedRule[*glaccount.GLAccount]("parent_account_active").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *glaccount.GLAccount,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.ParentID.IsNil() {
				return nil
			}

			parent := new(glaccount.GLAccount)
			err := db.DB().NewSelect().
				Model(parent).
				Column("id", "status").
				Where("id = ?", entity.ParentID).
				Where("organization_id = ?", valCtx.OrganizationID).
				Where("business_unit_id = ?", valCtx.BusinessUnitID).
				Scan(ctx)
			if err != nil {
				multiErr.Add("parentId", errortypes.ErrInvalid, "Parent account not found")
				return nil
			}

			if parent.Status != domaintypes.StatusActive {
				multiErr.Add("parentId", errortypes.ErrInvalid, "Parent account must be active")
			}

			return nil
		})
}

func createCircularReferenceRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*glaccount.GLAccount] {
	return validationframework.NewTenantedRule[*glaccount.GLAccount](
		"circular_reference_detection",
	).
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *glaccount.GLAccount,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.ParentID.IsNil() {
				return nil
			}

			if entity.ParentID == entity.ID {
				multiErr.Add("parentId", errortypes.ErrInvalid, "Account cannot be its own parent")
				return nil
			}

			visited := map[pulid.ID]struct{}{entity.ID: {}}
			currentParentID := entity.ParentID

			for range maxHierarchyDepth {
				if _, ok := visited[currentParentID]; ok {
					multiErr.Add(
						"parentId",
						errortypes.ErrInvalid,
						"Circular reference detected in account hierarchy",
					)
					return nil
				}

				visited[currentParentID] = struct{}{}

				var parentID *string
				err := db.DB().NewSelect().
					TableExpr("gl_accounts").
					Column("parent_id").
					Where("id = ?", currentParentID).
					Where("organization_id = ?", valCtx.OrganizationID).
					Where("business_unit_id = ?", valCtx.BusinessUnitID).
					Scan(ctx, &parentID)
				if err != nil {
					return nil
				}

				if parentID == nil || *parentID == "" {
					return nil
				}

				currentParentID = pulid.ID(*parentID)
			}

			multiErr.Add(
				"parentId",
				errortypes.ErrInvalid,
				"Account hierarchy is too deep (max 10 levels)",
			)
			return nil
		})
}

func createBalanceConsistencyRule() validationframework.TenantedRule[*glaccount.GLAccount] {
	return validationframework.NewTenantedRule[*glaccount.GLAccount]("balance_consistency").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityMedium).
		WithValidation(func(
			_ context.Context,
			entity *glaccount.GLAccount,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.DebitBalance < 0 {
				multiErr.Add(
					"debitBalance",
					errortypes.ErrInvalid,
					"Debit balance cannot be negative",
				)
			}

			if entity.CreditBalance < 0 {
				multiErr.Add(
					"creditBalance",
					errortypes.ErrInvalid,
					"Credit balance cannot be negative",
				)
			}

			return nil
		})
}

func createDeactivationProtectionRule(
	db *postgres.Connection,
) validationframework.TenantedRule[*glaccount.GLAccount] {
	return validationframework.NewTenantedRule[*glaccount.GLAccount]("deactivation_protection").
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *glaccount.GLAccount,
			valCtx *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Status == domaintypes.StatusActive {
				return nil
			}

			activeChildCount, err := db.DB().NewSelect().
				TableExpr("gl_accounts").
				Where("parent_id = ?", entity.ID).
				Where("organization_id = ?", valCtx.OrganizationID).
				Where("business_unit_id = ?", valCtx.BusinessUnitID).
				Where("status = ?", domaintypes.StatusActive).
				Count(ctx)
			if err != nil {
				return err
			}

			if activeChildCount > 0 {
				multiErr.Add(
					"status",
					errortypes.ErrInvalid,
					"Cannot deactivate account with active child accounts",
				)
			}

			if entity.CurrentBalance != 0 {
				multiErr.Add(
					"currentBalance",
					errortypes.ErrInvalid,
					"Cannot deactivate account with non-zero balance",
				)
			}

			return nil
		})
}
