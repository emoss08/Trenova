package agenttoolservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/internal/core/services/trailerservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	tractorRecordEntity    = "tractor"
	trailerRecordEntity    = "trailer"
	paramTrailerID         = "trailerId"
	paramNewLocationID     = "newLocationId"
	paramPrimaryWorkerID   = "primaryWorkerId"
	paramSecondaryWorkerID = "secondaryWorkerId"
	paramRegistrationState = "registrationState"
	paramMaxLoadWeight     = "maxLoadWeight"
	paramLastInspection    = "lastInspectionDate"
	mdMaxLoadWeight        = 200_000
	equipmentSupplierNote  = "from list_equipment_types"
	manufacturerSupplier   = "from list_equipment_manufacturers"
	fleetCodeSupplier      = "from list_fleet_codes"
)

var (
	equipmentDates = toolpreview.Types(map[string]assistantartifact.DisplayType{
		mdRegistrationExpiry: assistantartifact.DisplayDate,
		paramLastInspection:  assistantartifact.DisplayDate,
	})
	equipmentRefs = toolpreview.WithRefs(map[string]permission.Resource{
		paramPrimaryWorkerID:   permission.ResourceWorker,
		paramSecondaryWorkerID: permission.ResourceWorker,
		mdEquipmentTypeID:      permission.ResourceEquipmentType,
		mdManufacturerID:       permission.ResourceEquipmentManufacturer,
		mdFleetCodeID:          permission.ResourceFleetCode,
	})
)

type tractorKeeper interface {
	Get(ctx context.Context, req repositories.GetTractorByIDRequest) (*tractor.Tractor, error)
	PlanCreate(ctx context.Context, entity *tractor.Tractor) (*tractor.Tractor, error)
	Create(
		ctx context.Context,
		entity *tractor.Tractor,
		actor *serviceports.RequestActor,
	) (*tractor.Tractor, error)
	PlanUpdate(
		ctx context.Context,
		entity *tractor.Tractor,
	) (*serviceports.RecordChange[tractor.Tractor], error)
	Update(
		ctx context.Context,
		entity *tractor.Tractor,
		actor *serviceports.RequestActor,
	) (*tractor.Tractor, error)
	PlanLocate(
		ctx context.Context,
		req *repositories.LocateTractorRequest,
	) (*tractorservice.LocatePlan, error)
	Locate(
		ctx context.Context,
		req *repositories.LocateTractorRequest,
	) (*equipmentcontinuity.EquipmentContinuity, error)
}

type trailerKeeper interface {
	Get(ctx context.Context, req repositories.GetTrailerByIDRequest) (*trailer.Trailer, error)
	PlanCreate(ctx context.Context, entity *trailer.Trailer) (*trailer.Trailer, error)
	Create(
		ctx context.Context,
		entity *trailer.Trailer,
		actor *serviceports.RequestActor,
	) (*trailer.Trailer, error)
	PlanUpdate(
		ctx context.Context,
		entity *trailer.Trailer,
	) (*serviceports.RecordChange[trailer.Trailer], error)
	Update(
		ctx context.Context,
		entity *trailer.Trailer,
		actor *serviceports.RequestActor,
	) (*trailer.Trailer, error)
	PlanLocate(
		ctx context.Context,
		req *repositories.LocateTrailerRequest,
		actor *serviceports.RequestActor,
	) (*trailerservice.LocatePlan, error)
	Locate(
		ctx context.Context,
		req *repositories.LocateTrailerRequest,
		actor *serviceports.RequestActor,
	) (*equipmentcontinuity.EquipmentContinuity, error)
}

