package operationstools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/shipmenttools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolguard"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreset"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentserviceports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type shipmentCreator interface {
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewCreate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipmentserviceports.ShipmentCreatePlan, error)
}

type contractRatePreviewer interface {
	PreviewContractRate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipmentserviceports.ContractRateApplication, error)
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
type CreateShipmentTool struct {
	shipments shipmentCreator
	contracts contractRatePreviewer
	imports   importCompleter
	locations shipmentpreview.LocationZoneReader
	logger    *zap.Logger
}

type CreateShipmentDeps struct {
	Shipments shipmentCreator
	Contracts contractRatePreviewer
	Imports   importCompleter
	Locations shipmentpreview.LocationZoneReader
	Logger    *zap.Logger
}

func NewCreateShipmentTool(deps CreateShipmentDeps) agenttoolports.AgentTool {
	logger := deps.Logger
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CreateShipmentTool{
		shipments: deps.Shipments,
		contracts: deps.Contracts,
		imports:   deps.Imports,
		locations: deps.Locations,
		logger:    logger.Named("tool.create-shipment"),
	}
}

func (t *CreateShipmentTool) Name() string { return "shipment_create" }

func (t *CreateShipmentTool) SearchTerms() []string {
	return []string{"new shipment", "enter shipment", "book a load", "new load"}
}

func (t *CreateShipmentTool) Prerequisites() []string {
	return []string{
		agentpreset.ToolFindShipmentRecords,
		"customer_list",
		"masterdata_list_service_types",
		"masterdata_list_shipment_types",
		"formula_list_templates",
		"location_list",
		"masterdata_list_commodities",
		"masterdata_list_accessorial_charges",
		"equipment_list_types",
	}
}

func (t *CreateShipmentTool) Companions() []string {
	return []string{
		agentpreset.ToolCreateCustomer,
		agentpreset.ToolCreateLocation,
		agentpreset.ToolCreateCommodity,
	}
}

func (t *CreateShipmentTool) Recipe() []string {
	return []string{agentpreset.ToolFindShipmentRecords, "shipment_create"}
}

func (t *CreateShipmentTool) Description() string {
	return "Enter a new shipment, priced by the customer's rate agreement when no price is given; to copy one, use shipment_duplicate. " +
		"Match its customer, stops and commodities first with one shipment_match_records call, and propose what it says is missing instead of stopping. " +
		"Do not interview the person first: fill what the conversation, page and lookups give and call it; ask_user only for what it refuses as missing. " +
		"Price it the way the person did: \"$2,450 flat\" is the Flat Rate template, \"$2.10 a mile\" Per Mile, and charges they name are accessorial charges. " +
		"Given no price, leave the rating out: it finds the agreement itself; ask for one only when none covers the lane. " +
		"Add the freight and one move with its stops in travel order, each with a location and a local window; a window that has passed is entered as written. " +
		"The system assigns the pro number; the BOL is optional, and a BOL must be unique."
}

