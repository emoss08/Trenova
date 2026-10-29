package bcsession

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/shared/businesscentral"
)

const (
	postingSetupTTL      = 5 * time.Minute
	maxCachedCompanies   = 1024
	closedPeriodCode     = "posting-date"
	closedPeriodResolved = "Widen the allowed posting dates in Business Central's General " +
		"Ledger Setup, or send the record on the first open day"
)

type cachedSetup struct {
	setup    businesscentral.GeneralLedgerSetup
	loadedAt time.Time
}

type PostingSetups struct {
	now       func() time.Time
	mu        sync.Mutex
	byCompany map[string]*cachedSetup
}

func NewPostingSetups(now func() time.Time) *PostingSetups {
	return &PostingSetups{now: now, byCompany: make(map[string]*cachedSetup)}
}

func (s *PostingSetups) Store(company string, setup *businesscentral.GeneralLedgerSetup) {
	if company == "" || setup == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, known := s.byCompany[company]; !known && len(s.byCompany) >= maxCachedCompanies {
		clear(s.byCompany)
	}
	s.byCompany[company] = &cachedSetup{setup: *setup, loadedAt: s.now()}
}

func (s *PostingSetups) cached(company string) (businesscentral.GeneralLedgerSetup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.byCompany[company]
	if !ok || s.now().Sub(entry.loadedAt) > postingSetupTTL {
		return businesscentral.GeneralLedgerSetup{}, false
	}
	return entry.setup, true
}

func (s *PostingSetups) get(
	ctx context.Context,
	client *businesscentral.Client,
) (*businesscentral.GeneralLedgerSetup, error) {
	company := client.Ref().String()
	if setup, ok := s.cached(company); ok {
		return &setup, nil
	}
	setup, err := client.GeneralLedgerSetup(ctx)
	if err != nil {
		return nil, err
	}
	s.Store(company, setup)
	return setup, nil
}

func checkPostingDate(setup *businesscentral.GeneralLedgerSetup, date string) error {
	if strings.TrimSpace(date) == "" {
		return nil
	}
	allowed, err := setup.AllowsPosting(date)
	if err != nil {
		return err
	}
	if allowed {
		return nil
	}
	return &accountingsync.SyncError{
		Category: accountingsync.SyncErrorClosedPeriod,
		Code:     closedPeriodCode,
		Message: "The date " + date + " is outside the posting dates " + ProviderName +
			" allows",
		Resolution: closedPeriodResolved,
	}
}

func DocumentCurrency(setup *businesscentral.GeneralLedgerSetup, code string) string {
	value := strings.ToUpper(strings.TrimSpace(code))
	if value == "" || value == setup.LocalCurrencyCode {
		return ""
	}
	return value
}

type PostingContext struct {
	setup *businesscentral.GeneralLedgerSetup
}

func (c *Session) PostingFor(
	ctx context.Context,
	client *businesscentral.Client,
	date string,
) (*PostingContext, error) {
	setup, err := c.PostingSetup(ctx, client)
	if err != nil {
		return nil, err
	}
	if err = checkPostingDate(setup, date); err != nil {
		return nil, err
	}
	return &PostingContext{setup: setup}, nil
}

func (p *PostingContext) Currency(code string) string {
	return DocumentCurrency(p.setup, code)
}

func (c *Session) PostingSetup(
	ctx context.Context,
	client *businesscentral.Client,
) (*businesscentral.GeneralLedgerSetup, error) {
	return c.setups.get(ctx, client)
}
