package agenttoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	paramFuelPurchaseID       = "fuelPurchaseId"
	paramFuelCardID           = "fuelCardId"
	paramFuelImportID         = "fuelPurchaseImportId"
	paramTractorID            = "tractorId"
	paramJurisdictionID       = "jurisdictionId"
	paramPurchasedAt          = "purchasedAt"
	paramVendor               = "vendor"
	paramVendorCity           = "vendorCity"
	paramFuelType             = "fuelType"
	paramQuantity             = "quantity"
	paramQuantityUnit         = "quantityUnit"
	paramUnitPrice            = "unitPrice"
	paramTotalAmount          = "totalAmount"
	paramOdometer             = "odometer"
	paramCardLastFour         = "cardLastFour"
	paramTransactionReference = "transactionReference"
	paramTaxPaid              = "taxPaid"
	paramNotes                = "notes"
	paramAssignedTractorID    = "assignedTractorId"
	paramAssignedWorkerID     = "assignedWorkerId"

	fuelPurchaseRecordEntity = "fuel_purchase"
	keepWhenLeftOut          = " Leave it out to keep it."
	searchTermIFTA           = "ifta"
	fuelPurchaseKind         = "fuel purchase"
	maxFuelVendorText        = 150
	maxFuelReferenceText     = 100
	maxFuelNotes             = 2000
	maxOdometer              = 10_000_000
	fuelCardLastFourDigits   = 4
	fuelDiscardReason        = 500

	fuelPurchaseSupplier = "The fuel purchase, from list_fuel_purchases. Never guess one."
	fuelCardSupplier     = "The fuel card, from list_fuel_cards. Never guess one."
	fuelImportSupplier   = "The fuel purchase import, from list_fuel_purchase_imports. Never " +
		"guess one."
	fuelTaintHold = "A purchase read from a receipt or statement someone outside sent is " +
		"proposed, since it is a tax record the IFTA return is computed from."
)

var (
	fuelTypes     = agenttoolschema.Source("ifta.fuelType", domaintypes.IFTAFuelTypeValues())
	quantityUnits = agenttoolschema.Source(
		"fuelPurchase.quantityUnit",
		fuelpurchase.QuantityUnitValues(),
	)
	fuelPurchaseDates = map[string]assistantartifact.DisplayType{
		paramPurchasedAt: assistantartifact.DisplayDateTime,
	}
	fuelPurchaseRefs = toolpreview.WithRefs(map[string]permission.Resource{
		paramTractorID: permission.ResourceTractor,
		paramWorkerID:  permission.ResourceWorker,
	})
)

type fuelPurchaseKeeper interface {
	GetPurchase(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*fuelpurchase.FuelPurchase, error)
	PlanCreatePurchase(
		ctx context.Context,
		req *fuelpurchaseservice.CreatePurchaseRequest,
	) (*fuelpurchase.FuelPurchase, error)
	CreatePurchase(
		ctx context.Context,
		req *fuelpurchaseservice.CreatePurchaseRequest,
	) (*fuelpurchase.FuelPurchase, error)
	PlanUpdatePurchase(
		ctx context.Context,
		req *fuelpurchaseservice.UpdatePurchaseRequest,
	) (*fuelpurchaseservice.PurchaseChange, error)
	UpdatePurchase(
		ctx context.Context,
		req *fuelpurchaseservice.UpdatePurchaseRequest,
	) (*fuelpurchase.FuelPurchase, error)
	PlanDeletePurchase(
		ctx context.Context,
		req *fuelpurchaseservice.DeletePurchaseRequest,
	) (*fuelpurchase.FuelPurchase, error)
	DeletePurchase(ctx context.Context, req *fuelpurchaseservice.DeletePurchaseRequest) error
}

type fuelCardKeeper interface {
	GetCard(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*fuelpurchase.FuelCard, error)
	PlanAssignCard(
		ctx context.Context,
		req *fuelpurchaseservice.AssignCardRequest,
	) (*fuelpurchaseservice.CardChange, error)
	AssignCard(
		ctx context.Context,
		req *fuelpurchaseservice.AssignCardRequest,
	) (*fuelpurchase.FuelCard, error)
}

type fuelImportKeeper interface {
	GetImport(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		id pulid.ID,
	) (*fuelpurchase.ImportBatch, error)
	PlanCommit(
		ctx context.Context,
		req *fuelpurchaseservice.CommitRequest,
	) (*fuelpurchaseservice.CommitPlan, error)
	Commit(
		ctx context.Context,
		req *fuelpurchaseservice.CommitRequest,
	) (*fuelpurchase.ImportBatch, error)
	PlanResolveRows(
		ctx context.Context,
		req *fuelpurchaseservice.ResolveRowsRequest,
	) (*fuelpurchaseservice.ResolveRowsPlan, error)
	ResolveRows(
		ctx context.Context,
		req *fuelpurchaseservice.ResolveRowsRequest,
	) (*fuelpurchaseservice.ResolveRowsResult, error)
	PlanDiscard(
		ctx context.Context,
		req *fuelpurchaseservice.DiscardRequest,
	) (*fuelpurchaseservice.ImportBatchChange, error)
	Discard(
		ctx context.Context,
		req *fuelpurchaseservice.DiscardRequest,
	) (*fuelpurchase.ImportBatch, error)
}

type jurisdictionLabeler interface {
	GetJurisdiction(ctx context.Context, id pulid.ID) (*ifta.Jurisdiction, error)
}

