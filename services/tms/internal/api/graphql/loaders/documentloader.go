package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type documentsByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req repositories.BulkDeleteDocumentRequest,
	) ([]*document.Document, error)
}

type DocumentByIDLoaderFactoryParams struct {
	fx.In

	Repo repositories.DocumentRepository
}

type DocumentByIDLoaderFactory struct {
	documents documentsByIDsGetter
}

func NewDocumentByIDLoaderFactory(p DocumentByIDLoaderFactoryParams) *DocumentByIDLoaderFactory {
	return &DocumentByIDLoaderFactory{
		documents: p.Repo,
	}
}

func (f *DocumentByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *document.Document] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *DocumentByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*document.Document] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*document.Document, error) {
		return f.documents.GetByIDs(ctx, repositories.BulkDeleteDocumentRequest{
			IDs:        ids,
			TenantInfo: tenantInfo,
		})
	}, "Document not found within your organization")
}
