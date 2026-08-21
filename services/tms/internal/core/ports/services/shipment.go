package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
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

type ShipmentBillingReadinessPolicy struct {
	ShipmentBillingRequirementEnforcement tenant.EnforcementLevel            `json:"shipmentBillingRequirementEnforcement"`
	RateValidationEnforcement             tenant.EnforcementLevel            `json:"rateValidationEnforcement"`
	BillingExceptionDisposition           tenant.BillingExceptionDisposition `json:"billingExceptionDisposition"`
	NotifyOnBillingExceptions             bool                               `json:"notifyOnBillingExceptions"`
	ReadyToBillAssignmentMode             tenant.ReadyToBillAssignmentMode   `json:"readyToBillAssignmentMode"`
	BillingQueueTransferMode              tenant.BillingQueueTransferMode    `json:"billingQueueTransferMode"`
}

type ShipmentBillingValidation struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ShipmentBillingWarning struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

type ShipmentServiceFailureBillingContext struct {
	HasUnresolved     bool     `json:"hasUnresolved"`
	UnresolvedCount   int      `json:"unresolvedCount"`
	ServiceFailureIDs []string `json:"serviceFailureIds"`
}

type ShipmentBillingRequirement struct {
	DocumentTypeID   string   `json:"documentTypeId"`
	DocumentTypeCode string   `json:"documentTypeCode"`
	DocumentTypeName string   `json:"documentTypeName"`
	Satisfied        bool     `json:"satisfied"`
	DocumentCount    int      `json:"documentCount"`
	DocumentIDs      []string `json:"documentIds"`
}

type ShipmentBillingReadiness struct {
	ShipmentID                   string                               `json:"shipmentId"`
	ShipmentStatus               shipment.Status                      `json:"shipmentStatus"`
	Policy                       ShipmentBillingReadinessPolicy       `json:"policy"`
	Requirements                 []ShipmentBillingRequirement         `json:"requirements"`
	MissingRequirements          []ShipmentBillingRequirement         `json:"missingRequirements"`
	ValidationFailures           []ShipmentBillingValidation          `json:"validationFailures"`
	Warnings                     []ShipmentBillingWarning             `json:"warnings"`
	ServiceFailureContext        ShipmentServiceFailureBillingContext `json:"serviceFailureContext"`
	CanMarkReadyToInvoice        bool                                 `json:"canMarkReadyToInvoice"`
	ShouldAutoMarkReadyToInvoice bool                                 `json:"shouldAutoMarkReadyToInvoice"`
	ShouldAutoTransferToBilling  bool                                 `json:"shouldAutoTransferToBilling"`
}

type DistanceMoveResult struct {
	MoveID              pulid.ID `json:"moveId,omitempty"`
	MoveIndex           int      `json:"moveIndex"`
	Distance            float64  `json:"distance"`
	Source              string   `json:"source"`
	Provider            string   `json:"provider,omitempty"`
	RoutingType         string   `json:"routingType,omitempty"`
	DataVersion         string   `json:"dataVersion,omitempty"`
	DistanceUnits       string   `json:"distanceUnits,omitempty"`
	DistanceProfileID   string   `json:"distanceProfileId,omitempty"`
	DistanceProfileName string   `json:"distanceProfileName,omitempty"`
	Warnings            []string `json:"warnings,omitempty"`
	CalculatedAt        int64    `json:"calculatedAt"`
}

type DistanceCalculationResponse struct {
	ShipmentID    pulid.ID             `json:"shipmentId,omitempty"`
	TotalDistance float64              `json:"totalDistance"`
	Moves         []DistanceMoveResult `json:"moves"`
}

type DistanceCalculationService interface {
	ResolveForShipment(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*DistanceCalculationResponse, error)
	RecalculateShipment(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*DistanceCalculationResponse, error)
}

type TransferShipmentToBillingRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	BillType   billingqueue.BillType `json:"billType"`
}

type BulkTransferShipmentToBillingRequest struct {
	ShipmentIDs []pulid.ID            `json:"shipmentIds"`
	BillType    billingqueue.BillType `json:"billType"`
}

type BulkTransferToBillingResult struct {
	ShipmentID pulid.ID                       `json:"shipmentId"`
	Success    bool                           `json:"success"`
	Item       *billingqueue.BillingQueueItem `json:"item,omitempty"`
	Error      string                         `json:"error,omitempty"`
}

type BulkTransferToBillingResponse struct {
	Results      []BulkTransferToBillingResult `json:"results"`
	TotalCount   int                           `json:"totalCount"`
	SuccessCount int                           `json:"successCount"`
	ErrorCount   int                           `json:"errorCount"`
}

