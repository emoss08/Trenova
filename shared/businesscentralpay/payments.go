package businesscentralpay

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/businesscentral"
	"github.com/emoss08/trenova/shared/businesscentralerrors"
	"github.com/emoss08/trenova/shared/businesscentralref"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	journalsEntity             = "journals"
	generalLedgerEntriesEntity = "generalLedgerEntries"
	codeField                  = "code"
	documentNumberField        = "documentNumber"
)

type journalSpec struct {
	journals string
	payments string
}

func journalSpecOf(k businesscentralref.PartyKind) (journalSpec, error) {
	switch k {
	case businesscentralref.PartyCustomer:
		return journalSpec{journals: "customerPaymentJournals", payments: "customerPayments"}, nil
	case businesscentralref.PartyVendor:
		return journalSpec{journals: "vendorPaymentJournals", payments: "vendorPayments"}, nil
	default:
		return journalSpec{}, businesscentralerrors.ErrUnknownKind
	}
}

type PaymentJournal struct {
	ID                     string
	Code                   string
	DisplayName            string
	BalancingAccountID     string
	BalancingAccountNumber string
	LastModified           time.Time
}

type PaymentJournalInput struct {
	Code               string
	DisplayName        string
	BalancingAccountID string
}

type JournalPayment struct {
	ID                     string
	JournalID              string
	LineNumber             int
	PartyID                string
	PartyNumber            string
	PostingDate            string
	DocumentNumber         string
	ExternalDocumentNumber string
	Amount                 decimal.Decimal
	AppliesToInvoiceID     string
	AppliesToInvoiceNumber string
	Description            string
	LastModified           time.Time
	ETag                   string
}

type PaymentInput struct {
	PartyID                string
	PostingDate            string
	DocumentNumber         string
	ExternalDocumentNumber string
	Amount                 decimal.Decimal
	AppliesToInvoiceID     string
	Description            string
}

type PaymentRef struct {
	Kind      businesscentralref.PartyKind
	JournalID string
	PaymentID string
	ETag      string
}

type LedgerEntry struct {
	EntryNumber    int64
	PostingDate    string
	DocumentNumber string
	DocumentType   string
	AccountID      string
	AccountNumber  string
	Description    string
	DebitAmount    decimal.Decimal
	CreditAmount   decimal.Decimal
	LastModified   time.Time
}

func CustomerPaymentAmount(receipt decimal.Decimal) decimal.Decimal {
	return receipt.Neg()
}

type wirePaymentJournal struct {
	ID                     string                   `json:"id"`
	Code                   string                   `json:"code"`
	DisplayName            string                   `json:"displayName"`
	BalancingAccountID     string                   `json:"balancingAccountId"`
	BalancingAccountNumber string                   `json:"balancingAccountNumber"`
	LastModifiedDateTime   businesscentral.WireTime `json:"lastModifiedDateTime"`
}

func (w *wirePaymentJournal) journal() PaymentJournal {
	return PaymentJournal{
		ID:                     strings.ToLower(w.ID),
		Code:                   w.Code,
		DisplayName:            w.DisplayName,
		BalancingAccountID:     businesscentral.EmptyGUID(w.BalancingAccountID),
		BalancingAccountNumber: w.BalancingAccountNumber,
		LastModified:           w.LastModifiedDateTime.Time(),
	}
}

type paymentJournalBody struct {
	Code               string `json:"code"`
	DisplayName        string `json:"displayName,omitempty"`
	BalancingAccountID string `json:"balancingAccountId,omitempty"`
}

type wireJournalPayment struct {
	ID                     string                   `json:"id"`
	JournalID              string                   `json:"journalId"`
	LineNumber             int                      `json:"lineNumber"`
	CustomerID             string                   `json:"customerId"`
	CustomerNumber         string                   `json:"customerNumber"`
	VendorID               string                   `json:"vendorId"`
	VendorNumber           string                   `json:"vendorNumber"`
	PostingDate            string                   `json:"postingDate"`
	DocumentNumber         string                   `json:"documentNumber"`
	ExternalDocumentNumber string                   `json:"externalDocumentNumber"`
	Amount                 decimal.Decimal          `json:"amount"`
	AppliesToInvoiceID     string                   `json:"appliesToInvoiceId"`
	AppliesToInvoiceNumber string                   `json:"appliesToInvoiceNumber"`
	Description            string                   `json:"description"`
	LastModifiedDateTime   businesscentral.WireTime `json:"lastModifiedDateTime"`
	ETag                   string                   `json:"@odata.etag"`
}

