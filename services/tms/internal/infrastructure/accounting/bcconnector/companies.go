package bcconnector

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/businesscentral"
)

const (
	companySeparator      = " · "
	sandboxSuffix         = " (Sandbox)"
	productionEnvironment = "Production"
	endOfDayOffset        = 24*time.Hour - time.Second
	oneDay                = 24 * time.Hour
)

var (
	errNoCompany = errors.New(
		"business central offered no company this sign-in can open",
	)
	errNoAccess = errors.New(
		"the signed-in user must be able to open Business Central to connect it",
	)
)

func (c *Connector) Companies(
	ctx context.Context,
	grant *services.AccountingTokenGrant,
	_ string,
) ([]services.AccountingCompany, error) {
	if grant == nil {
		return nil, businesscentral.ErrAccessTokenRequired
	}
	tenant, err := businesscentral.TenantIDFromAccessToken(grant.AccessToken)
	if err != nil {
		return nil, err
	}
	discovery, err := businesscentral.NewDiscoveryClient(grant.AccessToken, c.options()...)
	if err != nil {
		return nil, err
	}
	environments, probing, err := listEnvironments(ctx, discovery)
	if err != nil {
		return nil, err
	}

	companies := make([]services.AccountingCompany, 0, len(environments))
	for idx := range environments {
		env := &environments[idx]
		if !businesscentral.ValidEnvironmentName(env.Name) {
			continue
		}
		listed, listErr := discovery.Companies(ctx, tenant, env.Name)
		switch {
		case listErr == nil:
			companies = appendCompanies(companies, tenant, env, listed)
		case probing:
			return nil, fmt.Errorf("%w: %w", errNoAccess, listErr)
		case refusesAccess(listErr), businesscentral.IsNotFound(listErr):
			continue
		default:
			return nil, listErr
		}
	}
	if len(companies) == 0 {
		return nil, errNoCompany
	}
	return companies, nil
}

func listEnvironments(
	ctx context.Context,
	discovery *businesscentral.DiscoveryClient,
) (environments []businesscentral.Environment, probing bool, err error) {
	environments, err = discovery.Environments(ctx)
	if err == nil {
		return environments, false, nil
	}
	if !refusesAccess(err) {
		return nil, false, err
	}
	return []businesscentral.Environment{{
		Type: businesscentral.EnvironmentTypeProduction,
		Name: productionEnvironment,
	}}, true, nil
}

func refusesAccess(err error) bool {
	return businesscentral.IsAuth(err) || businesscentral.IsForbidden(err)
}

func appendCompanies(
	companies []services.AccountingCompany,
	tenant string,
	env *businesscentral.Environment,
	listed []businesscentral.Company,
) []services.AccountingCompany {
	for idx := range listed {
		company := &listed[idx]
		ref, err := businesscentral.NewCompanyRef(tenant, env.Name, company.ID)
		if err != nil {
			continue
		}
		id := ref.String()
		if id == "" {
			continue
		}
		companies = append(companies, services.AccountingCompany{
			ID:   id,
			Name: companyLabel(company, env),
		})
	}
	return companies
}

func companyLabel(company *businesscentral.Company, env *businesscentral.Environment) string {
	name := strings.TrimSpace(company.DisplayName)
	if name == "" {
		name = strings.TrimSpace(company.Name)
	}
	label := name + companySeparator + env.Name
	if strings.EqualFold(env.Type, businesscentral.EnvironmentTypeSandbox) {
		label += sandboxSuffix
	}
	return label
}

type companyBooks struct {
	info       *businesscentral.CompanyInformation
	setup      *businesscentral.GeneralLedgerSetup
	periods    []businesscentral.AccountingPeriod
	currencies []businesscentral.Currency
	name       string
}

func (c *Connector) CompanyFacts(
	ctx context.Context,
	realmID, accessToken string,
) (*accountingsync.CompanyFacts, error) {
	client, err := c.client(services.AccountingDocumentAuth{
		RealmID:     realmID,
		AccessToken: accessToken,
	})
	if err != nil {
		return nil, err
	}
	books, err := c.readBooks(ctx, client, accessToken)
	if err != nil {
		return nil, err
	}
	return books.facts(), nil
}

