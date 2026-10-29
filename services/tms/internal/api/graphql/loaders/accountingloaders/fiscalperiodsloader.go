package accountingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalyearservice"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/fiscalperiodports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type fiscalPeriodsByFiscalYearIDsLister interface {
	ListPeriodsByFiscalYearIDs(
		ctx context.Context,
		req fiscalperiodports.ListByFiscalYearIDsRequest,
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

func newFiscalPeriodsByFiscalYearIDLoader(f *FiscalPeriodsByFiscalYearIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*fiscalperiod.FiscalPeriod] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *FiscalPeriodsByFiscalYearIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*fiscalperiod.FiscalPeriod] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*fiscalperiod.FiscalPeriod, error) {
			return f.periods.ListPeriodsByFiscalYearIDs(
				ctx,
				fiscalperiodports.ListByFiscalYearIDsRequest{
					TenantInfo:    tenantInfo,
					FiscalYearIDs: ids,
				},
			)
		},
	)
}

var fiscalPeriodsByFiscalYearIDKey = loadercore.NewKey(newFiscalPeriodsByFiscalYearIDLoader)

func FiscalPeriodsByFiscalYearID(
	ctx context.Context,
	key string,
) ([]*fiscalperiod.FiscalPeriod, error) {
	return loadercore.Load(ctx, fiscalPeriodsByFiscalYearIDKey, key)
}

func (f *FiscalPeriodsByFiscalYearIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