func (w *wireJournalPayment) payment() JournalPayment {
	return JournalPayment{
		ID:         strings.ToLower(w.ID),
		JournalID:  businesscentral.EmptyGUID(w.JournalID),
		LineNumber: w.LineNumber,
		PartyID: businesscentral.EmptyGUID(
			stringutils.FirstNonEmptyTrimmed(w.CustomerID, w.VendorID),
		),
		PartyNumber:            stringutils.FirstNonEmptyTrimmed(w.CustomerNumber, w.VendorNumber),
		PostingDate:            businesscentral.OutputDate(w.PostingDate),
		DocumentNumber:         w.DocumentNumber,
		ExternalDocumentNumber: w.ExternalDocumentNumber,
		Amount:                 w.Amount,
		AppliesToInvoiceID:     businesscentral.EmptyGUID(w.AppliesToInvoiceID),
		AppliesToInvoiceNumber: w.AppliesToInvoiceNumber,
		Description:            w.Description,
		LastModified:           w.LastModifiedDateTime.Time(),
		ETag:                   w.ETag,
	}
}

type paymentBody struct {
	CustomerID             string                  `json:"customerId,omitempty"`
	VendorID               string                  `json:"vendorId,omitempty"`
	PostingDate            string                  `json:"postingDate"`
	DocumentNumber         string                  `json:"documentNumber,omitempty"`
	ExternalDocumentNumber string                  `json:"externalDocumentNumber,omitempty"`
	Amount                 *businesscentral.Number `json:"amount"`
	AppliesToInvoiceID     string                  `json:"appliesToInvoiceId,omitempty"`
	Description            string                  `json:"description,omitempty"`
}

type wireLedgerEntry struct {
	EntryNumber          int64                    `json:"entryNumber"`
	PostingDate          string                   `json:"postingDate"`
	DocumentNumber       string                   `json:"documentNumber"`
	DocumentType         string                   `json:"documentType"`
	AccountID            string                   `json:"accountId"`
	AccountNumber        string                   `json:"accountNumber"`
	Description          string                   `json:"description"`
	DebitAmount          decimal.Decimal          `json:"debitAmount"`
	CreditAmount         decimal.Decimal          `json:"creditAmount"`
	LastModifiedDateTime businesscentral.WireTime `json:"lastModifiedDateTime"`
}

func (w *wireLedgerEntry) entry() LedgerEntry {
	return LedgerEntry{
		EntryNumber:    w.EntryNumber,
		PostingDate:    businesscentral.OutputDate(w.PostingDate),
		DocumentNumber: w.DocumentNumber,
		DocumentType:   businesscentral.EnumValue(w.DocumentType),
		AccountID:      businesscentral.EmptyGUID(w.AccountID),
		AccountNumber:  w.AccountNumber,
		Description:    w.Description,
		DebitAmount:    w.DebitAmount,
		CreditAmount:   w.CreditAmount,
		LastModified:   w.LastModifiedDateTime.Time(),
	}
}

func PaymentJournals(
	ctx context.Context,
	c *businesscentral.Client,
	kind businesscentralref.PartyKind,
	code string,
) ([]PaymentJournal, error) {
	spec, err := journalSpecOf(kind)
	if err != nil {
		return nil, err
	}
	filter := ""
	if trimmed := strings.TrimSpace(code); trimmed != "" {
		normalized, codeErr := journalCode(trimmed)
		if codeErr != nil {
			return nil, codeErr
		}
		literal, quoteErr := businesscentral.ODataString(normalized)
		if quoteErr != nil {
			return nil, quoteErr
		}
		filter = businesscentral.FilterEquals(codeField, literal)
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: spec.journals,
		Path:     c.CollectionPath(spec.journals),
		Filter:   filter,
	}, (*wirePaymentJournal).journal)
}

func CreatePaymentJournal(
	ctx context.Context,
	c *businesscentral.Client,
	kind businesscentralref.PartyKind,
	in *PaymentJournalInput,
) (*PaymentJournal, error) {
	spec, err := journalSpecOf(kind)
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: spec.journals + businesscentral.SuffixCreate,
		Method:   http.MethodPost,
		Path:     c.CollectionPath(spec.journals),
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, (*wirePaymentJournal).journal)
}

func (in *PaymentJournalInput) body() (paymentJournalBody, error) {
	code, err := journalCode(in.Code)
	if err != nil {
		return paymentJournalBody{}, err
	}
	name, err := businesscentral.CleanText(in.DisplayName, businesscentral.MaxDisplayNameLength)
	if err != nil {
		return paymentJournalBody{}, err
	}
	account, err := businesscentral.OptionalGUID(in.BalancingAccountID)
	if err != nil {
		return paymentJournalBody{}, err
	}
	return paymentJournalBody{Code: code, DisplayName: name, BalancingAccountID: account}, nil
}

func journalCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" || !businesscentral.TextWithin(code, businesscentral.MaxJournalCodeLength) ||
		businesscentral.HasControl(code) {
		return "", businesscentralerrors.ErrJournalCodeInvalid
	}
	return code, nil
}

