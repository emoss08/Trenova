package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
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

type contractRatePreviewer interface {
	PreviewContractRate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *serviceports.RequestActor,
	) (*serviceports.ContractRateApplication, error)
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
	contracts contractRatePreviewer
	imports   importCompleter
	locations locationZoneReader
	logger    *zap.Logger
}

type createShipmentDeps struct {
	Shipments shipmentCreator
	Contracts contractRatePreviewer
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
		contracts: deps.Contracts,
		imports:   deps.Imports,
		locations: deps.Locations,
		logger:    logger.Named("tool.create-shipment"),
	}
}

func (t *createShipmentTool) Name() string { return "create_shipment" }

func (t *createShipmentTool) SearchTerms() []string {
	return []string{"new shipment", "enter shipment"}
}

func (t *createShipmentTool) Prerequisites() []string {
	return []string{
		"list_customers",
		"list_service_types",
		"list_shipment_types",
		"list_formula_templates",
		"list_locations",
		"list_commodities",
		"list_accessorial_charges",
		"list_equipment_types",
	}
}

func (t *createShipmentTool) Recipe() []string {
	return []string{
		"list_customers",
		"list_service_types",
		"list_shipment_types",
		"list_locations",
		"create_shipment",
	}
}

func (t *createShipmentTool) Description() string {
	return "Enter a new shipment, priced by the customer's rate agreement when no price is " +
		"given; to copy one, use duplicate_shipment. Give the customer, " +
		"service type and shipment type by id. Price it the way the person did: \"$2,450 " +
		"flat\" is the Flat Rate template with baseRate 2450, \"$2.10 a mile\" Per Mile with " +
		"2.10, and charges they name are accessorial charges. Given no price, leave the rating " +
		"out and call it: it finds the agreement itself, so do not look one up. Only when " +
		"it says none covers the lane, ask the person for the price, offering " +
		"one to react to. Add the freight (pieces, weight, " +
		"commodities, temperatures) and one move with its stops in travel order, each with a " +
		"location id, a type and a local window such as 2026-10-01T08:00, from 0. Pass " +
		"sourceDocumentId for a shipment read from an uploaded document. The system assigns " +
		"the pro number; the BOL is optional, and a BOL must be unique."
}

func (t *createShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipment": shipmentDraftSchema(),
			"sourceDocumentId": agenttoolschema.RecordIDText(
				permission.ResourceDocument,
				"The uploaded document this shipment was read from, "+
					"when there is one: the documentId you read with get_shipment_draft, usually "+
					"this run's subject or the page.",
			),
		},
		toolschema.KeyRequired:             []string{"shipment"},
		toolschema.KeyAdditionalProperties: false,
	}
}

const paramAdditionalCharges = "additionalCharges"

func shipmentDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType:        toolschema.TypeObject,
		toolschema.KeyDescription: "The shipment to enter.",
		toolschema.KeyProperties: map[string]any{
			paramCustomerID: agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"The customer, from list_customers.",
			),
			"billToCustomerId": agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"Who is billed, when not the customer, from "+
					"list_customers.",
			),
			fieldServiceTypeID: agenttoolschema.RecordIDText(permission.ResourceServiceType,
				"From list_service_types."),
			fieldShipmentTypeID: agenttoolschema.RecordIDText(permission.ResourceShipmentType,
				"From list_shipment_types."),
			"formulaTemplateId": agenttoolschema.RecordIDText(
				permission.ResourceFormulaTemplate,
				"The rating method that prices the freight, from "+
					"list_formula_templates. Leave it out when the customer's rate agreement "+
					"covers the lane, and the contract's is used.",
			),
			"baseRate": amountProperty("The rate the rating method prices with, as the person " +
				"gave it: 2450 for a $2,450 flat load, 2.10 for $2.10 a mile."),
			"freightTerms": agenttoolschema.Enum(
				"Who pays the freight. Defaults to Prepaid.",
				agenttoolschema.FreightTerms,
			),
			previewFieldTractorTypeID: agenttoolschema.RecordIDText(
				permission.ResourceEquipmentType,
				"A tractor equipment type, from list_equipment_types.",
			),
			previewFieldTrailerTypeID: agenttoolschema.RecordIDText(
				permission.ResourceEquipmentType,
				"A trailer equipment type, from list_equipment_types.",
			),
			"bol": stringProperty("The customer's BOL or reference. It must be unique among "+
				"open shipments, so never reuse one from another shipment. Optional unless the "+
				"customer's billing requires a BOL; leave it out when none is known.", 100),
			"externalReference": stringProperty("The customer's own order or load number, "+
				"as it appears on their tender, email or rate confirmation. One live shipment "+
				"per customer may carry it, so a second booking of the same order is refused; "+
				"leave it out when the customer gave none.", 100),
			fieldPieces: integerProperty(
				"The total piece or handling-unit count.",
				0,
				1_000_000,
			),
			fieldWeight:      integerProperty("Pounds.", 0, 10_000_000),
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
			paramAdditionalCharges: map[string]any{
				toolschema.KeyType:        toolschema.TypeArray,
				toolschema.KeyDescription: "Accessorial charges agreed for the shipment.",
				toolschema.KeyItems:       chargeLineDraftSchema(),
			},
		},
		toolschema.KeyRequired: []string{
			paramCustomerID,
			fieldServiceTypeID,
			fieldShipmentTypeID,
			"moves",
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
			fieldLocationID: agenttoolschema.RecordIDText(
				permission.ResourceLocation,
				"The stop's location, from list_locations.",
			),
			fieldType: agenttoolschema.Enum(
				"What happens at the stop.",
				agenttoolschema.StopTypes,
			),
			"scheduleType": agenttoolschema.Enum(
				"Open for a window, Appointment for a fixed time. Defaults to Open.",
				agenttoolschema.StopScheduleTypes,
			),
			"sequence": integerProperty("The stop's place in the move, counting from 0. "+
				"Leave it out to take the order given.", 0, 100),
			"scheduledWindowStart": agenttoolschema.LocalDateTime("When the stop's window " +
				"opens. A part of the day the person named is a window, not a question: " +
				"morning 08:00 to 12:00, afternoon 13:00 to 17:00, evening 17:00 to 21:00. " +
				"A deadline alone (\"by 3pm\", \"by noon\") opens at 08:00 that day and closes " +
				"at the deadline. Use it and say which you used."),
			"scheduledWindowEnd": agenttoolschema.LocalDateTime(
				"When the stop's window closes: the deadline the person gave, or the end of " +
					"the part of the day they named.",
			),
			fieldPieces:   integerProperty("Pieces handled at the stop.", 0, 1_000_000),
			fieldWeight:   integerProperty("Pounds handled at the stop.", 0, 10_000_000),
			"addressLine": stringProperty("A dock or suite note for the stop.", 200),
		},
		toolschema.KeyRequired: []string{
			fieldLocationID,
			fieldType,
			fieldScheduledWindowStart,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func commodityDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"commodityId": agenttoolschema.RecordIDText(
				permission.ResourceCommodity,
				"The commodity, from list_commodities.",
			),
			fieldPieces: integerProperty("Pieces of it. Defaults to 1.", 0, 1_000_000),
			fieldWeight: integerProperty("Pounds of it.", 0, 10_000_000),
		},
		toolschema.KeyRequired:             []string{"commodityId"},
		toolschema.KeyAdditionalProperties: false,
	}
}

func chargeLineDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"accessorialChargeId": agenttoolschema.RecordIDText(
				permission.ResourceAccessorialCharge,
				"The accessorial, from list_accessorial_charges.",
			),
			"method": agenttoolschema.Enum(
				"How the amount is applied: Flat once, PerUnit times the unit, Percentage of "+
					"the linehaul.",
				agenttoolschema.AccessorialMethods,
			),
			"unit": integerProperty("How many units the charge covers; 1 for a flat charge.", 1,
				10_000),
			paramAmount: amountProperty("The charge's amount in major units, as a decimal such " +
				"as 125.00."),
		},
		toolschema.KeyRequired: []string{
			"accessorialChargeId", "method", "unit", paramAmount,
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
	if err = t.contractRating(ctx, entity, params.Actor); err != nil {
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

// contractRating seats the rating the customer's agreement prices the lane
// with when the call named none, as quote_shipment would have returned it, so
// the shipment is saved on the contract's rate rather than asking the person
// for a rating method the contract already decides.
func (t *createShipmentTool) contractRating(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *serviceports.RequestActor,
) error {
	if entity.FormulaTemplateID.IsNotNil() {
		return nil
	}
	if t.contracts == nil {
		return errNoRatingMethod("")
	}

	priced, err := t.contracts.PreviewContractRate(ctx, entity, actor)
	if err != nil {
		return err
	}
	if priced == nil || !priced.Applied || priced.Outcome != ratequote.OutcomeRated ||
		priced.FormulaTemplateID == nil || priced.FormulaTemplateID.IsNil() {
		reason := ""
		if priced != nil {
			reason = string(priced.Outcome)
		}

		return errNoRatingMethod(reason)
	}

	entity.FormulaTemplateID = *priced.FormulaTemplateID
	if !entity.BaseRate.Valid && priced.BaseRate.Valid {
		entity.BaseRate = priced.BaseRate
	}

	return nil
}

func errNoRatingMethod(outcome string) error {
	message := "No rate agreement prices this lane for this customer, so the price is the " +
		"person's to give. Ask them for it, offering one to react to (a figure from " +
		"quote_shipment or their recent loads on the lane, if you can get one), then send it " +
		"as formulaTemplateId from list_formula_templates and baseRate"
	if outcome != "" {
		message += " (the agreements answered " + outcome + ")"
	}

	return errortypes.NewValidationError("formulaTemplateId", errortypes.ErrRequired, message)
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

func (t *updateShipmentTool) SearchTerms() []string {
	return []string{"edit shipment", "pieces", "correct weight"}
}

func (t *updateShipmentTool) Prerequisites() []string {
	return []string{"search_shipments", "get_shipment"}
}

func (t *updateShipmentTool) Recipe() []string {
	return []string{"search_shipments", "get_shipment", "update_shipment"}
}

func (t *updateShipmentTool) Description() string {
	return "Change the details of a saved shipment: the customer, service type, shipment " +
		"type, equipment types, BOL, pieces, weight or temperature range. Send only " +
		"the fields to change. Stops, moves, rates and status are not changed here: " +
		"reschedule_stop moves an appointment, add_shipment_charge adds an accessorial, " +
		"and the move, hold and cancel tools do the rest."
}

func (t *updateShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"shipmentId": agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to change, from the page, list_shipments or search_shipments."),
			"customerId": agenttoolschema.RecordIDText(permission.ResourceCustomer,
				"The new customer, from list_customers."),
			"serviceTypeId": agenttoolschema.RecordIDText(permission.ResourceServiceType,
				"The new service type, from list_service_types."),
			"shipmentTypeId": agenttoolschema.RecordIDText(permission.ResourceShipmentType,
				"The new shipment type, from list_shipment_types."),
			"tractorTypeId": agenttoolschema.RecordIDText(permission.ResourceEquipmentType,
				"The new tractor equipment type, from list_equipment_types."),
			"trailerTypeId": agenttoolschema.RecordIDText(permission.ResourceEquipmentType,
				"The new trailer equipment type, from list_equipment_types."),
			"bol": map[string]any{
				"type":        "string",
				"description": "The customer's BOL or reference number.",
			},
			"pieces": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "The total piece or handling-unit count. Leave it out to keep it.",
			},
			"weight": map[string]any{
				"type":        "integer",
				"minimum":     1,
				"description": "Pounds. Leave it out to keep it.",
			},
			"temperatureMin": map[string]any{
				"type": "integer",
				"description": "Fahrenheit, with temperatureMax, for a load that needs temperature " +
					"control. Leave both out to keep the load as it is.",
			},
			"temperatureMax": map[string]any{
				"type":        "integer",
				"description": "Fahrenheit, with temperatureMin.",
			},
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
	if err := placeholderTemperatures(entity, patch); err != nil {
		return err
	}
	if patch.TemperatureMin != nil {
		entity.TemperatureMin = patch.TemperatureMin
	}
	if patch.TemperatureMax != nil {
		entity.TemperatureMax = patch.TemperatureMax
	}
	if entity.TemperatureMin != nil && entity.TemperatureMax != nil &&
		*entity.TemperatureMin > *entity.TemperatureMax {
		return errortypes.NewValidationError("temperatureMin", errortypes.ErrInvalid,
			"The minimum temperature is above the maximum")
	}

	return nil
}

func placeholderTemperatures(entity *shipment.Shipment, patch shipmentPatch) error {
	unset := entity.TemperatureMin == nil && entity.TemperatureMax == nil
	zero := func(value *int16) bool { return value != nil && *value == 0 }
	if unset && zero(patch.TemperatureMin) && zero(patch.TemperatureMax) {
		return errortypes.NewValidationError("temperatureMin", errortypes.ErrInvalid,
			"The load has no temperature control, and 0 to 0 °F would make it a frozen load. "+
				"Leave both temperatures out to keep it as it is, or give the range the "+
				"person asked for")
	}

	return nil
}

func (t *updateShipmentTool) Target(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, "shipmentId", permission.ResourceShipment)
}
