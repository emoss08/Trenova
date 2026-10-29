package documentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type documentsByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req documentports.BulkDeleteDocumentRequest,
	) ([]*document.Document, error)
}

type DocumentByIDLoaderFactoryParams struct {
	fx.In

	Repo documentports.DocumentRepository
}

type DocumentByIDLoaderFactory struct {
	documents documentsByIDsGetter
}

func NewDocumentByIDLoaderFactory(p DocumentByIDLoaderFactoryParams) *DocumentByIDLoaderFactory {
	return &DocumentByIDLoaderFactory{
		documents: p.Repo,
	}
}

func newDocumentByIDLoader(f *DocumentByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *document.Document] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *DocumentByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*document.Document] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*document.Document, error) {
			return f.documents.GetByIDs(ctx, documentports.BulkDeleteDocumentRequest{
				IDs:        ids,
				TenantInfo: tenantInfo,
			})
		},
		"Document not found within your organization",
	)
}

var documentByIDKey = loadercore.NewKey(newDocumentByIDLoader)

func DocumentByID(ctx context.Context, key string) (*document.Document, error) {
	return loadercore.Load(ctx, documentByIDKey, key)
}

func (f *DocumentByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