type tractorView struct {
	Code                    string `json:"code"`
	Status                  string `json:"status"`
	PrimaryWorkerID         string `json:"primaryWorkerId"`
	SecondaryWorkerID       string `json:"secondaryWorkerId,omitempty"`
	EquipmentTypeID         string `json:"equipmentTypeId"`
	EquipmentManufacturerID string `json:"equipmentManufacturerId"`
	FleetCodeID             string `json:"fleetCodeId,omitempty"`
	Make                    string `json:"make,omitempty"`
	Model                   string `json:"model,omitempty"`
	Year                    *int   `json:"year,omitempty"`
	Vin                     string `json:"vin,omitempty"`
	LicensePlateNumber      string `json:"licensePlateNumber,omitempty"`
	State                   string `json:"state,omitempty"`
	RegistrationNumber      string `json:"registrationNumber,omitempty"`
	RegistrationExpiry      *int64 `json:"registrationExpiry,omitempty"`
	FuelType                string `json:"fuelType"`
	IFTAQualified           bool   `json:"iftaQualified"`
	ExternalID              string `json:"externalId,omitempty"`
}

func idText(id pulid.ID) string {
	if id.IsNil() {
		return ""
	}

	return id.String()
}

func tractorViewOf(entity *tractor.Tractor, states map[pulid.ID]string) any {
	return &tractorView{
		Code:                    entity.Code,
		Status:                  string(entity.Status),
		PrimaryWorkerID:         idText(entity.PrimaryWorkerID),
		SecondaryWorkerID:       idText(entity.SecondaryWorkerID),
		EquipmentTypeID:         idText(entity.EquipmentTypeID),
		EquipmentManufacturerID: idText(entity.EquipmentManufacturerID),
		FleetCodeID:             idText(entity.FleetCodeID),
		Make:                    entity.Make,
		Model:                   entity.Model,
		Year:                    entity.Year,
		Vin:                     entity.Vin,
		LicensePlateNumber:      entity.LicensePlateNumber,
		State:                   stateName(states, entity.StateID),
		RegistrationNumber:      entity.RegistrationNumber,
		RegistrationExpiry:      entity.RegistrationExpiry,
		FuelType:                string(entity.FuelType),
		IFTAQualified:           entity.IFTAQualified,
		ExternalID:              entity.ExternalID,
	}
}

