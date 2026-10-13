package nsconnector

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/shopspring/decimal"
)

const (
	currencyBookTTL   = 5 * time.Minute
	maxCachedRealms   = 1024
	currencyErrorCode = "currency"
)

type session struct {
	client *netsuite.Client
	ref    netsuite.CompanyRef
	auth   accountingsyncports.AccountingDocumentAuth
}

func (c *Connector) open(auth accountingsyncports.AccountingDocumentAuth) (*session, error) {
	ref, err := netsuite.ParseCompanyRef(auth.RealmID)
	if err != nil {
		return nil, err
	}
	client, err := netsuite.NewAccountClient(netsuite.AccountConfig{
		AccountID:   ref.AccountID,
		AccessToken: auth.AccessToken,
	}, c.options()...)
	if err != nil {
		return nil, err
	}
	return &session{client: client, ref: ref, auth: auth}, nil
}

func (s *session) subsidiary() *recordRef {
	if s.ref.SubsidiaryID == "" {
		return nil
	}
	return &recordRef{ID: s.ref.SubsidiaryID}
}

type currencyBook struct {
	base     string
	ids      map[string]string
	loadedAt time.Time
}

type currencyBooks struct {
	now     func() time.Time
	mu      sync.Mutex
	byRealm map[string]*currencyBook
}

func newCurrencyBooks(now func() time.Time) *currencyBooks {
	return &currencyBooks{now: now, byRealm: make(map[string]*currencyBook)}
}

func (b *currencyBooks) cached(realm string) (*currencyBook, bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	entry, ok := b.byRealm[realm]
	if !ok || b.now().Sub(entry.loadedAt) > currencyBookTTL {
		return nil, false
	}
	return entry, true
}

func (b *currencyBooks) store(realm string, book *currencyBook) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, known := b.byRealm[realm]; !known && len(b.byRealm) >= maxCachedRealms {
		clear(b.byRealm)
	}
	book.loadedAt = b.now()
	b.byRealm[realm] = book
}

func (b *currencyBooks) get(ctx context.Context, s *session) (*currencyBook, error) {
	realm := s.ref.String()
	if book, ok := b.cached(realm); ok {
		return book, nil
	}
	rows, err := queryAll(ctx, s.client, currenciesSQL, func(r *currencyRow) string {
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
	home, err := homeOf(ctx, s.client, s.ref)
	if err != nil {
		return nil, err
	}
	book.base = strings.ToUpper(home.Currency.Value())
	b.store(realm, book)
	return book, nil
}

func homeOf(
	ctx context.Context,
	client *netsuite.Client,
	ref netsuite.CompanyRef,
) (*subsidiaryRow, error) {
	var rows []subsidiaryRow
	query := netsuite.Query{SQL: companySQL, Limit: 1}
	if ref.SubsidiaryID != "" {
		query = netsuite.Query{SQL: subsidiarySQL, Params: []any{ref.SubsidiaryID}, Limit: 1}
	}
	if _, err := client.Query(ctx, query, &rows); err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, netsuite.ErrUnexpectedPayload
	}
	return &rows[0], nil
}

type currencyFields struct {
	currency *recordRef
	rate     *amount
}

func (c *Connector) currencyFor(
	ctx context.Context,
	s *session,
	code string,
	rate decimal.Decimal,
) (currencyFields, error) {
	value := strings.ToUpper(strings.TrimSpace(code))
	base := strings.ToUpper(strings.TrimSpace(s.auth.CompanyCode))
	if value == "" || value == base {
		return currencyFields{}, nil
	}
	book, err := c.books.get(ctx, s)
	if err != nil {
		return currencyFields{}, err
	}
	if value == book.base {
		return currencyFields{}, nil
	}
	id := book.ids[value]
	if id == "" {
		return currencyFields{}, &accountingsync.SyncError{
			Category: accountingsync.SyncErrorCurrency,
			Code:     currencyErrorCode,
			Message:  "The currency " + value + " is not active in " + providerName,
			Resolution: "Add " + value + " under Lists > Accounting > Currencies in " +
				providerName + ", then retry",
		}
	}
	out := currencyFields{currency: &recordRef{ID: id}}
	if rate.IsPositive() {
		out.rate = amountOf(rate)
	}
	return out, nil
}
