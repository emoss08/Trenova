package resolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
)

// The HR pickers list in memory and filter here. A carrier has a handful of
// shifts and a handful of policies; a search index for either would be more
// machinery than the whole feature.

func (r *Resolver) resolveShiftTemplateSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.schedulingService.GetTemplate(ctx, req.tenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, shiftTemplateSelectOptionItem(entity))
		}
		return selectOptionConnection(items, len(items), 0)
	}

	templates, err := r.schedulingService.ListTemplates(
		ctx,
		&repositories.ListShiftTemplatesRequest{TenantInfo: req.tenantInfo, ActiveOnly: true},
	)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.selectQuery.Query))
	items := make([]selectOptionConnectionItem, 0, len(templates))
	for _, template := range templates {
		if query != "" &&
			!strings.Contains(strings.ToLower(template.Code), query) &&
			!strings.Contains(strings.ToLower(template.Name), query) {
			continue
		}
		items = append(items, shiftTemplateSelectOptionItem(template))
	}

	return selectOptionConnection(items, len(items), 0)
}

func shiftTemplateSelectOptionItem(entity *worker.ShiftTemplate) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Name,
			Description: stringPtr(entity.Code),
			Meta: map[string]any{
				"code":            entity.Code,
				"color":           entity.Color,
				"daysOfWeek":      entity.DaysOfWeek,
				"startMinute":     entity.StartMinute,
				"durationMinutes": entity.DurationMinutes,
				"cycleWeeks":      entity.CycleWeeks,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

func (r *Resolver) resolveWorkerPolicySelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.selfServiceService.GetPolicy(ctx, req.tenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, workerPolicySelectOptionItem(entity))
		}
		return selectOptionConnection(items, len(items), 0)
	}

	policies, err := r.selfServiceService.ListPolicies(
		ctx,
		&repositories.ListWorkerPoliciesRequest{TenantInfo: req.tenantInfo},
	)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.selectQuery.Query))
	items := make([]selectOptionConnectionItem, 0, len(policies))
	for _, policy := range policies {
		if policy.Status != domaintypes.StatusActive {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(policy.Code), query) &&
			!strings.Contains(strings.ToLower(policy.Title), query) {
			continue
		}
		items = append(items, workerPolicySelectOptionItem(policy))
	}

	return selectOptionConnection(items, len(items), 0)
}

func workerPolicySelectOptionItem(entity *worker.WorkerPolicy) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Title,
			Description: stringPtr(entity.Code),
			Meta: map[string]any{
				"code":         entity.Code,
				"versionLabel": entity.VersionLabel,
				"appliesTo":    string(entity.AppliesTo),
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}

// resolveJobPositionSelectOptions offers the half of the chart the picker is
// filling: driving titles for a worker's record, front-office titles for a
// user's membership. Offering both would put somebody on the wrong roster.
func (r *Resolver) resolveJobPositionSelectOptions(
	ctx context.Context,
	req selectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.ids) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.ids))
		for _, id := range req.ids {
			entity, err := r.orgStructureService.GetPosition(ctx, req.tenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, jobPositionSelectOptionItem(entity))
		}
		return selectOptionConnection(items, len(items), 0)
	}

	positions, err := r.orgStructureService.ListPositions(
		ctx,
		&repositories.ListJobPositionsRequest{
			TenantInfo:  req.tenantInfo,
			ActiveOnly:  true,
			DrivingOnly: selectOptionBoolFilter(req.filters, "drivingOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	nonDrivingOnly := selectOptionBoolFilter(req.filters, "nonDrivingOnly")
	query := strings.ToLower(strings.TrimSpace(req.selectQuery.Query))
	items := make([]selectOptionConnectionItem, 0, len(positions))
	for _, position := range positions {
		if nonDrivingOnly && position.IsDrivingPosition {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(position.Code), query) &&
			!strings.Contains(strings.ToLower(position.Title), query) {
			continue
		}
		items = append(items, jobPositionSelectOptionItem(position))
	}

	return selectOptionConnection(items, len(items), 0)
}

func jobPositionSelectOptionItem(entity *worker.JobPosition) selectOptionConnectionItem {
	return selectOptionConnectionItemFor(
		&gqlmodel.SelectOption{
			ID:          entity.ID.String(),
			Label:       entity.Title,
			Description: stringPtr(entity.Code),
			Meta: map[string]any{
				"code":              entity.Code,
				"department":        string(entity.Department),
				"isDrivingPosition": entity.IsDrivingPosition,
			},
		},
		entity.CreatedAt,
		entity.ID,
	)
}