var tractorRecord = &masterRecord[tractor.Tractor]{
	kind:     tractorRecordEntity,
	resource: permission.ResourceTractor,
	entity:   tractorRecordEntity,
	idParam:  paramTractorID,
	supplier: "from list_tractors",
	label:    func(entity *tractor.Tractor) string { return entity.Code },
	id:       func(entity *tractor.Tractor) pulid.ID { return entity.ID },
	version:  func(entity *tractor.Tractor) int64 { return entity.Version },
	detach: func(entity *tractor.Tractor) {
		entity.EquipmentType = nil
		entity.EquipmentManufacturer = nil
		entity.FleetCode = nil
		entity.State = nil
		entity.PrimaryWorker = nil
		entity.SecondaryWorker = nil
		entity.OwnerWorker = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	stateIDs: func(entity *tractor.Tractor) []pulid.ID { return []pulid.ID{entity.StateID} },
	view:     tractorViewOf,
	options:  []toolpreview.Option{equipmentRefs, equipmentDates},
}

func unitFields[T any](at unitAccess[T]) []masterField[T] {
	return []masterField[T]{
		masterText(mdCode, "The unit number, unique in this organization.", mdMaxUnitText, true,
			at.code),
		masterEnum(mdStatus, "Whether it can be dispatched. "+equipmentStatusNote+
			" Defaults to Available.", equipmentStatuses, at.status),
		masterID(mdEquipmentTypeID, permission.ResourceEquipmentType,
			"Its equipment type, "+equipmentSupplierNote+".", at.equipmentType),
		masterID(mdManufacturerID, permission.ResourceEquipmentManufacturer,
			"Who made it, "+manufacturerSupplier+".", at.manufacturer),
		masterOptionalID(mdFleetCodeID, permission.ResourceFleetCode,
			"The fleet it belongs to, "+fleetCodeSupplier+".", at.fleetCode),
		masterText(mdMake, "The make.", mdMaxUnitText, false, at.make),
		masterText(mdModel, "The model.", mdMaxUnitText, false, at.model),
		masterOptionalInt(mdYear, "The model year.", mdMinYear, mdMaxYear, at.year),
		masterUpperText(mdVIN, "The 17-character VIN.", mdMaxUnitText, at.vin),
		masterText(mdLicensePlate, "The license plate.", mdMaxUnitText, false, at.plate),
		masterText(mdRegistrationNumber, "The registration number.", mdMaxUnitText, false,
			at.registration),
		masterDay(mdRegistrationExpiry, "When the registration expires.", at.expiry),
		masterText(mdExternalID, "Its id in another system, such as a telematics provider.",
			mdMaxExternalID, false, at.externalID),
	}
}

type unitAccess[T any] struct {
	code          func(*T) *string
	status        func(*T) *domaintypes.EquipmentStatus
	equipmentType func(*T) *pulid.ID
	manufacturer  func(*T) *pulid.ID
	fleetCode     func(*T) *pulid.ID
	make          func(*T) *string
	model         func(*T) *string
	year          func(*T) **int
	vin           func(*T) *string
	plate         func(*T) *string
	registration  func(*T) *string
	expiry        func(*T) **int64
	externalID    func(*T) *string
}

func tractorFields() []masterField[tractor.Tractor] {
	fields := unitFields(unitAccess[tractor.Tractor]{
		code:          func(t *tractor.Tractor) *string { return &t.Code },
		status:        func(t *tractor.Tractor) *domaintypes.EquipmentStatus { return &t.Status },
		equipmentType: func(t *tractor.Tractor) *pulid.ID { return &t.EquipmentTypeID },
		manufacturer:  func(t *tractor.Tractor) *pulid.ID { return &t.EquipmentManufacturerID },
		fleetCode:     func(t *tractor.Tractor) *pulid.ID { return &t.FleetCodeID },
		make:          func(t *tractor.Tractor) *string { return &t.Make },
		model:         func(t *tractor.Tractor) *string { return &t.Model },
		year:          func(t *tractor.Tractor) **int { return &t.Year },
		vin:           func(t *tractor.Tractor) *string { return &t.Vin },
		plate:         func(t *tractor.Tractor) *string { return &t.LicensePlateNumber },
		registration:  func(t *tractor.Tractor) *string { return &t.RegistrationNumber },
		expiry:        func(t *tractor.Tractor) **int64 { return &t.RegistrationExpiry },
		externalID:    func(t *tractor.Tractor) *string { return &t.ExternalID },
	})

	return append(fields,
		masterID(paramPrimaryWorkerID, permission.ResourceWorker,
			"The driver who runs it, from search_worker or list_workers.",
			func(t *tractor.Tractor) *pulid.ID { return &t.PrimaryWorkerID }),
		masterOptionalID(paramSecondaryWorkerID, permission.ResourceWorker,
			"A team driver, from search_worker or list_workers.",
			func(t *tractor.Tractor) *pulid.ID { return &t.SecondaryWorkerID }),
		masterState(mdState, "The state it is registered in.", false,
			func(t *tractor.Tractor) *pulid.ID { return &t.StateID }),
		masterEnum(paramFuelType, "The fuel it burns; its miles land on this fuel's lines of "+
			"an IFTA return. Defaults to Diesel.", fuelTypes,
			func(t *tractor.Tractor) *domaintypes.IFTAFuelType { return &t.FuelType }),
		masterBool("iftaQualified", "Whether its miles and fuel enter the IFTA return. "+
			"Defaults to true.", func(t *tractor.Tractor) *bool { return &t.IFTAQualified }),
	)
}

func newCreateTractorTool(tractors tractorKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[tractor.Tractor]{
		record: tractorRecord,
		name:   "create_tractor",
		description: "Add a tractor to the fleet: its unit number, driver, equipment type, " +
			"manufacturer, make, model, VIN, registration and fuel. Check list_tractors " +
			"first; it is Available unless another status is given.",
		rationale: "Adds a tractor inside Trenova that dispatch can then assign; nothing is " +
			"sent and it can be marked Sold or out of service.",
		fields: tractorFields(),
		required: []string{
			mdCode, paramPrimaryWorkerID, mdEquipmentTypeID, mdManufacturerID,
		},
		searchTerms: []string{"new tractor", "add truck", "power unit", "new unit"},
		policy:      mdInternalAsk,
		states:      states,
		fresh: func(tenant pagination.TenantInfo) *tractor.Tractor {
			return &tractor.Tractor{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.EquipmentStatusAvailable,
				FuelType:       domaintypes.IFTAFuelTypeDiesel,
				IFTAQualified:  true,
				OwnershipType:  domaintypes.OwnershipTypeCompanyOwned,
			}
		},
		plan:   tractors.PlanCreate,
		create: tractors.Create,
	})
}

