package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
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

type shipmentCreator interface {
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *serviceports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewCreate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *serviceports.RequestActor,
	) (*serviceports.ShipmentCreatePlan, error)
}

// importCompleter closes the import assistant's conversation for a document
// once the shipment it was about exists, the way the shipment form does.
type importCompleter interface {
	CompleteHistory(ctx context.Context, documentID string, tenantInfo pagination.TenantInfo) error
}

// createShipmentTool enters a shipment: the customer, the service, the stops
// in order with their locations and windows, the freight and how it is rated.
// It reads an allow-listed draft rather than the record itself, so a model
// cannot set what the form does not let a person set, and what it builds is
// validated by exactly the rules a person's entry is.
type createShipmentTool struct {
	shipments shipmentCreator
	imports   importCompleter
	locations locationZoneReader
	logger    *zap.Logger
}

type createShipmentDeps struct {
	Shipments shipmentCreator
	Imports   importCompleter
	Locations locationZoneReader
	Logger    *zap.Logger
}

func newCreateShipmentTool(deps createShipmentDeps) serviceports.AgentTool {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &createShipmentTool{
		shipments: deps.Shipments,
		imports:   deps.Imports,
		locations: deps.Locations,
		logger:    logger.Named("tool.create-shipment"),
	}
}

func (t *createShipmentTool) Name() string { return "create_shipment" }

func (t *createShipmentTool) Description() string {
	return "Enter a new shipment. To copy an existing shipment use duplicate_shipment instead, " +
		"which carries its stops, commodities, charges and rating exactly. Give the customer, " +
		"service type, shipment type and rating method (formulaTemplateId, from " +
		"list_formula_templates) by id: the rating method is required, and a shipment nothing " +
		"can price is refused. Add the freight (pieces, weight, commodities, temperature range " +
		"when it matters), any accessorial charges, and one move with its stops in travel order: " +
		"each stop has a location id from list_locations, a type, and a scheduled window in " +
		"local time at the stop, such as 2026-10-01T08:00. Stops and moves count from 0. When the " +
		"shipment comes from an uploaded document, pass sourceDocumentId so the document is " +
		"linked and its import conversation closed. The pro number is assigned by the system. A " +
		"BOL must be unique among open shipments, so never reuse one; ask the person for it."
}

