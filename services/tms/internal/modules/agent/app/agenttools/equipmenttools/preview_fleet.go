package equipmenttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type equipmentKind[T any] struct {
	resource permission.Resource
	noun     string
	id       func(*T) pulid.ID
	code     func(*T) string
	version  func(*T) int64
	status   func(*T) *domaintypes.EquipmentStatus
}

var tractorKind = equipmentKind[tractor.Tractor]{
	resource: permission.ResourceTractor,
	noun:     "tractor",
	id:       func(unit *tractor.Tractor) pulid.ID { return unit.ID },
	code:     func(unit *tractor.Tractor) string { return unit.Code },
	version:  func(unit *tractor.Tractor) int64 { return unit.Version },
	status:   func(unit *tractor.Tractor) *domaintypes.EquipmentStatus { return &unit.Status },
}

var trailerKind = equipmentKind[trailer.Trailer]{
	resource: permission.ResourceTrailer,
	noun:     "trailer",
	id:       func(unit *trailer.Trailer) pulid.ID { return unit.ID },
	code:     func(unit *trailer.Trailer) string { return unit.Code },
	version:  func(unit *trailer.Trailer) int64 { return unit.Version },
	status:   func(unit *trailer.Trailer) *domaintypes.EquipmentStatus { return &unit.Status },
}

func (k equipmentKind[T]) setStatus(status domaintypes.EquipmentStatus) func(*T) error {
	return func(unit *T) error {
		*k.status(unit) = status

		return nil
	}
}

func (k equipmentKind[T]) PlanPreview(
	ids []pulid.ID,
	status domaintypes.EquipmentStatus,
	units []*T,
) (*agentpreview.ToolPreview, error) {
	byID := make(map[pulid.ID]*T, len(units))
	for _, unit := range units {
		if unit != nil {
			byID[k.id(unit)] = unit
		}
	}

	mutate := k.setStatus(status)
	changes := make([]*agentpreview.RecordChange, 0, len(ids))
	missing := make([]string, 0)
	for _, id := range ids {
		unit, found := byID[id]
		if !found {
			missing = append(missing, id.String())
			continue
		}
		change, err := toolpreview.Update(toolpreview.Record{
			Resource: k.resource,
			ID:       k.id(unit),
			Label:    k.code(unit),
			Version:  changepreview.PreviewVersion(k.version(unit)),
		}, unit, mutate, toolpreview.Only("status"))
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would set %s to %s.", changepreview.CountOf(len(ids), k.noun), status),
		changes...,
	)
	if len(missing) == 0 {
		return preview, nil
	}

	return toolpreview.Warn(
		preview,
		agentpreview.PreviewWarningWouldFail,
		fmt.Sprintf(
			"This would be refused as it stands: %s matches nothing in this organization.",
			changepreview.CountOf(len(missing), k.noun+" id"),
		),
		missing...,
	), nil
}

func (t *UpdateTractorStatusTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "tractorIds")
	if err != nil {
		return nil, err
	}

	units, err := t.tractors.GetByIDs(ctx, tractorports.GetTractorsByIDsRequest{
		TenantInfo: toolparams.TenantFrom(params),
		TractorIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	return tractorKind.PlanPreview(ids, status, units)
}

func (t *UpdateTrailerStatusTool) Preview(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*agentpreview.ToolPreview, error) {
	if err := toolguard.GuardPreview(t, params); err != nil {
		return nil, err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "trailerIds")
	if err != nil {
		return nil, err
	}

	units, err := t.trailers.GetByIDs(ctx, trailerports.GetTrailersByIDsRequest{
		TenantInfo: toolparams.TenantFrom(params),
		TrailerIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	return trailerKind.PlanPreview(ids, status, units)
}