func newUpdateTractorTool(tractors tractorKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[tractor.Tractor]{
		record: tractorRecord,
		name:   "update_tractor",
		description: "Change a tractor: its driver, team driver, equipment type, fleet, " +
			"make, model, VIN, plate, registration or fuel. Fields left out keep their " +
			"value. Use update_tractor_status for many units and locate_tractor to move one.",
		rationale:   "Changes a tractor inside Trenova; nothing is sent, and a later change puts it back.",
		fields:      tractorFields(),
		searchTerms: []string{"edit tractor", "reassign truck driver", "registration renewal"},
		policy:      mdInternalAsk,
		states:      states,
		get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*tractor.Tractor, error) {
			return tractors.Get(ctx, repositories.GetTractorByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   tractors.PlanUpdate,
		update: tractors.Update,
	})
}

type trailerView struct {
	Code                    string `json:"code"`
	Status                  string `json:"status"`
	EquipmentTypeID         string `json:"equipmentTypeId"`
	EquipmentManufacturerID string `json:"equipmentManufacturerId"`
	FleetCodeID             string `json:"fleetCodeId,omitempty"`
	Make                    string `json:"make,omitempty"`
	Model                   string `json:"model,omitempty"`
	Year                    *int   `json:"year,omitempty"`
	Vin                     string `json:"vin,omitempty"`
	LicensePlateNumber      string `json:"licensePlateNumber,omitempty"`
	RegistrationState       string `json:"registrationState,omitempty"`
	RegistrationNumber      string `json:"registrationNumber,omitempty"`
	RegistrationExpiry      *int64 `json:"registrationExpiry,omitempty"`
	MaxLoadWeight           *int   `json:"maxLoadWeight,omitempty"`
	LastInspectionDate      *int64 `json:"lastInspectionDate,omitempty"`
	ExternalID              string `json:"externalId,omitempty"`
}

func trailerViewOf(entity *trailer.Trailer, states map[pulid.ID]string) any {
	return &trailerView{
		Code:                    entity.Code,
		Status:                  string(entity.Status),
		EquipmentTypeID:         idText(entity.EquipmentTypeID),
		EquipmentManufacturerID: idText(entity.EquipmentManufacturerID),
		FleetCodeID:             idText(entity.FleetCodeID),
		Make:                    entity.Make,
		Model:                   entity.Model,
		Year:                    entity.Year,
		Vin:                     entity.Vin,
		LicensePlateNumber:      entity.LicensePlateNumber,
		RegistrationState:       stateName(states, entity.RegistrationStateID),
		RegistrationNumber:      entity.RegistrationNumber,
		RegistrationExpiry:      entity.RegistrationExpiry,
		MaxLoadWeight:           entity.MaxLoadWeight,
		LastInspectionDate:      entity.LastInspectionDate,
		ExternalID:              entity.ExternalID,
	}
}

