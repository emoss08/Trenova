package nsbinding

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/shopspring/decimal"
)

const (
	currencyBookTTL   = 5 * time.Minute
	maxCachedRealms   = 1024
	currencyErrorCode = "currency"
)

type Session struct {
	Client *netsuite.Client
	Ref    netsuite.CompanyRef
	auth   accountingconnectorports.AccountingDocumentAuth
}

func (c *Binding) Open(auth accountingconnectorports.AccountingDocumentAuth) (*Session, error) {
	ref, err := netsuite.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return nil, err
	}
	client, err := netsuite.NewAccountClient(netsuite.AccountConfig{
		AccountID:   ref.AccountID,
		AccessToken: auth.AccessToken,
	}, c.Options()...)
	if err != nil {
		return nil, err
	}
	return &Session{Client: client, Ref: ref, auth: auth}, nil
}

func (s *Session) Subsidiary() *RecordRef {
	if s.Ref.SubsidiaryID == "" {
		return nil
	}
	return &RecordRef{ID: s.Ref.SubsidiaryID}
}

type currencyBook struct {
	base     string
	ids      map[string]string
	loadedAt time.Time
}

type CurrencyBooks struct {
	now     func() time.Time
	mu      sync.Mutex
	byRealm map[string]*currencyBook
}

func NewCurrencyBooks(now func() time.Time) *CurrencyBooks {
	return &CurrencyBooks{now: now, byRealm: make(map[string]*currencyBook)}
}

func (b *CurrencyBooks) cached(realm string) (*currencyBook, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.byRealm[realm]
	if !ok || b.now().Sub(entry.loadedAt) > currencyBookTTL {
		return nil, false
	}
	return entry, true
}

func (b *CurrencyBooks) store(realm string, book *currencyBook) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, known := b.byRealm[realm]; !known && len(b.byRealm) >= maxCachedRealms {
		clear(b.byRealm)
	}
	book.loadedAt = b.now()
	b.byRealm[realm] = book
}

func (b *CurrencyBooks) get(ctx context.Context, s *Session) (*currencyBook, error) {
	realm := s.Ref.String()
	if book, ok := b.cached(realm); ok {
		return book, nil
	}
	rows, err := QueryAll(ctx, s.Client, CurrenciesSQL, func(r *CurrencyRow) string {
		return r.ID.Value()
	})
	if err != nil {
		return nil, err
	}
	book := &currencyBook{ids: make(map[string]string, len(rows))}
	for idx := range rows {
		symbol := strings.ToUpper(rows[idx].Symbol.Value())
		if symbol != "" {
			book.ids[symbol] = rows[idx].ID.Value()
		}
	}
	home, err := HomeOf(ctx, s.Client, s.Ref)
	if err != nil {
		return nil, err
	}
	book.base = strings.ToUpper(home.Currency.Value())
	b.store(realm, book)
	return book, nil
}

func HomeOf(
	ctx context.Context,
	client *netsuite.Client,
	ref netsuite.CompanyRef,
) (*SubsidiaryRow, error) {
	var rows []SubsidiaryRow
	query := netsuite.Query{SQL: CompanySQL, Limit: 1}
	if ref.SubsidiaryID != "" {
		query = netsuite.Query{SQL: subsidiarySQL, Params: []any{ref.SubsidiaryID}, Limit: 1}
	}
	if _, err := client.Query(ctx, query, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, netsuiteerrors.ErrUnexpectedPayload
	}
	return &rows[0], nil
}

type CurrencyFields struct {
	currency *RecordRef
	rate     *Amount
}

func (c *Binding) CurrencyFor(
	ctx context.Context,
	s *Session,
	code string,
	rate decimal.Decimal,
) (CurrencyFields, error) {
	value := strings.ToUpper(strings.TrimSpace(code))
	base := strings.ToUpper(strings.TrimSpace(s.auth.CompanyCode))
	if value == "" || value == base {
		return CurrencyFields{}, nil
	}
	book, err := c.books.get(ctx, s)
	if err != nil {
		return CurrencyFields{}, err
	}
	if value == book.base {
		return CurrencyFields{}, nil
	}
	id := book.ids[value]
	if id == "" {
		return CurrencyFields{}, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorCurrency,
			Code:     currencyErrorCode,
			Message:  "The currency " + value + " is not active in " + ProviderName,
			Resolution: "Add " + value + " under Lists > Accounting > Currencies in " +
				ProviderName + ", then retry",
		}
	}
	out := CurrencyFields{currency: &RecordRef{ID: id}}
	if rate.IsPositive() {
		out.rate = AmountOf(rate)
	}
	return out, nil
}

func (c *Binding) Clock() time.Time {
	return c.now()
}
