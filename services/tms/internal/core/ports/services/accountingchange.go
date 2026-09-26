package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/shopspring/decimal"
)

type AccountingChangeOperation string

const (
	AccountingChangeUpsert = AccountingChangeOperation("Upsert")
	AccountingChangeVoid   = AccountingChangeOperation("Void")
	AccountingChangeDelete = AccountingChangeOperation("Delete")
)

type AccountingChangeFeedLimits struct {
	MaxLookback    time.Duration
	ReportsDeletes bool
	MaxPerRead     int
}

type ReadAccountingChangesRequest struct {
	Auth           AccountingDocumentAuth
	Cursor         string
	Now            time.Time
	Payments       bool
	BillPayments   bool
	Documents      bool
	ReferenceKinds []accountingsync.ReferenceKind
}

type AccountingInboundLine struct {
	DocumentKind       accountingsync.InboundDocumentKind
	DocumentExternalID string
	Amount             decimal.Decimal
}

type AccountingInboundPayment struct {
	Kind              accountingsync.InboundChangeKind
	Operation         AccountingChangeOperation
	ExternalID        string
	Number            string
	ModifiedAt        int64
	ModifiedBy        string
	PartyExternalID   string
	PartyName         string
	TxnDate           string
	CurrencyCode      string
	Amount            decimal.Decimal
	Unapplied         decimal.Decimal
	MethodExternalID  string
	MethodName        string
	AccountExternalID string
	ReferenceNumber   string
	Lines             []AccountingInboundLine
}

type AccountingChangedReference struct {
	Object  *accountingsync.AccountingReferenceObject
	Deleted bool
}

type AccountingChangedDocument struct {
	ObjectTypes []accountingsync.SyncObjectType
	ExternalID  string
	Operation   AccountingChangeOperation
	ModifiedAt  int64
	ModifiedBy  string
}

type AccountingChangePage struct {
	Payments      []AccountingInboundPayment
	Documents     []AccountingChangedDocument
	References    []AccountingChangedReference
	NextCursor    string
	More          bool
	CursorExpired bool
}

type AccountingChangeReader interface {
	ChangeFeedLimits() AccountingChangeFeedLimits
	ChangeCursorAt(at time.Time) string
	ReadChanges(
		ctx context.Context,
		req *ReadAccountingChangesRequest,
	) (*AccountingChangePage, error)
}
