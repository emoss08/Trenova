package nsconnector

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuite"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
)

const (
	companySeparator = " · "
	sandboxSuffix    = " (Sandbox)"
	endOfDayOffset   = 24*time.Hour - time.Second
)

func (c *Connector) Companies(
	ctx context.Context,
	grant *accountingconnectorports.AccountingTokenGrant,
	callbackRealmID string,
) ([]accountingconnectorports.AccountingCompany, error) {
	if grant == nil {
		return nil, netsuiteerrors.ErrAccessTokenRequired
	}
	account, err := nsbinding.AccountOf(callbackRealmID)
	if err != nil {
		return nil, err
	}
	s, err := c.Open(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     account,
		AccessToken: grant.AccessToken,
	})
	if err != nil {
		return nil, err
	}
	subsidiaries, err := nsbinding.QueryAll(
		ctx,
		s.Client,
		subsidiariesSQL,
		func(r *nsbinding.SubsidiaryRow) string {
			return r.ID.Value()
		},
	)
	switch {
	case err == nil:
	case netsuiteerrors.IsUnsupportedTable(err):
		return singleCompany(ctx, s)
	case isUnreachable(err):
		return nil, fmt.Errorf("%w: %w", nsbinding.ErrUnreachable, err)
	default:
		return nil, err
	}
	companies := make([]accountingconnectorports.AccountingCompany, 0, len(subsidiaries))
	for idx := range subsidiaries {
		row := &subsidiaries[idx]
		ref, refErr := netsuite.NewCompanyRef(account, row.ID.Value())
		if refErr != nil {
			continue
		}
		companies = append(companies, accountingconnectorports.AccountingCompany{
			ID:   ref.String(),
			Name: companyLabel(row.Name.Value(), ref),
		})
	}
	if len(companies) == 0 {
		return nil, nsbinding.ErrNoCompany
	}
	return companies, nil
}

func singleCompany(
	ctx context.Context,
	s *nsbinding.Session,
) ([]accountingconnectorports.AccountingCompany, error) {
	home, err := nsbinding.HomeOf(ctx, s.Client, s.Ref)
	if err != nil {
		if isUnreachable(err) {
			return nil, fmt.Errorf("%w: %w", nsbinding.ErrUnreachable, err)
		}
		return nil, err
	}
	return []accountingconnectorports.AccountingCompany{{
		ID:   s.Ref.AccountID,
		Name: companyLabel(home.CompanyName.Value(), s.Ref),
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
) (*accountingconnection.CompanyFacts, error) {
	s, err := c.Open(accountingconnectorports.AccountingDocumentAuth{
		RealmID:     realmID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}
	facts, err := readFacts(ctx, s)
	if err != nil {
		if isUnreachable(err) {
			return nil, fmt.Errorf("%w: %w", nsbinding.ErrUnreachable, err)
		}
		return nil, err
	}
	return facts, nil
}

func readFacts(
	ctx context.Context,
	s *nsbinding.Session,
) (*accountingconnection.CompanyFacts, error) {
	var company []nsbinding.SubsidiaryRow
	if _, err := s.Client.Query(
		ctx,
		netsuite.Query{SQL: nsbinding.CompanySQL, Limit: 1},
		&company,
	); err != nil {
		return nil, err
	}
	if len(company) == 0 {
		return nil, netsuiteerrors.ErrUnexpectedPayload
	}
	home := &company[0]
	facts := &accountingconnection.CompanyFacts{
		CompanyName:  home.CompanyName.Value(),
		LegalName:    home.LegalName.Value(),
		Country:      strings.ToUpper(home.Country.Value()),
		HomeCurrency: strings.ToUpper(home.Currency.Value()),
	}
	if s.Ref.SubsidiaryID != "" {
		subsidiary, err := nsbinding.HomeOf(ctx, s.Client, s.Ref)
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

	periods, err := nsbinding.QueryAll(ctx, s.Client, periodsSQL, func(r *periodRow) string {
		return r.ID.Value()
	})
	if err != nil {
		return nil, err
	}
	currencies, err := nsbinding.QueryAll(
		ctx,
		s.Client,
		nsbinding.CurrenciesSQL,
		func(r *nsbinding.CurrencyRow) string {
			return r.ID.Value()
		},
	)
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
