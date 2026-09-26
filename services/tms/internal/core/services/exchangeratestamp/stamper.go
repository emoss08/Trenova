package exchangeratestamp

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/accountingcontrolpolicyservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Stamp struct {
	Rate decimal.Decimal
	Date int64
}

func (s *Stamp) Apply(rate *decimal.NullDecimal, date **int64) {
	if s == nil {
		return
	}
	*rate = decimal.NewNullDecimal(s.Rate)
	day := s.Date
	*date = &day
}

type Request struct {
	TenantInfo     pagination.TenantInfo
	CurrencyCode   string
	DocumentDate   int64
	AccountingDate int64
}

type Params struct {
	fx.In

	Logger   *zap.Logger
	Controls repositories.AccountingControlRepository
	Rates    services.ExchangeRateService
	Policy   *accountingcontrolpolicyservice.Service
}

type Stamper struct {
	l        *zap.Logger
	controls repositories.AccountingControlRepository
	rates    services.ExchangeRateService
	policy   *accountingcontrolpolicyservice.Service
}

func New(p Params) *Stamper {
	return &Stamper{
		l:        p.Logger.Named("service.exchange-rate-stamp"),
		controls: p.Controls,
		rates:    p.Rates,
		policy:   p.Policy,
	}
}

func QuoteDate(
	policy *accountingcontrolpolicyservice.Service,
	control *tenant.AccountingControl,
	documentDate, accountingDate int64,
) int64 {
	if policy == nil || control == nil {
		return timeutils.DayStartUTC(documentDate)
	}
	date, err := policy.ResolveFXQuoteDate(accountingcontrolpolicyservice.ResolveFXQuoteDateRequest{
		Policy:         control.ExchangeRateDatePolicy,
		DocumentDate:   documentDate,
		AccountingDate: accountingDate,
	})
	if err != nil {
		date = documentDate
	}
	return timeutils.DayStartUTC(date)
}

func Needed(control *tenant.AccountingControl, currencyCode string) (string, string, bool) {
	if control == nil {
		return "", "", false
	}
	code := strings.ToUpper(strings.TrimSpace(currencyCode))
	functional := strings.ToUpper(strings.TrimSpace(control.FunctionalCurrencyCode))
	if code == "" || functional == "" || code == functional {
		return code, functional, false
	}
	return code, functional, true
}

func (s *Stamper) Stamp(ctx context.Context, req *Request) (*Stamp, error) {
	if s == nil || req == nil {
		return nil, nil
	}
	control, err := s.controls.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	code, functional, needed := Needed(control, req.CurrencyCode)
	if !needed {
		return nil, nil
	}

	date := QuoteDate(s.policy, control, req.DocumentDate, req.AccountingDate)
	result, err := s.rates.CachedRate(ctx, req.TenantInfo, code, functional, time.Unix(date, 0).UTC())
	if err != nil && !errors.Is(err, services.ErrExchangeRateNotCached) {
		return nil, err
	}
	if result == nil {
		s.l.Debug("no cached exchange rate at posting; the sync resolves it",
			zap.String("from", code),
			zap.String("to", functional),
			zap.String("date", timeutils.FormatCalendarDate(date, time.UTC)),
		)
		return nil, nil
	}

	return &Stamp{Rate: result.Rate, Date: date}, nil
}

func (s *Stamper) StampInto(
	ctx context.Context,
	req *Request,
	rate *decimal.NullDecimal,
	date **int64,
) error {
	stamp, err := s.Stamp(ctx, req)
	if err != nil {
		return err
	}
	if stamp == nil {
		return nil
	}
	stamp.Apply(rate, date)
	return nil
}