var (
	_ fuelPurchaseKeeper = (*fuelpurchaseservice.Service)(nil)
	_ fuelCardKeeper     = (*fuelpurchaseservice.Service)(nil)
	_ fuelImportKeeper   = (*fuelpurchaseservice.Service)(nil)
)

func fuelInternalSpec(spec *receivableSpec) *receivableSpec {
	spec.egress = agent.EgressInternal
	if spec.defaultTier == "" {
		spec.defaultTier = agent.TierPropose
	}
	if spec.maxTier == "" {
		spec.maxTier = agent.TierActWithApproval
	}

	return spec
}

func fuelPurchaseProperties(forCorrection bool) map[string]any {
	keep := ""
	if forCorrection {
		keep = keepWhenLeftOut
	}

	properties := map[string]any{
		paramTractorID: stringProperty("The tractor the fuel went into, from list_tractors."+
			keep, 0),
		paramWorkerID: stringProperty("The driver who bought it, from search_worker or "+
			"list_workers. Leave it out to take the card's driver."+keep, 0),
		paramJurisdictionID: stringProperty("The state or province the fuel was bought in, "+
			"from list_ifta_jurisdictions; it decides which line of the return the tax-paid "+
			"gallons credit."+keep, 0),
		paramPurchasedAt: dateTimeProperty("When it was bought, as the receipt says." + keep),
		paramVendor:      stringProperty("The truck stop or vendor."+keep, maxFuelVendorText),
		paramVendorCity:  stringProperty("The vendor's city."+keep, maxFuelVendorText),
		paramFuelType: agenttoolschema.Enum("The fuel as IFTA classifies it; DEF, Reefer "+
			"and Other are spend only and never enter the return."+keep, fuelTypes),
		paramQuantity: stringProperty("The quantity bought, as a decimal such as 100.5, in "+
			"quantityUnit."+keep, 0),
		paramQuantityUnit: agenttoolschema.Enum("The unit the quantity is in. Defaults to "+
			"Gallon."+keep, quantityUnits),
		paramUnitPrice: stringProperty("The price per unit as a decimal such as 3.899."+
			keep, 0),
		paramTotalAmount: stringProperty("The total paid as a decimal such as 389.90."+
			keep, 0),
		paramCurrencyCode: stringProperty("The ISO currency code. Defaults to USD."+keep,
			currencyCodeLength),
		paramOdometer: integerProperty("The odometer reading at the pump."+keep, 0,
			maxOdometer),
		paramFuelCardID: stringProperty("The fuel card it was paid with, from "+
			"list_fuel_cards."+keep, 0),
		paramCardLastFour: stringProperty("The card's last four digits when the card is "+
			"not on file."+keep, fuelCardLastFourDigits),
		paramTransactionReference: stringProperty("The receipt or transaction number; a "+
			"purchase with a reference already on file is refused."+keep,
			maxFuelReferenceText),
		paramTaxPaid: booleanProperty("Whether fuel tax was paid at the pump. Defaults to " +
			"true; bulk or untaxed fuel earns no tax-paid credit." + keep),
		paramNotes: stringProperty("Anything the person reconciling it should know."+keep,
			maxFuelNotes),
	}
	if forCorrection {
		properties[paramFuelPurchaseID] = stringProperty(fuelPurchaseSupplier, 0)
	}

	return properties
}

type fuelPurchaseInput struct {
	values map[string]any
}

func (in fuelPurchaseInput) given(key string) bool {
	_, ok := in.values[key]

	return ok
}

func (in fuelPurchaseInput) applyIDs(purchase *fuelpurchase.FuelPurchase) error {
	for key, set := range map[string]func(pulid.ID){
		paramTractorID:      func(id pulid.ID) { purchase.TractorID = id },
		paramJurisdictionID: func(id pulid.ID) { purchase.JurisdictionID = id },
		paramWorkerID:       func(id pulid.ID) { purchase.WorkerID = &id },
		paramFuelCardID:     func(id pulid.ID) { purchase.FuelCardID = &id },
	} {
		if !in.given(key) {
			continue
		}
		id, err := requirePulid(in.values, key)
		if err != nil {
			return err
		}
		set(id)
	}

	return nil
}

func (in fuelPurchaseInput) applyTexts(purchase *fuelpurchase.FuelPurchase) error {
	for key, field := range map[string]struct {
		target *string
		limit  int
	}{
		paramVendor:               {&purchase.Vendor, maxFuelVendorText},
		paramVendorCity:           {&purchase.VendorCity, maxFuelVendorText},
		paramCardLastFour:         {&purchase.CardLastFour, fuelCardLastFourDigits},
		paramTransactionReference: {&purchase.TransactionReference, maxFuelReferenceText},
		paramNotes:                {&purchase.Notes, maxFuelNotes},
	} {
		text, err := optionalBoundedText(in.values, key, field.limit)
		if err != nil {
			return err
		}
		if text != nil {
			*field.target = *text
		}
	}

	return nil
}

func (in fuelPurchaseInput) applyAmounts(purchase *fuelpurchase.FuelPurchase) error {
	if in.given(paramQuantity) {
		quantity, present, err := optionalDecimal(in.values, paramQuantity)
		if err != nil {
			return err
		}
		if !present || !quantity.IsPositive() {
			return fmt.Errorf("parameter %q must be greater than zero", paramQuantity)
		}
		purchase.Quantity = quantity
	}
	if in.given(paramUnitPrice) {
		price, present, err := optionalDecimal(in.values, paramUnitPrice)
		if err != nil {
			return err
		}
		if present && !price.IsPositive() {
			return fmt.Errorf("parameter %q must be greater than zero", paramUnitPrice)
		}
		purchase.UnitPrice = decimal.NullDecimal{Decimal: price, Valid: present}
	}
	if in.given(paramTotalAmount) {
		total, err := requireMoney(in.values, paramTotalAmount)
		if err != nil {
			return err
		}
		purchase.TotalAmountMinor = total
	}
	if in.given(paramOdometer) {
		odometer, err := requireIntInRange(in.values, paramOdometer, 0, maxOdometer)
		if err != nil {
			return err
		}
		reading := int64(odometer)
		purchase.Odometer = &reading
	}

	return nil
}

