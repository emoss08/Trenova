package ratingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type FormulaLibraryByIDLoaderFactoryParams struct {
	fx.In

	LibraryRepo formulalibraryports.FormulaLibraryRepository
}

type FormulaLibraryByIDLoaderFactory struct {
	libraries formulalibraryports.FormulaLibraryRepository
}

func NewFormulaLibraryByIDLoaderFactory(
	p FormulaLibraryByIDLoaderFactoryParams,
) *FormulaLibraryByIDLoaderFactory {
	return &FormulaLibraryByIDLoaderFactory{libraries: p.LibraryRepo}
}

func newFormulaLibraryByIDLoader(f *FormulaLibraryByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *formulalibrary.FormulaLibrary] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *FormulaLibraryByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*formulalibrary.FormulaLibrary] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*formulalibrary.FormulaLibrary, error) {
			return f.libraries.GetByIDs(ctx, &formulalibraryports.GetLibrariesByIDsRequest{
				TenantInfo: tenantInfo,
				LibraryIDs: ids,
			})
		},
		"Formula library not found within your organization",
	)
}

var formulaLibraryByIDKey = loadercore.NewKey(newFormulaLibraryByIDLoader)

func FormulaLibraryByID(ctx context.Context, key string) (*formulalibrary.FormulaLibrary, error) {
	return loadercore.Load(ctx, formulaLibraryByIDKey, key)
}

func (f *FormulaLibraryByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
