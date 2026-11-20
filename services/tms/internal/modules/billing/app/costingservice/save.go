package costingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/billing/ports/costingcontrolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type CategoryChange struct {
	ID                  pulid.ID
	RateSource          costingcontrol.RateSource
	OverrideRatePerMile decimal.NullDecimal
	IsActive            bool
	GLAccountIDs        []pulid.ID
	Version             int64
}

type SaveRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      *permissionports.RequestActor
	Control    *costingcontrol.CostingControl
	Categories []*CategoryChange
}

type PlannedCategory struct {
	Category     *costingcontrol.CostCategory
	Original     *costingcontrol.CostCategory
	GLAccountIDs []pulid.ID
}

type SavePlan struct {
	Original       *costingcontrol.CostingControl
	Control        *costingcontrol.CostingControl
	ControlChanged bool
	Categories     []*PlannedCategory
	Unchanged      []*costingcontrol.CostCategory
}

func (s *Service) Save(
	ctx context.Context,
	req *SaveRequest,
) (*costingcontrol.CostingControl, error) {
	log := s.l.With(
		zap.String("operation", "Save"),
		zap.String("orgID", req.TenantInfo.OrgID.String()),
	)

	getReq := &costingcontrolports.GetCostingControlRequest{TenantInfo: req.TenantInfo}
	original, err := s.repo.GetByOrgID(ctx, getReq)
	if err != nil {
		log.Error("failed to get costing control", zap.Error(err))
		return nil, err
	}

	if req.Control == nil && len(req.Categories) == 0 {
		return original, nil
	}

	plan := s.planSave(ctx, req, original)
	if err = s.validator.ValidateSave(ctx, plan); err != nil {
		return nil, err
	}

	if err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		return s.writePlan(txCtx, req, plan)
	}); err != nil {
		log.Error("failed to save costing control", zap.Error(err))
		return nil, err
	}

	return s.repo.GetByOrgID(ctx, getReq)
}

func (s *Service) planSave(
	ctx context.Context,
	req *SaveRequest,
	original *costingcontrol.CostingControl,
) *SavePlan {
	plan := &SavePlan{
		Original:   original,
		Control:    original,
		Categories: make([]*PlannedCategory, 0, len(req.Categories)),
	}

	if req.Control != nil {
		control := *req.Control
		control.ID = original.ID
		control.BusinessUnitID = original.BusinessUnitID
		control.OrganizationID = original.OrganizationID
		control.CreatedAt = original.CreatedAt
		control.UpdatedAt = original.UpdatedAt
		control.FuelIndex = nil
		control.Categories = nil
		control.KeepFields(
			original,
			permissionports.VisibleFieldGroups(ctx, s.fields, req.Actor),
		)
		plan.Control = &control
		plan.ControlChanged = true
	}

	byID := make(map[pulid.ID]*costingcontrol.CostCategory, len(original.Categories))
	for _, category := range original.Categories {
		if category != nil {
			byID[category.ID] = category
		}
	}

	changed := make(map[pulid.ID]struct{}, len(req.Categories))
	for _, change := range req.Categories {
		existing := byID[change.ID]
		if existing == nil {
			plan.Categories = append(plan.Categories, &PlannedCategory{})
			continue
		}
		changed[change.ID] = struct{}{}
		plan.Categories = append(plan.Categories, &PlannedCategory{
			Category:     applyCategoryChange(existing, change),
			Original:     existing,
			GLAccountIDs: sliceutils.Dedupe(change.GLAccountIDs),
		})
	}

	for _, category := range original.Categories {
		if category == nil {
			continue
		}
		if _, ok := changed[category.ID]; !ok {
			plan.Unchanged = append(plan.Unchanged, category)
		}
	}

	return plan
}