func (t *CreateShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"shipment": shipmentDraftSchema(),
			"sourceDocumentId": agenttoolschema.RecordIDText(
				permission.ResourceDocument,
				"The uploaded document this shipment was read from, "+
					"when there is one: the documentId you read with intake_get_draft or "+
					"agent_read_attachment, usually a file on the message, this run's subject or "+
					"the page.",
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
			toolparams.ParamCustomerID: agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"The customer, from customer_list.",
			),
			"billToCustomerId": agenttoolschema.RecordIDText(
				permission.ResourceCustomer,
				"Who is billed, when not the customer, from "+
					"customer_list.",
			),
			changepreview.FieldServiceTypeID: agenttoolschema.RecordIDText(
				permission.ResourceServiceType,
				"From masterdata_list_service_types.",
			),
			changepreview.FieldShipmentTypeID: agenttoolschema.RecordIDText(
				permission.ResourceShipmentType,
				"From masterdata_list_shipment_types.",
			),
			"formulaTemplateId": agenttoolschema.RecordIDText(
				permission.ResourceFormulaTemplate,
				"The tariff that prices the freight, from formula_list_templates. Leave "+
					"it out when the customer's rate agreement covers the lane, and the "+
					"contract's is used, or when freightChargeAmount gives the price.",
			),
			"tariffInputs": agenttoolschema.StringMap("Values for the inputs the tariff declares, "+
				"which formula_get_template lists, as the person gave them: $2,450 for a "+
				"flat load, $2.10/mi for a rate a mile, 12% for a percent. Leave an input "+
				"out to take the tariff's default.", 4000),
			changepreview.FieldFreightChargeAmount: toolparams.AmountProperty(
				"The freight charge the person or the customer's rate confirmation gives, " +
					"such as 2450. The shipment is then rated by hand at it and no tariff " +
					"runs; leave formulaTemplateId out.",
			),
			"freightTerms": agenttoolschema.Enum(
				"Who pays the freight. Defaults to Prepaid.",
				agenttoolschema.FreightTerms,
			),
			changepreview.PreviewFieldTractorTypeID: agenttoolschema.RecordIDText(
				permission.ResourceEquipmentType,
				"A tractor equipment type, from equipment_list_types.",
			),
			changepreview.PreviewFieldTrailerTypeID: agenttoolschema.RecordIDText(
				permission.ResourceEquipmentType,
				"A trailer equipment type, from equipment_list_types.",
			),
			"bol": toolparams.StringProperty(
				"The bill of lading number. It must be unique among open shipments, so "+
					"never reuse one from another shipment. Optional unless the customer's "+
					"billing requires a BOL; leave it out when none is known.",
				100,
			),
			"externalReference": toolparams.StringProperty(
				"The customer's own order or load number, "+
					"as it appears on their tender, email or rate confirmation. It is checked apart "+
					"from the BOL, only against this customer's other live shipments, so a second "+
					"booking of the same order is refused; keep it as written unless that refusal "+
					"comes back, and leave it out when the customer gave none.",
				100,
			),
			changepreview.FieldPieces: toolparams.IntegerProperty(
				"The total piece or handling-unit count.",
				0,
				1_000_000,
			),
			changepreview.FieldWeight: toolparams.IntegerProperty("Pounds.", 0, 10_000_000),
			"temperatureMin": toolparams.IntegerProperty(
				"Fahrenheit, for reefer freight.",
				-100,
				200,
			),
			"temperatureMax": toolparams.IntegerProperty(
				"Fahrenheit, for reefer freight.",
				-100,
				200,
			),
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
			toolparams.ParamCustomerID,
			changepreview.FieldServiceTypeID,
			changepreview.FieldShipmentTypeID,
			"moves",
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func moveDraftSchema() map[string]any {
	return map[string]any{
		toolschema.KeyType: toolschema.TypeObject,
		toolschema.KeyProperties: map[string]any{
			"loaded": toolparams.BooleanProperty("False for a deadhead move. Defaults to true."),
			"sequence": toolparams.IntegerProperty(
				"The move's place, counting from 0. Leave it out to "+
					"take the order given.",
				0,
				50,
			),
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
			changepreview.FieldLocationID: agenttoolschema.RecordIDText(
				permission.ResourceLocation,
				"The stop's location, from location_list.",
			),
			changepreview.FieldType: agenttoolschema.Enum(
				"What happens at the stop.",
				agenttoolschema.StopTypes,
			),
			"scheduleType": agenttoolschema.Enum(
				"Open for a window, Appointment for a fixed time. Defaults to Open.",
				agenttoolschema.StopScheduleTypes,
			),
			"sequence": toolparams.IntegerProperty(
				"The stop's place in the move, counting from 0. "+
					"Leave it out to take the order given.",
				0,
				100,
			),
			"scheduledWindowStart": agenttoolschema.LocalDateTime("When the stop's window " +
				"opens. A part of the day the person named is a window, not a question: " +
				"morning 08:00 to 12:00, afternoon 13:00 to 17:00, evening 17:00 to 21:00. " +
				"A deadline alone (\"by 3pm\", \"by noon\") opens at 08:00 that day and closes " +
				"at the deadline. Use it and say which you used."),
			"scheduledWindowEnd": agenttoolschema.LocalDateTime(
				"When the stop's window closes: the deadline the person gave, or the end of " +
					"the part of the day they named.",
			),
			changepreview.FieldPieces: toolparams.IntegerProperty(
				"Pieces handled at the stop.",
				0,
				1_000_000,
			),
			changepreview.FieldWeight: toolparams.IntegerProperty(
				"Pounds handled at the stop.",
				0,
				10_000_000,
			),
			"addressLine": toolparams.StringProperty("A dock or suite note for the stop.", 200),
		},
		toolschema.KeyRequired: []string{
			changepreview.FieldLocationID,
			changepreview.FieldType,
			changepreview.FieldScheduledWindowStart,
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
				"The commodity, from masterdata_list_commodities.",
			),
			changepreview.FieldPieces: toolparams.IntegerProperty(
				"Pieces of it. Defaults to 1.",
				0,
				1_000_000,
			),
			changepreview.FieldWeight: toolparams.IntegerProperty("Pounds of it.", 0, 10_000_000),
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
				"The accessorial, from masterdata_list_accessorial_charges.",
			),
			"method": agenttoolschema.Enum(
				"How the amount is applied: Flat once, PerUnit times the unit, Percentage of "+
					"the linehaul.",
				agenttoolschema.AccessorialMethods,
			),
			"unit": toolparams.IntegerProperty(
				"How many units the charge covers; 1 for a flat charge.",
				1,
				10_000,
			),
			toolparams.ParamAmount: toolparams.AmountProperty(
				"The charge's amount in major units, as a decimal such " +
					"as 125.00.",
			),
		},
		toolschema.KeyRequired: []string{
			"accessorialChargeId", "method", "unit", toolparams.ParamAmount,
		},
		toolschema.KeyAdditionalProperties: false,
	}
}

