package businesscentral

import (
	"context"
	"net/http"
	"strings"
	"time"

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

func (k PartyKind) journalSpec() (journalSpec, error) {
	switch k {
	case PartyCustomer:
		return journalSpec{journals: "customerPaymentJournals", payments: "customerPayments"}, nil
	case PartyVendor:
		return journalSpec{journals: "vendorPaymentJournals", payments: "vendorPayments"}, nil
	default:
		return journalSpec{}, ErrUnknownKind
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
	Kind      PartyKind
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
	ID                     string   `json:"id"`
	Code                   string   `json:"code"`
	DisplayName            string   `json:"displayName"`
	BalancingAccountID     string   `json:"balancingAccountId"`
	BalancingAccountNumber string   `json:"balancingAccountNumber"`
	LastModifiedDateTime   wireTime `json:"lastModifiedDateTime"`
}

func (w *wirePaymentJournal) journal() PaymentJournal {
	return PaymentJournal{
		ID:                     strings.ToLower(w.ID),
		Code:                   w.Code,
		DisplayName:            w.DisplayName,
		BalancingAccountID:     emptyGUID(w.BalancingAccountID),
		BalancingAccountNumber: w.BalancingAccountNumber,
		LastModified:           w.LastModifiedDateTime.time(),
	}
}

type paymentJournalBody struct {
	Code               string `json:"code"`
	DisplayName        string `json:"displayName,omitempty"`
	BalancingAccountID string `json:"balancingAccountId,omitempty"`
}

type wireJournalPayment struct {
	ID                     string          `json:"id"`
	JournalID              string          `json:"journalId"`
	LineNumber             int             `json:"lineNumber"`
	CustomerID             string          `json:"customerId"`
	CustomerNumber         string          `json:"customerNumber"`
	VendorID               string          `json:"vendorId"`
	VendorNumber           string          `json:"vendorNumber"`
	PostingDate            string          `json:"postingDate"`
	DocumentNumber         string          `json:"documentNumber"`
	ExternalDocumentNumber string          `json:"externalDocumentNumber"`
	Amount                 decimal.Decimal `json:"amount"`
	AppliesToInvoiceID     string          `json:"appliesToInvoiceId"`
	AppliesToInvoiceNumber string          `json:"appliesToInvoiceNumber"`
	Description            string          `json:"description"`
	LastModifiedDateTime   wireTime        `json:"lastModifiedDateTime"`
	ETag                   string          `json:"@odata.etag"`
}

func (w *wireJournalPayment) payment() JournalPayment {
	return JournalPayment{
		ID:                     strings.ToLower(w.ID),
		JournalID:              emptyGUID(w.JournalID),
		LineNumber:             w.LineNumber,
		PartyID:                emptyGUID(firstNonBlank(w.CustomerID, w.VendorID)),
		PartyNumber:            firstNonBlank(w.CustomerNumber, w.VendorNumber),
		PostingDate:            outputDate(w.PostingDate),
		DocumentNumber:         w.DocumentNumber,
		ExternalDocumentNumber: w.ExternalDocumentNumber,
		Amount:                 w.Amount,
		AppliesToInvoiceID:     emptyGUID(w.AppliesToInvoiceID),
		AppliesToInvoiceNumber: w.AppliesToInvoiceNumber,
		Description:            w.Description,
		LastModified:           w.LastModifiedDateTime.time(),
		ETag:                   w.ETag,
	}
}

type paymentBody struct {
	CustomerID             string  `json:"customerId,omitempty"`
	VendorID               string  `json:"vendorId,omitempty"`
	PostingDate            string  `json:"postingDate"`
	DocumentNumber         string  `json:"documentNumber,omitempty"`
	ExternalDocumentNumber string  `json:"externalDocumentNumber,omitempty"`
	Amount                 *number `json:"amount"`
	AppliesToInvoiceID     string  `json:"appliesToInvoiceId,omitempty"`
	Description            string  `json:"description,omitempty"`
}

type wireLedgerEntry struct {
	EntryNumber          int64           `json:"entryNumber"`
	PostingDate          string          `json:"postingDate"`
	DocumentNumber       string          `json:"documentNumber"`
	DocumentType         string          `json:"documentType"`
	AccountID            string          `json:"accountId"`
	AccountNumber        string          `json:"accountNumber"`
	Description          string          `json:"description"`
	DebitAmount          decimal.Decimal `json:"debitAmount"`
	CreditAmount         decimal.Decimal `json:"creditAmount"`
	LastModifiedDateTime wireTime        `json:"lastModifiedDateTime"`
}

func (w *wireLedgerEntry) entry() LedgerEntry {
	return LedgerEntry{
		EntryNumber:    w.EntryNumber,
		PostingDate:    outputDate(w.PostingDate),
		DocumentNumber: w.DocumentNumber,
		DocumentType:   enumValue(w.DocumentType),
		AccountID:      emptyGUID(w.AccountID),
		AccountNumber:  w.AccountNumber,
		Description:    w.Description,
		DebitAmount:    w.DebitAmount,
		CreditAmount:   w.CreditAmount,
		LastModified:   w.LastModifiedDateTime.time(),
	}
}

func (c *Client) PaymentJournals(
	ctx context.Context,
	kind PartyKind,
	code string,
) ([]PaymentJournal, error) {
	spec, err := kind.journalSpec()
	if err != nil {
		return nil, err
	}
	filter := ""
	if trimmed := strings.TrimSpace(code); trimmed != "" {
		normalized, codeErr := journalCode(trimmed)
		if codeErr != nil {
			return nil, codeErr
		}
		literal, quoteErr := odataString(normalized)
		if quoteErr != nil {
			return nil, quoteErr
		}
		filter = filterEquals(codeField, literal)
	}
	return collect(ctx, c.core, &listCall{
		endpoint: spec.journals,
		path:     c.collectionPath(spec.journals),
		filter:   filter,
	}, (*wirePaymentJournal).journal)
}

func (c *Client) CreatePaymentJournal(
	ctx context.Context,
	kind PartyKind,
	in *PaymentJournalInput,
) (*PaymentJournal, error) {
	spec, err := kind.journalSpec()
	if err != nil {
		return nil, err
	}
	body, err := in.body()
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: spec.journals + suffixCreate,
		method:   http.MethodPost,
		path:     c.collectionPath(spec.journals),
		body:     body,
		expected: []int{http.StatusCreated},
	}, (*wirePaymentJournal).journal)
}