var trailerRecord = &masterRecord[trailer.Trailer]{
	kind:     trailerRecordEntity,
	resource: permission.ResourceTrailer,
	entity:   trailerRecordEntity,
	idParam:  paramTrailerID,
	supplier: "from list_trailers",
	label:    func(entity *trailer.Trailer) string { return entity.Code },
	id:       func(entity *trailer.Trailer) pulid.ID { return entity.ID },
	version:  func(entity *trailer.Trailer) int64 { return entity.Version },
	detach: func(entity *trailer.Trailer) {
		entity.EquipmentType = nil
		entity.EquipmentManufacturer = nil
		entity.FleetCode = nil
		entity.RegistrationState = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	stateIDs: func(entity *trailer.Trailer) []pulid.ID {
		return []pulid.ID{entity.RegistrationStateID}
	},
	view:    trailerViewOf,
	options: []toolpreview.Option{equipmentRefs, equipmentDates},
}

func trailerFields() []masterField[trailer.Trailer] {
	fields := unitFields(unitAccess[trailer.Trailer]{
		code:          func(t *trailer.Trailer) *string { return &t.Code },
		status:        func(t *trailer.Trailer) *domaintypes.EquipmentStatus { return &t.Status },
		equipmentType: func(t *trailer.Trailer) *pulid.ID { return &t.EquipmentTypeID },
		manufacturer:  func(t *trailer.Trailer) *pulid.ID { return &t.EquipmentManufacturerID },
		fleetCode:     func(t *trailer.Trailer) *pulid.ID { return &t.FleetCodeID },
		make:          func(t *trailer.Trailer) *string { return &t.Make },
		model:         func(t *trailer.Trailer) *string { return &t.Model },
		year:          func(t *trailer.Trailer) **int { return &t.Year },
		vin:           func(t *trailer.Trailer) *string { return &t.Vin },
		plate:         func(t *trailer.Trailer) *string { return &t.LicensePlateNumber },
		registration:  func(t *trailer.Trailer) *string { return &t.RegistrationNumber },
		expiry:        func(t *trailer.Trailer) **int64 { return &t.RegistrationExpiry },
		externalID:    func(t *trailer.Trailer) *string { return &t.ExternalID },
	})

	return append(fields,
		masterState(paramRegistrationState, "The state it is registered in.", false,
			func(t *trailer.Trailer) *pulid.ID { return &t.RegistrationStateID }),
		masterOptionalInt(paramMaxLoadWeight, "The most it may carry, in pounds.", 0,
			mdMaxLoadWeight, func(t *trailer.Trailer) **int { return &t.MaxLoadWeight }),
		masterDay(paramLastInspection, "When it was last inspected.",
			func(t *trailer.Trailer) **int64 { return &t.LastInspectionDate }),
	)
}

func newCreateTrailerTool(trailers trailerKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[trailer.Trailer]{
		record: trailerRecord,
		name:   "create_trailer",
		description: "Add a trailer to the fleet: its unit number, equipment type, " +
			"manufacturer, make, model, VIN, registration, load limit and last inspection. " +
			"Check list_trailers first; it is Available unless another status is given.",
		rationale: "Adds a trailer inside Trenova that dispatch can then assign; nothing is " +
			"sent and it can be marked Sold or out of service.",
		fields:      trailerFields(),
		required:    []string{mdCode, mdEquipmentTypeID, mdManufacturerID},
		searchTerms: []string{"new trailer", "add trailer", "reefer", "dry van"},
		policy:      mdInternalAsk,
		states:      states,
		fresh: func(tenant pagination.TenantInfo) *trailer.Trailer {
			return &trailer.Trailer{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.EquipmentStatusAvailable,
				OwnershipType:  domaintypes.OwnershipTypeCompanyOwned,
			}
		},
		plan:   trailers.PlanCreate,
		create: trailers.Create,
	})
}

func newUpdateTrailerTool(trailers trailerKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[trailer.Trailer]{
		record: trailerRecord,
		name:   "update_trailer",
		description: "Change a trailer: its equipment type, fleet, make, model, VIN, plate, " +
			"registration, load limit or last inspection. Fields left out keep their value. " +
			"Use update_trailer_status for many units and locate_trailer to move one.",
		rationale:   "Changes a trailer inside Trenova; nothing is sent, and a later change puts it back.",
		fields:      trailerFields(),
		searchTerms: []string{"edit trailer", "trailer inspection", "registration renewal"},
		policy:      mdInternalAsk,
		states:      states,
		get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*trailer.Trailer, error) {
			return trailers.Get(ctx, repositories.GetTrailerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		plan:   trailers.PlanUpdate,
		update: trailers.Update,
	})
}