func (in fuelPurchaseInput) applyChoices(purchase *fuelpurchase.FuelPurchase) error {
	if in.given(paramPurchasedAt) {
		purchasedAt, err := requireDateTime(in.values, paramPurchasedAt)
		if err != nil {
			return err
		}
		purchase.PurchasedAt = purchasedAt
	}
	if in.given(paramFuelType) {
		fuelType, err := requireEnum(in.values, paramFuelType, fuelTypes.Values)
		if err != nil {
			return err
		}
		purchase.FuelType = fuelType
	}
	if in.given(paramQuantityUnit) {
		unit, err := requireEnum(in.values, paramQuantityUnit, quantityUnits.Values)
		if err != nil {
			return err
		}
		purchase.QuantityUnit = unit
	}
	if in.given(paramCurrencyCode) {
		code := strings.ToUpper(strings.TrimSpace(optionalString(in.values, paramCurrencyCode)))
		if len(code) != currencyCodeLength {
			return fmt.Errorf("%s must be a three-letter code such as USD", paramCurrencyCode)
		}
		purchase.CurrencyCode = code
	}
	taxPaid, err := optionalBoolPointer(in.values, paramTaxPaid)
	if err != nil {
		return err
	}
	if taxPaid != nil {
		purchase.TaxPaid = *taxPaid
	}

	return nil
}

func (in fuelPurchaseInput) apply(purchase *fuelpurchase.FuelPurchase) error {
	for _, step := range []func(*fuelpurchase.FuelPurchase) error{
		in.applyIDs,
		in.applyTexts,
		in.applyAmounts,
		in.applyChoices,
	} {
		if err := step(purchase); err != nil {
			return err
		}
	}

	return nil
}

func newFuelPurchaseDraft(
	params *serviceports.ToolExecuteParams,
) (*fuelpurchaseservice.CreatePurchaseRequest, error) {
	purchase := &fuelpurchase.FuelPurchase{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		QuantityUnit:   fuelpurchase.QuantityUnitGallon,
		CurrencyCode:   money.DefaultCurrencyCode,
		TaxPaid:        true,
	}
	if err := (fuelPurchaseInput{values: params.Params}).apply(purchase); err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo: tenantFrom(*params),
		Purchase:   purchase,
		UserID:     params.Actor.UserID,
	}, nil
}

type fuelPurchaseView struct {
	TractorID            string `json:"tractorId"`
	WorkerID             string `json:"workerId,omitempty"`
	Jurisdiction         string `json:"jurisdiction"`
	PurchasedAt          int64  `json:"purchasedAt"`
	Vendor               string `json:"vendor,omitempty"`
	VendorCity           string `json:"vendorCity,omitempty"`
	FuelType             string `json:"fuelType"`
	Quantity             string `json:"quantity"`
	QuantityUnit         string `json:"quantityUnit"`
	Gallons              string `json:"gallons"`
	UnitPrice            string `json:"unitPrice,omitempty"`
	TotalAmount          string `json:"totalAmount"`
	CurrencyCode         string `json:"currencyCode"`
	TaxPaid              bool   `json:"taxPaid"`
	Odometer             int64  `json:"odometer,omitempty"`
	CardLastFour         string `json:"cardLastFour,omitempty"`
	TransactionReference string `json:"transactionReference,omitempty"`
	Notes                string `json:"notes,omitempty"`
}

func fuelPurchaseViewOf(
	purchase *fuelpurchase.FuelPurchase,
	jurisdiction string,
) *fuelPurchaseView {
	view := &fuelPurchaseView{
		TractorID:            purchase.TractorID.String(),
		Jurisdiction:         jurisdiction,
		PurchasedAt:          purchase.PurchasedAt,
		Vendor:               purchase.Vendor,
		VendorCity:           purchase.VendorCity,
		FuelType:             string(purchase.FuelType),
		Quantity:             purchase.Quantity.String(),
		QuantityUnit:         string(purchase.QuantityUnit),
		Gallons:              purchase.Gallons.StringFixed(3),
		TotalAmount:          money.DecimalFromMinor(purchase.TotalAmountMinor).StringFixed(2),
		CurrencyCode:         purchase.CurrencyCode,
		TaxPaid:              purchase.TaxPaid,
		CardLastFour:         purchase.CardLastFour,
		TransactionReference: purchase.TransactionReference,
		Notes:                purchase.Notes,
	}
	if purchase.WorkerID != nil {
		view.WorkerID = purchase.WorkerID.String()
	}
	if purchase.UnitPrice.Valid {
		view.UnitPrice = purchase.UnitPrice.Decimal.String()
	}
	if purchase.Odometer != nil {
		view.Odometer = *purchase.Odometer
	}

	return view
}

func jurisdictionLabel(
	ctx context.Context,
	jurisdictions jurisdictionLabeler,
	id pulid.ID,
) string {
	if id.IsNil() {
		return ""
	}
	jurisdiction, err := jurisdictions.GetJurisdiction(ctx, id)
	if err != nil || jurisdiction == nil {
		return id.String()
	}

	return jurisdiction.Code + " (" + jurisdiction.Name + ")"
}

