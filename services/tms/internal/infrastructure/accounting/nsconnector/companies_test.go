package nsconnector

import (
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/accounting/nsbinding"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/shared/netsuiteerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompaniesOffersEveryActiveSubsidiaryUnderOneWorld(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		if call.queries("FROM subsidiary s") {
			return fixtureReply(t, http.StatusOK, "subsidiaries.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	companies, err := conn.Companies(t.Context(), &accountingconnectorports.AccountingTokenGrant{
		AccessToken: "access-token",
	}, "1234567_sb1")
	require.NoError(t, err)
	assert.Equal(t, []accountingconnectorports.AccountingCompany{
		{ID: "1234567_SB1-s1", Name: "Acme Holdings · 1234567_SB1 (Sandbox)"},
		{ID: "1234567_SB1-s3", Name: "Acme US · 1234567_SB1 (Sandbox)"},
	}, companies)

	call, ok := fake.query("FROM subsidiary s")
	require.True(t, ok)
	assert.Contains(t, call.SQL, "s.isinactive = 'F'")
	assert.Contains(t, call.SQL, "s.iselimination = 'F'")
	assert.Equal(t, "Bearer access-token", call.Auth)
}

func TestCompaniesWithoutOneWorldIsTheAccount(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: func(call nsCall) reply {
		switch {
		case call.queries("FROM subsidiary s"):
			return fixtureReply(t, http.StatusBadRequest, "error_unsupported_table.json")
		case call.queries("FROM companyinformation"):
			return fixtureReply(t, http.StatusOK, "company_information.json")
		}
		return reply{}
	}}
	conn := testConnector(t, fake)
	companies, err := conn.Companies(t.Context(), &accountingconnectorports.AccountingTokenGrant{
		AccessToken: "access-token",
	}, "1234567")
	require.NoError(t, err)
	assert.Equal(t, []accountingconnectorports.AccountingCompany{
		{ID: "1234567", Name: "Acme Freight · 1234567"},
	}, companies)
}

func TestCompaniesFailures(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{})
	_, err := conn.Companies(t.Context(), &accountingconnectorports.AccountingTokenGrant{
		AccessToken: "access-token",
	}, testAccount)
	require.ErrorIs(t, err, nsbinding.ErrNoCompany)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))

	_, err = conn.Companies(t.Context(), nil, testAccount)
	require.ErrorIs(t, err, netsuiteerrors.ErrAccessTokenRequired)
	_, err = conn.Companies(t.Context(), &accountingconnectorports.AccountingTokenGrant{
		AccessToken: "access-token",
	}, "")
	require.ErrorIs(t, err, nsbinding.ErrNoAccount)

	unreachable := testConnector(t, &fakeNS{respond: func(nsCall) reply {
		return fixtureReply(t, http.StatusNotFound, "error_404_nonexistent.json")
	}})
	_, err = unreachable.Companies(t.Context(), &accountingconnectorports.AccountingTokenGrant{
		AccessToken: "access-token",
	}, testAccount)
	require.ErrorIs(t, err, nsbinding.ErrUnreachable)
}

func factsResponder(t *testing.T) func(call nsCall) reply {
	return func(call nsCall) reply {
		switch {
		case call.queries("FROM companyinformation"):
			return fixtureReply(t, http.StatusOK, "company_information.json")
		case call.queries("FROM subsidiary s"):
			return fixtureReply(t, http.StatusOK, "subsidiary.json")
		case call.queries("FROM accountingperiod p"):
			return fixtureReply(t, http.StatusOK, "accounting_periods.json")
		case call.queries("FROM currency c"):
			return fixtureReply(t, http.StatusOK, "currencies.json")
		}
		return reply{}
	}
}

func TestCompanyFactsReadsTheSubsidiaryBooks(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: factsResponder(t)}
	conn := testConnector(t, fake)
	facts, err := conn.CompanyFacts(t.Context(), testRef().String(), "access-token")
	require.NoError(t, err)

	closed := time.Date(2026, 5, 31, 23, 59, 59, 0, time.UTC).Unix()
	assert.Equal(t, &accountingconnection.CompanyFacts{
		CompanyName:          "Acme Canada",
		LegalName:            "Acme Freight Canada ULC",
		Country:              "CA",
		HomeCurrency:         "CAD",
		MultiCurrencyEnabled: true,
		BooksClosedThrough:   &closed,
		FiscalYearStartMonth: time.April,
		ShortCode:            "CAD",
	}, facts)

	call, ok := fake.query("FROM subsidiary s")
	require.True(t, ok)
	assert.Equal(t, []any{testSubsidiary}, call.Params)
}

func TestCompanyFactsWithoutASubsidiaryUseCompanyInformation(t *testing.T) {
	t.Parallel()

	fake := &fakeNS{respond: factsResponder(t)}
	conn := testConnector(t, fake)
	facts, err := conn.CompanyFacts(t.Context(), testAccount, "access-token")
	require.NoError(t, err)
	assert.Equal(t, "Acme Freight", facts.CompanyName)
	assert.Equal(t, "Acme Freight Holdings, Inc.", facts.LegalName)
	assert.Equal(t, "USD", facts.HomeCurrency)
	assert.Equal(t, "USD", facts.ShortCode)
	_, ok := fake.query("FROM subsidiary s")
	assert.False(t, ok)
}

func TestCompanyFactsSurfacesAnUnreachableAccount(t *testing.T) {
	t.Parallel()

	conn := testConnector(t, &fakeNS{respond: func(nsCall) reply {
		return fixtureReply(t, http.StatusNotFound, "error_404_nonexistent.json")
	}})
	_, err := conn.CompanyFacts(t.Context(), testAccount, "access-token")
	require.ErrorIs(t, err, nsbinding.ErrUnreachable)
	assert.Equal(t, accountingsync.ErrorCategoryConfiguration, conn.ClassifyError(err))

	failing := testConnector(t, &fakeNS{respond: func(nsCall) reply {
		return fixtureReply(t, http.StatusUnauthorized, "error_401_invalid_login.json")
	}})
	_, err = failing.CompanyFacts(t.Context(), testAccount, "access-token")
	require.Error(t, err)
	assert.Equal(t, accountingsync.ErrorCategoryUnauthorized, failing.ClassifyError(err))

	_, err = failing.CompanyFacts(t.Context(), "bad realm", "access-token")
	require.ErrorIs(t, err, netsuiteerrors.ErrInvalidCompanyRef)
}

func TestBooksClosedThroughIgnoresOpenAndYearPeriods(t *testing.T) {
	t.Parallel()

	assert.Nil(t, booksClosedThrough(nil))
	assert.Equal(t, time.Month(0), fiscalYearStart(nil))
	periods := sortedPeriods([]periodRow{
		{StartDate: "2026-01-01", EndDate: "2026-12-31", Year: true, Closed: true},
		{StartDate: "bad", EndDate: "2026-01-31", Posting: true, Closed: true},
		{StartDate: "2026-02-01", EndDate: "2026-02-28", Posting: true},
	})
	require.Len(t, periods, 2)
	assert.Nil(t, booksClosedThrough(periods))
	assert.Equal(t, time.January, fiscalYearStart(periods))
}
