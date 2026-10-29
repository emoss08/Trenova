package factoringledgerports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringledger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type FundingItemInput struct {
	ItemID           pulid.ID                       `json:"itemId"`
	Outcome          factoringledger.FundingOutcome `json:"outcome"`
	AdvanceMinor     *int64                         `json:"advanceMinor"`
	FeeMinor         *int64                         `json:"feeMinor"`
	Reason           string                         `json:"reason"`
	RemittanceLineID pulid.ID                       `json:"-"`
}

type RecordFundingRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	SubmissionID pulid.ID              `json:"submissionId"`
	Version      int64                 `json:"version"`
	FundedOn     int64                 `json:"fundedOn"`
	Items        []FundingItemInput    `json:"items"`
	Memo         string                `json:"memo"`
}

type ReserveMovementRequest struct {
	TenantInfo       pagination.TenantInfo `json:"-"`
	SubmissionID     pulid.ID              `json:"submissionId"`
	ItemID           pulid.ID              `json:"itemId"`
	Version          int64                 `json:"version"`
	OccurredOn       int64                 `json:"occurredOn"`
	CashMinor        *int64                `json:"cashMinor"`
	Reason           string                `json:"reason"`
	RemittanceLineID pulid.ID              `json:"-"`
}

type PlannedLine struct {
	Role      factoring.AccountRole `json:"role"`
	AccountID pulid.ID              `json:"accountId"`
	Debit     int64                 `json:"debit"`
	Credit    int64                 `json:"credit"`
}

type PlannedMovement struct {
	ItemID           pulid.ID                 `json:"itemId"`
	InvoiceID        pulid.ID                 `json:"invoiceId"`
	InvoiceNumber    string                   `json:"invoiceNumber"`
	CustomerID       pulid.ID                 `json:"customerId"`
	Movement         factoringledger.Movement `json:"movement"`
	Lines            []PlannedLine            `json:"lines"`
	RequestedDate    int64                    `json:"requestedDate"`
	AccountingDate   int64                    `json:"accountingDate"`
	FiscalPeriodID   pulid.ID                 `json:"fiscalPeriodId"`
	FiscalPeriodName string                   `json:"fiscalPeriodName"`
	EntryStatus      string                   `json:"entryStatus"`
	RequiresApproval bool                     `json:"requiresApproval"`
}

func (m *PlannedMovement) Redated() bool {
	return m != nil && m.AccountingDate != m.RequestedDate
}

type FundingPlan struct {
	Before       *factoringsubmission.Submission `json:"before"`
	After        *factoringsubmission.Submission `json:"after"`
	Movements    []*PlannedMovement              `json:"movements"`
	FundedCount  int                             `json:"fundedCount"`
	RejectCount  int                             `json:"rejectCount"`
	AdvanceMinor int64                           `json:"advanceMinor"`
	FeeMinor     int64                           `json:"feeMinor"`
	ReserveMinor int64                           `json:"reserveMinor"`
}

type ReserveMovementPlan struct {
	Before   *factoringsubmission.Submission     `json:"before"`
	After    *factoringsubmission.Submission     `json:"after"`
	Item     *factoringsubmission.SubmissionItem `json:"item"`
	Movement *PlannedMovement                    `json:"movement"`
	Reverses *factoringledger.ReserveEntry       `json:"reverses"`
}

type FundingResult struct {
	Submission *factoringsubmission.Submission `json:"submission"`
	Entries    []*factoringledger.ReserveEntry `json:"entries"`
}

type ReserveMovementResult struct {
	Submission *factoringsubmission.Submission `json:"submission"`
	Entry      *factoringledger.ReserveEntry   `json:"entry"`
}

type LedgerService interface {
	ListEntries(
		ctx context.Context,
		req *ListReserveEntryConnectionRequest,
	) (*pagination.CursorListResult[*factoringledger.ReserveEntry], error)
	GetBalance(ctx context.Context, req GetReserveBalanceRequest) (*ReserveBalance, error)
	PlanFunding(
		ctx context.Context,
		req *RecordFundingRequest,
		actor *permissionports.RequestActor,
	) (*FundingPlan, error)
	RecordFunding(
		ctx context.Context,
		req *RecordFundingRequest,
		actor *permissionports.RequestActor,
	) (*FundingResult, error)
	PlanRelease(
		ctx context.Context,
		req *ReserveMovementRequest,
		actor *permissionports.RequestActor,
	) (*ReserveMovementPlan, error)
	ReleaseReserve(
		ctx context.Context,
		req *ReserveMovementRequest,
		actor *permissionports.RequestActor,
	) (*ReserveMovementResult, error)
	PlanChargeback(
		ctx context.Context,
		req *ReserveMovementRequest,
		actor *permissionports.RequestActor,
	) (*ReserveMovementPlan, error)
	RecordChargeback(
		ctx context.Context,
		req *ReserveMovementRequest,
		actor *permissionports.RequestActor,
	) (*ReserveMovementResult, error)
	PlanChargebackReversal(
		ctx context.Context,
		req *ReserveMovementRequest,
		actor *permissionports.RequestActor,
	) (*ReserveMovementPlan, error)
	ReverseChargeback(
		ctx context.Context,
		req *ReserveMovementRequest,
		actor *permissionports.RequestActor,
	) (*ReserveMovementResult, error)
}
