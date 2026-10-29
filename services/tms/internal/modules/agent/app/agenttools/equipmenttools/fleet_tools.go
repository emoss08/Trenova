package equipmenttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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

var equipmentStatuses = agenttoolschema.Source("equipment.status", []domaintypes.EquipmentStatus{
	domaintypes.EquipmentStatusAvailable,
	domaintypes.EquipmentStatusOOS,
	domaintypes.EquipmentStatusAtMaintenance,
	domaintypes.EquipmentStatusSold,
})

const equipmentStatusNote = "Available means it can be dispatched. " +
	"OutOfService means it cannot, for any reason other than scheduled work. " +
	"AtMaintenance means it is in the shop. Sold means it has left the fleet " +
	"and is not coming back."

type tractorStatusUpdater interface {
	GetByIDs(
		ctx context.Context,
		req tractorports.GetTractorsByIDsRequest,
	) ([]*tractor.Tractor, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *tractorports.BulkUpdateTractorStatusRequest,
	) ([]*tractor.Tractor, error)
}

type trailerStatusUpdater interface {
	GetByIDs(
		ctx context.Context,
		req trailerports.GetTrailersByIDsRequest,
	) ([]*trailer.Trailer, error)
	BulkUpdateStatus(
		ctx context.Context,
		req *trailerports.BulkUpdateTrailerStatusRequest,
	) ([]*trailer.Trailer, error)
}

type UpdateTractorStatusTool struct {
	tractors tractorStatusUpdater
}

func NewUpdateTractorStatusTool(tractors tractorStatusUpdater) agenttoolports.AgentTool {
	return &UpdateTractorStatusTool{tractors: tractors}
}

func (t *UpdateTractorStatusTool) Name() string { return "equipment_set_tractor_status" }

func (t *UpdateTractorStatusTool) SearchTerms() []string {
	return []string{"out of service", "maintenance", "available"}
}

func (t *UpdateTractorStatusTool) Prerequisites() []string {
	return []string{tractorRecord.Lister}
}

func (t *UpdateTractorStatusTool) Description() string {
	return "Change the status of one or more tractors, which is what makes a truck " +
		"available to dispatch or takes it off the board. " + equipmentStatusNote +
		" Get the ids from equipment_list_tractors; a unit number is not an id."
}

func (t *UpdateTractorStatusTool) ParamSchema() map[string]any {
	return equipmentStatusSchema(
		"tractorIds",
		"The tractors to move, by id from equipment_list_tractors. One id is the normal case.",
		permission.ResourceTractor,
	)
}

func (t *UpdateTractorStatusTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceTractor,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Changes a tractor's status inside Trenova.",
	}
}

func (t *UpdateTractorStatusTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "tractorIds")
	if err != nil {
		return err
	}

	_, err = t.tractors.BulkUpdateStatus(ctx, &tractorports.BulkUpdateTractorStatusRequest{
		TenantInfo: toolparams.TenantFrom(params),
		TractorIDs: ids,
		Status:     status,
	})

	return err
}

type UpdateTrailerStatusTool struct {
	trailers trailerStatusUpdater
}

func NewUpdateTrailerStatusTool(trailers trailerStatusUpdater) agenttoolports.AgentTool {
	return &UpdateTrailerStatusTool{trailers: trailers}
}

func (t *UpdateTrailerStatusTool) Name() string { return "equipment_set_trailer_status" }

func (t *UpdateTrailerStatusTool) SearchTerms() []string {
	return []string{"out of service", "maintenance", "available"}
}

func (t *UpdateTrailerStatusTool) Prerequisites() []string {
	return []string{trailerRecord.Lister}
}

func (t *UpdateTrailerStatusTool) Description() string {
	return "Change the status of one or more trailers, which is what makes a trailer " +
		"available to dispatch or takes it off the board. " + equipmentStatusNote +
		" Get the ids from equipment_list_trailers; a unit number is not an id."
}

func (t *UpdateTrailerStatusTool) ParamSchema() map[string]any {
	return equipmentStatusSchema(
		"trailerIds",
		"The trailers to move, by id from equipment_list_trailers. One id is the normal case.",
		permission.ResourceTrailer,
	)
}

func (t *UpdateTrailerStatusTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agenttooltrust.ToolKindAction,
		Resource:      permission.ResourceTrailer,
		Operation:     permission.OpUpdate,
		Scope:         agenttooltrust.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierAutoExecute,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agenttooltrust.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agenttooltrust.ExternalReadNever,
		Rationale:     "Changes a trailer's status inside Trenova.",
	}
}

func (t *UpdateTrailerStatusTool) Execute(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) error {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return err
	}

	ids, status, err := equipmentStatusArgs(params.Params, "trailerIds")
	if err != nil {
		return err
	}

	_, err = t.trailers.BulkUpdateStatus(ctx, &trailerports.BulkUpdateTrailerStatusRequest{
		TenantInfo: toolparams.TenantFrom(params),
		TrailerIDs: ids,
		Status:     status,
	})

	return err
}

func equipmentStatusSchema(
	idsKey, idsDescription string,
	resource permission.Resource,
) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			idsKey: agenttoolschema.OfResource(map[string]any{
				"type":                 "array",
				toolparams.SchemaItems: map[string]any{"type": "string"},
				"minItems":             1,
				"maxItems":             maxEquipmentPerStatusChange,
				"description":          idsDescription,
			}, resource),
			changepreview.FieldStatus: agenttoolschema.Enum(
				"The status to set. "+equipmentStatusNote, equipmentStatuses,
			),
		},
		"required":             []string{idsKey, "status"},
		"additionalProperties": false,
	}
}

func equipmentStatusArgs(
	params map[string]any,
	idsKey string,
) ([]pulid.ID, domaintypes.EquipmentStatus, error) {
	ids, err := toolparams.RequirePulidSlice(params, idsKey, maxEquipmentPerStatusChange)
	if err != nil {
		return nil, "", err
	}

	raw := toolparams.OptionalString(params, "status")
	status, err := domaintypes.EquipmentStatusFromString(raw)
	if err != nil {
		return nil, "", fmt.Errorf(
			"parameter %q must be exactly one of Available, OutOfService, AtMaintenance "+
				"or Sold, not %q", "status", raw,
		)
	}

	return ids, status, nil
}