func applyCategoryChange(
	existing *costingcontrol.CostCategory,
	change *CategoryChange,
) *costingcontrol.CostCategory {
	return &costingcontrol.CostCategory{
		ID:                   existing.ID,
		BusinessUnitID:       existing.BusinessUnitID,
		OrganizationID:       existing.OrganizationID,
		CostingControlID:     existing.CostingControlID,
		Category:             existing.Category,
		Name:                 existing.Name,
		CostBehavior:         existing.CostBehavior,
		RateSource:           change.RateSource,
		BenchmarkRatePerMile: existing.BenchmarkRatePerMile,
		OverrideRatePerMile:  change.OverrideRatePerMile,
		IsActive:             change.IsActive,
		SortOrder:            existing.SortOrder,
		CreatedAt:            existing.CreatedAt,
		UpdatedAt:            existing.UpdatedAt,
		Version:              change.Version,
	}
}

func (s *Service) writePlan(ctx context.Context, req *SaveRequest, plan *SavePlan) error {
	entries := make([]auditports.BulkLogEntry, 0, len(plan.Categories)+1)

	if plan.ControlChanged {
		updated, err := s.repo.Update(ctx, plan.Control)
		if err != nil {
			return err
		}
		before := controlAuditState(plan.Original)
		after := controlAuditState(updated)
		entries = append(entries, auditports.BulkLogEntry{
			Params: auditParams(req, updated.GetID(), after, before),
			Options: []auditports.LogOption{
				auditservice.WithComment("Costing control updated"),
				auditservice.WithDiff(before, after),
				auditservice.WithCritical(),
			},
		})
	}

	if len(plan.Categories) > 0 {
		categories := make([]*costingcontrol.CostCategory, 0, len(plan.Categories))
		links := make([]costingcontrolports.CategoryGLAccounts, 0, len(plan.Categories))
		for _, planned := range plan.Categories {
			categories = append(categories, planned.Category)
			links = append(links, costingcontrolports.CategoryGLAccounts{
				CostCategoryID: planned.Category.ID,
				GLAccountIDs:   planned.GLAccountIDs,
			})
		}

		if err := s.repo.UpdateCategories(ctx, categories); err != nil {
			return err
		}
		if err := s.repo.ReplaceCategoryGLAccounts(
			ctx,
			&costingcontrolports.ReplaceCategoryGLAccountsRequest{
				TenantInfo: req.TenantInfo,
				Categories: links,
			},
		); err != nil {
			return err
		}

		for _, planned := range plan.Categories {
			before := categoryAuditState(planned.Original, glAccountIDsOf(planned.Original))
			after := categoryAuditState(planned.Category, planned.GLAccountIDs)
			entries = append(entries, auditports.BulkLogEntry{
				Params: auditParams(req, planned.Category.GetID(), after, before),
				Options: []auditports.LogOption{
					auditservice.WithComment("Cost category updated"),
					auditservice.WithDiff(before, after),
				},
			})
		}
	}

	return s.auditService.LogActions(ctx, entries)
}

func auditParams(
	req *SaveRequest,
	resourceID pulid.ID,
	current, previous map[string]any,
) *auditports.LogActionParams {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceCostingControl,
		ResourceID:     resourceID.String(),
		Operation:      permission.OpUpdate,
		CurrentState:   current,
		PreviousState:  previous,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}
	if req.Actor != nil {
		params.UserID = req.Actor.UserID
	}

	return params
}

func controlAuditState(control *costingcontrol.CostingControl) map[string]any {
	state := *control
	state.FuelIndex = nil
	state.Categories = nil

	return jsonutils.MustToJSON(&state)
}

func glAccountIDsOf(category *costingcontrol.CostCategory) []pulid.ID {
	ids := make([]pulid.ID, 0, len(category.GLAccounts))
	for _, link := range category.GLAccounts {
		if link != nil {
			ids = append(ids, link.GLAccountID)
		}
	}

	return ids
}

func categoryAuditState(
	category *costingcontrol.CostCategory,
	glAccountIDs []pulid.ID,
) map[string]any {
	state := *category
	state.GLAccounts = nil
	snapshot := jsonutils.MustToJSON(&state)

	ids := make([]string, 0, len(glAccountIDs))
	for _, id := range glAccountIDs {
		ids = append(ids, id.String())
	}
	snapshot["glAccountIds"] = ids

	return snapshot
}
