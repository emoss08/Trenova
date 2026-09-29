package agenttoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	carrierPaymentMethod = "paymentMethod"
	carrierPaymentTerm   = "paymentTermDays"
	carrierRemitName     = "remitToName"
	carrierRemitLine1    = "remitAddressLine1"
	carrierRemitLine2    = "remitAddressLine2"
	carrierRemitCity     = "remitCity"
	carrierRemitState    = "remitState"
	carrierRemitPostal   = "remitPostalCode"
	carrierTaxID         = "taxId"
	carrierTaxIDType     = "taxIdType"
	carrierW9OnFile      = "w9OnFile"
	carrier1099          = "is1099Eligible"
	carrierMaxTaxID      = 20
	carrierMaxSCAC       = 4
	carrierStatusNote    = "Active carriers can be tendered freight; Inactive ones are kept " +
		"on file; DoNotUse marks a carrier nobody may book."
)

var (
	carrierStatuses   = agenttoolschema.Source("carrier.status", carrier.StatusValues())
	carrierTypes      = agenttoolschema.Source("carrier.type", carrier.TypeValues())
	carrierPayMethods = agenttoolschema.Source(
		"carrier.paymentMethod",
		carrier.PaymentMethodValues(),
	)
	carrierTaxIDTypes = agenttoolschema.Source("carrier.taxIdType", carrier.TaxIDTypeValues())
	carrierMoneyKeys  = []string{
		carrierPaymentMethod, carrierPaymentTerm, carrierRemitName, carrierRemitLine1,
		carrierRemitLine2, carrierRemitCity, carrierRemitState, carrierRemitPostal,
		carrierTaxID, carrierTaxIDType, carrierW9OnFile, carrier1099,
	}
)

type carrierKeeper interface {
	Get(ctx context.Context, req repositories.GetCarrierByIDRequest) (*carrier.Carrier, error)
	PlanCreate(ctx context.Context, entity *carrier.Carrier) (*carrier.Carrier, error)
	Create(
		ctx context.Context,
		entity *carrier.Carrier,
		actor *serviceports.RequestActor,
	) (*carrier.Carrier, error)
	PlanUpdate(
		ctx context.Context,
		entity *carrier.Carrier,
	) (*serviceports.RecordChange[carrier.Carrier], error)
	Update(
		ctx context.Context,
		entity *carrier.Carrier,
		actor *serviceports.RequestActor,
	) (*carrier.Carrier, error)
	PlanBulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateCarrierStatusRequest,
	) ([]serviceports.RecordChange[carrier.Carrier], error)
	BulkUpdateStatus(
		ctx context.Context,
		req *repositories.BulkUpdateCarrierStatusRequest,
	) ([]*carrier.Carrier, error)
}

type carrierView struct {
	Code              string `json:"code"`
	Name              string `json:"name"`
	DBAName           string `json:"dbaName,omitempty"`
	CarrierType       string `json:"carrierType"`
	Status            string `json:"status"`
	DOTNumber         string `json:"dotNumber,omitempty"`
	MCNumber          string `json:"mcNumber,omitempty"`
	SCAC              string `json:"scac,omitempty"`
	AddressLine1      string `json:"addressLine1,omitempty"`
	AddressLine2      string `json:"addressLine2,omitempty"`
	City              string `json:"city,omitempty"`
	State             string `json:"state,omitempty"`
	PostalCode        string `json:"postalCode,omitempty"`
	Phone             string `json:"phone,omitempty"`
	Email             string `json:"email,omitempty"`
	PaymentMethod     string `json:"paymentMethod"`
	PaymentTermDays   int    `json:"paymentTermDays"`
	RemitToName       string `json:"remitToName,omitempty"`
	RemitAddressLine1 string `json:"remitAddressLine1,omitempty"`
	RemitAddressLine2 string `json:"remitAddressLine2,omitempty"`
	RemitCity         string `json:"remitCity,omitempty"`
	RemitState        string `json:"remitState,omitempty"`
	RemitPostalCode   string `json:"remitPostalCode,omitempty"`
	TaxIDType         string `json:"taxIdType,omitempty"`
	W9OnFile          bool   `json:"w9OnFile"`
	Is1099Eligible    bool   `json:"is1099Eligible"`
	Notes             string `json:"notes,omitempty"`
}