func (t *CreateShipmentTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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
		Artifact:      changepreview.ShipmentRecordEntity,
		Rationale: "A new load commits a customer's freight and the money that follows, so " +
			"no desk books one unattended.",
	}
}

func (t *CreateShipmentTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
) error {
	_, err := t.ExecuteWithResult(ctx, params)

	return err
}

func (t *CreateShipmentTool) ExecuteWithResult(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolResultReporter interface passes params by value
) (*agent.ToolExecutionResult, error) {
	if err := toolguard.GuardExecute(t, params); err != nil {
		return nil, err
	}

	entity, err := t.Draft(ctx, &params)
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
			toolparams.TenantFrom(params),
		); completeErr != nil {
			t.logger.Warn("shipment created but the import conversation could not be closed",
				zap.String("sourceDocumentId", entity.SourceDocumentID),
				zap.String("shipmentId", created.ID.String()),
				zap.Error(completeErr),
			)
		}
	}

	return shipmentpreview.ShipmentResult(changepreview.ResultCreated, created), nil
}

// Validate runs the create plan the preview runs, so a shipment the service
// would refuse, a BOL it already has or a rate it cannot find among them,
// is refused to the model before it is proposed to a person.
func (t *CreateShipmentTool) Validate(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolValidator interface passes params by value
) error {
	return toolspec.PreviewValidates(ctx, t, &params)
}

