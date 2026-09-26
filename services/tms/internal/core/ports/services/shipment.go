package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accessorialcharge"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/modeprofile"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
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

// ShipmentBillingPayerReadiness is one payer's standing on a shipment: what
// they owe, whether their credit allows billing, and whether their profile
// lets clean freight skip review.
type ShipmentBillingPayerReadiness struct {
	PayerID                  pulid.ID              `json:"payerId"`
	PayerName                string                `json:"payerName"`
	PayerCode                string                `json:"payerCode"`
	IsPrimary                bool                  `json:"isPrimary"`
	ShareAmount              decimal.Decimal       `json:"shareAmount"`
	CreditStatus             customer.CreditStatus `json:"creditStatus"`
	CreditHold               bool                  `json:"creditHold"`
	ShouldAutoApproveBilling bool                  `json:"shouldAutoApproveBilling"`
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
	// ShouldAutoApproveBilling means this shipment may clear the billing queue
	// without a biller looking at it, because it has no requirement or rate issue
	// and every payer asked for clean freight to pass straight through.
	ShouldAutoApproveBilling bool `json:"shouldAutoApproveBilling"`
	// Payers is every customer with a share of this shipment, the shipment's own
	// payer first. Requirements above are the union of theirs.
	Payers []ShipmentBillingPayerReadiness `json:"payers"`
}

type JurisdictionMileResult struct {
	CountryCode      string  `json:"countryCode"`
	JurisdictionCode string  `json:"jurisdictionCode"`
	Distance         float64 `json:"distance"`
	DistanceUnits    string  `json:"distanceUnits"`
	Loaded           bool    `json:"loaded"`
}

type DistanceMoveResult struct {
	MoveID              pulid.ID                 `json:"moveId,omitempty"`
	MoveIndex           int                      `json:"moveIndex"`
	Distance            float64                  `json:"distance"`
	Source              string                   `json:"source"`
	Provider            string                   `json:"provider,omitempty"`
	RoutingType         string                   `json:"routingType,omitempty"`
	DataVersion         string                   `json:"dataVersion,omitempty"`
	DistanceUnits       string                   `json:"distanceUnits,omitempty"`
	DistanceProfileID   string                   `json:"distanceProfileId,omitempty"`
	DistanceProfileName string                   `json:"distanceProfileName,omitempty"`
	Warnings            []string                 `json:"warnings,omitempty"`
	JurisdictionMiles   []JurisdictionMileResult `json:"jurisdictionMiles,omitempty"`
	CalculatedAt        int64                    `json:"calculatedAt"`
}

type RecalculateMoveJurisdictionMilesRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	UserID         pulid.ID              `json:"userId"`
}

type BackfillJurisdictionMilesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
	MaxMoves   int                   `json:"maxMoves"`
	DryRun     bool                  `json:"dryRun"`
	UserID     pulid.ID              `json:"userId"`
}

type BackfillJurisdictionMilesResult struct {
	Started           bool            `json:"started"`
	DryRun            bool            `json:"dryRun"`
	UnattributedMoves int             `json:"unattributedMoves"`
	UnattributedMiles decimal.Decimal `json:"unattributedMiles"`
	WorkflowID        string          `json:"workflowId,omitempty"`
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
	RecalculateMoveJurisdictionMiles(
		ctx context.Context,
		req RecalculateMoveJurisdictionMilesRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
	BackfillJurisdictionMiles(
		ctx context.Context,
		req BackfillJurisdictionMilesRequest,
	) (*BackfillJurisdictionMilesResult, error)
}

type TransferShipmentToBillingRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	BillType   billingqueue.BillType `json:"billType"`
}

const (
	MaxBulkTransferToBillingShipments = 100
	MaxBillingTransferCandidateIDs    = billingtransfer.MaxRunShipments
)

type BillingTransferFailureCode = billingtransfer.FailureCode

const (
	BillingTransferFailureNotFound           = billingtransfer.FailureNotFound
	BillingTransferFailureInvalidStatus      = billingtransfer.FailureInvalidStatus
	BillingTransferFailureAlreadyTransferred = billingtransfer.FailureAlreadyTransferred
	BillingTransferFailureRequirementsUnmet  = billingtransfer.FailureRequirementsUnmet
	BillingTransferFailureRateValidation     = billingtransfer.FailureRateValidation
	BillingTransferFailureReturnToOperations = billingtransfer.FailureReturnToOperations
	BillingTransferFailureUnexpected         = billingtransfer.FailureUnexpected
)