func carrierViewOf(entity *carrier.Carrier, states map[pulid.ID]string) any {
	view := &carrierView{
		Code:              entity.Code,
		Name:              entity.Name,
		DBAName:           entity.DBAName,
		CarrierType:       string(entity.CarrierType),
		Status:            string(entity.Status),
		DOTNumber:         entity.DOTNumber,
		MCNumber:          entity.MCNumber,
		SCAC:              entity.SCAC,
		AddressLine1:      entity.AddressLine1,
		AddressLine2:      entity.AddressLine2,
		City:              entity.City,
		PostalCode:        entity.PostalCode,
		Phone:             entity.Phone,
		Email:             entity.Email,
		PaymentMethod:     string(entity.PaymentMethod),
		PaymentTermDays:   entity.PaymentTermDays,
		RemitToName:       entity.RemitToName,
		RemitAddressLine1: entity.RemitAddressLine1,
		RemitAddressLine2: entity.RemitAddressLine2,
		RemitCity:         entity.RemitCity,
		RemitPostalCode:   entity.RemitPostalCode,
		W9OnFile:          entity.W9OnFile,
		Is1099Eligible:    entity.Is1099Eligible,
		Notes:             entity.Notes,
		State:             stateNamePointer(states, entity.StateID),
		RemitState:        stateNamePointer(states, entity.RemitStateID),
	}
	if entity.TaxIDType != nil {
		view.TaxIDType = string(*entity.TaxIDType)
	}

	return view
}

var carrierRecord = &masterRecord[carrier.Carrier]{
	kind:     "carrier",
	resource: permission.ResourceCarrier,
	entity:   carrierRecordEntity,
	idParam:  paramCarrierID,
	idsParam: "carrierIds",
	supplier: "from list_carriers",
	label:    func(entity *carrier.Carrier) string { return entity.Name },
	id:       func(entity *carrier.Carrier) pulid.ID { return entity.ID },
	version:  func(entity *carrier.Carrier) int64 { return entity.Version },
	detach: func(entity *carrier.Carrier) {
		entity.State = nil
		entity.RemitState = nil
		entity.BusinessUnit = nil
		entity.Organization = nil
	},
	stateIDs: func(entity *carrier.Carrier) []pulid.ID {
		ids := make([]pulid.ID, 0, 2)
		for _, id := range []*pulid.ID{entity.StateID, entity.RemitStateID} {
			if id != nil {
				ids = append(ids, *id)
			}
		}

		return ids
	},
	view: carrierViewOf,
}

