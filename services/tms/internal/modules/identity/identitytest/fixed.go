package identitytest

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/app/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/exchangeratestamp"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/reference/ports/exchangerateports"
	"github.com/emoss08/trenova/internal/testutil/mocks/exchangerateportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

func NewFixedForTest(
	t *testing.T,
	functionalCurrency string,
	rate decimal.Decimal,
) *exchangeratestamp.Stamper {
	t.Helper()

	controls := tenantportsmock.NewMockAccountingControlRepository(t)
	controls.EXPECT().
		GetByOrgID(mock.Anything, mock.Anything).
		Return(&accountingcontrol.AccountingControl{
			FunctionalCurrencyCode: functionalCurrency,
			ExchangeRateDatePolicy: accountingcontrol.ExchangeRateDatePolicyDocumentDate,
		}, nil).
		Maybe()
	rates := exchangerateportsmock.NewMockExchangeRateService(t)
	rates.EXPECT().
		CachedRate(mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(&exchangerateports.RateConversionResult{Rate: rate}, nil).
		Maybe()

	return exchangeratestamp.New(exchangeratestamp.Params{
		Logger:   zap.NewNop(),
		Controls: controls,
		Rates:    rates,
		Policy: accountingcontrolpolicyservice.New(
			accountingcontrolpolicyservice.Params{Logger: zap.NewNop()},
		),
	})
}
