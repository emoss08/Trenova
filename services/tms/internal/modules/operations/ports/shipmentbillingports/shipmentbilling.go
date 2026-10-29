package shipmentbillingports

import (
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingtransfer"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type ShipmentBillingReadinessPolicy struct {
	ShipmentBillingRequirementEnforcement tenant.EnforcementLevel                    `json:"shipmentBillingRequirementEnforcement"`
	RateValidationEnforcement             tenant.EnforcementLevel                    `json:"rateValidationEnforcement"`
	BillingExceptionDisposition           billingcontrol.BillingExceptionDisposition `json:"billingExceptionDisposition"`
	NotifyOnBillingExceptions             bool                                       `json:"notifyOnBillingExceptions"`
	ReadyToBillAssignmentMode             billingcontrol.ReadyToBillAssignmentMode   `json:"readyToBillAssignmentMode"`
	BillingQueueTransferMode              billingcontrol.BillingQueueTransferMode    `json:"billingQueueTransferMode"`
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
	DocumentTypeID      string                              `json:"documentTypeId"`
	DocumentTypeCode    string                              `json:"documentTypeCode"`
	DocumentTypeName    string                              `json:"documentTypeName"`
	Satisfied           bool                                `json:"satisfied"`
	DocumentCount       int                                 `json:"documentCount"`
	DocumentIDs         []string                            `json:"documentIds"`
	IneligibleDocuments []ShipmentBillingIneligibleDocument `json:"ineligibleDocuments"`
}

type ShipmentBillingIneligibleDocument struct {
	DocumentID string            `json:"documentId"`
	Standing   document.Standing `json:"standing"`
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
	ShipmentStatus               shipmentvocab.Status                 `json:"shipmentStatus"`
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

type TransferShipmentToBillingRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	BillType   billingqueue.BillType `json:"billType"`
}

const MaxBulkTransferToBillingShipments = 100

const MaxBillingTransferCandidateIDs = billingtransfer.MaxRunShipments

type BillingTransferFailureCode = billingtransfer.FailureCode

const BillingTransferFailureNotFound = billingtransfer.FailureNotFound

const BillingTransferFailureInvalidStatus = billingtransfer.FailureInvalidStatus

const BillingTransferFailureAlreadyTransferred = billingtransfer.FailureAlreadyTransferred

const BillingTransferFailureRequirementsUnmet = billingtransfer.FailureRequirementsUnmet

const BillingTransferFailureRateValidation = billingtransfer.FailureRateValidation

const BillingTransferFailureReturnToOperations = billingtransfer.FailureReturnToOperations

const BillingTransferFailureUnexpected = billingtransfer.FailureUnexpected

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
	Status shipmentvocab.Status     `json:"status"`
}

// ListAutoTransferCandidateIDsRequest asks for the shipments an automatic
// transfer may take now. ScanLimit bounds how many recently changed candidates
// are evaluated to find them.
type ListAutoTransferCandidateIDsRequest struct {
	TenantInfo pagination.TenantInfo `json:"tenantInfo"`
	Limit      int                   `json:"limit"`
	ScanLimit  int                   `json:"scanLimit"`
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

// Transfers reports whether the outcome puts the shipment in the queue.
func (o BillingTransferOutcome) Transfers() bool {
	return o == BillingTransferOutcomeTransfer || o == BillingTransferOutcomeMarkReadyAndTransfer
}

// BillingTransferOutcomeTransfer queues the shipment for billing.
const BillingTransferOutcomeTransfer = BillingTransferOutcome("Transfer")

// BillingTransferOutcomeMarkReadyAndTransfer marks a completed shipment
// ready to invoice and then queues it.
const BillingTransferOutcomeMarkReadyAndTransfer = BillingTransferOutcome("MarkReadyAndTransfer")

// BillingTransferOutcomeRefused leaves the shipment where it is; the
// failure code says why.
const BillingTransferOutcomeRefused = BillingTransferOutcome("Refused")

// BillingTransferOutcomeReturnToOperations leaves the shipment with
// operations, whose issues the organization corrects there.
const BillingTransferOutcomeReturnToOperations = BillingTransferOutcome("ReturnToOperations")

// BillingTransferDecision is one shipment's answer: what a transfer would do
// and the readiness it was decided from, by the same checks a transfer makes.
type BillingTransferDecision struct {
	ShipmentID pulid.ID             `json:"shipmentId"`
	ProNumber  string               `json:"proNumber"`
	Status     shipmentvocab.Status `json:"status"`
	// BillingTransferStatus is the stage the shipment is at now: empty for
	// one billing has never received, SentBackToOps for one it returned.
	BillingTransferStatus shipmentvocab.BillingTransferStatus `json:"billingTransferStatus"`
	CustomerID            pulid.ID                            `json:"customerId"`
	CustomerName          string                              `json:"customerName"`
	TotalCharge           decimal.NullDecimal                 `json:"totalCharge"`
	CurrencyCode          string                              `json:"currencyCode"`
	DeliveredAt           *int64                              `json:"deliveredAt"`
	Outcome               BillingTransferOutcome              `json:"outcome"`
	FailureCode           BillingTransferFailureCode          `json:"failureCode,omitempty"`
	Reason                string                              `json:"reason,omitempty"`
	MissingRequirements   []ShipmentBillingRequirement        `json:"missingRequirements"`
	ValidationFailures    []ShipmentBillingValidation         `json:"validationFailures"`
	Warnings              []ShipmentBillingWarning            `json:"warnings"`
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
	Status                      shipmentvocab.Status
	MarkCompletedReadyToInvoice bool
}

type BillingTransferCandidates struct {
	Decisions  []BillingTransferDecision `json:"decisions"`
	TotalCount int                       `json:"totalCount"`
	HasMore    bool                      `json:"hasMore"`
}

func (r *BulkTransferToBillingResponse) WithoutFormulaContent() *BulkTransferToBillingResponse {
	if r == nil {
		return nil
	}

	results := make([]BulkTransferToBillingResult, len(r.Results))
	for i := range r.Results {
		results[i] = r.Results[i]
		results[i].Item = results[i].Item.WithoutFormulaContent()
		results[i].Items = billingqueue.WithoutFormulaContent(results[i].Items)
	}

	redacted := *r
	redacted.Results = results

	return &redacted
}