func fuelPurchaseRecord(purchase *fuelpurchase.FuelPurchase) toolpreview.Record {
	label := purchase.TransactionReference
	if label == "" {
		label = purchase.Gallons.StringFixed(3) + " gallons of " + purchase.FuelType.Label()
	}

	return toolpreview.Record{
		Resource: permission.ResourceFuelPurchase,
		ID:       purchase.ID,
		Label:    label,
		Version:  pinnedVersion(purchase.Version),
	}
}

func fuelPurchaseResult(
	action string,
	purchase *fuelpurchase.FuelPurchase,
) *agent.ToolExecutionResult {
	return &agent.ToolExecutionResult{
		Action: action,
		Kind:   fuelPurchaseKind,
		Name:   fuelPurchaseRecord(purchase).Label,
		IDs:    map[string]string{paramFuelPurchaseID: purchase.ID.String()},
		Record: recordOf(fuelPurchaseRecordEntity, purchase.ID),
	}
}

func targetFuelPurchase(params map[string]any) (serviceports.ToolTarget, bool) {
	return targetOf(params, paramFuelPurchaseID, permission.ResourceFuelPurchase)
}

type labelledPurchase struct {
	purchase     *fuelpurchase.FuelPurchase
	jurisdiction string
}

func newRecordFuelPurchaseTool(
	purchases fuelPurchaseKeeper,
	jurisdictions jurisdictionLabeler,
) serviceports.AgentTool {
	return newReportingReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "record_fuel_purchase",
		description: "Record fuel bought for a tractor from a receipt or a statement line: " +
			"where it was bought for IFTA, when, what fuel, how much and what it cost. It is " +
			"a tax record the quarter's IFTA return is computed from. Check " +
			"list_fuel_purchases first, since a reference already on file is refused.",
		resource:   permission.ResourceFuelPurchase,
		operation:  permission.OpCreate,
		artifact:   fuelPurchaseRecordEntity,
		reversible: true,
		taintHold:  fuelTaintHold,
		rationale: "Adds a fuel purchase inside Trenova that the next IFTA return counts; " +
			"nothing is sent, and delete_fuel_purchase removes it.",
		properties: fuelPurchaseProperties(false),
		required: []string{
			paramTractorID,
			paramJurisdictionID,
			paramPurchasedAt,
			paramFuelType,
			paramQuantity,
			paramTotalAmount,
		},
		searchTerms: []string{"fuel", "receipt", "diesel", "gallons", searchTermIFTA, "pump"},
	}), receivablePlan[*fuelpurchaseservice.CreatePurchaseRequest, *labelledPurchase]{
		request: newFuelPurchaseDraft,
		plan: func(
			ctx context.Context,
			req *fuelpurchaseservice.CreatePurchaseRequest,
			_ *serviceports.ToolExecuteParams,
		) (*labelledPurchase, error) {
			planned, err := purchases.PlanCreatePurchase(ctx, req)
			if err != nil {
				return nil, err
			}

			return &labelledPurchase{
				purchase:     planned,
				jurisdiction: jurisdictionLabel(ctx, jurisdictions, planned.JurisdictionID),
			}, nil
		},
		refused: func(*fuelpurchaseservice.CreatePurchaseRequest) string {
			return "Would record a fuel purchase."
		},
		render: func(
			_ *fuelpurchaseservice.CreatePurchaseRequest,
			plan *labelledPurchase,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Create(
				toolpreview.Record{
					Resource: permission.ResourceFuelPurchase,
					Label:    fuelPurchaseRecord(plan.purchase).Label,
				},
				fuelPurchaseViewOf(plan.purchase, plan.jurisdiction),
				fuelPurchaseRefs,
				toolpreview.Types(fuelPurchaseDates),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would record %s gallons of %s bought in %s on %s for %s %s.",
				plan.purchase.Gallons.StringFixed(3),
				plan.purchase.FuelType.Label(),
				plan.jurisdiction,
				dayLabel(plan.purchase.PurchasedAt),
				money.DecimalFromMinor(plan.purchase.TotalAmountMinor).StringFixed(2),
				plan.purchase.CurrencyCode,
			), change), nil
		},
		run: func(
			ctx context.Context,
			req *fuelpurchaseservice.CreatePurchaseRequest,
			_ *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			created, err := purchases.CreatePurchase(ctx, req)
			if err != nil {
				return nil, err
			}

			return fuelPurchaseResult("recorded", created), nil
		},
	})
}

type fuelPurchaseCorrection struct {
	id     pulid.ID
	values map[string]any
}

func fuelPurchaseCorrectionFrom(
	params *serviceports.ToolExecuteParams,
) (*fuelPurchaseCorrection, error) {
	id, err := requirePulid(params.Params, paramFuelPurchaseID)
	if err != nil {
		return nil, err
	}
	values := make(map[string]any, len(params.Params))
	for key, value := range params.Params {
		if key != paramFuelPurchaseID {
			values[key] = value
		}
	}
	if len(values) == 0 {
		return nil, errors.New("name at least one field of the purchase to correct")
	}

	return &fuelPurchaseCorrection{id: id, values: values}, nil
}

