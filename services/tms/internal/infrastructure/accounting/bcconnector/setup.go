package bcconnector

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
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

type postingSetups struct {
	now       func() time.Time
	mu        sync.Mutex
	byCompany map[string]*cachedSetup
}

func newPostingSetups(now func() time.Time) *postingSetups {
	return &postingSetups{now: now, byCompany: make(map[string]*cachedSetup)}
}

func (s *postingSetups) store(company string, setup *businesscentral.GeneralLedgerSetup) {
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

func (s *postingSetups) cached(company string) (businesscentral.GeneralLedgerSetup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.byCompany[company]
	if !ok || s.now().Sub(entry.loadedAt) > postingSetupTTL {
		return businesscentral.GeneralLedgerSetup{}, false
	}
	return entry.setup, true
}

func (s *postingSetups) get(
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
	s.store(company, setup)
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
		Message: "The date " + date + " is outside the posting dates " + providerName +
			" allows",
		Resolution: closedPeriodResolved,
	}
}

func documentCurrency(setup *businesscentral.GeneralLedgerSetup, code string) string {
	value := strings.ToUpper(strings.TrimSpace(code))
	if value == "" || value == setup.LocalCurrencyCode {
		return ""
	}
	return value
}

type postingContext struct {
	setup *businesscentral.GeneralLedgerSetup
}

func (c *Connector) postingFor(
	ctx context.Context,
	client *businesscentral.Client,
	date string,
) (*postingContext, error) {
	setup, err := c.setups.get(ctx, client)
	if err != nil {
		return nil, err
	}
	if err = checkPostingDate(setup, date); err != nil {
		return nil, err
	}
	return &postingContext{setup: setup}, nil
}

func (p *postingContext) currency(code string) string {
	return documentCurrency(p.setup, code)
}