type BulkTransferShipmentToBillingRequest struct {
	ShipmentIDs                 []pulid.ID            `json:"shipmentIds"`
	BillType                    billingqueue.BillType `json:"billType"`
	MarkCompletedReadyToInvoice bool                  `json:"markCompletedReadyToInvoice"`

	// OnResult is called as each shipment is answered for, before the whole
	// batch returns. A caller that has to report progress — or that must not
	// lose what already happened if the batch dies partway — records from here
	// rather than waiting for the response.
	OnResult func(BulkTransferToBillingResult) `json:"-"`

	// SuppressExceptionNotifications stops the per-shipment billing-exception
	// notification. A bulk run can carry thousands of shipments, and one global
	// notification per problem shipment is a storm nobody reads; the run's own
	// item rows carry the same missing requirements and validation failures in
	// a form somebody can actually work through.
	SuppressExceptionNotifications bool `json:"-"`
}

func (r *BulkTransferShipmentToBillingRequest) Validate() *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	switch {
	case len(r.ShipmentIDs) == 0:
		multiErr.Add("shipmentIds", errortypes.ErrRequired, "At least one shipment ID is required")
	case len(r.ShipmentIDs) > MaxBulkTransferToBillingShipments:
		multiErr.Add(
			"shipmentIds",
			errortypes.ErrInvalid,
			"A bulk transfer can include at most {0} shipments per request",
			MaxBulkTransferToBillingShipments,
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

type BulkTransferToBillingResult struct {
	ShipmentID           pulid.ID                       `json:"shipmentId"`
	ProNumber            string                         `json:"proNumber,omitempty"`
	Success              bool                           `json:"success"`
	MarkedReadyToInvoice bool                           `json:"markedReadyToInvoice"`
	Item                 *billingqueue.BillingQueueItem `json:"item,omitempty"`
	// Items is every queue item the transfer created, one per payer; Item is
	// the primary payer's and stays for callers that expect one.
	Items               []*billingqueue.BillingQueueItem `json:"items"`
	FailureCode         BillingTransferFailureCode       `json:"failureCode,omitempty"`
	Error               string                           `json:"error,omitempty"`
	Err                 error                            `json:"-"`
	MissingRequirements []ShipmentBillingRequirement     `json:"missingRequirements"`
	ValidationFailures  []ShipmentBillingValidation      `json:"validationFailures"`
}

type BulkTransferToBillingResponse struct {
	Results      []BulkTransferToBillingResult `json:"results"`
	TotalCount   int                           `json:"totalCount"`
	SuccessCount int                           `json:"successCount"`
	ErrorCount   int                           `json:"errorCount"`
}

type ListBillingTransferCandidateIDsRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
	Status shipment.Status          `json:"status"`
}

type BillingTransferCandidateIDsResponse struct {
	IDs        []pulid.ID `json:"ids"`
	TotalCount int        `json:"totalCount"`
	Truncated  bool       `json:"truncated"`
}

// PlanBillingTransfersRequest asks what transferring shipments to billing
// would do, without doing it.
type PlanBillingTransfersRequest struct {
	TenantInfo                  pagination.TenantInfo
	ShipmentIDs                 []pulid.ID
	MarkCompletedReadyToInvoice bool
}

// BillingTransferOutcome is what a transfer would do to one shipment.
type BillingTransferOutcome string

const (
	// BillingTransferOutcomeTransfer queues the shipment for billing.
	BillingTransferOutcomeTransfer = BillingTransferOutcome("Transfer")
	// BillingTransferOutcomeMarkReadyAndTransfer marks a completed shipment
	// ready to invoice and then queues it.
	BillingTransferOutcomeMarkReadyAndTransfer = BillingTransferOutcome("MarkReadyAndTransfer")
	// BillingTransferOutcomeRefused leaves the shipment where it is; the
	// failure code says why.
	BillingTransferOutcomeRefused = BillingTransferOutcome("Refused")
	// BillingTransferOutcomeReturnToOperations leaves the shipment with
	// operations, whose issues the organization corrects there.
	BillingTransferOutcomeReturnToOperations = BillingTransferOutcome("ReturnToOperations")
)

// Transfers reports whether the outcome puts the shipment in the queue.
func (o BillingTransferOutcome) Transfers() bool {
	return o == BillingTransferOutcomeTransfer || o == BillingTransferOutcomeMarkReadyAndTransfer
}

// BillingTransferDecision is one shipment's answer: what a transfer would do
// and the readiness it was decided from, by the same checks a transfer makes.
type BillingTransferDecision struct {
	ShipmentID pulid.ID        `json:"shipmentId"`
	ProNumber  string          `json:"proNumber"`
	Status     shipment.Status `json:"status"`
	// BillingTransferStatus is the stage the shipment is at now: empty for
	// one billing has never received, SentBackToOps for one it returned.
	BillingTransferStatus shipment.BillingTransferStatus `json:"billingTransferStatus"`
	CustomerID            pulid.ID                       `json:"customerId"`
	CustomerName          string                         `json:"customerName"`
	TotalCharge           decimal.NullDecimal            `json:"totalCharge"`
	DeliveredAt           *int64                         `json:"deliveredAt"`
	Outcome               BillingTransferOutcome         `json:"outcome"`
	FailureCode           BillingTransferFailureCode     `json:"failureCode,omitempty"`
	Reason                string                         `json:"reason,omitempty"`
	MissingRequirements   []ShipmentBillingRequirement   `json:"missingRequirements"`
	ValidationFailures    []ShipmentBillingValidation    `json:"validationFailures"`
	Warnings              []ShipmentBillingWarning       `json:"warnings"`
	// AutoApprove means at least one payer's item would clear the queue on
	// its own, by the organization's deterministic rule.
	AutoApprove bool `json:"autoApprove"`
	PayerCount  int  `json:"payerCount"`
	// Version is the shipment's version the decision was read at.
	Version int64 `json:"version"`
}

// BillingTransferPlan is every requested shipment's decision, in request
// order, with the count of each outcome.
type BillingTransferPlan struct {
	Decisions []BillingTransferDecision `json:"decisions"`
	Transfer  int                       `json:"transfer"`
	Refused   int                       `json:"refused"`
	Returned  int                       `json:"returned"`
}

// ListBillingTransferCandidatesRequest is the transfer dialog's list with a
// decision on each row.
type ListBillingTransferCandidatesRequest struct {
	Filter                      *pagination.QueryOptions
	Status                      shipment.Status
	MarkCompletedReadyToInvoice bool
}

type BillingTransferCandidates struct {
	Decisions  []BillingTransferDecision `json:"decisions"`
	TotalCount int                       `json:"totalCount"`
	HasMore    bool                      `json:"hasMore"`
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

type ShipmentRatingOutcome struct {
	Adopted       bool
	Amount        decimal.Decimal
	Currency      string
	AgreementName string
	Explanation   string
}

type ShipmentCreatePlan struct {
	Shipment *shipment.Shipment
	Rating   *ShipmentRatingOutcome
}

// ShipmentCancelPreview is a shipment before and after a cancellation.
type ShipmentCancelPreview struct {
	Before *shipment.Shipment
	After  *shipment.Shipment
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
	// PreviewCancel checks a cancellation as Cancel does and returns the
	// shipment before and as cancelling it would leave it, writing nothing.
	PreviewCancel(
		ctx context.Context,
		req *repositories.CancelShipmentRequest,
		actor *RequestActor,
	) (*ShipmentCancelPreview, error)
	// PreviewContractRate answers what the agreements would charge for a
	// shipment that has not been saved, which is what the billing panel offers
	// before anyone commits to it. Nothing is written.
	PreviewContractRate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*ContractRateApplication, error)
	PreviewCreate(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*ShipmentCreatePlan, error)
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
	TransferToBillingItems(
		ctx context.Context,
		req *TransferShipmentToBillingRequest,
		actor *RequestActor,
	) (*TransferToBillingResult, error)
	BulkTransferToBilling(
		ctx context.Context,
		req *BulkTransferShipmentToBillingRequest,
		actor *RequestActor,
	) (*BulkTransferToBillingResponse, error)
	ListBillingTransferCandidateIDs(
		ctx context.Context,
		req *ListBillingTransferCandidateIDsRequest,
	) (*BillingTransferCandidateIDsResponse, error)
	// PlanBillingTransfers answers, for each shipment, what BulkTransferToBilling
	// would do to it, by the same checks and without writing anything.
	PlanBillingTransfers(
		ctx context.Context,
		req *PlanBillingTransfersRequest,
	) (*BillingTransferPlan, error)
	// ListBillingTransferCandidates is a page of the shipments the transfer
	// dialog offers, oldest first, each with its decision.
	ListBillingTransferCandidates(
		ctx context.Context,
		req *ListBillingTransferCandidatesRequest,
	) (*BillingTransferCandidates, error)
}
