package shipmentserviceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentbillingports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratequote"
	"github.com/emoss08/trenova/internal/modules/reference/domain/modeprofile"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ShipmentUIPolicy struct {
	AllowMoveRemovals      bool                        `json:"allowMoveRemovals"`
	CheckForDuplicateBOLs  bool                        `json:"checkForDuplicateBols"`
	CheckHazmatSegregation bool                        `json:"checkHazmatSegregation"`
	MaxShipmentWeightLimit int32                       `json:"maxShipmentWeightLimit"`
	Profile                *modeprofile.ResolvedPolicy `json:"profile,omitempty"`
}

type ShipmentMutationObserver interface {
	AfterShipmentUpdate(
		ctx context.Context,
		original *shipment.Shipment,
		updated *shipment.Shipment,
		actor *permissionports.RequestActor,
	) error
}

type MoveStatusObserver interface {
	AfterMoveStatusChange(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		move *shipment.ShipmentMove,
		previous shipment.MoveStatus,
	) error
}

// AutoRateShipmentRequest asks the rate agreements to price a shipment again,
// replacing whatever it currently charges.
//
// It is always somebody's deliberate act. A contract prices a shipment once,
// when it is created; after that its rating method, tariff inputs and accessorials
// are ordinary fields, and this is the one thing that overwrites them.
type AutoRateShipmentRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ShipmentID pulid.ID              `json:"-"`
}

