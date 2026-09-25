package repositories

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PayableKind string

const (
	PayableCarrier = PayableKind("Carrier")
	PayableDriver  = PayableKind("Driver")
)

type PayableJournalLine struct {
	AccountID   pulid.ID
	AccountCode string
	AccountName string
	DebitMinor  int64
	CreditMinor int64
}

func (l *PayableJournalLine) NetMinor() int64 {
	return l.DebitMinor - l.CreditMinor
}

type PayableSettlement struct {
	Kind             PayableKind
	ID               pulid.ID
	Number           string
	PartyID          pulid.ID
	PartyName        string
	OwnerOperator    bool
	PeriodStart      int64
	PeriodEnd        int64
	PayDate          int64
	PostedAt         *int64
	PaidAt           *int64
	NetMinor         int64
	ShipmentCount    int
	CurrencyCode     string
	PaymentMethod    string
	PaymentReference string
	PayableAccountID pulid.ID
	BankAccountID    pulid.ID
	Lines            []PayableJournalLine
	InvoiceNumbers   []string
}

type GetPayableSettlementRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       PayableKind
	ID         pulid.ID
}

type AccountingPayablesSource interface {
	GetSettlement(ctx context.Context, req *GetPayableSettlementRequest) (*PayableSettlement, error)
}
