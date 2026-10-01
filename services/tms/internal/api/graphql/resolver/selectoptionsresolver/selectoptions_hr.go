package selectoptionsresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (r *Deps) resolveShiftTemplateSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.SchedulingService.GetTemplate(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, shiftTemplateSelectOptionItem(entity))
		}
		return selectOptionConnection(items, len(items), 0)
	}

	templates, err := r.SchedulingService.ListTemplates(
		ctx,
		&repositories.ListShiftTemplatesRequest{TenantInfo: req.TenantInfo, ActiveOnly: true},
	)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))
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
			Description: stringutils.Ptr(entity.Code),
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

func (r *Deps) resolveWorkerPolicySelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.SelfServiceService.GetPolicy(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, workerPolicySelectOptionItem(entity))
		}
		return selectOptionConnection(items, len(items), 0)
	}

	policies, err := r.SelfServiceService.ListPolicies(
		ctx,
		&repositories.ListWorkerPoliciesRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}

	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))
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
			Description: stringutils.Ptr(entity.Code),
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
func (r *Deps) resolveJobPositionSelectOptions(
	ctx context.Context,
	req *SelectOptionsRequest,
) (*gqlmodel.SelectOptionConnection, error) {
	if len(req.IDs) > 0 {
		items := make([]selectOptionConnectionItem, 0, len(req.IDs))
		for _, id := range req.IDs {
			entity, err := r.OrgStructureService.GetPosition(ctx, req.TenantInfo, id)
			if err != nil {
				return nil, err
			}
			items = append(items, jobPositionSelectOptionItem(entity))
		}
		return selectOptionConnection(items, len(items), 0)
	}

	positions, err := r.OrgStructureService.ListPositions(
		ctx,
		&repositories.ListJobPositionsRequest{
			TenantInfo:  req.TenantInfo,
			ActiveOnly:  true,
			DrivingOnly: selectOptionBoolFilter(req.Filters, "drivingOnly"),
		},
	)
	if err != nil {
		return nil, err
	}

	nonDrivingOnly := selectOptionBoolFilter(req.Filters, "nonDrivingOnly")
	query := strings.ToLower(strings.TrimSpace(req.SelectQuery.Query))
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
			Description: stringutils.Ptr(entity.Code),
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