func (t *CreateShipmentTool) Draft(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (*shipment.Shipment, error) {
	draft := new(shipmenttools.ShipmentDraft)
	if err := toolparams.DecodeParam(params.Params, "shipment", draft); err != nil {
		return nil, err
	}

	tenantInfo := toolparams.TenantFrom(*params)
	zones, err := shipmentpreview.ReadLocationZones(
		ctx, t.locations, tenantInfo, params.Timezone, draft.LocationIDs(),
	)
	if err != nil {
		return nil, err
	}

	entity, err := draft.Build(tenantInfo, zones)
	if err != nil {
		return nil, err
	}
	if err = t.contractRating(ctx, entity, params.Actor); err != nil {
		return nil, err
	}

	if sourceDocumentID := toolparams.OptionalString(
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
// with when the call named none, as rate_quote would have returned it, so
// the shipment is saved on the contract's rate rather than asking the person
// for a rating method the contract already decides.
func (t *CreateShipmentTool) contractRating(
	ctx context.Context,
	entity *shipment.Shipment,
	actor *permissionports.RequestActor,
) error {
	if entity.FormulaTemplateID.IsNotNil() || entity.FreightChargeAmount.Valid {
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
	if len(entity.TariffInputs) == 0 {
		entity.TariffInputs = priced.Inputs.Clone()
	}

	return nil
}

func errNoRatingMethod(outcome string) error {
	message := "No rate agreement prices this lane for this customer, so the price is the " +
		"one the person or the customer's own rate confirmation gives. When neither gives " +
		"one, ask the person for it, offering one to react to (a figure from " +
		"rate_quote or their recent loads on the lane, if you can get one). Send it as " +
		"freightChargeAmount, or as formulaTemplateId from formula_list_templates with " +
		"its inputs"
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
type UpdateShipmentTool struct {
	shipments shipmentpreview.ShipmentWriter
	partners  shipmentpreview.ShipmentPartnerNotices
}

func NewUpdateShipmentTool(
	shipments shipmentpreview.ShipmentWriter,
	partners shipmentpreview.ShipmentPartnerNotices,
) agenttoolports.AgentTool {
	return &UpdateShipmentTool{shipments: shipments, partners: partners}
}

func (t *UpdateShipmentTool) Name() string { return "shipment_update" }

func (t *UpdateShipmentTool) SearchTerms() []string {
	return []string{"edit shipment", "pieces", "correct weight"}
}

func (t *UpdateShipmentTool) Prerequisites() []string {
	return []string{"shipment_list", "shipment_get"}
}

func (t *UpdateShipmentTool) Recipe() []string {
	return []string{"shipment_list", "shipment_get", "shipment_update"}
}

func (t *UpdateShipmentTool) Description() string {
	return "Change the details of a saved shipment: the customer, service type, shipment " +
		"type, equipment types, BOL, pieces, weight or temperature range. Send only " +
		"the fields to change. Stops, moves, rates and status are not changed here: " +
		"shipment_reschedule_stop moves an appointment, shipment_add_charge adds an accessorial, " +
		"and the move, hold and cancel tools do the rest."
}

func (t *UpdateShipmentTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			toolparams.ParamShipmentID: agenttoolschema.RecordIDText(permission.ResourceShipment,
				"The shipment to change, from the page, shipment_list."),
			"customerId": agenttoolschema.RecordIDText(permission.ResourceCustomer,
				"The new customer, from customer_list."),
			"serviceTypeId": agenttoolschema.RecordIDText(permission.ResourceServiceType,
				"The new service type, from masterdata_list_service_types."),
			"shipmentTypeId": agenttoolschema.RecordIDText(permission.ResourceShipmentType,
				"The new shipment type, from masterdata_list_shipment_types."),
			"tractorTypeId": agenttoolschema.RecordIDText(permission.ResourceEquipmentType,
				"The new tractor equipment type, from equipment_list_types."),
			"trailerTypeId": agenttoolschema.RecordIDText(permission.ResourceEquipmentType,
				"The new trailer equipment type, from equipment_list_types."),
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
		"required":             []string{toolparams.ParamShipmentID},
		"additionalProperties": false,
	}
}

func (t *UpdateShipmentTool) Policy() agenttoolports.ToolPolicy {
	return agenttoolports.ToolPolicy{
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

// shipmentPatch is what shipment_update may change. Pointers say "absent
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

func (t *UpdateShipmentTool) Execute(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams,
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
func (t *UpdateShipmentTool) plan(
	ctx context.Context,
	params *agenttoolports.ToolExecuteParams,
) (before, after *shipment.Shipment, err error) {
	if err = toolguard.GuardExecute(t, *params); err != nil {
		return nil, nil, err
	}

	shipmentID, err := toolparams.RequirePulid(params.Params, toolparams.ParamShipmentID)
	if err != nil {
		return nil, nil, err
	}

	var patch shipmentPatch
	if err = jsonutils.Convert(params.Params, &patch); err != nil {
		return nil, nil, fmt.Errorf("shipment_update parameters: %w", err)
	}
	if patch.empty() {
		return nil, nil, errortypes.NewValidationError(
			toolparams.ParamShipmentID,
			errortypes.ErrInvalid,
			"Nothing to change: send at least one field besides the shipment id",
		)
	}

	entity, err := t.shipments.Get(ctx, &shipmentports.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: toolparams.TenantFrom(*params),
		ShipmentOptions: shipmentports.ShipmentOptions{
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

func (t *UpdateShipmentTool) Target(params map[string]any) (agenttoolports.ToolTarget, bool) {
	return toolparams.TargetOf(params, toolparams.ParamShipmentID, permission.ResourceShipment)
}
