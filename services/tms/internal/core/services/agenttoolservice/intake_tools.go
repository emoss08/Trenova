package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// shipmentWriter is the slice of the shipment service the intake tools use.
type shipmentWriter interface {
	Get(ctx context.Context, req *repositories.GetShipmentByIDRequest) (*shipment.Shipment, error)
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *serviceports.RequestActor,
	) (*shipment.Shipment, error)
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *serviceports.RequestActor,
	) (*shipment.Shipment, error)
}

// importCompleter closes the import assistant's conversation for a document
// once the shipment it was about exists, the way the shipment form does.
type importCompleter interface {
	CompleteHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) error
}

// createShipmentTool enters a shipment: the customer, the service, the stops
// in order with their locations and windows, the freight. It takes the same
// shape the shipment form sends, so what an agent builds from a document is
// validated by exactly the rules a person's entry is.
type createShipmentTool struct {
	shipments shipmentWriter
	imports   importCompleter
	logger    *zap.Logger
}

func newCreateShipmentTool(
	shipments shipmentWriter,
	imports importCompleter,
	logger *zap.Logger,
) serviceports.AgentTool {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &createShipmentTool{shipments: shipments, imports: imports, logger: logger.Named("tool.create-shipment")}
}

func (t *createShipmentTool) Name() string { return "create_shipment" }

func (t *createShipmentTool) Description() string {
	return "Enter a new shipment. Give the customer, service type and shipment type by " +
		"id, the BOL or reference, the freight (pieces, weight, temperature range when " +
		"it matters), and one move with its stops in travel order: each stop has a " +
		"location id from list_locations, a type, and a scheduled window. Rate fields " +
		"are optional; a shipment with none is rated from the customer's agreements. " +
		"When the shipment comes from an uploaded document, pass sourceDocumentId so " +
		"the document is linked and its import conversation closed. The pro number is " +
		"assigned by the system."
}

func (t *createShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipment": map[string]any{
				"type":        "object",
				"description": "The shipment, in the shape the shipment form sends.",
				"properties": map[string]any{
					"customerId": map[string]any{
						"type":        "string",
						"description": "The customer, from list_customers.",
					},
					"serviceTypeId": map[string]any{
						"type":        "string",
						"description": "From list_service_types.",
					},
					"shipmentTypeId": map[string]any{
						"type":        "string",
						"description": "From list_shipment_types.",
					},
					"tractorTypeId": map[string]any{
						"type":        "string",
						"description": "A tractor equipment type, from list_equipment_types.",
					},
					"trailerTypeId": map[string]any{
						"type":        "string",
						"description": "A trailer equipment type, from list_equipment_types.",
					},
					"bol":            map[string]any{"type": "string", "description": "The customer's BOL or reference."},
					"pieces":         map[string]any{"type": "integer"},
					"weight":         map[string]any{"type": "integer", "description": "Pounds."},
					"temperatureMin": map[string]any{"type": "integer", "description": "Fahrenheit, for reefer freight."},
					"temperatureMax": map[string]any{"type": "integer", "description": "Fahrenheit, for reefer freight."},
					"ratingUnit":     map[string]any{"type": "integer", "description": "Defaults to 1."},
					"freightChargeAmount": map[string]any{
						"type":        "string",
						"description": "The agreed freight charge as a decimal string, only when the customer gave one.",
					},
					"moves": map[string]any{
						"type":        "array",
						"description": "Usually one move. Each has its stops in travel order.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"loaded":   map[string]any{"type": "boolean"},
								"sequence": map[string]any{"type": "integer"},
								"stops": map[string]any{
									"type": "array",
									"items": map[string]any{
										"type": "object",
										"properties": map[string]any{
											"locationId": map[string]any{"type": "string"},
											"type": map[string]any{
												"type": "string",
												"enum": []string{"Pickup", "Delivery", "SplitPickup", "SplitDelivery"},
											},
											"scheduleType": map[string]any{
												"type": "string",
												"enum": []string{"Open", "Appointment"},
											},
											"sequence":             map[string]any{"type": "integer"},
											"scheduledWindowStart": map[string]any{"type": "integer", "description": "Unix seconds."},
											"scheduledWindowEnd":   map[string]any{"type": "integer", "description": "Unix seconds."},
											"pieces":               map[string]any{"type": "integer"},
											"weight":               map[string]any{"type": "integer"},
											"addressLine":          map[string]any{"type": "string"},
										},
										"required":             []string{"locationId", "type", "sequence", "scheduledWindowStart"},
										"additionalProperties": false,
									},
								},
							},
							"required":             []string{"stops"},
							"additionalProperties": false,
						},
					},
				},
				"required":             []string{"customerId", "serviceTypeId", "moves"},
				"additionalProperties": true,
			},
			"sourceDocumentId": map[string]any{
				"type": "string",
				"description": "The uploaded document this shipment was read from, when there is " +
					"one: the documentId you read with get_shipment_draft, usually this run's " +
					"subject or the page.",
			},
		},
		"required":             []string{"shipment"},
		"additionalProperties": false,
	}
}