// ContractRateApplication describes what the rate agreements would charge, or
// did charge, for a shipment.
//
// It is the answer to both questions the billing panel asks: what would this
// load rate at, and what happened when I asked for it to be rated. The same
// shape serves both because a preview and an application differ only in whether
// the numbers were kept.
type ContractRateApplication struct {
	Applied bool              `json:"applied"`
	Outcome ratequote.Outcome `json:"outcome"`

	AgreementID   *pulid.ID `json:"agreementId,omitempty"`
	AgreementName string    `json:"agreementName,omitempty"`
	RuleID        *pulid.ID `json:"ruleId,omitempty"`
	RuleLabel     string    `json:"ruleLabel,omitempty"`

	FormulaTemplateID   *pulid.ID          `json:"formulaTemplateId,omitempty"`
	FormulaTemplateName string             `json:"formulaTemplateName,omitempty"`
	TemplateVersion     int64              `json:"templateVersion,omitempty"`
	Inputs              tarifftypes.Inputs `json:"inputs,omitempty"`
	Refusal             string             `json:"refusal,omitempty"`

	FreightLines      []shipment.FreightLine `json:"freightLines"`
	FreightAmount     decimal.Decimal        `json:"freightAmount"`
	OtherChargeAmount decimal.Decimal        `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal        `json:"totalChargeAmount"`

	Charges []ContractRateCharge `json:"charges"`

	PreviousFreightAmount decimal.Decimal `json:"previousFreightAmount"`

	Explanation string `json:"explanation,omitempty"`
}

type ContractRateCharge struct {
	AccessorialChargeID pulid.ID             `json:"accessorialChargeId"`
	Code                string               `json:"code"`
	Description         string               `json:"description,omitempty"`
	Owner               shipment.SystemOwner `json:"owner"`
	Label               string               `json:"label,omitempty"`
	Amount              decimal.Decimal      `json:"amount"`
	Quantity            decimal.NullDecimal  `json:"quantity"`
	Rate                decimal.NullDecimal  `json:"rate"`
	RateUnit            string               `json:"rateUnit,omitempty"`
}

type ShipmentRatingOutcome struct {
	Adopted       bool
	Amount        decimal.Decimal
	Currency      string
	AgreementName string
	Explanation   string
}

type ShipmentCreatePlan struct {
	Shipment   *shipment.Shipment
	Rating     *ShipmentRatingOutcome
	Advisories []*errortypes.AdvisoryError
}

// ShipmentCancelPreview is a shipment before and after a cancellation.
type ShipmentCancelPreview struct {
	Before *shipment.Shipment
	After  *shipment.Shipment
}

type ShipmentChangePreview struct {
	Before *shipment.Shipment
	After  *shipment.Shipment
}

type ShipmentAutoRatePreview struct {
	Before      *shipment.Shipment
	After       *shipment.Shipment
	Application *ContractRateApplication
}

type ShipmentDuplicatePreview struct {
	Source *shipment.Shipment
	Copies []*shipment.Shipment
}

type ShipmentService interface {
	List(
		ctx context.Context,
		req *shipmentports.ListShipmentsRequest,
	) (*pagination.CursorListResult[*shipment.Shipment], error)
	Get(
		ctx context.Context,
		req *shipmentports.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
	GetByIDs(
		ctx context.Context,
		req *shipmentports.GetShipmentsByIDsRequest,
	) ([]*shipment.Shipment, error)
	SelectOptions(
		ctx context.Context,
		req *shipmentports.ShipmentSelectOptionsRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	GetUIPolicy(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentUIPolicy, error)
	GetBillingReadiness(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentbillingports.ShipmentBillingReadiness, error)
	GetPreviousRates(
		ctx context.Context,
		req *shipmentports.GetPreviousRatesRequest,
	) (*pagination.ListResult[*shipmentports.PreviousRateSummary], error)
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	GetUnassigned(
		ctx context.Context,
		req *shipmentports.GetUnassignedShipmentsRequest,
	) (*pagination.CursorListResult[*shipment.Shipment], error)
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	Cancel(
		ctx context.Context,
		req *shipmentports.CancelShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	// PreviewCancel checks a cancellation as Cancel does and returns the
	// shipment before and as cancelling it would leave it, writing nothing.
	PreviewCancel(
		ctx context.Context,
		req *shipmentports.CancelShipmentRequest,
		actor *permissionports.RequestActor,
	) (*ShipmentCancelPreview, error)
	// PreviewContractRate answers what the agreements would charge for a
	// shipment that has not been saved, which is what the billing panel offers
	// before anyone commits to it. Nothing is written.
	PreviewContractRate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*ContractRateApplication, error)
	PreviewCreate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *permissionports.RequestActor,
	) (*ShipmentCreatePlan, error)
	// AutoRate prices a saved shipment from its contract again, overwriting its
	// rating method, base rate and contract accessorials.
	AutoRate(
		ctx context.Context,
		req *AutoRateShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, *ContractRateApplication, error)
	PreviewAutoRate(
		ctx context.Context,
		req *AutoRateShipmentRequest,
		actor *permissionports.RequestActor,
	) (*ShipmentAutoRatePreview, error)
	Uncancel(
		ctx context.Context,
		req *shipmentports.UncancelShipmentRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewUncancel(
		ctx context.Context,
		req *shipmentports.UncancelShipmentRequest,
	) (*ShipmentChangePreview, error)
	TransferOwnership(
		ctx context.Context,
		req *shipmentports.TransferOwnershipRequest,
		actor *permissionports.RequestActor,
	) (*shipment.Shipment, error)
	PreviewTransferOwnership(
		ctx context.Context,
		req *shipmentports.TransferOwnershipRequest,
		actor *permissionports.RequestActor,
	) (*ShipmentChangePreview, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		req *shipmentports.DuplicateBOLCheckRequest,
	) error
	CheckHazmatSegregation(
		ctx context.Context,
		req *shipmentports.CheckHazmatSegregationRequest,
	) error
	CalculateLoadingOptimization(
		ctx context.Context,
		req *LoadingOptimizationRequest,
	) (*LoadingOptimizationResult, error)
	GetDelayedShipments(
		ctx context.Context,
		req *shipmentports.GetDelayedShipmentsRequest,
	) ([]*shipment.Shipment, error)
	DelayShipments(
		ctx context.Context,
		req *shipmentports.DelayShipmentsRequest,
		actor *permissionports.RequestActor,
	) ([]*shipment.Shipment, error)
	GetAutoCancelableShipments(
		ctx context.Context,
		req *shipmentports.GetAutoCancelableShipmentsRequest,
	) ([]*shipment.Shipment, error)
	AutoCancelShipments(
		ctx context.Context,
		req *shipmentports.AutoCancelShipmentsRequest,
		actor *permissionports.RequestActor,
	) ([]*shipment.Shipment, error)
	Duplicate(
		ctx context.Context,
		req *shipmentports.BulkDuplicateShipmentRequest,
	) (*shipmentports.ShipmentDuplicateWorkflowResponse, error)
	PreviewDuplicate(
		ctx context.Context,
		req *shipmentports.BulkDuplicateShipmentRequest,
	) (*ShipmentDuplicatePreview, error)
	CalculateTotals(
		ctx context.Context,
		entity *shipment.Shipment,
		userID pulid.ID,
	) (*shipmentports.ShipmentTotalsResponse, error)
	CalculateDistance(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*shipmentdistanceports.DistanceCalculationResponse, error)
	RecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentdistanceports.DistanceCalculationResponse, error)
	PreviewRecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*shipmentdistanceports.ShipmentDistancePreview, error)
	AutoMarkReadyToInvoiceIfEligible(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*shipment.Shipment, error)
	TransferToBilling(
		ctx context.Context,
		req *shipmentbillingports.TransferShipmentToBillingRequest,
		actor *permissionports.RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	TransferToBillingItems(
		ctx context.Context,
		req *shipmentbillingports.TransferShipmentToBillingRequest,
		actor *permissionports.RequestActor,
	) (*billingqueueports.TransferToBillingResult, error)
	BulkTransferToBilling(
		ctx context.Context,
		req *shipmentbillingports.BulkTransferShipmentToBillingRequest,
		actor *permissionports.RequestActor,
	) (*shipmentbillingports.BulkTransferToBillingResponse, error)
	ListBillingTransferCandidateIDs(
		ctx context.Context,
		req *shipmentbillingports.ListBillingTransferCandidateIDsRequest,
	) (*shipmentbillingports.BillingTransferCandidateIDsResponse, error)
	ListAutoTransferCandidateIDs(
		ctx context.Context,
		req *shipmentbillingports.ListAutoTransferCandidateIDsRequest,
	) ([]pulid.ID, error)
	// PlanBillingTransfers answers, for each shipment, what BulkTransferToBilling
	// would do to it, by the same checks and without writing anything.
	PlanBillingTransfers(
		ctx context.Context,
		req *shipmentbillingports.PlanBillingTransfersRequest,
	) (*shipmentbillingports.BillingTransferPlan, error)
	// ListBillingTransferCandidates is a page of the shipments the transfer
	// dialog offers, oldest first, each with its decision.
	ListBillingTransferCandidates(
		ctx context.Context,
		req *shipmentbillingports.ListBillingTransferCandidatesRequest,
	) (*shipmentbillingports.BillingTransferCandidates, error)
}
