package exchangeratestamptest

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/services/exchangeratestamp"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func NewFixedForTest(t *testing.T, functionalCurrency string, rate decimal.Decimal) *exchangeratestamp.Stamper {
	t.Helper()

	controls := mocks.NewMockAccountingControlRepository(t)
	controls.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&tenant.AccountingControl{
			FunctionalCurrencyCode: functionalCurrency,
			ExchangeRateDatePolicy: tenant.ExchangeRateDatePolicyDocumentDate,
		}, nil).
		Maybe()
	rates := mocks.NewMockExchangeRateService(t)
	rates.EXPECT().
		CachedRate(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&services.RateConversionResult{Rate: rate}, nil).
		Maybe()

	return exchangeratestamp.New(exchangeratestamp.Params{
		Logger:   zap.NewNop(),
		Controls: controls,
		Rates:    rates,
		Policy:   accountingcontrolpolicyservice.New(accountingcontrolpolicyservice.Params{Logger: zap.NewNop()}),
	})
}