func (t *createShipmentTool) Reversible() bool { return false }

func (t *createShipmentTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

func (t *createShipmentTool) PermissionOperation() permission.Operation {
	return permission.OpCreate
}

// RequiresIdempotencyKey is true because a shipment created twice is two
// loads on the board and two invoices, and the runtime retries writes.
func (t *createShipmentTool) RequiresIdempotencyKey() bool { return true }

func (t *createShipmentTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

// TierLimit holds a new load at a decision whatever the agent has earned. A
// shipment commits a customer's freight and the money that follows it, and the
// organization decided no desk creates one unattended — the mailbox that lets
// the intake desk answer a status question on its own does not let it book.
func (t *createShipmentTool) TierLimit(
	context.Context,
	serviceports.ToolExecuteParams,
) agent.AutonomyTier {
	return agent.TierActWithApproval
}

func (t *createShipmentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	entity := new(shipment.Shipment)
	if err := decodeParam(params.Params, "shipment", entity); err != nil {
		return err
	}

	// The tenant is the actor's, whatever the model wrote; a model cannot
	// enter a shipment for another organization by naming it.
	tenantInfo := tenantFrom(params)
	entity.ID = pulid.Nil
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	entity.EnteredByID = params.Actor.UserIDOrNil()
	entity.Status = shipment.StatusNew
	scopeShipmentChildren(entity, tenantInfo)

	if sourceDocumentID := optionalString(params.Params, "sourceDocumentId"); sourceDocumentID != "" {
		if _, err := pulid.Parse(sourceDocumentID); err != nil {
			return errortypes.NewValidationError(
				"sourceDocumentId",
				errortypes.ErrInvalid,
				"Source document id is not an id",
			)
		}
		entity.SourceDocumentID = sourceDocumentID
	}

	created, err := t.shipments.Create(ctx, entity, params.Actor)
	if err != nil {
		return err
	}

	if entity.SourceDocumentID != "" && t.imports != nil {
		if completeErr := t.imports.CompleteHistory(ctx, entity.SourceDocumentID, tenantInfo); completeErr != nil {
			t.logger.Warn("shipment created but the import conversation could not be closed",
				zap.String("sourceDocumentId", entity.SourceDocumentID),
				zap.String("shipmentId", created.ID.String()),
				zap.Error(completeErr),
			)
		}
	}

	return nil
}

// scopeShipmentChildren stamps the tenant on every move and stop and clears
// any id the model invented, so children are created under the shipment
// rather than pointed at rows that may belong to someone else.
func scopeShipmentChildren(entity *shipment.Shipment, tenantInfo pagination.TenantInfo) {
	for _, move := range entity.Moves {
		if move == nil {
			continue
		}
		move.ID = pulid.Nil
		move.ShipmentID = pulid.Nil
		move.OrganizationID = tenantInfo.OrgID
		move.BusinessUnitID = tenantInfo.BuID
		if move.Status == "" {
			move.Status = shipment.MoveStatusNew
		}
		for _, stop := range move.Stops {
			if stop == nil {
				continue
			}
			stop.ID = pulid.Nil
			stop.ShipmentMoveID = pulid.Nil
			stop.OrganizationID = tenantInfo.OrgID
			stop.BusinessUnitID = tenantInfo.BuID
			if stop.Status == "" {
				stop.Status = shipment.StopStatusNew
			}
			if stop.ScheduleType == "" {
				stop.ScheduleType = shipment.StopScheduleTypeOpen
			}
		}
	}
}

// updateShipmentTool changes the details of a saved shipment that a person
// would change on the form: who it is for, what service, what freight. It
// deliberately cannot touch stops, moves, rates or status: those have their
// own tools and their own rules, and a patch that could reach them would be
// a way around both.
type updateShipmentTool struct {
	shipments shipmentWriter
}

func newUpdateShipmentTool(shipments shipmentWriter) serviceports.AgentTool {
	return &updateShipmentTool{shipments: shipments}
}

func (t *updateShipmentTool) Name() string { return "update_shipment" }

func (t *updateShipmentTool) Description() string {
	return "Change the details of a saved shipment: the customer, service type, shipment " +
		"type, equipment types, BOL, pieces, weight or temperature range. Send only " +
		"the fields to change. Stops, moves, rates and status are not changed here; " +
		"use the move, hold and cancel tools for those."
}

func (t *updateShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": map[string]any{
				"type": "string",
				"description": "The shipment to change, from the page, list_shipments or " +
					"search_shipments.",
			},
			"customerId": map[string]any{
				"type":        "string",
				"description": "The new customer, from list_customers.",
			},
			"serviceTypeId": map[string]any{
				"type":        "string",
				"description": "The new service type, from list_service_types.",
			},
			"shipmentTypeId": map[string]any{
				"type":        "string",
				"description": "The new shipment type, from list_shipment_types.",
			},
			"tractorTypeId": map[string]any{
				"type":        "string",
				"description": "The new tractor equipment type, from list_equipment_types.",
			},
			"trailerTypeId": map[string]any{
				"type":        "string",
				"description": "The new trailer equipment type, from list_equipment_types.",
			},
			"bol": map[string]any{
				"type":        "string",
				"description": "The customer's BOL or reference number.",
			},
			"pieces": map[string]any{
				"type":        "integer",
				"description": "The total piece or handling-unit count.",
			},
			"weight":         map[string]any{"type": "integer", "description": "Pounds."},
			"temperatureMin": map[string]any{"type": "integer", "description": "Fahrenheit."},
			"temperatureMax": map[string]any{"type": "integer", "description": "Fahrenheit."},
		},
		"required":             []string{"shipmentId"},
		"additionalProperties": false,
	}
}