func (c *fuelPurchaseCorrection) request(
	ctx context.Context,
	purchases fuelPurchaseKeeper,
	params *serviceports.ToolExecuteParams,
) (*fuelpurchaseservice.UpdatePurchaseRequest, error) {
	stored, err := purchases.GetPurchase(ctx, tenantFrom(*params), c.id)
	if err != nil {
		return nil, err
	}

	corrected := *stored
	corrected.Tractor = nil
	corrected.Worker = nil
	corrected.FuelCard = nil
	corrected.Jurisdiction = nil
	corrected.ImportBatch = nil
	if err = (fuelPurchaseInput{values: c.values}).apply(&corrected); err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.UpdatePurchaseRequest{
		TenantInfo: tenantFrom(*params),
		Purchase:   &corrected,
		UserID:     params.Actor.UserID,
	}, nil
}

type labelledPurchaseChange struct {
	change  *fuelpurchaseservice.PurchaseChange
	request *fuelpurchaseservice.UpdatePurchaseRequest
	before  string
	after   string
}

func newCorrectFuelPurchaseTool(
	purchases fuelPurchaseKeeper,
	jurisdictions jurisdictionLabeler,
) serviceports.AgentTool {
	return newReportingReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "correct_fuel_purchase",
		description: "Correct a recorded fuel purchase: the tractor, jurisdiction, day, " +
			"fuel type, quantity, amount or any other field that was keyed wrong. Fields left " +
			"out keep their value. A finalized IFTA return keeps its figures until it is " +
			"reopened or amended.",
		resource:   permission.ResourceFuelPurchase,
		operation:  permission.OpUpdate,
		artifact:   fuelPurchaseRecordEntity,
		reversible: true,
		taintHold:  fuelTaintHold,
		rationale: "Changes a fuel purchase inside Trenova; nothing is sent, and a later " +
			"correction changes it back.",
		properties: fuelPurchaseProperties(true),
		required:   []string{paramFuelPurchaseID},
		target:     targetFuelPurchase,
	}), receivablePlan[*fuelPurchaseCorrection, *labelledPurchaseChange]{
		request: fuelPurchaseCorrectionFrom,
		plan: func(
			ctx context.Context,
			correction *fuelPurchaseCorrection,
			params *serviceports.ToolExecuteParams,
		) (*labelledPurchaseChange, error) {
			req, err := correction.request(ctx, purchases, params)
			if err != nil {
				return nil, err
			}
			change, err := purchases.PlanUpdatePurchase(ctx, req)
			if err != nil {
				return nil, err
			}

			return &labelledPurchaseChange{
				change:  change,
				request: req,
				before:  jurisdictionLabel(ctx, jurisdictions, change.Before.JurisdictionID),
				after:   jurisdictionLabel(ctx, jurisdictions, change.After.JurisdictionID),
			}, nil
		},
		refused: func(*fuelPurchaseCorrection) string {
			return "Would correct a fuel purchase."
		},
		render: func(
			_ *fuelPurchaseCorrection,
			plan *labelledPurchaseChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				fuelPurchaseRecord(plan.change.Before),
				fuelPurchaseViewOf(plan.change.Before, plan.before),
				fuelPurchaseViewOf(plan.change.After, plan.after),
				fuelPurchaseRefs,
				toolpreview.Types(fuelPurchaseDates),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would correct the fuel purchase %s bought on %s; a finalized IFTA return "+
					"keeps its figures until it is reopened or amended.",
				fuelPurchaseRecord(plan.change.Before).Label,
				dayLabel(plan.change.Before.PurchasedAt),
			), change), nil
		},
		run: func(
			ctx context.Context,
			correction *fuelPurchaseCorrection,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := correction.request(ctx, purchases, params)
			if err != nil {
				return nil, err
			}
			updated, err := purchases.UpdatePurchase(ctx, req)
			if err != nil {
				return nil, err
			}

			return fuelPurchaseResult("corrected", updated), nil
		},
	})
}

func deleteFuelPurchaseRequest(
	ctx context.Context,
	purchases fuelPurchaseKeeper,
	params *serviceports.ToolExecuteParams,
	id pulid.ID,
) (*fuelpurchaseservice.DeletePurchaseRequest, error) {
	stored, err := purchases.GetPurchase(ctx, tenantFrom(*params), id)
	if err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.DeletePurchaseRequest{
		TenantInfo: tenantFrom(*params),
		ID:         stored.ID,
		Version:    stored.Version,
		UserID:     params.Actor.UserID,
	}, nil
}

func newDeleteFuelPurchaseTool(
	purchases fuelPurchaseKeeper,
	jurisdictions jurisdictionLabeler,
) serviceports.AgentTool {
	return newReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "delete_fuel_purchase",
		description: "Propose deleting a fuel purchase recorded in error, such as a " +
			"receipt entered twice. It is removed outright; a return already generated for " +
			"its quarter keeps its figures until it is recomputed. To fix a wrong field use " +
			"correct_fuel_purchase instead.",
		resource:  permission.ResourceFuelPurchase,
		operation: permission.OpDelete,
		maxTier:   agent.TierPropose,
		rationale: "Removes a tax record the IFTA return is computed from, and nothing " +
			"brings it back but entering it again, so a person always decides.",
		properties: map[string]any{
			paramFuelPurchaseID: stringProperty(fuelPurchaseSupplier, 0),
		},
		required: []string{paramFuelPurchaseID},
		target:   targetFuelPurchase,
	}), receivablePlan[pulid.ID, *labelledPurchase]{
		request: func(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
			return requirePulid(params.Params, paramFuelPurchaseID)
		},
		plan: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*labelledPurchase, error) {
			req, err := deleteFuelPurchaseRequest(ctx, purchases, params, id)
			if err != nil {
				return nil, err
			}
			planned, err := purchases.PlanDeletePurchase(ctx, req)
			if err != nil {
				return nil, err
			}

			return &labelledPurchase{
				purchase:     planned,
				jurisdiction: jurisdictionLabel(ctx, jurisdictions, planned.JurisdictionID),
			}, nil
		},
		refused: func(pulid.ID) string {
			return "Would delete a fuel purchase."
		},
		render: func(_ pulid.ID, plan *labelledPurchase) (*agent.ToolPreview, error) {
			change, err := toolpreview.Delete(
				fuelPurchaseRecord(plan.purchase),
				fuelPurchaseViewOf(plan.purchase, plan.jurisdiction),
				fuelPurchaseRefs,
				toolpreview.Types(fuelPurchaseDates),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would delete the fuel purchase %s bought on %s; a return already generated "+
					"for its quarter keeps its figures until it is recomputed.",
				fuelPurchaseRecord(plan.purchase).Label,
				dayLabel(plan.purchase.PurchasedAt),
			), change), nil
		},
		run: func(
			ctx context.Context,
			id pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := deleteFuelPurchaseRequest(ctx, purchases, params, id)
			if err != nil {
				return nil, err
			}

			return nil, purchases.DeletePurchase(ctx, req)
		},
	})
}

