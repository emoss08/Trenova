package settlementshared

import (
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type Action string

const (
	ActionSubmit           = Action("Submit")
	ActionApprove          = Action("Approve")
	ActionReject           = Action("Reject")
	ActionPost             = Action("Post")
	ActionMarkPaid         = Action("MarkPaid")
	ActionVoid             = Action("Void")
	ActionRecalculate      = Action("Recalculate")
	ActionAddAdjustment    = Action("AddAdjustment")
	ActionRemoveAdjustment = Action("RemoveAdjustment")
)

func (a Action) IsValid() bool {
	switch a {
	case ActionSubmit,
		ActionApprove,
		ActionReject,
		ActionPost,
		ActionMarkPaid,
		ActionVoid,
		ActionRecalculate,
		ActionAddAdjustment,
		ActionRemoveAdjustment:
		return true
	default:
		return false
	}
}

type AdjustmentInput struct {
	Description string
	AmountMinor int64
	Quantity    decimal.Decimal
	Rate        decimal.Decimal
	PayCodeID   *pulid.ID
	GLAccountID *pulid.ID
}

type ActionRequest struct {
	TenantInfo       pagination.TenantInfo
	SettlementID     pulid.ID
	Action           Action
	Reason           string
	PaymentMethod    string
	PaymentReference string
	Adjustment       *AdjustmentInput
	LineID           pulid.ID
}

type JournalLine struct {
	AccountID   pulid.ID
	DebitMinor  int64
	CreditMinor int64
}

type JournalPlan struct {
	AccountingDate   int64
	FiscalYearID     pulid.ID
	FiscalPeriodID   pulid.ID
	EntryStatus      string
	RequiresApproval bool
	Description      string
	Lines            []JournalLine
}

func (p *JournalPlan) TotalDebitMinor() int64 {
	var total int64
	for _, line := range p.Lines {
		total += line.DebitMinor
	}
	return total
}

type ActionPlan[T any] struct {
	Before   T
	After    T
	Journal  *JournalPlan
	AutoPost bool
	Refusal  error
}

func (p *ActionPlan[T]) Refused() bool {
	return p.Refusal != nil
}

func IsRefusal(err error) bool {
	return errortypes.IsError(err) ||
		errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err)
}
