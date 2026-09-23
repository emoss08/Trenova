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
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
)

// maxEquipmentPerStatusChange bounds how much of a fleet one call may move.
//
// The service takes a bulk request and would happily accept the whole yard. A
// chat turn is not where a hundred units change status at once, and a model
// that misreads "all the reefers" as every trailer on the property should hit a
// refusal rather than a completed write.
const maxEquipmentPerStatusChange = 25

const equipmentStatusNote = "Available means it can be dispatched. " +
	"OutOfService means it cannot, for any reason other than scheduled work. " +
	"AtMaintenance means it is in the shop. Sold means it has left the fleet " +
	"and is not coming back."

type tractorStatusUpdater interface {
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateTractorStatusRequest,
	) ([]*tractor.Tractor, error)
}

type trailerStatusUpdater interface {
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateTrailerStatusRequest,
	) ([]*trailer.Trailer, error)
}

type updateTractorStatusTool struct {
	tractors tractorStatusUpdater
}

func newUpdateTractorStatusTool(tractors tractorStatusUpdater) serviceports.AgentTool {
	return &updateTractorStatusTool{tractors: tractors}
}

func (t *updateTractorStatusTool) Name() string { return "update_tractor_status" }

func (t *updateTractorStatusTool) Description() string {
	return "Change the status of one or more tractors, which is what makes a truck " +
		"available to dispatch or takes it off the board. " + equipmentStatusNote +
		" Get the ids from list_tractors; a unit number is not an id."
}

func (t *updateTractorStatusTool) ParamSchema() map[string]any {
	return equipmentStatusSchema(
		"tractorIds",
		"The tractors to move, by id from list_tractors. One id is the normal case.",
	)
}

func (t *updateTractorStatusTool) Reversible() bool { return true }

func (t *updateTractorStatusTool) PermissionResource() permission.Resource {
	return permission.ResourceTractor
}

func (t *updateTractorStatusTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *updateTractorStatusTool) RequiresIdempotencyKey() bool { return false }

func (t *updateTractorStatusTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *updateTractorStatusTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "tractorIds")
	if err != nil {
		return err
	}

	_, err = t.tractors.BulkUpdateStatus(ctx, &repositories.BulkUpdateTractorStatusRequest{
		TenantInfo: tenantFrom(params),
		TractorIDs: ids,
		Status:     status,
	})

	return err
}

type updateTrailerStatusTool struct {
	trailers trailerStatusUpdater
}

func newUpdateTrailerStatusTool(trailers trailerStatusUpdater) serviceports.AgentTool {
	return &updateTrailerStatusTool{trailers: trailers}
}

func (t *updateTrailerStatusTool) Name() string { return "update_trailer_status" }

func (t *updateTrailerStatusTool) Description() string {
	return "Change the status of one or more trailers, which is what makes a trailer " +
		"available to dispatch or takes it off the board. " + equipmentStatusNote +
		" Get the ids from list_trailers; a unit number is not an id."
}

func (t *updateTrailerStatusTool) ParamSchema() map[string]any {
	return equipmentStatusSchema(
		"trailerIds",
		"The trailers to move, by id from list_trailers. One id is the normal case.",
	)
}

func (t *updateTrailerStatusTool) Reversible() bool { return true }

func (t *updateTrailerStatusTool) PermissionResource() permission.Resource {
	return permission.ResourceTrailer
}

func (t *updateTrailerStatusTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *updateTrailerStatusTool) RequiresIdempotencyKey() bool { return false }

func (t *updateTrailerStatusTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *updateTrailerStatusTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "trailerIds")
	if err != nil {
		return err
	}

	_, err = t.trailers.BulkUpdateStatus(ctx, &repositories.BulkUpdateTrailerStatusRequest{
		TenantInfo: tenantFrom(params),
		TrailerIDs: ids,
		Status:     status,
	})

	return err
}

func equipmentStatusSchema(idsKey, idsDescription string) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			idsKey: map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"minItems":    1,
				"maxItems":    maxEquipmentPerStatusChange,
				"description": idsDescription,
			},
			"status": map[string]any{
				"type": "string",
				"enum": []string{
					string(domaintypes.EquipmentStatusAvailable),
					string(domaintypes.EquipmentStatusOOS),
					string(domaintypes.EquipmentStatusAtMaintenance),
					string(domaintypes.EquipmentStatusSold),
				},
				"description": "The status to set. " + equipmentStatusNote,
			},
		},
		"required":             []string{idsKey, "status"},
		"additionalProperties": false,
	}
}

func equipmentStatusArgs(
	params map[string]any,
	idsKey string,
) ([]pulid.ID, domaintypes.EquipmentStatus, error) {
	ids, err := requirePulidSlice(params, idsKey, maxEquipmentPerStatusChange)
	if err != nil {
		return nil, "", err
	}

	raw := optionalString(params, "status")
	status, err := domaintypes.EquipmentStatusFromString(raw)
	if err != nil {
		return nil, "", fmt.Errorf(
			"parameter %q must be exactly one of Available, OutOfService, AtMaintenance "+
				"or Sold, not %q", "status", raw,
		)
	}

	return ids, status, nil
}