type fuelCardAssignment struct {
	id      pulid.ID
	tractor *pulid.ID
	worker  *pulid.ID
}

func fuelCardAssignmentFrom(params *serviceports.ToolExecuteParams) (*fuelCardAssignment, error) {
	id, err := requirePulid(params.Params, paramFuelCardID)
	if err != nil {
		return nil, err
	}
	tractor, err := optionalPulidParam(params.Params, paramAssignedTractorID)
	if err != nil {
		return nil, err
	}
	worker, err := optionalPulidParam(params.Params, paramAssignedWorkerID)
	if err != nil {
		return nil, err
	}
	if tractor == nil && worker == nil {
		return nil, errors.New("name the tractor, the driver or both the card goes to")
	}

	return &fuelCardAssignment{id: id, tractor: tractor, worker: worker}, nil
}

func (a *fuelCardAssignment) request(
	ctx context.Context,
	cards fuelCardKeeper,
	params *serviceports.ToolExecuteParams,
) (*fuelpurchaseservice.AssignCardRequest, error) {
	stored, err := cards.GetCard(ctx, tenantFrom(*params), a.id)
	if err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.AssignCardRequest{
		TenantInfo:        tenantFrom(*params),
		ID:                stored.ID,
		Version:           stored.Version,
		AssignedTractorID: a.tractor,
		AssignedWorkerID:  a.worker,
		UserID:            params.Actor.UserID,
	}, nil
}

type fuelCardView struct {
	Label             string `json:"label"`
	Status            string `json:"status"`
	AssignedTractorID string `json:"assignedTractorId,omitempty"`
	AssignedWorkerID  string `json:"assignedWorkerId,omitempty"`
}

func fuelCardViewOf(card *fuelpurchase.FuelCard) *fuelCardView {
	view := &fuelCardView{Label: card.Label, Status: string(card.Status)}
	if card.AssignedTractorID != nil {
		view.AssignedTractorID = card.AssignedTractorID.String()
	}
	if card.AssignedWorkerID != nil {
		view.AssignedWorkerID = card.AssignedWorkerID.String()
	}

	return view
}

func newAssignFuelCardTool(cards fuelCardKeeper) serviceports.AgentTool {
	return newReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "assign_fuel_card",
		description: "Assign a fuel card to the tractor or driver that carries it, most " +
			"often a card a feed discovered. Assigning a suspended card also activates it, " +
			"and once assigned, resolve_fuel_purchase_import_rows posts the purchases that " +
			"waited for it.",
		resource:   permission.ResourceFuelCard,
		operation:  permission.OpUpdate,
		reversible: true,
		rationale: "Changes which tractor a card's purchases count against inside Trenova; " +
			"nothing is sent, and assigning it again changes it back.",
		properties: map[string]any{
			paramFuelCardID: stringProperty(fuelCardSupplier, 0),
			paramAssignedTractorID: stringProperty("The tractor that carries the card, from "+
				"list_tractors.", 0),
			paramAssignedWorkerID: stringProperty("The driver who carries the card, from "+
				"search_worker or list_workers.", 0),
		},
		required: []string{paramFuelCardID},
		target: func(params map[string]any) (serviceports.ToolTarget, bool) {
			return targetOf(params, paramFuelCardID, permission.ResourceFuelCard)
		},
	}), receivablePlan[*fuelCardAssignment, *fuelpurchaseservice.CardChange]{
		request: fuelCardAssignmentFrom,
		plan: func(
			ctx context.Context,
			assignment *fuelCardAssignment,
			params *serviceports.ToolExecuteParams,
		) (*fuelpurchaseservice.CardChange, error) {
			req, err := assignment.request(ctx, cards, params)
			if err != nil {
				return nil, err
			}

			return cards.PlanAssignCard(ctx, req)
		},
		refused: func(*fuelCardAssignment) string {
			return "Would assign a fuel card."
		},
		render: func(
			_ *fuelCardAssignment,
			plan *fuelpurchaseservice.CardChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				toolpreview.Record{
					Resource: permission.ResourceFuelCard,
					ID:       plan.Before.ID,
					Label:    plan.Before.Provider.Label() + " card ending " + plan.Before.LastFour,
					Version:  pinnedVersion(plan.Before.Version),
				},
				fuelCardViewOf(plan.Before),
				fuelCardViewOf(plan.After),
				toolpreview.WithRefs(map[string]permission.Resource{
					paramAssignedTractorID: permission.ResourceTractor,
					paramAssignedWorkerID:  permission.ResourceWorker,
				}),
			)
			if err != nil {
				return nil, err
			}

			summary := fmt.Sprintf("Would assign the %s card ending %s.",
				plan.Before.Provider.Label(), plan.Before.LastFour)
			if plan.Before.Status != plan.After.Status {
				summary = fmt.Sprintf("Would assign the %s card ending %s and make it %s.",
					plan.Before.Provider.Label(), plan.Before.LastFour,
					strings.ToLower(plan.After.Status.Label()))
			}

			return toolpreview.Build(summary, change), nil
		},
		run: func(
			ctx context.Context,
			assignment *fuelCardAssignment,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := assignment.request(ctx, cards, params)
			if err != nil {
				return nil, err
			}
			_, err = cards.AssignCard(ctx, req)

			return nil, err
		},
	})
}

