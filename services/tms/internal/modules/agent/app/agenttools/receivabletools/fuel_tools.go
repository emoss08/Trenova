package receivabletools

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/ifta"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const paramFuelPurchaseID = "fuelPurchaseId"

const paramFuelCardID = "fuelCardId"

const paramFuelImportID = "fuelPurchaseImportId"

const paramJurisdictionID = "jurisdictionId"

const paramPurchasedAt = "purchasedAt"

const noonOffsetSeconds = 12 * 3600

const paramVendor = "vendor"

const paramVendorCity = "vendorCity"

const paramQuantity = "quantity"

const paramQuantityUnit = "quantityUnit"

const paramUnitPrice = "unitPrice"

const paramTotalAmount = "totalAmount"

const paramOdometer = "odometer"

const paramCardLastFour = "cardLastFour"

const paramTransactionReference = "transactionReference"

const paramTaxPaid = "taxPaid"

const paramNotes = "notes"

const paramAssignedTractorID = "assignedTractorId"

const paramAssignedWorkerID = "assignedWorkerId"

const fuelPurchaseRecordEntity = "fuel_purchase"

const searchTermIFTA = "ifta"

const fuelPurchaseKind = "fuel purchase"

const maxFuelVendorText = 150

const maxFuelReferenceText = 100

const maxFuelNotes = 2000

const maxOdometer = 10_000_000

const fuelCardLastFourDigits = 4

const fuelDiscardReason = 500

const fuelPurchaseSupplier = "The fuel purchase, from fuel_list_purchases. Never guess one."

const fuelCardSupplier = "The fuel card, from fuel_list_cards. Never guess one."

const fuelImportSupplier = "The fuel purchase import, from fuel_list_purchase_imports. Never " +
	"guess one."

const fuelTaintHold = "A purchase read from a receipt or statement someone outside sent is " +
	"proposed, since it is a tax record the IFTA return is computed from."

var quantityUnits = agenttoolschema.Source(
	"fuelPurchase.quantityUnit",
	fuelpurchase.QuantityUnitValues(),
)

var fuelPurchaseDates = map[string]assistantartifact.DisplayType{
	paramPurchasedAt: assistantartifact.DisplayDateTime,
}

var fuelPurchaseRefs = toolpreview.WithRefs(map[string]permission.Resource{
	common.ParamTractorID: permission.ResourceTractor,
	common.ParamWorkerID:  permission.ResourceWorker,
})