func (c *Connector) readBooks(
	ctx context.Context,
	client *businesscentral.Client,
	accessToken string,
) (*companyBooks, error) {
	var (
		books companyBooks
		err   error
	)
	if books.info, err = client.CompanyInformation(ctx); err != nil {
		return nil, err
	}
	if books.setup, err = client.GeneralLedgerSetup(ctx); err != nil {
		return nil, err
	}
	if books.periods, err = client.AccountingPeriods(ctx); err != nil {
		return nil, err
	}
	if books.currencies, err = client.Currencies(ctx); err != nil {
		return nil, err
	}
	if books.name, err = c.companyName(ctx, client.Ref(), accessToken); err != nil {
		return nil, err
	}
	c.setups.store(client.Ref().String(), books.setup)
	return &books, nil
}

func (c *Connector) companyName(
	ctx context.Context,
	ref businesscentral.CompanyRef,
	accessToken string,
) (string, error) {
	discovery, err := businesscentral.NewDiscoveryClient(accessToken, c.options()...)
	if err != nil {
		return "", err
	}
	companies, err := discovery.Companies(ctx, ref.TenantID, ref.Environment)
	if err != nil {
		return "", err
	}
	for idx := range companies {
		if sameID(companies[idx].ID, ref.CompanyID) {
			return strings.TrimSpace(companies[idx].Name), nil
		}
	}
	return "", nil
}

func (b *companyBooks) facts() *accountingsync.CompanyFacts {
	home := b.setup.LocalCurrencyCode
	if home == "" {
		home = b.info.CurrencyCode
	}
	name := b.name
	if name == "" {
		name = strings.TrimSpace(b.info.DisplayName)
	}
	periods := sortedPeriods(b.periods)
	return &accountingsync.CompanyFacts{
		CompanyName:          name,
		LegalName:            b.info.DisplayName,
		Country:              b.info.CountryRegionCode,
		HomeCurrency:         home,
		MultiCurrencyEnabled: hasForeignCurrency(b.currencies, home),
		BooksClosedThrough:   booksClosedThrough(b.setup, periods),
		FiscalYearStartMonth: fiscalYearStart(periods),
	}
}

func sortedPeriods(periods []businesscentral.AccountingPeriod) []businesscentral.AccountingPeriod {
	out := make([]businesscentral.AccountingPeriod, 0, len(periods))
	for idx := range periods {
		if _, err := businesscentral.ParseDate(periods[idx].StartingDate); err == nil {
			out = append(out, periods[idx])
		}
	}
	slices.SortFunc(out, func(a, b businesscentral.AccountingPeriod) int {
		return strings.Compare(a.StartingDate, b.StartingDate)
	})
	return out
}

func hasForeignCurrency(currencies []businesscentral.Currency, home string) bool {
	for idx := range currencies {
		code := strings.ToUpper(strings.TrimSpace(currencies[idx].Code))
		if code != "" && code != home {
			return true
		}
	}
	return false
}

func booksClosedThrough(
	setup *businesscentral.GeneralLedgerSetup,
	periods []businesscentral.AccountingPeriod,
) *int64 {
	var closed time.Time
	if from, err := businesscentral.ParseDate(setup.AllowPostingFrom); err == nil {
		closed = from.Add(-oneDay)
	}
	for idx := 0; idx+1 < len(periods); idx++ {
		if !periods[idx].Closed {
			continue
		}
		next, err := businesscentral.ParseDate(periods[idx+1].StartingDate)
		if err != nil {
			continue
		}
		if end := next.Add(-oneDay); end.After(closed) {
			closed = end
		}
	}
	if closed.IsZero() {
		return nil
	}
	value := closed.Add(endOfDayOffset).Unix()
	return &value
}

func fiscalYearStart(periods []businesscentral.AccountingPeriod) time.Month {
	for idx := len(periods) - 1; idx >= 0; idx-- {
		if !periods[idx].NewFiscalYear {
			continue
		}
		start, err := businesscentral.ParseDate(periods[idx].StartingDate)
		if err != nil {
			return 0
		}
		return start.Month()
	}
	return 0
}