func Payments(
	ctx context.Context,
	c *businesscentral.Client,
	kind businesscentralref.PartyKind,
	journalID, documentNumber string,
) ([]JournalPayment, error) {
	spec, err := journalSpecOf(kind)
	if err != nil {
		return nil, err
	}
	journal, err := businesscentral.GUID(journalID)
	if err != nil {
		return nil, err
	}
	filter := ""
	if trimmed := strings.TrimSpace(documentNumber); trimmed != "" {
		literal, quoteErr := documentNumberLiteral(trimmed)
		if quoteErr != nil {
			return nil, quoteErr
		}
		filter = businesscentral.FilterEquals(documentNumberField, literal)
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: spec.payments,
		Path:     c.EntityPath(spec.journals, journal) + "/" + spec.payments,
		Filter:   filter,
	}, (*wireJournalPayment).payment)
}

func CreatePayment(
	ctx context.Context,
	c *businesscentral.Client,
	kind businesscentralref.PartyKind,
	journalID string,
	in *PaymentInput,
) (*JournalPayment, error) {
	spec, err := journalSpecOf(kind)
	if err != nil {
		return nil, err
	}
	journal, err := businesscentral.GUID(journalID)
	if err != nil {
		return nil, err
	}
	body, err := in.body(kind)
	if err != nil {
		return nil, err
	}
	return businesscentral.FetchOne(ctx, c.Core(), &businesscentral.Call{
		Endpoint: spec.payments + businesscentral.SuffixCreate,
		Method:   http.MethodPost,
		Path:     c.EntityPath(spec.journals, journal) + "/" + spec.payments,
		Body:     body,
		Expected: []int{http.StatusCreated},
	}, (*wireJournalPayment).payment)
}

func DeletePayment(ctx context.Context, c *businesscentral.Client, ref *PaymentRef) error {
	spec, err := journalSpecOf(ref.Kind)
	if err != nil {
		return err
	}
	journal, err := businesscentral.GUID(ref.JournalID)
	if err != nil {
		return err
	}
	payment, err := businesscentral.GUID(ref.PaymentID)
	if err != nil {
		return err
	}
	path := c.EntityPath(spec.journals, journal) + "/" + spec.payments + "(" + payment + ")"
	return c.Remove(ctx, spec.payments+businesscentral.SuffixDelete, path, ref.ETag)
}

func (in *PaymentInput) body(kind businesscentralref.PartyKind) (paymentBody, error) {
	party, err := businesscentral.GUID(in.PartyID)
	if err != nil {
		if strings.TrimSpace(in.PartyID) == "" {
			return paymentBody{}, businesscentralerrors.ErrPartyRequired
		}
		return paymentBody{}, err
	}
	if in.Amount.IsZero() || (kind == businesscentralref.PartyVendor && in.Amount.IsNegative()) {
		return paymentBody{}, businesscentralerrors.ErrAmountInvalid
	}
	body := paymentBody{Amount: businesscentral.NumberOf(in.Amount)}
	if kind == businesscentralref.PartyCustomer {
		body.CustomerID = party
	} else {
		body.VendorID = party
	}
	if body.PostingDate, err = businesscentral.InputDate(in.PostingDate, true); err != nil {
		return paymentBody{}, err
	}
	body.DocumentNumber, err = businesscentral.CleanText(
		in.DocumentNumber,
		businesscentral.MaxDocumentNumberLength,
	)
	if err != nil {
		return paymentBody{}, err
	}
	body.ExternalDocumentNumber, err = businesscentral.CleanText(
		in.ExternalDocumentNumber, businesscentral.MaxExternalDocumentNumberLength,
	)
	if err != nil {
		return paymentBody{}, err
	}
	if body.Description, err = businesscentral.CleanText(in.Description, businesscentral.MaxDescriptionLength); err != nil {
		return paymentBody{}, err
	}
	if body.AppliesToInvoiceID, err = businesscentral.OptionalGUID(in.AppliesToInvoiceID); err != nil {
		return paymentBody{}, err
	}
	return body, nil
}

func PostJournal(ctx context.Context, c *businesscentral.Client, journalID string) error {
	id, err := businesscentral.GUID(journalID)
	if err != nil {
		return err
	}
	return c.Invoke(
		ctx,
		"journals-post",
		c.ActionPath(journalsEntity, id, businesscentral.ActionPost),
	)
}

func GeneralLedgerEntries(
	ctx context.Context,
	c *businesscentral.Client,
	documentNumber string,
) ([]LedgerEntry, error) {
	literal, err := documentNumberLiteral(strings.TrimSpace(documentNumber))
	if err != nil {
		return nil, err
	}
	return businesscentral.Collect(ctx, c.Core(), &businesscentral.ListCall{
		Endpoint: "general-ledger-entries",
		Path:     c.CollectionPath(generalLedgerEntriesEntity),
		Filter:   businesscentral.FilterEquals(documentNumberField, literal),
	}, (*wireLedgerEntry).entry)
}

func documentNumberLiteral(value string) (string, error) {
	if !businesscentral.TextWithin(value, businesscentral.MaxDocumentNumberLength) {
		return "", businesscentralerrors.ErrInvalidFilter
	}
	return businesscentral.ODataString(value)
}