type ShipmentMutationObserver interface {
	AfterShipmentUpdate(
		ctx context.Context,
		original *shipment.Shipment,
		updated *shipment.Shipment,
		actor *RequestActor,
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
// when it is created; after that its rating method, base rate and accessorials
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
	// Applied is false when no contract covered the lane, in which case nothing
	// was changed and the outcome says why.
	Applied bool              `json:"applied"`
	Outcome ratequote.Outcome `json:"outcome"`

	AgreementID   *pulid.ID `json:"agreementId,omitempty"`
	AgreementName string    `json:"agreementName,omitempty"`
	RuleID        *pulid.ID `json:"ruleId,omitempty"`
	RuleLabel     string    `json:"ruleLabel,omitempty"`

	// FormulaTemplateID and BaseRate are what the contract seats on the
	// shipment: its rating method, and the rate that method prices with.
	FormulaTemplateID   *pulid.ID           `json:"formulaTemplateId,omitempty"`
	FormulaTemplateName string              `json:"formulaTemplateName,omitempty"`
	BaseRate            decimal.NullDecimal `json:"baseRate,omitempty"`

	LinehaulAmount    decimal.Decimal `json:"linehaulAmount"`
	OtherChargeAmount decimal.Decimal `json:"otherChargeAmount"`
	TotalChargeAmount decimal.Decimal `json:"totalChargeAmount"`

	// Accessorials are the charges the contract's own schedule applies. They
	// are listed rather than counted because a rater accepting a re-rate is
	// agreeing to each of them, and a total hides which ones appeared.
	Accessorials []ContractRateAccessorial `json:"accessorials,omitempty"`

	// PreviousLinehaulAmount is what the shipment charged before, so the dialog
	// can state the change rather than only the result.
	PreviousLinehaulAmount decimal.Decimal `json:"previousLinehaulAmount"`

	Explanation string `json:"explanation,omitempty"`
}

// ContractRateAccessorial is one charge a contract's accessorial schedule
// applies automatically.
type ContractRateAccessorial struct {
	AccessorialChargeID pulid.ID                 `json:"accessorialChargeId"`
	Description         string                   `json:"description,omitempty"`
	Method              accessorialcharge.Method `json:"method"`
	Amount              decimal.Decimal          `json:"amount"`
	Unit                int16                    `json:"unit"`
}

type ShipmentService interface {
	List(
		ctx context.Context,
		req *repositories.ListShipmentsRequest,
	) (*pagination.CursorListResult[*shipment.Shipment], error)
	Get(
		ctx context.Context,
		req *repositories.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
	GetByIDs(
		ctx context.Context,
		req *repositories.GetShipmentsByIDsRequest,
	) ([]*shipment.Shipment, error)
	SelectOptions(
		ctx context.Context,
		req *repositories.ShipmentSelectOptionsRequest,
	) (*pagination.ListResult[*shipment.Shipment], error)
	GetUIPolicy(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentUIPolicy, error)
	GetBillingReadiness(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentBillingReadiness, error)
	GetPreviousRates(
		ctx context.Context,
		req *repositories.GetPreviousRatesRequest,
	) (*pagination.ListResult[*repositories.PreviousRateSummary], error)
	Create(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	GetUnassigned(
		ctx context.Context,
		req *repositories.GetUnassignedShipmentsRequest,
	) (*pagination.CursorListResult[*shipment.Shipment], error)
	Update(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	Cancel(
		ctx context.Context,
		req *repositories.CancelShipmentRequest,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	// PreviewContractRate answers what the agreements would charge for a
	// shipment that has not been saved, which is what the billing panel offers
	// before anyone commits to it. Nothing is written.
	PreviewContractRate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*ContractRateApplication, error)
	// AutoRate prices a saved shipment from its contract again, overwriting its
	// rating method, base rate and contract accessorials.
	AutoRate(
		ctx context.Context,
		req *AutoRateShipmentRequest,
		actor *RequestActor,
	) (*shipment.Shipment, *ContractRateApplication, error)
	Uncancel(
		ctx context.Context,
		req *repositories.UncancelShipmentRequest,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	TransferOwnership(
		ctx context.Context,
		req *repositories.TransferOwnershipRequest,
		actor *RequestActor,
	) (*shipment.Shipment, error)
	CheckForDuplicateBOLs(
		ctx context.Context,
		req *repositories.DuplicateBOLCheckRequest,
	) error
	CheckHazmatSegregation(
		ctx context.Context,
		req *repositories.CheckHazmatSegregationRequest,
	) error
	CalculateLoadingOptimization(
		ctx context.Context,
		req *repositories.LoadingOptimizationRequest,
	) (*repositories.LoadingOptimizationResult, error)
	GetDelayedShipments(
		ctx context.Context,
		req *repositories.GetDelayedShipmentsRequest,
	) ([]*shipment.Shipment, error)
	DelayShipments(
		ctx context.Context,
		req *repositories.DelayShipmentsRequest,
		actor *RequestActor,
	) ([]*shipment.Shipment, error)
	GetAutoCancelableShipments(
		ctx context.Context,
		req *repositories.GetAutoCancelableShipmentsRequest,
	) ([]*shipment.Shipment, error)
	AutoCancelShipments(
		ctx context.Context,
		req *repositories.AutoCancelShipmentsRequest,
		actor *RequestActor,
	) ([]*shipment.Shipment, error)
	Duplicate(
		ctx context.Context,
		req *repositories.BulkDuplicateShipmentRequest,
	) (*repositories.ShipmentDuplicateWorkflowResponse, error)
	CalculateTotals(
		ctx context.Context,
		entity *shipment.Shipment,
		userID pulid.ID,
	) (*repositories.ShipmentTotalsResponse, error)
	CalculateDistance(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*DistanceCalculationResponse, error)
	RecalculateDistance(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*DistanceCalculationResponse, error)
	AutoMarkReadyToInvoiceIfEligible(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
		userID pulid.ID,
	) (*shipment.Shipment, error)
	TransferToBilling(
		ctx context.Context,
		req *TransferShipmentToBillingRequest,
		actor *RequestActor,
	) (*billingqueue.BillingQueueItem, error)
	BulkTransferToBilling(
		ctx context.Context,
		req *BulkTransferShipmentToBillingRequest,
		actor *RequestActor,
	) (*BulkTransferToBillingResponse, error)
}
