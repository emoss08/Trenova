package nsconnector

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/netsuite"
)

const (
	companySeparator = " · "
	sandboxSuffix    = " (Sandbox)"
	endOfDayOffset   = 24*time.Hour - time.Second
)

var errNoCompany = errors.New("netsuite offered no active subsidiary this sign-in can open")

func (c *Connector) Companies(
	ctx context.Context,
	grant *accountingsyncports.AccountingTokenGrant,
	callbackRealmID string,
) ([]accountingsyncports.AccountingCompany, error) {
	if grant == nil {
		return nil, netsuite.ErrAccessTokenRequired
	}
	account, err := accountOf(callbackRealmID)
	if err != nil {
		return nil, err
	}
	s, err := c.open(accountingsyncports.AccountingDocumentAuth{
		RealmID:     account,
		AccessToken: grant.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	subsidiaries, err := queryAll(ctx, s.client, subsidiariesSQL, func(r *subsidiaryRow) string {
		return r.ID.Value()
	})
	switch {
	case err == nil:
	case netsuite.IsUnsupportedTable(err):
		return singleCompany(ctx, s)
	case isUnreachable(err):
		return nil, fmt.Errorf("%w: %w", errUnreachable, err)
	default:
		return nil, err
	}
	companies := make([]accountingsyncports.AccountingCompany, 0, len(subsidiaries))
	for idx := range subsidiaries {
		row := &subsidiaries[idx]
		ref, refErr := netsuite.NewCompanyRef(account, row.ID.Value())
		if refErr != nil {
			continue
		}
		companies = append(companies, accountingsyncports.AccountingCompany{
			ID:   ref.String(),
			Name: companyLabel(row.Name.Value(), ref),
		})
	}
	if len(companies) == 0 {
		return nil, errNoCompany
	}
	return companies, nil
}

func singleCompany(
	ctx context.Context,
	s *session,
) ([]accountingsyncports.AccountingCompany, error) {
	home, err := homeOf(ctx, s.client, s.ref)
	if err != nil {
		if isUnreachable(err) {
			return nil, fmt.Errorf("%w: %w", errUnreachable, err)
		}
		return nil, err
	}
	return []accountingsyncports.AccountingCompany{{
		ID:   s.ref.AccountID,
		Name: companyLabel(home.CompanyName.Value(), s.ref),
	}}, nil
}

func companyLabel(name string, ref netsuite.CompanyRef) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = ref.AccountID
	} else {
		label += companySeparator + ref.AccountID
	}
	if ref.Sandbox() {
		label += sandboxSuffix
	}
	return label
}

func (c *Connector) CompanyFacts(
	ctx context.Context,
	realmID, accessToken string,
) (*accountingsync.CompanyFacts, error) {
	s, err := c.open(accountingsyncports.AccountingDocumentAuth{
		RealmID:     realmID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}
	facts, err := readFacts(ctx, s)
	if err != nil {
		if isUnreachable(err) {
			return nil, fmt.Errorf("%w: %w", errUnreachable, err)
		}
		return nil, err
	}
	return facts, nil
}

func readFacts(ctx context.Context, s *session) (*accountingsync.CompanyFacts, error) {
	var company []subsidiaryRow
	if _, err := s.client.Query(
		ctx,
		netsuite.Query{SQL: companySQL, Limit: 1},
		&company,
	); err != nil {
		return nil, err
	}
	if len(company) == 0 {
		return nil, netsuite.ErrUnexpectedPayload
	}
	home := &company[0]
	facts := &accountingsync.CompanyFacts{
		CompanyName:  home.CompanyName.Value(),
		LegalName:    home.LegalName.Value(),
		Country:      strings.ToUpper(home.Country.Value()),
		HomeCurrency: strings.ToUpper(home.Currency.Value()),
	}
	if s.ref.SubsidiaryID != "" {
		subsidiary, err := homeOf(ctx, s.client, s.ref)
		if err != nil {
			return nil, err
		}
		facts.CompanyName = subsidiary.Name.Value()
		if legal := subsidiary.LegalName.Value(); legal != "" {
			facts.LegalName = legal
		}
		if country := subsidiary.Country.Value(); country != "" {
			facts.Country = strings.ToUpper(country)
		}
		facts.HomeCurrency = strings.ToUpper(subsidiary.Currency.Value())
	}
	if facts.LegalName == "" {
		facts.LegalName = facts.CompanyName
	}
	facts.ShortCode = facts.HomeCurrency

	periods, err := queryAll(ctx, s.client, periodsSQL, func(r *periodRow) string {
		return r.ID.Value()
	})
	if err != nil {
		return nil, err
	}
	currencies, err := queryAll(ctx, s.client, currenciesSQL, func(r *currencyRow) string {
		return r.ID.Value()
	})
	if err != nil {
		return nil, err
	}
	facts.MultiCurrencyEnabled = len(currencies) > 1
	sorted := sortedPeriods(periods)
	facts.BooksClosedThrough = booksClosedThrough(sorted)
	facts.FiscalYearStartMonth = fiscalYearStart(sorted)
	return facts, nil
}

type datedPeriod struct {
	row   *periodRow
	start time.Time
	end   time.Time
}

func sortedPeriods(rows []periodRow) []datedPeriod {
	out := make([]datedPeriod, 0, len(rows))
	for idx := range rows {
		start, startErr := netsuite.ParseQueryDate(rows[idx].StartDate.Value())
		end, endErr := netsuite.ParseQueryDate(rows[idx].EndDate.Value())
		if startErr != nil || endErr != nil {
			continue
		}
		out = append(out, datedPeriod{row: &rows[idx], start: start, end: end})
	}
	slices.SortStableFunc(out, func(a, b datedPeriod) int {
		return a.start.Compare(b.start)
	})
	return out
}

func booksClosedThrough(periods []datedPeriod) *int64 {
	var closed time.Time
	for idx := range periods {
		period := &periods[idx]
		if !period.row.Posting.Value() {
			continue
		}
		if !period.row.Closed.Value() && !period.row.AllLocked.Value() {
			continue
		}
		if period.end.After(closed) {
			closed = period.end
		}
	}
	if closed.IsZero() {
		return nil
	}
	value := closed.Add(endOfDayOffset).Unix()
	return &value
}

func fiscalYearStart(periods []datedPeriod) time.Month {
	for idx := len(periods) - 1; idx >= 0; idx-- {
		if periods[idx].row.Year.Value() {
			return periods[idx].start.Month()
		}
	}
	return 0
}