func targetFuelImport(
	resource permission.Resource,
) func(map[string]any) (serviceports.ToolTarget, bool) {
	return func(params map[string]any) (serviceports.ToolTarget, bool) {
		return targetOf(params, paramFuelImportID, resource)
	}
}

func fuelImportRecord(batch *fuelpurchase.ImportBatch) toolpreview.Record {
	label := batch.FileName
	if label == "" {
		label = batch.Provider.Label() + " feed run"
	}

	return toolpreview.Record{
		Resource: permission.ResourceFuelPurchaseImport,
		ID:       batch.ID,
		Label:    label,
		Version:  pinnedVersion(batch.Version),
	}
}

type fuelImportView struct {
	Status         string `json:"status"`
	RowCount       int    `json:"rowCount"`
	ReadyToCommit  int    `json:"readyToCommit"`
	ErrorCount     int    `json:"errorCount"`
	CommittedCount int    `json:"committedCount"`
}

func fuelImportViewOf(batch *fuelpurchase.ImportBatch) *fuelImportView {
	view := &fuelImportView{
		Status:         string(batch.Status),
		RowCount:       batch.RowCount,
		ErrorCount:     batch.ErrorCount,
		CommittedCount: batch.CommittedCount,
	}
	if batch.Summary != nil {
		view.ReadyToCommit = batch.Summary.NewCount
	}

	return view
}

func loadFuelImport(
	ctx context.Context,
	imports fuelImportKeeper,
	params *serviceports.ToolExecuteParams,
) (*fuelpurchase.ImportBatch, error) {
	id, err := requirePulid(params.Params, paramFuelImportID)
	if err != nil {
		return nil, err
	}

	return imports.GetImport(ctx, tenantFrom(*params), id)
}

func fuelImportIDFrom(params *serviceports.ToolExecuteParams) (pulid.ID, error) {
	return requirePulid(params.Params, paramFuelImportID)
}

func newCommitFuelPurchaseImportTool(imports fuelImportKeeper) serviceports.AgentTool {
	return newReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "commit_fuel_purchase_import",
		description: "Commit a staged fuel card statement: every row ready to commit becomes " +
			"a fuel purchase in one step, and rows whose reference is already on file are " +
			"skipped. Read it with list_fuel_purchase_imports first; rows in error stay " +
			"behind.",
		resource:  permission.ResourceFuelPurchase,
		operation: permission.OpImport,
		rationale: "Adds a statement's purchases inside Trenova, which the IFTA return counts; " +
			"nothing is sent, but undoing it means deleting each purchase.",
		properties: map[string]any{
			paramFuelImportID: stringProperty(fuelImportSupplier, 0),
		},
		required: []string{paramFuelImportID},
		target:   targetFuelImport(permission.ResourceFuelPurchase),
	}), receivablePlan[pulid.ID, *fuelpurchaseservice.CommitPlan]{
		request: fuelImportIDFrom,
		plan: func(
			ctx context.Context,
			_ pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*fuelpurchaseservice.CommitPlan, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}

			return imports.PlanCommit(ctx, &fuelpurchaseservice.CommitRequest{
				TenantInfo: tenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})
		},
		refused: func(pulid.ID) string {
			return "Would commit a fuel purchase import."
		},
		render: func(_ pulid.ID, plan *fuelpurchaseservice.CommitPlan) (*agent.ToolPreview, error) {
			committed := *plan.Batch
			committed.Status = fuelpurchase.ImportStatusCommitted
			committed.CommittedCount = len(plan.Purchases)
			change, err := toolpreview.Changed(
				fuelImportRecord(plan.Batch),
				fuelImportViewOf(plan.Batch),
				fuelImportViewOf(&committed),
			)
			if err != nil {
				return nil, err
			}

			gallons, total := decimal.Zero, int64(0)
			for _, purchase := range plan.Purchases {
				gallons = gallons.Add(purchase.QuantityUnit.ToGallons(purchase.Quantity))
				total += purchase.TotalAmountMinor
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would commit %s from %s: %s gallons for %s %s.",
				countOf(len(plan.Purchases), fuelPurchaseKind),
				fuelImportRecord(plan.Batch).Label,
				gallons.StringFixed(3),
				money.DecimalFromMinor(total).StringFixed(2),
				plan.Batch.DefaultCurrency,
			), change), nil
		},
		run: func(
			ctx context.Context,
			_ pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}
			_, err = imports.Commit(ctx, &fuelpurchaseservice.CommitRequest{
				TenantInfo: tenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})

			return nil, err
		},
	})
}

