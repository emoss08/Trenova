package exchangeratestamp

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	documentDate   = int64(1_775_800_000)
	accountingDate = int64(1_776_000_000)
)

func newStamper(
	t *testing.T,
	control *tenant.AccountingControl,
) (*Stamper, *mocks.MockExchangeRateService) {
	t.Helper()
	controls := mocks.NewMockAccountingControlRepository(t)
	if control == nil {
		controls.EXPECT().
			GetByOrgID(mock.Anything, mock.Anything).
			Return(nil, errortypes.NewNotFoundError("Accounting control not found"))
	} else {
		controls.EXPECT().GetByOrgID(mock.Anything, mock.Anything).Return(control, nil)
	}
	rates := mocks.NewMockExchangeRateService(t)
	return New(Params{
		Logger:   zap.NewNop(),
		Controls: controls,
		Rates:    rates,
		Policy:   accountingcontrolpolicyservice.New(accountingcontrolpolicyservice.Params{Logger: zap.NewNop()}),
	}), rates
}

func request(currency string) *Request {
	return &Request{
		TenantInfo:     pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		CurrencyCode:   currency,
		DocumentDate:   documentDate,
		AccountingDate: accountingDate,
	}
}

func TestStampRecordsTheCachedRateOnTheDayThePolicyNames(t *testing.T) {
	t.Parallel()

	stamper, rates := newStamper(t, &tenant.AccountingControl{
		FunctionalCurrencyCode: "USD",
		ExchangeRateDatePolicy: tenant.ExchangeRateDatePolicyAccountingDate,
	})
	rates.EXPECT().
		CachedRate(mock.Anything, mock.Anything, "CAD", "USD", time.Unix(1_775_952_000, 0).UTC()).
		Return(&services.RateConversionResult{Rate: decimal.RequireFromString("0.7312")}, nil)

	stamp, err := stamper.Stamp(t.Context(), request("cad"))
	require.NoError(t, err)
	require.NotNil(t, stamp)
	require.True(t, stamp.Rate.Equal(decimal.RequireFromString("0.7312")))
	require.Equal(t, int64(1_775_952_000), stamp.Date)

	var rate decimal.NullDecimal
	var date *int64
	stamp.Apply(&rate, &date)
	require.True(t, rate.Valid)
	require.NotNil(t, date)
	require.Equal(t, stamp.Date, *date)
}

func TestStampUsesTheDocumentDateUnderThatPolicy(t *testing.T) {
	t.Parallel()

	stamper, rates := newStamper(t, &tenant.AccountingControl{
		FunctionalCurrencyCode: "USD",
		ExchangeRateDatePolicy: tenant.ExchangeRateDatePolicyDocumentDate,
	})
	rates.EXPECT().
		CachedRate(mock.Anything, mock.Anything, "EUR", "USD", time.Unix(1_775_779_200, 0).UTC()).
		Return(&services.RateConversionResult{Rate: decimal.RequireFromString("1.08")}, nil)

	stamp, err := stamper.Stamp(t.Context(), request("EUR"))
	require.NoError(t, err)
	require.NotNil(t, stamp)
	require.Equal(t, int64(1_775_779_200), stamp.Date)
}

func TestStampIsNothingWhenTheDocumentIsInTheFunctionalCurrency(t *testing.T) {
	t.Parallel()

	stamper, _ := newStamper(t, &tenant.AccountingControl{FunctionalCurrencyCode: "USD"})

	stamp, err := stamper.Stamp(t.Context(), request("usd"))
	require.NoError(t, err)
	require.Nil(t, stamp)
}

func TestStampIsNothingWhenNoRateIsCached(t *testing.T) {
	t.Parallel()

	stamper, rates := newStamper(t, &tenant.AccountingControl{FunctionalCurrencyCode: "USD"})
	rates.EXPECT().
		CachedRate(mock.Anything, mock.Anything, "CAD", "USD", mock.Anything).
		Return(nil, services.ErrExchangeRateNotCached)

	stamp, err := stamper.Stamp(t.Context(), request("CAD"))
	require.NoError(t, err)
	require.Nil(t, stamp)
}

func TestStampIsNothingWithoutAccountingControlOrStamper(t *testing.T) {
	t.Parallel()

	stamper, _ := newStamper(t, nil)
	stamp, err := stamper.Stamp(t.Context(), request("CAD"))
	require.NoError(t, err)
	require.Nil(t, stamp)

	var none *Stamper
	stamp, err = none.Stamp(t.Context(), request("CAD"))
	require.NoError(t, err)
	require.Nil(t, stamp)

	var rate decimal.NullDecimal
	var date *int64
	stamp.Apply(&rate, &date)
	require.False(t, rate.Valid)
	require.Nil(t, date)
}

func TestStampReportsARateLookupFailure(t *testing.T) {
	t.Parallel()

	stamper, rates := newStamper(t, &tenant.AccountingControl{FunctionalCurrencyCode: "USD"})
	rates.EXPECT().
		CachedRate(mock.Anything, mock.Anything, "CAD", "USD", mock.Anything).
		Return(nil, errortypes.NewBusinessError("failed to retrieve cached exchange rate"))

	stamp, err := stamper.Stamp(t.Context(), request("CAD"))
	require.Error(t, err)
	require.Nil(t, stamp)
}
