package quickbooks

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/shopspring/decimal"
)

type PostingType string

const (
	PostingDebit  = PostingType("Debit")
	PostingCredit = PostingType("Credit")
)

type JournalEntityType string

const (
	JournalEntityCustomer = JournalEntityType("Customer")
	JournalEntityVendor   = JournalEntityType("Vendor")
)

const lineDetailJournalEntry = "JournalEntryLineDetail"

var (
	ErrJournalUnbalanced = errors.New(
		"quickbooks: a journal entry's debits and credits must be equal",
	)
	ErrJournalLinesTooFew  = errors.New("quickbooks: a journal entry needs at least two lines")
	ErrJournalAccount      = errors.New("quickbooks: every journal line needs an account")
	ErrJournalPostingType  = errors.New("quickbooks: a journal line is either a debit or a credit")
	ErrJournalEntityTarget = errors.New(
		"quickbooks: a journal line's name needs both a type and an id",
	)
)

type JournalLine struct {
	PostingType PostingType
	AccountID   string
	Amount      decimal.Decimal
	Description string
	EntityType  JournalEntityType
	EntityID    string
}

type JournalTxn struct {
	DocNumber    string
	TxnDate      string
	CurrencyCode string
	ExchangeRate decimal.Decimal
	PrivateNote  string
	Lines        []JournalLine
}

type journalEntity struct {
	Type      string   `json:"Type"`
	EntityRef refValue `json:"EntityRef"`
}

type journalLineDetail struct {
	PostingType string         `json:"PostingType"`
	AccountRef  refValue       `json:"AccountRef"`
	Entity      *journalEntity `json:"Entity,omitempty"`
}

type wireJournalLine struct {
	DetailType             string            `json:"DetailType"`
	Amount                 money             `json:"Amount"`
	Description            string            `json:"Description,omitempty"`
	JournalEntryLineDetail journalLineDetail `json:"JournalEntryLineDetail"`
}

type journalBody struct {
	ID           string            `json:"Id,omitempty"`
	SyncToken    string            `json:"SyncToken,omitempty"`
	DocNumber    string            `json:"DocNumber,omitempty"`
	TxnDate      string            `json:"TxnDate,omitempty"`
	CurrencyRef  *refValue         `json:"CurrencyRef,omitempty"`
	ExchangeRate *exchangeRate     `json:"ExchangeRate,omitempty"`
	PrivateNote  string            `json:"PrivateNote,omitempty"`
	Line         []wireJournalLine `json:"Line"`
}

func (b *journalBody) identify(id, syncToken string) {
	b.ID = id
	b.SyncToken = syncToken
}

func journalBodyOf(requestID string, txn *JournalTxn) (*journalBody, error) {
	if err := validateRequestID(requestID); err != nil {
		return nil, err
	}
	if txn == nil || len(txn.Lines) < 2 {
		return nil, ErrJournalLinesTooFew
	}
	docNumber := strings.TrimSpace(txn.DocNumber)
	if utf8.RuneCountInString(docNumber) > MaxDocNumberLength {
		return nil, ErrDocNumberTooLong
	}

	body := &journalBody{
		DocNumber:   docNumber,
		TxnDate:     txn.TxnDate,
		PrivateNote: truncate(txn.PrivateNote, MaxPrivateNoteLength),
		Line:        make([]wireJournalLine, 0, len(txn.Lines)),
	}
	if currency := strings.TrimSpace(txn.CurrencyCode); currency != "" {
		body.CurrencyRef = &refValue{Value: strings.ToUpper(currency)}
		body.ExchangeRate = exchangeRateOf(txn.ExchangeRate)
	}

	debits, credits := decimal.Zero, decimal.Zero
	for idx := range txn.Lines {
		line := &txn.Lines[idx]
		wire, err := journalLineOf(line)
		if err != nil {
			return nil, err
		}
		if line.PostingType == PostingDebit {
			debits = debits.Add(line.Amount)
		} else {
			credits = credits.Add(line.Amount)
		}
		body.Line = append(body.Line, wire)
	}
	if !debits.Round(2).Equal(credits.Round(2)) {
		return nil, ErrJournalUnbalanced
	}
	return body, nil
}

func journalLineOf(line *JournalLine) (wireJournalLine, error) {
	if line.PostingType != PostingDebit && line.PostingType != PostingCredit {
		return wireJournalLine{}, ErrJournalPostingType
	}
	account := strings.TrimSpace(line.AccountID)
	if account == "" {
		return wireJournalLine{}, ErrJournalAccount
	}
	if line.Amount.IsNegative() {
		return wireJournalLine{}, ErrNegativeAmount
	}
	wire := wireJournalLine{
		DetailType:  lineDetailJournalEntry,
		Amount:      money(line.Amount),
		Description: truncate(line.Description, MaxLineDescription),
		JournalEntryLineDetail: journalLineDetail{
			PostingType: string(line.PostingType),
			AccountRef:  refValue{Value: account},
		},
	}
	entityID := strings.TrimSpace(line.EntityID)
	switch {
	case line.EntityType == "" && entityID == "":
	case line.EntityType != JournalEntityCustomer && line.EntityType != JournalEntityVendor,
		entityID == "":
		return wireJournalLine{}, ErrJournalEntityTarget
	default:
		wire.JournalEntryLineDetail.Entity = &journalEntity{
			Type:      string(line.EntityType),
			EntityRef: refValue{Value: entityID},
		}
	}
	return wire, nil
}

func (c *Client) CreateJournalEntry(
	ctx context.Context,
	requestID string,
	txn *JournalTxn,
) (*TxnResult, error) {
	body, err := journalBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return c.writeTxn(ctx, &txnWrite{requestID: requestID, kind: TxnJournalEntry, body: body})
}

func (c *Client) UpdateJournalEntry(
	ctx context.Context,
	requestID, id string,
	txn *JournalTxn,
) (*TxnResult, error) {
	body, err := journalBodyOf(requestID, txn)
	if err != nil {
		return nil, err
	}
	return c.updateTxn(
		ctx,
		&txnUpdate{requestID: requestID, kind: TxnJournalEntry, id: id, body: body},
	)
}