type equipmentPlace struct {
	Location string `json:"location"`
}

func placeOf(loc *location.Location, fallback pulid.ID) *equipmentPlace {
	if loc != nil && loc.Name != "" {
		return &equipmentPlace{Location: loc.Name}
	}
	if fallback.IsNil() {
		return &equipmentPlace{}
	}

	return &equipmentPlace{Location: fallback.String()}
}

func currentPlace(current *equipmentcontinuity.EquipmentContinuity) pulid.ID {
	if current == nil {
		return pulid.Nil
	}

	return current.CurrentLocationID
}

type locateRequest struct {
	unitID     pulid.ID
	locationID pulid.ID
}

func locateRequestFrom(
	unitKey string,
) func(*serviceports.ToolExecuteParams) (*locateRequest, error) {
	return func(params *serviceports.ToolExecuteParams) (*locateRequest, error) {
		unitID, err := requirePulid(params.Params, unitKey)
		if err != nil {
			return nil, err
		}
		locationID, err := requirePulid(params.Params, paramNewLocationID)
		if err != nil {
			return nil, err
		}

		return &locateRequest{unitID: unitID, locationID: locationID}, nil
	}
}

func locateProperties(
	unitKey, unit, supplier string,
	resource permission.Resource,
) map[string]any {
	return map[string]any{
		unitKey: agenttoolschema.RecordIDText(resource,
			fmt.Sprintf("The %s, %s. Never guess one.", unit, supplier)),
		paramNewLocationID: agenttoolschema.RecordIDText(permission.ResourceLocation,
			"Where it is now, from list_locations. Never guess one."),
	}
}

func newLocateTractorTool(tractors tractorKeeper) serviceports.AgentTool {
	build := func(req *locateRequest, params *serviceports.ToolExecuteParams) *repositories.LocateTractorRequest {
		return &repositories.LocateTractorRequest{
			TenantInfo:    tenantFrom(*params),
			TractorID:     req.unitID,
			NewLocationID: req.locationID,
		}
	}

	return newReportingReceivableTool(&receivableSpec{
		name: "locate_tractor",
		description: "Record where a tractor actually is when it was moved outside a " +
			"shipment, so dispatch plans from the right place. A tractor on a move in " +
			"progress cannot be located.",
		resource:    permission.ResourceTractor,
		operation:   permission.OpUpdate,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		artifact:    tractorRecordEntity,
		rationale: "Moves where Trenova thinks a tractor is; nothing is sent and locating it " +
			"again moves it back.",
		properties: locateProperties(paramTractorID, tractorRecordEntity, "from list_tractors",
			permission.ResourceTractor),
		required:    []string{paramTractorID, paramNewLocationID},
		searchTerms: []string{"locate tractor", "relocate truck", "move tractor", "yard"},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramTractorID, permission.ResourceTractor)
		},
	}, receivablePlan[*locateRequest, *tractorservice.LocatePlan]{
		request: locateRequestFrom(paramTractorID),
		plan: func(
			ctx context.Context,
			req *locateRequest,
			params *serviceports.ToolExecuteParams,
		) (*tractorservice.LocatePlan, error) {
			return tractors.PlanLocate(ctx, build(req, params))
		},
		refused: func(*locateRequest) string { return "Would locate a tractor." },
		render: func(req *locateRequest, plan *tractorservice.LocatePlan) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				tractorRecord.record(plan.Tractor),
				placeOf(nil, currentPlace(plan.Current)),
				placeOf(plan.Location, req.locationID),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would record tractor %s at %s.",
				plan.Tractor.Code, placeOf(plan.Location, req.locationID).Location), change), nil
		},
		run: func(
			ctx context.Context,
			req *locateRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if _, err := tractors.Locate(ctx, build(req, params)); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "located",
				Kind:   tractorRecordEntity,
				IDs:    map[string]string{paramTractorID: req.unitID.String()},
				Record: recordOf(tractorRecordEntity, req.unitID),
			}, nil
		},
	})
}