func (t *createShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipment": shipmentDraftSchema(),
			"sourceDocumentId": idProperty("The uploaded document this shipment was read from, " +
				"when there is one: the documentId you read with get_shipment_draft, usually " +
				"this run's subject or the page."),
		},
		toolschema.KeyRequired:             []string{"shipment"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func shipmentDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeObject,
		toolschema.KeyDescription: "The shipment to enter.",
		toolschema.KeyProperties: map[string]any{
			"customerId": idProperty("The customer, from list_customers."),
			"billToCustomerId": idProperty("Who is billed, when not the customer, from " +
				"list_customers."),
			"serviceTypeId":  idProperty("From list_service_types."),
			"shipmentTypeId": idProperty("From list_shipment_types."),
			"formulaTemplateId": idProperty("The rating method that prices the freight, from " +
				"list_formula_templates. Required: a rate agreement covering the lane may " +
				"replace it with its own when the shipment is saved."),
			"baseRate": amountProperty("The rate the rating method multiplies, as a decimal " +
				"such as 2.45, only when the customer agreed one."),
			"freightTerms": agenttoolschema.Enum(
				"Who pays the freight. Defaults to Prepaid.",
				agenttoolschema.FreightTerms,
			),
			"tractorTypeId": idProperty("A tractor equipment type, from list_equipment_types."),
			"trailerTypeId": idProperty("A trailer equipment type, from list_equipment_types."),
			"bol": stringProperty("The customer's BOL or reference. It must be unique among "+
				"open shipments, so never reuse one from another shipment. Optional unless the "+
				"customer's billing requires a BOL; leave it out when none is known.", 100),
			"pieces":         integerProperty("The total piece or handling-unit count.", 0, 1_000_000),
			"weight":         integerProperty("Pounds.", 0, 10_000_000),
			"temperatureMin": integerProperty("Fahrenheit, for reefer freight.", -100, 200),
			"temperatureMax": integerProperty("Fahrenheit, for reefer freight.", -100, 200),
			"ratingUnit": integerProperty("How many units the rating method prices. "+
				"Defaults to 1.", 1, 1_000_000),
			"moves": map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyDescription: "Usually one move, sequence 0. Each has its stops in " +
					"travel order.",
				toolschema.KeyMinItems: 1,
				toolschema.KeyItems:    moveDraftSchema(),
			},
			"commodities": map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "What is hauled, one line per commodity.",
				toolschema.KeyItems:       commodityDraftSchema(),
			},
			"additionalCharges": map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Accessorial charges agreed for the shipment.",
				toolschema.KeyItems:       chargeLineDraftSchema(),
			},
		},
		toolschema.KeyRequired: []string{
			"customerId", "serviceTypeId", "shipmentTypeId", "formulaTemplateId", "moves",
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func moveDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"loaded": booleanProperty("False for a deadhead move. Defaults to true."),
			"sequence": integerProperty("The move's place, counting from 0. Leave it out to "+
				"take the order given.", 0, 50),
			"stops": map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "The stops in travel order: a pickup first, a delivery last.",
				toolschema.KeyMinItems:    2,
				toolschema.KeyItems:       stopDraftSchema(),
			},
		},
		toolschema.KeyRequired:             []string{"stops"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func stopDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"locationId": idProperty("The stop's location, from list_locations."),
			"type": agenttoolschema.Enum(
				"What happens at the stop.",
				agenttoolschema.StopTypes,
			),
			"scheduleType": agenttoolschema.Enum(
				"Open for a window, Appointment for a fixed time. Defaults to Open.",
				agenttoolschema.StopScheduleTypes,
			),
			"sequence": integerProperty("The stop's place in the move, counting from 0. "+
				"Leave it out to take the order given.", 0, 100),
			"scheduledWindowStart": localTimeProperty("When the stop's window opens."),
			"scheduledWindowEnd":   localTimeProperty("When the stop's window closes, if it has one."),
			"pieces":               integerProperty("Pieces handled at the stop.", 0, 1_000_000),
			"weight":               integerProperty("Pounds handled at the stop.", 0, 10_000_000),
			"addressLine":          stringProperty("A dock or suite note for the stop.", 200),
		},
		toolschema.KeyRequired:             []string{"locationId", "type", "scheduledWindowStart"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func commodityDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"commodityId": idProperty("The commodity, from list_commodities."),
			"pieces":      integerProperty("Pieces of it. Defaults to 1.", 0, 1_000_000),
			"weight":      integerProperty("Pounds of it.", 0, 10_000_000),
		},
		toolschema.KeyRequired:             []string{"commodityId"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func chargeLineDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"accessorialChargeId": idProperty("The accessorial, from list_accessorial_charges."),
			"method": agenttoolschema.Enum(
				"How the amount is applied: Flat once, PerUnit times the unit, Percentage of "+
					"the linehaul.",
				agenttoolschema.AccessorialMethods,
			),
			"unit": integerProperty("How many units the charge covers; 1 for a flat charge.", 1,
				10_000),
			"amount": amountProperty("The charge's amount in major units, as a decimal such " +
				"as 125.00."),
		},
		toolschema.KeyRequired: []string{
			"accessorialChargeId", "method", "unit", "amount",
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *createShipmentTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpCreate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressInternal},
		Effect:        agent.ToolEffectChange,
		Idempotent:    true,
		ReadsExternal: agent.ExternalReadNever,
		Artifact:      shipmentRecordEntity,
		Rationale: "A new load commits a customer's freight and the money that follows, so " +
			"no desk books one unattended.",
	}
}

func (t *createShipmentTool) Execute(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *createShipmentTool) ExecuteWithResult(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := guardExecute(t, params); err != nil {
		return nil, err
	}

	entity, err := t.draft(ctx, &params)
	if err != nil {
		return nil, err
	}

	created, err := t.shipments.Create(ctx, entity, params.Actor)
	if err != nil {
		return nil, err
	}

	if entity.SourceDocumentID != "" && t.imports != nil {
		if completeErr := t.imports.CompleteHistory(
			ctx,
			entity.SourceDocumentID,
			tenantFrom(params),
		); completeErr != nil {
			t.logger.Warn("shipment created but the import conversation could not be closed",
				zap.String("sourceDocumentId", entity.SourceDocumentID),
				zap.String("shipmentId", created.ID.String()),
				zap.Error(completeErr),
			)
		}
	}

	return shipmentResult(resultCreated, created), nil
}

