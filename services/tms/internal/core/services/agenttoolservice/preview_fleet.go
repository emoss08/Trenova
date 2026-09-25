package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

var (
	_ serviceports.ToolPreviewer = (*updateTractorStatusTool)(nil)
	_ serviceports.ToolPreviewer = (*updateTrailerStatusTool)(nil)
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

func (k equipmentKind[T]) preview(
	ids []pulid.ID,
	status domaintypes.EquipmentStatus,
	units []*T,
) (*agent.ToolPreview, error) {
	byID := make(map[pulid.ID]*T, len(units))
	for _, unit := range units {
		if unit != nil {
			byID[k.id(unit)] = unit
		}
	}

	mutate := k.setStatus(status)
	changes := make([]*agent.RecordChange, 0, len(ids))
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
			Version:  previewVersion(k.version(unit)),
		}, unit, mutate, toolpreview.Only("status"))
		if err != nil {
			return nil, err
		}
		changes = append(changes, change)
	}

	preview := toolpreview.Build(
		fmt.Sprintf("Would set %s to %s.", countOf(len(ids), k.noun), status),
		changes...,
	)
	if len(missing) == 0 {
		return preview, nil
	}

	return toolpreview.Warn(
		preview,
		agent.PreviewWarningWouldFail,
		fmt.Sprintf(
			"This would be refused as it stands: %s matches nothing in this organization.",
			countOf(len(missing), k.noun+" id"),
		),
		missing...,
	), nil
}

func (t *updateTractorStatusTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "tractorIds")
	if err != nil {
		return nil, err
	}

	units, err := t.tractors.GetByIDs(ctx, repositories.GetTractorsByIDsRequest{
		TenantInfo: tenantFrom(params),
		TractorIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	return tractorKind.preview(ids, status, units)
}

func (t *updateTrailerStatusTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "trailerIds")
	if err != nil {
		return nil, err
	}

	units, err := t.trailers.GetByIDs(ctx, repositories.GetTrailersByIDsRequest{
		TenantInfo: tenantFrom(params),
		TrailerIDs: ids,
	})
	if err != nil {
		return nil, err
	}

	return trailerKind.preview(ids, status, units)
}
