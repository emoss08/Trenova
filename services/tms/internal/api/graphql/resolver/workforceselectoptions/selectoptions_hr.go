package workforceselectoptions

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/selectoption"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpolicy"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerschedule"
	"github.com/emoss08/trenova/shared/stringutils"
)

func shiftTemplateSelectOptionItem(
	entity *workerschedule.ShiftTemplate,
) selectoption.Item {
	return selectoption.ItemFor(
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

func workerPolicySelectOptionItem(entity *workerpolicy.WorkerPolicy) selectoption.Item {
	return selectoption.ItemFor(
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

func jobPositionSelectOptionItem(entity *worker.JobPosition) selectoption.Item {
	return selectoption.ItemFor(
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