// Validate runs the create plan the preview runs, so a shipment the service
// would refuse, a BOL it already has or a rate it cannot find among them,
// is refused to the model before it is proposed to a person.
func (t *createShipmentTool) Validate(
	ctx context.Context,
	params serviceports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return previewValidates(ctx, t, &params)
}

func (t *createShipmentTool) draft(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (*shipment.Shipment, error) {
	draft := new(shipmentDraft)
	if err := decodeParam(params.Params, "shipment", draft); err != nil {
		return nil, err
	}

	tenantInfo := tenantFrom(*params)
	zones, err := readLocationZones(
		ctx, t.locations, tenantInfo, params.Timezone, draft.locationIDs(),
	)
	if err != nil {
		return nil, err
	}

	entity, err := draft.build(tenantInfo, zones)
	if err != nil {
		return nil, err
	}

	if sourceDocumentID := optionalString(
		params.Params,
		"sourceDocumentId",
	); sourceDocumentID != "" {
		if _, err = pulid.Parse(sourceDocumentID); err != nil {
			return nil, errortypes.NewValidationError(
				"sourceDocumentId",
				errortypes.ErrInvalid,
				"Source document id is not an id",
			)
		}
		entity.SourceDocumentID = sourceDocumentID
	}

	return entity, nil
}

// updateShipmentTool changes the details of a saved shipment that a person
// would change on the form: who it is for, what service, what freight. It
// deliberately cannot touch stops, moves, rates or status: those have their
// own tools and their own rules, and a patch that could reach them would be
// a way around both.
type updateShipmentTool struct {
	shipments shipmentWriter
	partners  shipmentPartnerNotices
}

func newUpdateShipmentTool(
	shipments shipmentWriter,
	partners shipmentPartnerNotices,
) serviceports.AgentTool {
	return &updateShipmentTool{shipments: shipments, partners: partners}
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

func (t *updateShipmentTool) Policy() serviceports.ToolPolicy {
	return serviceports.ToolPolicy{
		Name:          t.Name(),
		Kind:          agent.ToolKindAction,
		Resource:      permission.ResourceShipment,
		Operation:     permission.OpUpdate,
		Scope:         agent.ToolScopeTenant,
		DefaultTier:   agent.TierActWithApproval,
		MaxTier:       agent.TierActWithApproval,
		Egress:        []agent.EgressClass{agent.EgressExternalRecipient},
		Effect:        agent.ToolEffectChange,
		Reversible:    true,
		ReadsExternal: agent.ExternalReadNever,
		Rationale: "A changed shipment is sent as an EDI tender change to the trading " +
			"partners it was tendered to.",
	}
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
	_, entity, err := t.plan(ctx, &params)
	if err != nil {
		return err
	}

	_, err = t.shipments.Update(ctx, entity, params.Actor)

	return err
}

// plan loads the shipment and applies the patch to it: the shipment as it is,
// and as the write would save it.
func (t *updateShipmentTool) plan(
	ctx context.Context,
	params *serviceports.ToolExecuteParams,
) (before, after *shipment.Shipment, err error) {
	if err = guardExecute(t, *params); err != nil {
		return nil, nil, err
	}

	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, nil, err
	}

	var patch shipmentPatch
	if err = jsonutils.Convert(params.Params, &patch); err != nil {
		return nil, nil, fmt.Errorf("update_shipment parameters: %w", err)
	}
	if patch.empty() {
		return nil, nil, errortypes.NewValidationError(
			"shipmentId",
			errortypes.ErrInvalid,
			"Nothing to change: send at least one field besides the shipment id",
		)
	}

	entity, err := t.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantFrom(*params),
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
		},
	})
	if err != nil {
		return nil, nil, err
	}

	patched := *entity
	if err = applyShipmentPatch(&patched, patch); err != nil {
		return nil, nil, err
	}

	return entity, &patched, nil
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
