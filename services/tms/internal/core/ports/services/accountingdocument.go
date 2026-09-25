package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/shopspring/decimal"
)

type AccountingDocumentAuth struct {
	RealmID     string
	AccessToken string
}

type AccountingDocumentLimits struct {
	MaxDocNumberLength int
	SupportsDebitMemo  bool
	CanVoidCreditMemo  bool
}

type AccountingDocumentLine struct {
	Description    string
	ItemExternalID string
	Quantity       decimal.Decimal
	UnitPrice      decimal.Decimal
	Amount         decimal.Decimal
	ServiceDate    string
}

type AccountingSalesDocument struct {
	Auth               AccountingDocumentAuth
	RequestID          string
	Kind               accountingsync.SyncObjectType
	CustomerExternalID string
	DocNumber          string
	TxnDate            string
	DueDate            string
	TermExternalID     string
	CurrencyCode       string
	PrivateNote        string
	CustomerMemo       string
	Lines              []AccountingDocumentLine
	ApplyToExternalID  string
	ApplyAmount        decimal.Decimal
	Refs               map[string]string
}

type AccountingPaymentApplication struct {
	InvoiceExternalID        string
	InvoiceNumber            string
	AppliedAmount            decimal.Decimal
	ShortPayAmount           decimal.Decimal
	ShortPayCreditExternalID string
}

type AccountingPaymentDocument struct {
	Auth                     AccountingDocumentAuth
	RequestID                string
	ExternalID               string
	CustomerExternalID       string
	TxnDate                  string
	CurrencyCode             string
	PaymentMethodExternalID  string
	DepositAccountExternalID string
	ReferenceNumber          string
	PrivateNote              string
	TotalAmount              decimal.Decimal
	Applications             []AccountingPaymentApplication
	ShortPayItemExternalID   string
	Refs                     map[string]string
}

type AccountingCreditApplicationDocument struct {
	Auth                 AccountingDocumentAuth
	RequestID            string
	CustomerExternalID   string
	TxnDate              string
	CurrencyCode         string
	InvoiceExternalID    string
	CreditMemoExternalID string
	Amount               decimal.Decimal
	PrivateNote          string
}

type AccountingCustomerDocument struct {
	Auth       AccountingDocumentAuth
	RequestID  string
	ExternalID string
	Party      AccountingPartyDraft
}

type AccountingDocumentRef struct {
	Auth       AccountingDocumentAuth
	RequestID  string
	Kind       accountingsync.SyncObjectType
	ExternalID string
	Refs       map[string]string
}

type AccountingFindDocumentRequest struct {
	Auth      AccountingDocumentAuth
	Kind      accountingsync.SyncObjectType
	DocNumber string
}

type AccountingDocumentResult struct {
	ExternalID string
	DocNumber  string
	Refs       map[string]string
}

type AccountingDocumentWriter interface {
	DocumentLimits() AccountingDocumentLimits
	DocumentURL(kind accountingsync.SyncObjectType, externalID string) string
	ClassifyDocumentError(err error) *accountingsync.SyncError
	UpsertCustomer(
		ctx context.Context,
		doc *AccountingCustomerDocument,
	) (*AccountingDocumentResult, error)
	CreateSalesDocument(
		ctx context.Context,
		doc *AccountingSalesDocument,
	) (*AccountingDocumentResult, error)
	VoidSalesDocument(
		ctx context.Context,
		ref *AccountingDocumentRef,
	) (*AccountingDocumentResult, error)
	SavePayment(
		ctx context.Context,
		doc *AccountingPaymentDocument,
	) (*AccountingDocumentResult, error)
	VoidPayment(
		ctx context.Context,
		ref *AccountingDocumentRef,
	) (*AccountingDocumentResult, error)
	CreateCreditApplication(
		ctx context.Context,
		doc *AccountingCreditApplicationDocument,
	) (*AccountingDocumentResult, error)
	VoidCreditApplication(
		ctx context.Context,
		ref *AccountingDocumentRef,
	) (*AccountingDocumentResult, error)
	FindSalesDocument(
		ctx context.Context,
		req *AccountingFindDocumentRequest,
	) (*AccountingDocumentResult, bool, error)
}