func newResolveFuelPurchaseImportRowsTool(imports fuelImportKeeper) serviceports.AgentTool {
	return newReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "resolve_fuel_purchase_import_rows",
		description: "Work out again the rows a fuel import is still holding, against the " +
			"cards, tractors and jurisdictions on file now, without fetching the statement " +
			"again. Run it after assign_fuel_card or once a missing tractor exists. A feed " +
			"posts the rows that now resolve; an upload waits for " +
			"commit_fuel_purchase_import.",
		resource:  permission.ResourceFuelPurchaseImport,
		operation: permission.OpImport,
		rationale: "Re-matches held statement rows inside Trenova and, for a feed, posts the " +
			"purchases that now resolve; nothing is sent.",
		properties: map[string]any{
			paramFuelImportID: stringProperty(fuelImportSupplier, 0),
		},
		required: []string{paramFuelImportID},
		target:   targetFuelImport(permission.ResourceFuelPurchaseImport),
	}), receivablePlan[pulid.ID, *fuelpurchaseservice.ResolveRowsPlan]{
		request: fuelImportIDFrom,
		plan: func(
			ctx context.Context,
			_ pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*fuelpurchaseservice.ResolveRowsPlan, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}

			return imports.PlanResolveRows(ctx, &fuelpurchaseservice.ResolveRowsRequest{
				TenantInfo: tenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})
		},
		refused: func(pulid.ID) string {
			return "Would work out a fuel import's held rows again."
		},
		render: func(
			_ pulid.ID,
			plan *fuelpurchaseservice.ResolveRowsPlan,
		) (*agent.ToolPreview, error) {
			outcome := "they wait for commit_fuel_purchase_import"
			if plan.Batch.IsFeed() {
				outcome = "the feed posts them as purchases"
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would work out %s held in %s again: %d now resolve and %d still wait; %s.",
				countOf(plan.Reviewed, "row"),
				fuelImportRecord(plan.Batch).Label,
				plan.Resolved,
				plan.Queued,
				outcome,
			)), nil
		},
		run: func(
			ctx context.Context,
			_ pulid.ID,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}
			_, err = imports.ResolveRows(ctx, &fuelpurchaseservice.ResolveRowsRequest{
				TenantInfo: tenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})

			return nil, err
		},
	})
}

type fuelImportDiscard struct {
	id     pulid.ID
	reason string
}

func newDiscardFuelPurchaseImportTool(imports fuelImportKeeper) serviceports.AgentTool {
	discardRequest := func(
		ctx context.Context,
		discard *fuelImportDiscard,
		params *serviceports.ToolExecuteParams,
	) (*fuelpurchaseservice.DiscardRequest, error) {
		batch, err := imports.GetImport(ctx, tenantFrom(*params), discard.id)
		if err != nil {
			return nil, err
		}

		return &fuelpurchaseservice.DiscardRequest{
			TenantInfo: tenantFrom(*params),
			BatchID:    batch.ID,
			Version:    batch.Version,
			Reason:     discard.reason,
			UserID:     params.Actor.UserID,
		}, nil
	}

	return newReceivableTool(fuelInternalSpec(&receivableSpec{
		name: "discard_fuel_purchase_import",
		description: "Discard a fuel card statement that was staged but should not be " +
			"committed, such as the wrong month or a file loaded twice. Nothing it holds " +
			"becomes a purchase, and it can be staged again later.",
		resource:   permission.ResourceFuelPurchase,
		operation:  permission.OpCancel,
		reversible: true,
		rationale: "Closes a staged statement inside Trenova without adding any purchase; it " +
			"can be staged again.",
		properties: map[string]any{
			paramFuelImportID: stringProperty(fuelImportSupplier, 0),
			paramReason: stringProperty("Why it is being discarded, for whoever finds it "+
				"later.", fuelDiscardReason),
		},
		required: []string{paramFuelImportID},
		target:   targetFuelImport(permission.ResourceFuelPurchase),
	}), receivablePlan[*fuelImportDiscard, *fuelpurchaseservice.ImportBatchChange]{
		request: func(params *serviceports.ToolExecuteParams) (*fuelImportDiscard, error) {
			id, err := requirePulid(params.Params, paramFuelImportID)
			if err != nil {
				return nil, err
			}
			reason, err := boundedText(params.Params, paramReason, fuelDiscardReason)
			if err != nil {
				return nil, err
			}

			return &fuelImportDiscard{id: id, reason: reason}, nil
		},
		plan: func(
			ctx context.Context,
			discard *fuelImportDiscard,
			params *serviceports.ToolExecuteParams,
		) (*fuelpurchaseservice.ImportBatchChange, error) {
			req, err := discardRequest(ctx, discard, params)
			if err != nil {
				return nil, err
			}

			return imports.PlanDiscard(ctx, req)
		},
		refused: func(*fuelImportDiscard) string {
			return "Would discard a fuel purchase import."
		},
		render: func(
			_ *fuelImportDiscard,
			plan *fuelpurchaseservice.ImportBatchChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				fuelImportRecord(plan.Before),
				fuelImportViewOf(plan.Before),
				fuelImportViewOf(plan.After),
			)
			if err != nil {
				return nil, err
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would discard %s (now %s); none of its rows becomes a purchase.",
				fuelImportRecord(plan.Before).Label,
				plan.Before.Status,
			), change), nil
		},
		run: func(
			ctx context.Context,
			discard *fuelImportDiscard,
			params *serviceports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			req, err := discardRequest(ctx, discard, params)
			if err != nil {
				return nil, err
			}
			_, err = imports.Discard(ctx, req)

			return nil, err
		},
	})
}