func (in *PaymentJournalInput) body() (paymentJournalBody, error) {
	code, err := journalCode(in.Code)
	if err != nil {
		return paymentJournalBody{}, err
	}
	name, err := cleanText(in.DisplayName, maxDisplayNameLength)
	if err != nil {
		return paymentJournalBody{}, err
	}
	account, err := optionalGUID(in.BalancingAccountID)
	if err != nil {
		return paymentJournalBody{}, err
	}
	return paymentJournalBody{Code: code, DisplayName: name, BalancingAccountID: account}, nil
}

func journalCode(raw string) (string, error) {
	code := strings.ToUpper(strings.TrimSpace(raw))
	if code == "" || !textWithin(code, MaxJournalCodeLength) || hasControl(code) {
		return "", ErrJournalCodeInvalid
	}
	return code, nil
}

func (c *Client) Payments(
	ctx context.Context,
	kind PartyKind,
	journalID, documentNumber string,
) ([]JournalPayment, error) {
	spec, err := kind.journalSpec()
	if err != nil {
		return nil, err
	}
	journal, err := guid(journalID)
	if err != nil {
		return nil, err
	}
	filter := ""
	if trimmed := strings.TrimSpace(documentNumber); trimmed != "" {
		literal, quoteErr := documentNumberLiteral(trimmed)
		if quoteErr != nil {
			return nil, quoteErr
		}
		filter = filterEquals(documentNumberField, literal)
	}
	return collect(ctx, c.core, &listCall{
		endpoint: spec.payments,
		path:     c.entityPath(spec.journals, journal) + "/" + spec.payments,
		filter:   filter,
	}, (*wireJournalPayment).payment)
}

