package equipmenttools

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/locationtools"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/masterdatakit"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/app/trailerservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/equipmentcontinuity"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/trailer"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/trailerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
)

const tractorRecordEntity = "tractor"

const trailerRecordEntity = "trailer"

const paramTrailerID = "trailerId"

const paramNewLocationID = "newLocationId"

const paramPrimaryWorkerID = "primaryWorkerId"

const paramSecondaryWorkerID = "secondaryWorkerId"

const paramRegistrationState = "registrationState"

const paramMaxLoadWeight = "maxLoadWeight"

const paramLastInspection = "lastInspectionDate"

const mdMaxLoadWeight = 200_000

const equipmentSupplierNote = "from equipment_list_types"

const manufacturerSupplier = "from equipment_list_manufacturers"

const fleetCodeSupplier = "from masterdata_list_fleet_codes"

var equipmentDates = toolpreview.Types(map[string]assistantartifact.DisplayType{
	masterdatakit.MdRegistrationExpiry: assistantartifact.DisplayDate,
	paramLastInspection:                assistantartifact.DisplayDate,
})

var equipmentRefs = toolpreview.WithRefs(map[string]permission.Resource{
	paramPrimaryWorkerID:            permission.ResourceWorker,
	paramSecondaryWorkerID:          permission.ResourceWorker,
	masterdatakit.MdEquipmentTypeID: permission.ResourceEquipmentType,
	masterdatakit.MdManufacturerID:  permission.ResourceEquipmentManufacturer,
	masterdatakit.MdFleetCodeID:     permission.ResourceFleetCode,
})

type TractorKeeper interface {
	Get(ctx context.Context, req tractorports.GetTractorByIDRequest) (*tractor.Tractor, error)
	PlanCreate(ctx context.Context, entity *tractor.Tractor) (*tractor.Tractor, error)
	Create(
		ctx context.Context,
		entity *tractor.Tractor,
		actor *permissionports.RequestActor,
	) (*tractor.Tractor, error)
	PlanUpdate(
		ctx context.Context,
		entity *tractor.Tractor,
	) (*kernelports.RecordChange[tractor.Tractor], error)
	Update(
		ctx context.Context,
		entity *tractor.Tractor,
		actor *permissionports.RequestActor,
	) (*tractor.Tractor, error)
	PlanLocate(
		ctx context.Context,
		req *tractorports.LocateTractorRequest,
	) (*tractorservice.LocatePlan, error)
	Locate(
		ctx context.Context,
		req *tractorports.LocateTractorRequest,
	) (*equipmentcontinuity.EquipmentContinuity, error)
}