func carrierFields() []masterField[carrier.Carrier] {
	return []masterField[carrier.Carrier]{
		masterText(mdCode, "The carrier's short code, unique in this organization.", mdMaxCode,
			true, func(c *carrier.Carrier) *string { return &c.Code }),
		masterText(mdName, "The carrier's legal name.", mdMaxName, true,
			func(c *carrier.Carrier) *string { return &c.Name }),
		masterText("dbaName", "The name it does business as, when that differs.", mdMaxName,
			false, func(c *carrier.Carrier) *string { return &c.DBAName }),
		masterEnum("carrierType", "What kind of carrier it is. Defaults to Common.",
			carrierTypes, func(c *carrier.Carrier) *carrier.Type { return &c.CarrierType }),
		masterText(mdDOTNumber, "The USDOT number, digits only.", mdMaxNumber, false,
			func(c *carrier.Carrier) *string { return &c.DOTNumber }),
		masterText(mdMCNumber, "The MC docket number, digits only.", mdMaxNumber, false,
			func(c *carrier.Carrier) *string { return &c.MCNumber }),
		masterUpperText("scac", "The two to four letter SCAC.", carrierMaxSCAC,
			func(c *carrier.Carrier) *string { return &c.SCAC }),
		masterText(mdAddressLine1, "The street address.", mdMaxAddress, false,
			func(c *carrier.Carrier) *string { return &c.AddressLine1 }),
		masterText(mdAddressLine2, "Suite, building or unit.", mdMaxAddress, false,
			func(c *carrier.Carrier) *string { return &c.AddressLine2 }),
		masterText(mdCity, "The city.", mdMaxCity, false,
			func(c *carrier.Carrier) *string { return &c.City }),
		masterStatePointer(mdState, "The state of the street address.",
			func(c *carrier.Carrier) **pulid.ID { return &c.StateID }),
		masterText(mdPostalCode, "The ZIP code.", mdMaxPostalCode, false,
			func(c *carrier.Carrier) *string { return &c.PostalCode }),
		masterText(mdPhone, "The dispatch phone number.", mdMaxPhone, false,
			func(c *carrier.Carrier) *string { return &c.Phone }),
		masterText(mdEmail, "The dispatch email address.", mdMaxName, false,
			func(c *carrier.Carrier) *string { return &c.Email }),
		masterText(mdNotes, "Anything colleagues booking this carrier should know.",
			mdMaxNotes, false, func(c *carrier.Carrier) *string { return &c.Notes }),
		masterEnum(carrierPaymentMethod, "How the carrier is paid. Defaults to Check. "+
			"Payment details are only ever proposed and change on a person's approval.",
			carrierPayMethods,
			func(c *carrier.Carrier) *carrier.PaymentMethod { return &c.PaymentMethod }),
		masterInt(carrierPaymentTerm, "Days after the invoice the carrier is paid. "+
			"Defaults to 30.", 0, mdMaxPaymentTerm,
			func(c *carrier.Carrier) *int { return &c.PaymentTermDays }),
		masterText(carrierRemitName, "Who payment is made out to.", mdMaxName, false,
			func(c *carrier.Carrier) *string { return &c.RemitToName }),
		masterText(carrierRemitLine1, "The remit-to street address.", mdMaxAddress, false,
			func(c *carrier.Carrier) *string { return &c.RemitAddressLine1 }),
		masterText(carrierRemitLine2, "The remit-to suite or unit.", mdMaxAddress, false,
			func(c *carrier.Carrier) *string { return &c.RemitAddressLine2 }),
		masterText(carrierRemitCity, "The remit-to city.", mdMaxCity, false,
			func(c *carrier.Carrier) *string { return &c.RemitCity }),
		masterStatePointer(carrierRemitState, "The remit-to state.",
			func(c *carrier.Carrier) **pulid.ID { return &c.RemitStateID }),
		masterText(carrierRemitPostal, "The remit-to ZIP code.", mdMaxPostalCode, false,
			func(c *carrier.Carrier) *string { return &c.RemitPostalCode }),
		masterText(carrierTaxID, "The tax id payments are reported under.", carrierMaxTaxID,
			false, func(c *carrier.Carrier) *string { return &c.TaxID }),
		masterEnumPointer(carrierTaxIDType, "Whether the tax id is an EIN or an SSN; "+
			"required with a tax id.", carrierTaxIDTypes,
			func(c *carrier.Carrier) **carrier.TaxIDType { return &c.TaxIDType }),
		masterBool(carrierW9OnFile, "Whether a signed W-9 is on file.",
			func(c *carrier.Carrier) *bool { return &c.W9OnFile }),
		masterBool(carrier1099, "Whether payments are reported on a 1099.",
			func(c *carrier.Carrier) *bool { return &c.Is1099Eligible }),
	}
}

func carrierPolicy() masterPolicy {
	policy := mdInternalAsk
	policy.moneyKeys = carrierMoneyKeys

	return policy
}

func carrierGetter(carriers carrierKeeper) func(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*carrier.Carrier, error) {
	return func(
		ctx context.Context,
		tenant pagination.TenantInfo,
		id pulid.ID,
	) (*carrier.Carrier, error) {
		return carriers.Get(ctx, repositories.GetCarrierByIDRequest{
			ID:         id,
			TenantInfo: tenant,
			CarrierFilterOptions: repositories.CarrierFilterOptions{
				IncludeContacts:          true,
				IncludeInsurancePolicies: true,
				IncludeEDIChannels:       true,
			},
		})
	}
}