func newLocateTrailerTool(trailers trailerKeeper) serviceports.AgentTool {
	build := func(req *locateRequest, params *serviceports.ToolExecuteParams) *repositories.LocateTrailerRequest {
		return &repositories.LocateTrailerRequest{
			TenantInfo:    tenantFrom(*params),
			TrailerID:     req.unitID,
			NewLocationID: req.locationID,
		}
	}

	return newReportingReceivableTool(&receivableSpec{
		name: "locate_trailer",
		description: "Record where a trailer actually is when it was moved outside a " +
			"shipment. It adds a completed empty move to the shipment that last used it and " +
			"re-rates that shipment, so a person always approves it.",
		resource:    permission.ResourceTrailer,
		operation:   permission.OpUpdate,
		egress:      agent.EgressMoney,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierPropose,
		personOnly:  true,
		artifact:    trailerRecordEntity,
		rationale: "Adds an empty move to the trailer's last shipment and re-rates it, which " +
			"can change what that customer is charged, so it runs only on a person's approval.",
		properties: locateProperties(paramTrailerID, trailerRecordEntity, "from list_trailers",
			permission.ResourceTrailer),
		required:    []string{paramTrailerID, paramNewLocationID},
		searchTerms: []string{"locate trailer", "trailer location", "move trailer", "drop yard"},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramTrailerID, permission.ResourceTrailer)
		},
	}, receivablePlan[*locateRequest, *trailerservice.LocatePlan]{
		request: locateRequestFrom(paramTrailerID),
		plan: func(
			ctx context.Context,
			req *locateRequest,
			params *serviceports.ToolExecuteParams,
		) (*trailerservice.LocatePlan, error) {
			return trailers.PlanLocate(ctx, build(req, params), params.Actor)
		},
		refused: func(*locateRequest) string { return "Would locate a trailer." },
		render:  renderTrailerLocate,
		run: func(
			ctx context.Context,
			req *locateRequest,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			if _, err := trailers.Locate(ctx, build(req, params), params.Actor); err != nil {
				return nil, err
			}

			return &agent.ToolExecutionResult{
				Action: "located",
				Kind:   trailerRecordEntity,
				IDs:    map[string]string{paramTrailerID: req.unitID.String()},
				Record: recordOf(trailerRecordEntity, req.unitID),
			}, nil
		},
	})
}

func renderTrailerLocate(
	req *locateRequest,
	plan *trailerservice.LocatePlan,
) (*agent.ToolPreview, error) {
	moved, err := toolpreview.Changed(
		trailerRecord.record(plan.Trailer),
		placeOf(nil, currentPlace(plan.Current)),
		placeOf(plan.Location, req.locationID),
	)
	if err != nil {
		return nil, err
	}

	shipmentChange := toolpreview.Money(toolpreview.Record{
		Resource: permission.ResourceShipment,
		ID:       plan.Previous.ID,
		Label:    plan.Previous.ProNumber,
		Version:  pinnedVersion(plan.Previous.Version),
	}, toolpreview.MoneyBlock(money.DefaultCurrencyCode, agent.MoneyLine{
		Label:  "Total charge",
		Before: plan.Previous.TotalChargeAmount,
		After:  plan.Updated.TotalChargeAmount,
	}))

	return toolpreview.Build(fmt.Sprintf(
		"Would record trailer %s at %s, adding a completed empty move to shipment %s and "+
			"re-rating it.",
		plan.Trailer.Code, placeOf(plan.Location, req.locationID).Location,
		plan.Previous.ProNumber,
	), moved, shipmentChange), nil
}