type TrailerKeeper interface {
	Get(ctx context.Context, req trailerports.GetTrailerByIDRequest) (*trailer.Trailer, error)
	PlanCreate(ctx context.Context, entity *trailer.Trailer) (*trailer.Trailer, error)
	Create(
		ctx context.Context,
		entity *trailer.Trailer,
		actor *permissionports.RequestActor,
	) (*trailer.Trailer, error)
	PlanUpdate(
		ctx context.Context,
		entity *trailer.Trailer,
	) (*kernelports.RecordChange[trailer.Trailer], error)
	Update(
		ctx context.Context,
		entity *trailer.Trailer,
		actor *permissionports.RequestActor,
	) (*trailer.Trailer, error)
	PlanLocate(
		ctx context.Context,
		req *trailerports.LocateTrailerRequest,
		actor *permissionports.RequestActor,
	) (*trailerservice.LocatePlan, error)
	Locate(
		ctx context.Context,
		req *trailerports.LocateTrailerRequest,
		actor *permissionports.RequestActor,
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

func tractorViewOf(entity *tractor.Tractor, states map[pulid.ID]string) any {
	return &tractorView{
		Code:                    entity.Code,
		Status:                  string(entity.Status),
		PrimaryWorkerID:         masterdatakit.IDText(entity.PrimaryWorkerID),
		SecondaryWorkerID:       masterdatakit.IDText(entity.SecondaryWorkerID),
		EquipmentTypeID:         masterdatakit.IDText(entity.EquipmentTypeID),
		EquipmentManufacturerID: masterdatakit.IDText(entity.EquipmentManufacturerID),
		FleetCodeID:             masterdatakit.IDText(entity.FleetCodeID),
		Make:                    entity.Make,
		Model:                   entity.Model,
		Year:                    entity.Year,
		Vin:                     entity.Vin,
		LicensePlateNumber:      entity.LicensePlateNumber,
		State:                   masterdatakit.StateName(states, entity.StateID),
		RegistrationNumber:      entity.RegistrationNumber,
		RegistrationExpiry:      entity.RegistrationExpiry,
		FuelType:                string(entity.FuelType),
		IFTAQualified:           entity.IFTAQualified,
		ExternalID:              entity.ExternalID,
	}
}

var tractorRecord = &masterdatakit.Record[tractor.Tractor]{
	Kind:     tractorRecordEntity,
	Resource: permission.ResourceTractor,
	Entity:   tractorRecordEntity,
	IDParam:  toolparams.ParamTractorID,
	Lister:   "equipment_list_tractors",
	Label:    func(entity *tractor.Tractor) string { return entity.Code },
	ID:       func(entity *tractor.Tractor) pulid.ID { return entity.ID },
	Version:  func(entity *tractor.Tractor) int64 { return entity.Version },
	Detach: func(entity *tractor.Tractor) {
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
	StateIDs: func(entity *tractor.Tractor) []pulid.ID { return []pulid.ID{entity.StateID} },
	View:     tractorViewOf,
	Options:  []toolpreview.Option{equipmentRefs, equipmentDates},
}

func unitFields[T any](at unitAccess[T]) []masterdatakit.Field[T] {
	return []masterdatakit.Field[T]{
		masterdatakit.Text(
			masterdatakit.MdCode,
			"The unit number, unique in this organization.",
			masterdatakit.MdMaxUnitText,
			true,
			at.code,
		),
		masterdatakit.Enum(
			masterdatakit.MdStatus,
			"Whether it can be dispatched. "+equipmentStatusNote+
				" Defaults to Available.",
			equipmentStatuses,
			at.status,
		),
		masterdatakit.ID(masterdatakit.MdEquipmentTypeID, permission.ResourceEquipmentType,
			"Its equipment type, "+equipmentSupplierNote+".", at.equipmentType),
		masterdatakit.ID(masterdatakit.MdManufacturerID, permission.ResourceEquipmentManufacturer,
			"Who made it, "+manufacturerSupplier+".", at.manufacturer),
		masterdatakit.OptionalID(masterdatakit.MdFleetCodeID, permission.ResourceFleetCode,
			"The fleet it belongs to, "+fleetCodeSupplier+".", at.fleetCode),
		masterdatakit.Text(
			masterdatakit.MdMake,
			"The make.",
			masterdatakit.MdMaxUnitText,
			false,
			at.make,
		),
		masterdatakit.Text(
			masterdatakit.MdModel,
			"The model.",
			masterdatakit.MdMaxUnitText,
			false,
			at.model,
		),
		masterdatakit.OptionalInt(
			masterdatakit.MdYear,
			"The model year.",
			masterdatakit.MdMinYear,
			masterdatakit.MdMaxYear,
			at.year,
		),
		masterdatakit.UpperText(
			masterdatakit.MdVIN,
			"The 17-character VIN.",
			masterdatakit.MdMaxUnitText,
			at.vin,
		),
		masterdatakit.Text(
			masterdatakit.MdLicensePlate,
			"The license plate.",
			masterdatakit.MdMaxUnitText,
			false,
			at.plate,
		),
		masterdatakit.Text(
			masterdatakit.MdRegistrationNumber,
			"The registration number.",
			masterdatakit.MdMaxUnitText,
			false,
			at.registration,
		),
		masterdatakit.Day(
			masterdatakit.MdRegistrationExpiry,
			"When the registration expires.",
			at.expiry,
		),
		masterdatakit.Text(
			masterdatakit.MdExternalID,
			"Its id in another system, such as a telematics provider.",
			masterdatakit.MdMaxExternalID,
			false,
			at.externalID,
		),
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

func tractorFields() []masterdatakit.Field[tractor.Tractor] {
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

	return append(
		fields,
		masterdatakit.ID(paramPrimaryWorkerID, permission.ResourceWorker,
			"The driver who runs it, from worker_list.",
			func(t *tractor.Tractor) *pulid.ID { return &t.PrimaryWorkerID }),
		masterdatakit.OptionalID(paramSecondaryWorkerID, permission.ResourceWorker,
			"A team driver, from worker_list.",
			func(t *tractor.Tractor) *pulid.ID { return &t.SecondaryWorkerID }),
		masterdatakit.State(masterdatakit.MdState, "The state it is registered in.", false,
			func(t *tractor.Tractor) *pulid.ID { return &t.StateID }),
		masterdatakit.Enum(
			toolparams.ParamFuelType,
			"The fuel it burns; its miles land on this fuel's lines of "+
				"an IFTA return. Defaults to Diesel.",
			toolparams.FuelTypes,
			func(t *tractor.Tractor) *domaintypes.IFTAFuelType { return &t.FuelType },
		),
		masterdatakit.Bool("iftaQualified", "Whether its miles and fuel enter the IFTA return. "+
			"Defaults to true.", func(t *tractor.Tractor) *bool { return &t.IFTAQualified }),
	)
}

func NewCreateTractorTool(
	tractors TractorKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewCreateTool(&masterdatakit.CreateSpec[tractor.Tractor]{
		Record: tractorRecord,
		Name:   "equipment_create_tractor",
		Description: "Add a tractor to the fleet: its unit number, driver, equipment type, " +
			"manufacturer, make, model, VIN, registration and fuel. Check equipment_list_tractors " +
			"first; it is Available unless another status is given.",
		Rationale: "Adds a tractor inside Trenova that dispatch can then assign; nothing is " +
			"sent and it can be marked Sold or out of service.",
		Fields: tractorFields(),
		Required: []string{
			masterdatakit.MdCode,
			paramPrimaryWorkerID,
			masterdatakit.MdEquipmentTypeID,
			masterdatakit.MdManufacturerID,
		},
		SearchTerms: []string{"new tractor", "add truck", "power unit", "new unit"},
		Policy:      masterdatakit.MdInternalAsk,
		States:      states,
		Fresh: func(tenant pagination.TenantInfo) *tractor.Tractor {
			return &tractor.Tractor{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.EquipmentStatusAvailable,
				FuelType:       domaintypes.IFTAFuelTypeDiesel,
				IFTAQualified:  true,
				OwnershipType:  domaintypes.OwnershipTypeCompanyOwned,
			}
		},
		Plan:   tractors.PlanCreate,
		Create: tractors.Create,
	})
}

func NewUpdateTractorTool(
	tractors TractorKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(&masterdatakit.UpdateSpec[tractor.Tractor]{
		Record: tractorRecord,
		Name:   "equipment_update_tractor",
		Description: "Change a tractor: its driver, team driver, equipment type, fleet, " +
			"make, model, VIN, plate, registration or fuel. Fields left out keep their " +
			"value. Use equipment_set_tractor_status for many units and equipment_set_tractor_location to move one.",
		Rationale:   "Changes a tractor inside Trenova; nothing is sent, and a later change puts it back.",
		Fields:      tractorFields(),
		SearchTerms: []string{"edit tractor", "reassign truck driver", "registration renewal"},
		Policy:      masterdatakit.MdInternalAsk,
		States:      states,
		Get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*tractor.Tractor, error) {
			return tractors.Get(ctx, tractorports.GetTractorByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		Plan:   tractors.PlanUpdate,
		Update: tractors.Update,
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
		EquipmentTypeID:         masterdatakit.IDText(entity.EquipmentTypeID),
		EquipmentManufacturerID: masterdatakit.IDText(entity.EquipmentManufacturerID),
		FleetCodeID:             masterdatakit.IDText(entity.FleetCodeID),
		Make:                    entity.Make,
		Model:                   entity.Model,
		Year:                    entity.Year,
		Vin:                     entity.Vin,
		LicensePlateNumber:      entity.LicensePlateNumber,
		RegistrationState:       masterdatakit.StateName(states, entity.RegistrationStateID),
		RegistrationNumber:      entity.RegistrationNumber,
		RegistrationExpiry:      entity.RegistrationExpiry,
		MaxLoadWeight:           entity.MaxLoadWeight,
		LastInspectionDate:      entity.LastInspectionDate,
		ExternalID:              entity.ExternalID,
	}
}

var trailerRecord = &masterdatakit.Record[trailer.Trailer]{
	Kind:     trailerRecordEntity,
	Resource: permission.ResourceTrailer,
	Entity:   trailerRecordEntity,
	IDParam:  paramTrailerID,
	Lister:   "equipment_list_trailers",
	Label:    func(entity *trailer.Trailer) string { return entity.Code },
	ID:       func(entity *trailer.Trailer) pulid.ID { return entity.ID },
	Version:  func(entity *trailer.Trailer) int64 { return entity.Version },
	Detach: func(entity *trailer.Trailer) {
		entity.EquipmentType = nil
		entity.EquipmentManufacturer = nil
		entity.FleetCode = nil
		entity.RegistrationState = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	StateIDs: func(entity *trailer.Trailer) []pulid.ID {
		return []pulid.ID{entity.RegistrationStateID}
	},
	View:    trailerViewOf,
	Options: []toolpreview.Option{equipmentRefs, equipmentDates},
}

func trailerFields() []masterdatakit.Field[trailer.Trailer] {
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
		masterdatakit.State(paramRegistrationState, "The state it is registered in.", false,
			func(t *trailer.Trailer) *pulid.ID { return &t.RegistrationStateID }),
		masterdatakit.OptionalInt(paramMaxLoadWeight, "The most it may carry, in pounds.", 0,
			mdMaxLoadWeight, func(t *trailer.Trailer) **int { return &t.MaxLoadWeight }),
		masterdatakit.Day(paramLastInspection, "When it was last inspected.",
			func(t *trailer.Trailer) **int64 { return &t.LastInspectionDate }),
	)
}

func NewCreateTrailerTool(
	trailers TrailerKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewCreateTool(&masterdatakit.CreateSpec[trailer.Trailer]{
		Record: trailerRecord,
		Name:   "equipment_create_trailer",
		Description: "Add a trailer to the fleet: its unit number, equipment type, " +
			"manufacturer, make, model, VIN, registration, load limit and last inspection. " +
			"Check equipment_list_trailers first; it is Available unless another status is given.",
		Rationale: "Adds a trailer inside Trenova that dispatch can then assign; nothing is " +
			"sent and it can be marked Sold or out of service.",
		Fields: trailerFields(),
		Required: []string{
			masterdatakit.MdCode,
			masterdatakit.MdEquipmentTypeID,
			masterdatakit.MdManufacturerID,
		},
		SearchTerms: []string{"new trailer", "add trailer", "reefer", "dry van"},
		Policy:      masterdatakit.MdInternalAsk,
		States:      states,
		Fresh: func(tenant pagination.TenantInfo) *trailer.Trailer {
			return &trailer.Trailer{
				OrganizationID: tenant.OrgID,
				BusinessUnitID: tenant.BuID,
				Status:         domaintypes.EquipmentStatusAvailable,
				OwnershipType:  domaintypes.OwnershipTypeCompanyOwned,
			}
		},
		Plan:   trailers.PlanCreate,
		Create: trailers.Create,
	})
}

func NewUpdateTrailerTool(
	trailers TrailerKeeper,
	states masterdatakit.StateLookup,
) agenttoolports.AgentTool {
	return masterdatakit.NewUpdateTool(&masterdatakit.UpdateSpec[trailer.Trailer]{
		Record: trailerRecord,
		Name:   "equipment_update_trailer",
		Description: "Change a trailer: its equipment type, fleet, make, model, VIN, plate, " +
			"registration, load limit or last inspection. Fields left out keep their value. " +
			"Use equipment_set_trailer_status for many units and equipment_set_trailer_location to move one.",
		Rationale:   "Changes a trailer inside Trenova; nothing is sent, and a later change puts it back.",
		Fields:      trailerFields(),
		SearchTerms: []string{"edit trailer", "trailer inspection", "registration renewal"},
		Policy:      masterdatakit.MdInternalAsk,
		States:      states,
		Get: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			id pulid.ID,
		) (*trailer.Trailer, error) {
			return trailers.Get(ctx, trailerports.GetTrailerByIDRequest{
				ID:         id,
				TenantInfo: tenant,
			})
		},
		Plan:   trailers.PlanUpdate,
		Update: trailers.Update,
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
) func(*agenttoolports.ToolExecuteParams) (*locateRequest, error) {
	return func(params *agenttoolports.ToolExecuteParams) (*locateRequest, error) {
		unitID, err := toolparams.RequirePulid(params.Params, unitKey)
		if err != nil {
			return nil, err
		}
		locationID, err := toolparams.RequirePulid(params.Params, paramNewLocationID)
		if err != nil {
			return nil, err
		}

		return &locateRequest{unitID: unitID, locationID: locationID}, nil
	}
}

func locateProperties(
	unitKey, unit, lister string,
	resource permission.Resource,
) map[string]any {
	return map[string]any{
		unitKey: agenttoolschema.RecordIDText(resource,
			fmt.Sprintf("The %s, from %s. Never guess one.", unit, lister)),
		paramNewLocationID: agenttoolschema.RecordIDText(permission.ResourceLocation,
			"Where it is now, from location_list. Never guess one."),
	}
}

func NewLocateTractorTool(tractors TractorKeeper) agenttoolports.AgentTool {
	build := func(req *locateRequest, params *agenttoolports.ToolExecuteParams) *tractorports.LocateTractorRequest {
		return &tractorports.LocateTractorRequest{
			TenantInfo:    toolparams.TenantFrom(*params),
			TractorID:     req.unitID,
			NewLocationID: req.locationID,
		}
	}

	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name: "equipment_set_tractor_location",
		Description: "Record where a tractor actually is when it was moved outside a " +
			"shipment, so dispatch plans from the right place. A tractor on a move in " +
			"progress cannot be located.",
		Resource:    permission.ResourceTractor,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Artifact:    tractorRecordEntity,
		Rationale: "Moves where Trenova thinks a tractor is; nothing is sent and locating it " +
			"again moves it back.",
		Properties: locateProperties(
			toolparams.ParamTractorID,
			tractorRecordEntity,
			tractorRecord.Lister,
			permission.ResourceTractor,
		),
		Required:      []string{toolparams.ParamTractorID, paramNewLocationID},
		SearchTerms:   []string{"locate tractor", "relocate truck", "move tractor", "yard"},
		Prerequisites: []string{tractorRecord.Lister, locationtools.Lister},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(
				params,
				toolparams.ParamTractorID,
				permission.ResourceTractor,
			)
		},
	}, toolspec.ReceivablePlan[*locateRequest, *tractorservice.LocatePlan]{
		Request: locateRequestFrom(toolparams.ParamTractorID),
		Plan: func(
			ctx context.Context,
			req *locateRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*tractorservice.LocatePlan, error) {
			return tractors.PlanLocate(ctx, build(req, params))
		},
		Refused: func(*locateRequest) string { return "Would locate a tractor." },
		Render: func(req *locateRequest, plan *tractorservice.LocatePlan) (*agentpreview.ToolPreview, error) {
			change, err := toolpreview.Changed(
				tractorRecord.PreviewRecord(plan.Tractor),
				placeOf(nil, currentPlace(plan.Current)),
				placeOf(plan.Location, req.locationID),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf("Would record tractor %s at %s.",
				plan.Tractor.Code, placeOf(plan.Location, req.locationID).Location), change), nil
		},
		Run: func(
			ctx context.Context,
			req *locateRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if _, err := tractors.Locate(ctx, build(req, params)); err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "located",
				Kind:   tractorRecordEntity,
				IDs:    map[string]string{toolparams.ParamTractorID: req.unitID.String()},
				Record: changepreview.RecordOf(tractorRecordEntity, req.unitID),
			}, nil
		},
	})
}

func NewLocateTrailerTool(trailers TrailerKeeper) agenttoolports.AgentTool {
	build := func(req *locateRequest, params *agenttoolports.ToolExecuteParams) *trailerports.LocateTrailerRequest {
		return &trailerports.LocateTrailerRequest{
			TenantInfo:    toolparams.TenantFrom(*params),
			TrailerID:     req.unitID,
			NewLocationID: req.locationID,
		}
	}

	return toolspec.NewReportingReceivableTool(&toolspec.ReceivableSpec{
		Name: "equipment_set_trailer_location",
		Description: "Record where a trailer actually is when it was moved outside a " +
			"shipment. It adds a completed empty move to the shipment that last used it and " +
			"re-rates that shipment, so a person always approves it.",
		Resource:    permission.ResourceTrailer,
		Operation:   permission.OpUpdate,
		Egress:      agent.EgressMoney,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierPropose,
		PersonOnly:  true,
		Artifact:    trailerRecordEntity,
		Rationale: "Adds an empty move to the trailer's last shipment and re-rates it, which " +
			"can change what that customer is charged, so it runs only on a person's approval.",
		Properties: locateProperties(paramTrailerID, trailerRecordEntity, trailerRecord.Lister,
			permission.ResourceTrailer),
		Required:      []string{paramTrailerID, paramNewLocationID},
		SearchTerms:   []string{"locate trailer", "trailer location", "move trailer", "drop yard"},
		Prerequisites: []string{trailerRecord.Lister, locationtools.Lister},
		Target: func(params map[string]any) (agenttoolports.ToolTarget, bool) {
			return toolparams.TargetOf(params, paramTrailerID, permission.ResourceTrailer)
		},
	}, toolspec.ReceivablePlan[*locateRequest, *trailerservice.LocatePlan]{
		Request: locateRequestFrom(paramTrailerID),
		Plan: func(
			ctx context.Context,
			req *locateRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*trailerservice.LocatePlan, error) {
			return trailers.PlanLocate(ctx, build(req, params), params.Actor)
		},
		Refused: func(*locateRequest) string { return "Would locate a trailer." },
		Render:  renderTrailerLocate,
		Run: func(
			ctx context.Context,
			req *locateRequest,
			params *agenttoolports.ToolExecuteParams,
		) (*agentrecord.ToolExecutionResult, error) {
			if _, err := trailers.Locate(ctx, build(req, params), params.Actor); err != nil {
				return nil, err
			}

			return &agentrecord.ToolExecutionResult{
				Action: "located",
				Kind:   trailerRecordEntity,
				IDs:    map[string]string{paramTrailerID: req.unitID.String()},
				Record: changepreview.RecordOf(trailerRecordEntity, req.unitID),
			}, nil
		},
	})
}

func renderTrailerLocate(
	req *locateRequest,
	plan *trailerservice.LocatePlan,
) (*agentpreview.ToolPreview, error) {
	moved, err := toolpreview.Changed(
		trailerRecord.PreviewRecord(plan.Trailer),
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
		Version:  changepreview.PinnedVersion(plan.Previous.Version),
	}, toolpreview.MoneyBlock(money.DefaultCurrencyCode, agentpreview.MoneyLine{
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