func (c *Client) CreatePayment(
	ctx context.Context,
	kind PartyKind,
	journalID string,
	in *PaymentInput,
) (*JournalPayment, error) {
	spec, err := kind.journalSpec()
	if err != nil {
		return nil, err
	}
	journal, err := guid(journalID)
	if err != nil {
		return nil, err
	}
	body, err := in.body(kind)
	if err != nil {
		return nil, err
	}
	return fetchOne(ctx, c.core, &call{
		endpoint: spec.payments + suffixCreate,
		method:   http.MethodPost,
		path:     c.entityPath(spec.journals, journal) + "/" + spec.payments,
		body:     body,
		expected: []int{http.StatusCreated},
	}, (*wireJournalPayment).payment)
}

func (c *Client) DeletePayment(ctx context.Context, ref *PaymentRef) error {
	spec, err := ref.Kind.journalSpec()
	if err != nil {
		return err
	}
	journal, err := guid(ref.JournalID)
	if err != nil {
		return err
	}
	payment, err := guid(ref.PaymentID)
	if err != nil {
		return err
	}
	path := c.entityPath(spec.journals, journal) + "/" + spec.payments + "(" + payment + ")"
	return c.remove(ctx, spec.payments+suffixDelete, path, ref.ETag)
}

func (in *PaymentInput) body(kind PartyKind) (paymentBody, error) {
	party, err := guid(in.PartyID)
	if err != nil {
		if strings.TrimSpace(in.PartyID) == "" {
			return paymentBody{}, ErrPartyRequired
		}
		return paymentBody{}, err
	}
	if in.Amount.IsZero() || (kind == PartyVendor && in.Amount.IsNegative()) {
		return paymentBody{}, ErrAmountInvalid
	}
	body := paymentBody{Amount: numberOf(in.Amount)}
	if kind == PartyCustomer {
		body.CustomerID = party
	} else {
		body.VendorID = party
	}
	if body.PostingDate, err = inputDate(in.PostingDate, true); err != nil {
		return paymentBody{}, err
	}
	body.DocumentNumber, err = cleanText(in.DocumentNumber, MaxDocumentNumberLength)
	if err != nil {
		return paymentBody{}, err
	}
	body.ExternalDocumentNumber, err = cleanText(
		in.ExternalDocumentNumber, MaxExternalDocumentNumberLength,
	)
	if err != nil {
		return paymentBody{}, err
	}
	if body.Description, err = cleanText(in.Description, MaxDescriptionLength); err != nil {
		return paymentBody{}, err
	}
	if body.AppliesToInvoiceID, err = optionalGUID(in.AppliesToInvoiceID); err != nil {
		return paymentBody{}, err
	}
	return body, nil
}

func (c *Client) PostJournal(ctx context.Context, journalID string) error {
	id, err := guid(journalID)
	if err != nil {
		return err
	}
	return c.invoke(ctx, "journals-post", c.actionPath(journalsEntity, id, actionPost))
}

func (c *Client) GeneralLedgerEntries(
	ctx context.Context,
	documentNumber string,
) ([]LedgerEntry, error) {
	literal, err := documentNumberLiteral(strings.TrimSpace(documentNumber))
	if err != nil {
		return nil, err
	}
	return collect(ctx, c.core, &listCall{
		endpoint: "general-ledger-entries",
		path:     c.collectionPath(generalLedgerEntriesEntity),
		filter:   filterEquals(documentNumberField, literal),
	}, (*wireLedgerEntry).entry)
}

func documentNumberLiteral(value string) (string, error) {
	if !textWithin(value, MaxDocumentNumberLength) {
		return "", ErrInvalidFilter
	}
	return odataString(value)
}
