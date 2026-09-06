package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fiscalyearservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type fiscalPeriodsByFiscalYearIDsLister interface {
	ListPeriodsByFiscalYearIDs(
		ctx context.Context,
		req repositories.ListByFiscalYearIDsRequest,
	) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error)
}

type FiscalPeriodsByFiscalYearIDLoaderFactoryParams struct {
	fx.In

	FiscalYearService *fiscalyearservice.Service
}

type FiscalPeriodsByFiscalYearIDLoaderFactory struct {
	periods fiscalPeriodsByFiscalYearIDsLister
}

func NewFiscalPeriodsByFiscalYearIDLoaderFactory(
	p FiscalPeriodsByFiscalYearIDLoaderFactoryParams,
) *FiscalPeriodsByFiscalYearIDLoaderFactory {
	return &FiscalPeriodsByFiscalYearIDLoaderFactory{
		periods: p.FiscalYearService,
	}
}

func (f *FiscalPeriodsByFiscalYearIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*fiscalperiod.FiscalPeriod] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *FiscalPeriodsByFiscalYearIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*fiscalperiod.FiscalPeriod] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
			return f.periods.ListPeriodsByFiscalYearIDs(
				ctx,
				repositories.ListByFiscalYearIDsRequest{
					TenantInfo:    tenantInfo,
					FiscalYearIDs: ids,
				},
			)
		},
	)
}