func (t *updateShipmentTool) Reversible() bool { return true }

func (t *updateShipmentTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

func (t *updateShipmentTool) PermissionOperation() permission.Operation {
	return permission.OpUpdate
}

func (t *updateShipmentTool) RequiresIdempotencyKey() bool { return false }

func (t *updateShipmentTool) DefaultAutonomyTier() agent.AutonomyTier {
	return agent.TierActWithApproval
}

// shipmentPatch is what update_shipment may change. Pointers say "absent
// leaves it alone"; the allowlist is the struct itself.
type shipmentPatch struct {
	CustomerID     *string `json:"customerId"`
	ServiceTypeID  *string `json:"serviceTypeId"`
	ShipmentTypeID *string `json:"shipmentTypeId"`
	TractorTypeID  *string `json:"tractorTypeId"`
	TrailerTypeID  *string `json:"trailerTypeId"`
	BOL            *string `json:"bol"`
	Pieces         *int64  `json:"pieces"`
	Weight         *int64  `json:"weight"`
	TemperatureMin *int16  `json:"temperatureMin"`
	TemperatureMax *int16  `json:"temperatureMax"`
}

func (p shipmentPatch) empty() bool {
	return p.CustomerID == nil && p.ServiceTypeID == nil && p.ShipmentTypeID == nil &&
		p.TractorTypeID == nil && p.TrailerTypeID == nil && p.BOL == nil &&
		p.Pieces == nil && p.Weight == nil && p.TemperatureMin == nil && p.TemperatureMax == nil
}

func (t *updateShipmentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	if err := guardExecute(t, params); err != nil {
		return err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return err
	}

	var patch shipmentPatch
	if err = jsonutils.Convert(params.Params, &patch); err != nil {
		return fmt.Errorf("update_shipment parameters: %w", err)
	}
	if patch.empty() {
		return errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Nothing to change: send at least one field besides the shipment id",
		)
	}

	tenantInfo := tenantFrom(params)
	entity, err := t.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantInfo,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return err
	}

	if err = applyShipmentPatch(entity, patch); err != nil {
		return err
	}

	_, err = t.shipments.Update(ctx, entity, params.Actor)

	return err
}

func applyShipmentPatch(entity *shipment.Shipment, patch shipmentPatch) error {
	ids := []struct {
		name  string
		value *string
		dst   *pulid.ID
	}{
		{"customerId", patch.CustomerID, &entity.CustomerID},
		{"serviceTypeId", patch.ServiceTypeID, &entity.ServiceTypeID},
		{"shipmentTypeId", patch.ShipmentTypeID, &entity.ShipmentTypeID},
		{"tractorTypeId", patch.TractorTypeID, &entity.TractorTypeID},
		{"trailerTypeId", patch.TrailerTypeID, &entity.TrailerTypeID},
	}
	for _, field := range ids {
		if field.value == nil {
			continue
		}
		id, err := pulid.Parse(*field.value)
		if err != nil {
			return fmt.Errorf("parameter %q is not an id", field.name)
		}
		*field.dst = id
	}

	if patch.BOL != nil {
		entity.BOL = *patch.BOL
	}
	if patch.Pieces != nil {
		entity.Pieces = patch.Pieces
	}
	if patch.Weight != nil {
		entity.Weight = patch.Weight
	}
	if patch.TemperatureMin != nil {
		entity.TemperatureMin = patch.TemperatureMin
	}
	if patch.TemperatureMax != nil {
		entity.TemperatureMax = patch.TemperatureMax
	}

	return nil
}

func (t *updateShipmentTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}