type FuelPurchaseKeeper interface {
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

type FuelCardKeeper interface {
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

type FuelImportKeeper interface {
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

func fuelPurchaseProperties(forCorrection bool) map[string]any {
	keep := ""
	if forCorrection {
		keep = common.KeepWhenLeftOut
	}

	properties := map[string]any{
		common.ParamTractorID: agenttoolschema.RecordIDText(permission.ResourceTractor,
			"The tractor the fuel went into, from equipment_list_tractors."+keep),
		common.ParamWorkerID: agenttoolschema.RecordIDText(permission.ResourceWorker,
			"The driver who bought it, from worker_search or worker_list. Leave it out to "+
				"take the card's driver."+keep),
		paramJurisdictionID: agenttoolschema.KindID("The state or province the fuel was "+
			"bought in, from fuel_list_ifta_jurisdictions; it decides which line of the return the "+
			"tax-paid gallons credit."+keep, permission.KindIFTAJurisdiction),
		paramPurchasedAt: map[string]any{
			toolschema.KeyType: toolschema.TypeString,
			toolschema.KeyDescription: "When it was bought: the receipt's date and time with its " +
				"offset, such as 2026-10-08T14:30:00-05:00, or the day alone (2026-10-08, " +
				"yesterday) when nobody gave a time; a day is recorded at noon in the " +
				"organization's time zone, which is what the IFTA quarter needs." + keep,
		},
		paramVendor:     common.StringProperty("The truck stop or vendor."+keep, maxFuelVendorText),
		paramVendorCity: common.StringProperty("The vendor's city."+keep, maxFuelVendorText),
		common.ParamFuelType: agenttoolschema.Enum("The fuel as IFTA classifies it; DEF, Reefer "+
			"and Other are spend only and never enter the return."+keep, common.FuelTypes),
		paramQuantity: common.StringProperty("The quantity bought, as a decimal such as 100.5, in "+
			"quantityUnit."+keep, 0),
		paramQuantityUnit: agenttoolschema.Enum("The unit the quantity is in. Defaults to "+
			"Gallon."+keep, quantityUnits),
		paramUnitPrice: common.StringProperty("The price per unit as a decimal such as 3.899."+
			keep, 0),
		paramTotalAmount: common.StringProperty("The total paid as a decimal such as 389.90, as "+
			"the receipt says. Leave it out when you have unitPrice and the quantity: it is "+
			"worked out to the cent from them."+keep, 0),
		paramCurrencyCode: common.StringProperty("The ISO currency code. Defaults to USD."+keep,
			common.CurrencyCodeLength),
		paramOdometer: common.IntegerProperty("The odometer reading at the pump."+keep, 0,
			maxOdometer),
		paramFuelCardID: agenttoolschema.RecordIDText(permission.ResourceFuelCard,
			"The fuel card it was paid with, from fuel_list_cards."+keep),
		paramCardLastFour: common.StringProperty("The card's last four digits when the card is "+
			"not on file."+keep, fuelCardLastFourDigits),
		paramTransactionReference: common.StringProperty("The receipt or transaction number; a "+
			"purchase with a reference already on file is refused."+keep,
			maxFuelReferenceText),
		paramTaxPaid: common.BooleanProperty("Whether fuel tax was paid at the pump. Defaults to " +
			"true; bulk or untaxed fuel earns no tax-paid credit." + keep),
		paramNotes: common.StringProperty("Anything the person reconciling it should know."+keep,
			maxFuelNotes),
	}
	if forCorrection {
		properties[paramFuelPurchaseID] = agenttoolschema.RecordIDText(
			permission.ResourceFuelPurchase, fuelPurchaseSupplier,
		)
	}

	return properties
}

type fuelPurchaseInput struct {
	values   map[string]any
	timezone string
}

func (in fuelPurchaseInput) given(key string) bool {
	_, ok := in.values[key]

	return ok
}

func (in fuelPurchaseInput) applyIDs(purchase *fuelpurchase.FuelPurchase) error {
	for key, set := range map[string]func(pulid.ID){
		common.ParamTractorID: func(id pulid.ID) { purchase.TractorID = id },
		paramJurisdictionID:   func(id pulid.ID) { purchase.JurisdictionID = id },
		common.ParamWorkerID:  func(id pulid.ID) { purchase.WorkerID = &id },
		paramFuelCardID:       func(id pulid.ID) { purchase.FuelCardID = &id },
	} {
		if !in.given(key) {
			continue
		}
		id, err := common.RequirePulid(in.values, key)
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
		text, err := common.OptionalBoundedText(in.values, key, field.limit)
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
		quantity, present, err := common.OptionalDecimal(in.values, paramQuantity)
		if err != nil {
			return err
		}
		if !present || !quantity.IsPositive() {
			return fmt.Errorf("parameter %q must be greater than zero", paramQuantity)
		}
		purchase.Quantity = quantity
	}
	if in.given(paramUnitPrice) {
		price, present, err := common.OptionalDecimal(in.values, paramUnitPrice)
		if err != nil {
			return err
		}
		if present && !price.IsPositive() {
			return fmt.Errorf("parameter %q must be greater than zero", paramUnitPrice)
		}
		purchase.UnitPrice = decimal.NullDecimal{Decimal: price, Valid: present}
	}
	if in.given(paramTotalAmount) {
		total, err := common.RequireMoney(in.values, paramTotalAmount)
		if err != nil {
			return err
		}
		purchase.TotalAmountMinor = total
	}
	if in.given(paramOdometer) {
		odometer, err := common.RequireIntInRange(in.values, paramOdometer, 0, maxOdometer)
		if err != nil {
			return err
		}
		reading := int64(odometer)
		purchase.Odometer = &reading
	}

	return nil
}

func (in fuelPurchaseInput) purchasedAt() (int64, error) {
	raw, err := common.RequireString(in.values, paramPurchasedAt)
	if err != nil {
		return 0, err
	}
	if instant, parseErr := common.ParseDateTime(paramPurchasedAt, raw); parseErr == nil {
		return instant, nil
	}

	if strings.Contains(raw, ":") {
		return 0, common.DateRefusal(paramPurchasedAt, raw, toolschema.FormatDateTime)
	}
	clk := filtercatalog.NewClock(in.timezone)
	day, err := filtercatalog.CoerceDate(paramPurchasedAt, raw, clk)
	if err != nil {
		return 0, common.DateRefusal(paramPurchasedAt, raw, toolschema.FormatDateTime)
	}

	return clk.DayStart(day) + noonOffsetSeconds, nil
}

func (in fuelPurchaseInput) applyChoices(purchase *fuelpurchase.FuelPurchase) error {
	if in.given(paramPurchasedAt) {
		purchasedAt, err := in.purchasedAt()
		if err != nil {
			return err
		}
		purchase.PurchasedAt = purchasedAt
	}
	if in.given(common.ParamFuelType) {
		fuelType, err := common.RequireEnum(
			in.values,
			common.ParamFuelType,
			common.FuelTypes.Values,
		)
		if err != nil {
			return err
		}
		purchase.FuelType = fuelType
	}
	if in.given(paramQuantityUnit) {
		unit, err := common.RequireEnum(in.values, paramQuantityUnit, quantityUnits.Values)
		if err != nil {
			return err
		}
		purchase.QuantityUnit = unit
	}
	if in.given(paramCurrencyCode) {
		code := strings.ToUpper(
			strings.TrimSpace(common.OptionalString(in.values, paramCurrencyCode)),
		)
		if len(code) != common.CurrencyCodeLength {
			return fmt.Errorf("%s must be a three-letter code such as USD", paramCurrencyCode)
		}
		purchase.CurrencyCode = code
	}
	taxPaid, err := common.OptionalBoolPointer(in.values, paramTaxPaid)
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
	params *agentports.ToolExecuteParams,
) (*fuelpurchaseservice.CreatePurchaseRequest, error) {
	purchase := &fuelpurchase.FuelPurchase{
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		QuantityUnit:   fuelpurchase.QuantityUnitGallon,
		CurrencyCode:   money.DefaultCurrencyCode,
		TaxPaid:        true,
	}
	input := fuelPurchaseInput{values: params.Params, timezone: params.Timezone}
	if err := input.apply(purchase); err != nil {
		return nil, err
	}
	if purchase.TotalAmountMinor == 0 {
		if !purchase.UnitPrice.Valid || !purchase.Quantity.IsPositive() {
			return nil, fmt.Errorf(
				"give %q, the total paid, or %q with %q so the total can be worked out",
				paramTotalAmount, paramUnitPrice, paramQuantity,
			)
		}
		purchase.TotalAmountMinor = money.MinorUnits(
			purchase.Quantity.Mul(purchase.UnitPrice.Decimal).Round(2),
		)
	}

	return &fuelpurchaseservice.CreatePurchaseRequest{
		TenantInfo: common.TenantFrom(*params),
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
		Version:  common.PinnedVersion(purchase.Version),
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
		Record: common.RecordOf(fuelPurchaseRecordEntity, purchase.ID),
	}
}

func targetFuelPurchase(params map[string]any) (agentports.ToolTarget, bool) {
	return common.TargetOf(params, paramFuelPurchaseID, permission.ResourceFuelPurchase)
}

type labelledPurchase struct {
	purchase     *fuelpurchase.FuelPurchase
	jurisdiction string
}

func NewRecordFuelPurchaseTool(
	purchases FuelPurchaseKeeper,
	jurisdictions jurisdictionLabeler,
) agentports.AgentTool {
	return common.NewReportingReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name:   "fuel_record_purchase",
		Recipe: []string{"fuel_list_purchases", "fuel_record_purchase"},
		Description: "Record fuel bought for a tractor from a receipt or a statement line: " +
			"where it was bought for IFTA, when, what fuel, how much and what it cost. It is " +
			"a tax record the quarter's IFTA return is computed from. Check " +
			"fuel_list_purchases first, since a reference already on file is refused.",
		Resource:   permission.ResourceFuelPurchase,
		Operation:  permission.OpCreate,
		Artifact:   fuelPurchaseRecordEntity,
		Reversible: true,
		TaintHold:  fuelTaintHold,
		Rationale: "Adds a fuel purchase inside Trenova that the next IFTA return counts; " +
			"nothing is sent, and fuel_delete_purchase removes it.",
		Properties: fuelPurchaseProperties(false),
		Required: []string{
			common.ParamTractorID,
			paramJurisdictionID,
			paramPurchasedAt,
			common.ParamFuelType,
			paramQuantity,
		},
		SearchTerms: []string{"fuel", "receipt", "diesel", "gallons", searchTermIFTA, "pump"},
	}), common.ReceivablePlan[*fuelpurchaseservice.CreatePurchaseRequest, *labelledPurchase]{
		Request: newFuelPurchaseDraft,
		Plan: func(
			ctx context.Context,
			req *fuelpurchaseservice.CreatePurchaseRequest,
			_ *agentports.ToolExecuteParams,
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
		Refused: func(*fuelpurchaseservice.CreatePurchaseRequest) string {
			return "Would record a fuel purchase."
		},
		Render: func(
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
				common.DayLabel(plan.purchase.PurchasedAt),
				money.DecimalFromMinor(plan.purchase.TotalAmountMinor).StringFixed(2),
				plan.purchase.CurrencyCode,
			), change), nil
		},
		Run: func(
			ctx context.Context,
			req *fuelpurchaseservice.CreatePurchaseRequest,
			_ *agentports.ToolExecuteParams,
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
	params *agentports.ToolExecuteParams,
) (*fuelPurchaseCorrection, error) {
	id, err := common.RequirePulid(params.Params, paramFuelPurchaseID)
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
	purchases FuelPurchaseKeeper,
	params *agentports.ToolExecuteParams,
) (*fuelpurchaseservice.UpdatePurchaseRequest, error) {
	stored, err := purchases.GetPurchase(ctx, common.TenantFrom(*params), c.id)
	if err != nil {
		return nil, err
	}

	corrected := *stored
	corrected.Tractor = nil
	corrected.Worker = nil
	corrected.FuelCard = nil
	corrected.Jurisdiction = nil
	corrected.ImportBatch = nil
	if err = (fuelPurchaseInput{values: c.values, timezone: params.Timezone}).apply(
		&corrected,
	); err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.UpdatePurchaseRequest{
		TenantInfo: common.TenantFrom(*params),
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

func NewCorrectFuelPurchaseTool(
	purchases FuelPurchaseKeeper,
	jurisdictions jurisdictionLabeler,
) agentports.AgentTool {
	return common.NewReportingReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name:        "fuel_correct_purchase",
		SearchTerms: []string{"fix fuel purchase", "keyed wrong", "wrong gallons"},
		Description: "Correct a recorded fuel purchase: the tractor, jurisdiction, day, " +
			"fuel type, quantity, amount or any other field that was keyed wrong. Fields left " +
			"out keep their value. A finalized IFTA return keeps its figures until it is " +
			"reopened or amended.",
		Resource:   permission.ResourceFuelPurchase,
		Operation:  permission.OpUpdate,
		Artifact:   fuelPurchaseRecordEntity,
		Reversible: true,
		TaintHold:  fuelTaintHold,
		Rationale: "Changes a fuel purchase inside Trenova; nothing is sent, and a later " +
			"correction changes it back.",
		Properties: fuelPurchaseProperties(true),
		Required:   []string{paramFuelPurchaseID},
		Target:     targetFuelPurchase,
	}), common.ReceivablePlan[*fuelPurchaseCorrection, *labelledPurchaseChange]{
		Request: fuelPurchaseCorrectionFrom,
		Plan: func(
			ctx context.Context,
			correction *fuelPurchaseCorrection,
			params *agentports.ToolExecuteParams,
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
		Refused: func(*fuelPurchaseCorrection) string {
			return "Would correct a fuel purchase."
		},
		Render: func(
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
				common.DayLabel(plan.change.Before.PurchasedAt),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			correction *fuelPurchaseCorrection,
			params *agentports.ToolExecuteParams,
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
	purchases FuelPurchaseKeeper,
	params *agentports.ToolExecuteParams,
	id pulid.ID,
) (*fuelpurchaseservice.DeletePurchaseRequest, error) {
	stored, err := purchases.GetPurchase(ctx, common.TenantFrom(*params), id)
	if err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.DeletePurchaseRequest{
		TenantInfo: common.TenantFrom(*params),
		ID:         stored.ID,
		Version:    stored.Version,
		UserID:     params.Actor.UserID,
	}, nil
}

func NewDeleteFuelPurchaseTool(
	purchases FuelPurchaseKeeper,
	jurisdictions jurisdictionLabeler,
) agentports.AgentTool {
	return common.NewReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name: "fuel_delete_purchase",
		Description: "Propose deleting a fuel purchase recorded in error, such as a " +
			"receipt entered twice. It is removed outright; a return already generated for " +
			"its quarter keeps its figures until it is recomputed. To fix a wrong field use " +
			"fuel_correct_purchase instead.",
		Resource:  permission.ResourceFuelPurchase,
		Operation: permission.OpDelete,
		MaxTier:   agent.TierPropose,
		Rationale: "Removes a tax record the IFTA return is computed from, and nothing " +
			"brings it back but entering it again, so a person always decides.",
		Properties: map[string]any{
			paramFuelPurchaseID: agenttoolschema.RecordIDText(permission.ResourceFuelPurchase,
				fuelPurchaseSupplier),
		},
		Required: []string{paramFuelPurchaseID},
		Target:   targetFuelPurchase,
	}), common.ReceivablePlan[pulid.ID, *labelledPurchase]{
		Request: func(params *agentports.ToolExecuteParams) (pulid.ID, error) {
			return common.RequirePulid(params.Params, paramFuelPurchaseID)
		},
		Plan: func(
			ctx context.Context,
			id pulid.ID,
			params *agentports.ToolExecuteParams,
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
		Refused: func(pulid.ID) string {
			return "Would delete a fuel purchase."
		},
		Render: func(_ pulid.ID, plan *labelledPurchase) (*agent.ToolPreview, error) {
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
				common.DayLabel(plan.purchase.PurchasedAt),
			), change), nil
		},
		Run: func(
			ctx context.Context,
			id pulid.ID,
			params *agentports.ToolExecuteParams,
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

func fuelCardAssignmentFrom(params *agentports.ToolExecuteParams) (*fuelCardAssignment, error) {
	id, err := common.RequirePulid(params.Params, paramFuelCardID)
	if err != nil {
		return nil, err
	}
	tractor, err := common.OptionalPulidParam(params.Params, paramAssignedTractorID)
	if err != nil {
		return nil, err
	}
	worker, err := common.OptionalPulidParam(params.Params, paramAssignedWorkerID)
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
	cards FuelCardKeeper,
	params *agentports.ToolExecuteParams,
) (*fuelpurchaseservice.AssignCardRequest, error) {
	stored, err := cards.GetCard(ctx, common.TenantFrom(*params), a.id)
	if err != nil {
		return nil, err
	}

	return &fuelpurchaseservice.AssignCardRequest{
		TenantInfo:        common.TenantFrom(*params),
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

func NewAssignFuelCardTool(cards FuelCardKeeper) agentports.AgentTool {
	return common.NewReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name:        "fuel_assign_card",
		SearchTerms: []string{"issue new fuel card", "fuel card to tractor", "hand out fuel card"},
		Description: "Assign a fuel card to the tractor or driver that carries it, most " +
			"often a card a feed discovered. Assigning a suspended card also activates it, " +
			"and once assigned, fuel_resolve_purchase_import_rows posts the purchases that " +
			"waited for it.",
		Resource:   permission.ResourceFuelCard,
		Operation:  permission.OpUpdate,
		Reversible: true,
		Rationale: "Changes which tractor a card's purchases count against inside Trenova; " +
			"nothing is sent, and assigning it again changes it back.",
		Properties: map[string]any{
			paramFuelCardID: agenttoolschema.RecordIDText(permission.ResourceFuelCard,
				fuelCardSupplier),
			paramAssignedTractorID: agenttoolschema.RecordIDText(permission.ResourceTractor,
				"The tractor that carries the card, from equipment_list_tractors."),
			paramAssignedWorkerID: agenttoolschema.RecordIDText(permission.ResourceWorker,
				"The driver who carries the card, from worker_search or worker_list."),
		},
		Required: []string{paramFuelCardID},
		Target: func(params map[string]any) (agentports.ToolTarget, bool) {
			return common.TargetOf(params, paramFuelCardID, permission.ResourceFuelCard)
		},
	}), common.ReceivablePlan[*fuelCardAssignment, *fuelpurchaseservice.CardChange]{
		Request: fuelCardAssignmentFrom,
		Plan: func(
			ctx context.Context,
			assignment *fuelCardAssignment,
			params *agentports.ToolExecuteParams,
		) (*fuelpurchaseservice.CardChange, error) {
			req, err := assignment.request(ctx, cards, params)
			if err != nil {
				return nil, err
			}

			return cards.PlanAssignCard(ctx, req)
		},
		Refused: func(*fuelCardAssignment) string {
			return "Would assign a fuel card."
		},
		Render: func(
			_ *fuelCardAssignment,
			plan *fuelpurchaseservice.CardChange,
		) (*agent.ToolPreview, error) {
			change, err := toolpreview.Changed(
				toolpreview.Record{
					Resource: permission.ResourceFuelCard,
					ID:       plan.Before.ID,
					Label:    plan.Before.Provider.Label() + " card ending " + plan.Before.LastFour,
					Version:  common.PinnedVersion(plan.Before.Version),
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
		Run: func(
			ctx context.Context,
			assignment *fuelCardAssignment,
			params *agentports.ToolExecuteParams,
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
) func(map[string]any) (agentports.ToolTarget, bool) {
	return func(params map[string]any) (agentports.ToolTarget, bool) {
		return common.TargetOf(params, paramFuelImportID, resource)
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
		Version:  common.PinnedVersion(batch.Version),
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
	imports FuelImportKeeper,
	params *agentports.ToolExecuteParams,
) (*fuelpurchase.ImportBatch, error) {
	id, err := common.RequirePulid(params.Params, paramFuelImportID)
	if err != nil {
		return nil, err
	}

	return imports.GetImport(ctx, common.TenantFrom(*params), id)
}

func fuelImportIDFrom(params *agentports.ToolExecuteParams) (pulid.ID, error) {
	return common.RequirePulid(params.Params, paramFuelImportID)
}

func NewCommitFuelPurchaseImportTool(imports FuelImportKeeper) agentports.AgentTool {
	return common.NewReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name:   "fuel_commit_purchase_import",
		Recipe: []string{"fuel_list_purchase_imports", "fuel_commit_purchase_import"},
		Description: "Commit a staged fuel card statement: every row ready to commit becomes " +
			"a fuel purchase in one step, and rows whose reference is already on file are " +
			"skipped. Read it with fuel_list_purchase_imports first; rows in error stay " +
			"behind.",
		Resource:  permission.ResourceFuelPurchase,
		Operation: permission.OpImport,
		Rationale: "Adds a statement's purchases inside Trenova, which the IFTA return counts; " +
			"nothing is sent, but undoing it means deleting each purchase.",
		Properties: map[string]any{
			paramFuelImportID: agenttoolschema.RecordIDText(permission.ResourceFuelPurchaseImport,
				fuelImportSupplier),
		},
		Required: []string{paramFuelImportID},
		Target:   targetFuelImport(permission.ResourceFuelPurchase),
	}), common.ReceivablePlan[pulid.ID, *fuelpurchaseservice.CommitPlan]{
		Request: fuelImportIDFrom,
		Plan: func(
			ctx context.Context,
			_ pulid.ID,
			params *agentports.ToolExecuteParams,
		) (*fuelpurchaseservice.CommitPlan, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}

			return imports.PlanCommit(ctx, &fuelpurchaseservice.CommitRequest{
				TenantInfo: common.TenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})
		},
		Refused: func(pulid.ID) string {
			return "Would commit a fuel purchase import."
		},
		Render: func(_ pulid.ID, plan *fuelpurchaseservice.CommitPlan) (*agent.ToolPreview, error) {
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
				common.CountOf(len(plan.Purchases), fuelPurchaseKind),
				fuelImportRecord(plan.Batch).Label,
				gallons.StringFixed(3),
				money.DecimalFromMinor(total).StringFixed(2),
				plan.Batch.DefaultCurrency,
			), change), nil
		},
		Run: func(
			ctx context.Context,
			_ pulid.ID,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}
			_, err = imports.Commit(ctx, &fuelpurchaseservice.CommitRequest{
				TenantInfo: common.TenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})

			return nil, err
		},
	})
}

func NewResolveFuelPurchaseImportRowsTool(imports FuelImportKeeper) agentports.AgentTool {
	return common.NewReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name: "fuel_resolve_purchase_import_rows",
		Description: "Work out again the rows a fuel import is still holding, against the " +
			"cards, tractors and jurisdictions on file now, without fetching the statement " +
			"again. Run it after fuel_assign_card or once a missing tractor exists. A feed " +
			"posts the rows that now resolve; an upload waits for " +
			"fuel_commit_purchase_import.",
		Resource:  permission.ResourceFuelPurchaseImport,
		Operation: permission.OpImport,
		Rationale: "Re-matches held statement rows inside Trenova and, for a feed, posts the " +
			"purchases that now resolve; nothing is sent.",
		Properties: map[string]any{
			paramFuelImportID: agenttoolschema.RecordIDText(permission.ResourceFuelPurchaseImport,
				fuelImportSupplier),
		},
		Required: []string{paramFuelImportID},
		Target:   targetFuelImport(permission.ResourceFuelPurchaseImport),
	}), common.ReceivablePlan[pulid.ID, *fuelpurchaseservice.ResolveRowsPlan]{
		Request: fuelImportIDFrom,
		Plan: func(
			ctx context.Context,
			_ pulid.ID,
			params *agentports.ToolExecuteParams,
		) (*fuelpurchaseservice.ResolveRowsPlan, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}

			return imports.PlanResolveRows(ctx, &fuelpurchaseservice.ResolveRowsRequest{
				TenantInfo: common.TenantFrom(*params),
				BatchID:    batch.ID,
				Version:    batch.Version,
				UserID:     params.Actor.UserID,
			})
		},
		Refused: func(pulid.ID) string {
			return "Would work out a fuel import's held rows again."
		},
		Render: func(
			_ pulid.ID,
			plan *fuelpurchaseservice.ResolveRowsPlan,
		) (*agent.ToolPreview, error) {
			outcome := "they wait for fuel_commit_purchase_import"
			if plan.Batch.IsFeed() {
				outcome = "the feed posts them as purchases"
			}

			return toolpreview.Build(fmt.Sprintf(
				"Would work out %s held in %s again: %d now resolve and %d still wait; %s.",
				common.CountOf(plan.Reviewed, "row"),
				fuelImportRecord(plan.Batch).Label,
				plan.Resolved,
				plan.Queued,
				outcome,
			)), nil
		},
		Run: func(
			ctx context.Context,
			_ pulid.ID,
			params *agentports.ToolExecuteParams,
		) (*agent.ToolExecutionResult, error) {
			batch, err := loadFuelImport(ctx, imports, params)
			if err != nil {
				return nil, err
			}
			_, err = imports.ResolveRows(ctx, &fuelpurchaseservice.ResolveRowsRequest{
				TenantInfo: common.TenantFrom(*params),
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

func NewDiscardFuelPurchaseImportTool(imports FuelImportKeeper) agentports.AgentTool {
	discardRequest := func(
		ctx context.Context,
		discard *fuelImportDiscard,
		params *agentports.ToolExecuteParams,
	) (*fuelpurchaseservice.DiscardRequest, error) {
		batch, err := imports.GetImport(ctx, common.TenantFrom(*params), discard.id)
		if err != nil {
			return nil, err
		}

		return &fuelpurchaseservice.DiscardRequest{
			TenantInfo: common.TenantFrom(*params),
			BatchID:    batch.ID,
			Version:    batch.Version,
			Reason:     discard.reason,
			UserID:     params.Actor.UserID,
		}, nil
	}

	return common.NewReceivableTool(common.FuelInternalSpec(&common.ReceivableSpec{
		Name: "fuel_discard_purchase_import",
		Description: "Discard a fuel card statement that was staged but should not be " +
			"committed, such as the wrong month or a file loaded twice. Nothing it holds " +
			"becomes a purchase, and it can be staged again later.",
		Resource:   permission.ResourceFuelPurchase,
		Operation:  permission.OpCancel,
		Reversible: true,
		Rationale: "Closes a staged statement inside Trenova without adding any purchase; it " +
			"can be staged again.",
		Properties: map[string]any{
			paramFuelImportID: agenttoolschema.RecordIDText(permission.ResourceFuelPurchaseImport,
				fuelImportSupplier),
			common.ParamReason: common.StringProperty(
				"Why it is being discarded, for whoever finds it "+
					"later.",
				fuelDiscardReason,
			),
		},
		Required: []string{paramFuelImportID},
		Target:   targetFuelImport(permission.ResourceFuelPurchase),
	}), common.ReceivablePlan[*fuelImportDiscard, *fuelpurchaseservice.ImportBatchChange]{
		Request: func(params *agentports.ToolExecuteParams) (*fuelImportDiscard, error) {
			id, err := common.RequirePulid(params.Params, paramFuelImportID)
			if err != nil {
				return nil, err
			}
			reason, err := common.BoundedText(params.Params, common.ParamReason, fuelDiscardReason)
			if err != nil {
				return nil, err
			}

			return &fuelImportDiscard{id: id, reason: reason}, nil
		},
		Plan: func(
			ctx context.Context,
			discard *fuelImportDiscard,
			params *agentports.ToolExecuteParams,
		) (*fuelpurchaseservice.ImportBatchChange, error) {
			req, err := discardRequest(ctx, discard, params)
			if err != nil {
				return nil, err
			}

			return imports.PlanDiscard(ctx, req)
		},
		Refused: func(*fuelImportDiscard) string {
			return "Would discard a fuel purchase import."
		},
		Render: renderFuelImportDiscard,
		Run: func(
			ctx context.Context,
			discard *fuelImportDiscard,
			params *agentports.ToolExecuteParams,
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

func renderFuelImportDiscard(
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
}