func newCreateCarrierTool(carriers carrierKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterCreateTool(&masterCreateSpec[carrier.Carrier]{
		record: carrierRecord,
		name:   "create_carrier",
		description: "Add a carrier the organization can tender freight to: its code, name, " +
			"USDOT and MC numbers, SCAC, address and dispatch contact. Check list_carriers " +
			"first. Vetting and compliance are vet_carrier's; payment and remit-to details " +
			"are only proposed.",
		rationale: "Adds a carrier inside Trenova that colleagues can then book; nothing is " +
			"sent. Payment and remit-to details decide where money goes, so a call that " +
			"sets them is money and runs only on a person's approval.",
		fields:      carrierFields(),
		required:    []string{mdCode, mdName},
		searchTerms: []string{"new carrier", "add carrier", "onboard carrier", "trucking company"},
		policy:      carrierPolicy(),
		states:      states,
		fresh: func(tenant pagination.TenantInfo) *carrier.Carrier {
			return &carrier.Carrier{
				OrganizationID:   tenant.OrgID,
				BusinessUnitID:   tenant.BuID,
				Status:           carrier.StatusActive,
				CarrierType:      carrier.TypeCommon,
				ComplianceStatus: carrier.ComplianceStatusPending,
				SafetyRating:     carrier.SafetyRatingNotRated,
				PaymentMethod:    carrier.PaymentMethodCheck,
				PaymentTermDays:  30,
			}
		},
		plan:   carriers.PlanCreate,
		create: carriers.Create,
	})
}

func newUpdateCarrierTool(carriers carrierKeeper, states stateLookup) serviceports.AgentTool {
	return newMasterUpdateTool(&masterUpdateSpec[carrier.Carrier]{
		record: carrierRecord,
		name:   "update_carrier",
		description: "Change a carrier's details: name, numbers, SCAC, address, dispatch " +
			"contact, notes, or payment and remit-to details. Fields left out keep their " +
			"value. Use update_carrier_status to activate or retire it and vet_carrier for " +
			"compliance.",
		rationale: "Changes a carrier inside Trenova and keeps its contacts, insurance and EDI " +
			"channels; nothing is sent. A call that changes payment or remit-to details is " +
			"money and runs only on a person's approval.",
		fields:      carrierFields(),
		searchTerms: []string{"edit carrier", "carrier address", "remit to", "carrier details"},
		policy:      carrierPolicy(),
		states:      states,
		get:         carrierGetter(carriers),
		plan:        carriers.PlanUpdate,
		update:      carriers.Update,
	})
}

func newUpdateCarrierStatusTool(
	carriers carrierKeeper,
	states stateLookup,
) serviceports.AgentTool {
	return newMasterStatusTool(&masterStatusSpec[carrier.Carrier, carrier.Status]{
		record: carrierRecord,
		name:   "update_carrier_status",
		description: "Set one or more carriers Active, Inactive or DoNotUse, which decides " +
			"whether they can be tendered freight. Get the ids from list_carriers.",
		rationale:   "Changes whether carriers are offered for booking inside Trenova; nothing is sent.",
		searchTerms: []string{"deactivate carrier", "do not use", "reactivate carrier"},
		statuses:    carrierStatuses,
		statusNote:  carrierStatusNote,
		policy:      mdInternalStatus,
		states:      states,
		plan: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status carrier.Status,
		) ([]serviceports.RecordChange[carrier.Carrier], error) {
			return carriers.PlanBulkUpdateStatus(ctx, &repositories.BulkUpdateCarrierStatusRequest{
				TenantInfo: tenant,
				CarrierIDs: ids,
				Status:     status,
			})
		},
		run: func(
			ctx context.Context,
			tenant pagination.TenantInfo,
			ids []pulid.ID,
			status carrier.Status,
		) ([]*carrier.Carrier, error) {
			return carriers.BulkUpdateStatus(ctx, &repositories.BulkUpdateCarrierStatusRequest{
				TenantInfo: tenant,
				CarrierIDs: ids,
				Status:     status,
			})
		},
	})
}
