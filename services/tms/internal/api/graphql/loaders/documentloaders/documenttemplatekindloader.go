package documentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/documents/app/documenttemplateservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type documentTemplateKindsGetter interface {
	KindsByTemplateIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		templateIDs []pulid.ID,
	) (map[pulid.ID]documenttemplate.Kind, error)
}

type DocumentTemplateKindByTemplateIDLoaderFactoryParams struct {
	fx.In

	DocumentTemplateService *documenttemplateservice.Service
}

type DocumentTemplateKindByTemplateIDLoaderFactory struct {
	kinds documentTemplateKindsGetter
}

func NewDocumentTemplateKindByTemplateIDLoaderFactory(
	p DocumentTemplateKindByTemplateIDLoaderFactoryParams,
) *DocumentTemplateKindByTemplateIDLoaderFactory {
	return &DocumentTemplateKindByTemplateIDLoaderFactory{
		kinds: p.DocumentTemplateService,
	}
}

func newDocumentTemplateKindByTemplateIDLoader(f *DocumentTemplateKindByTemplateIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, documenttemplate.Kind] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *DocumentTemplateKindByTemplateIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[documenttemplate.Kind] {
	return func(ctx context.Context, keys []string) ([]documenttemplate.Kind, []error) {
		values := make([]documenttemplate.Kind, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := loadercore.ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		kinds, err := f.kinds.KindsByTemplateIDs(ctx, tenantInfo, ids)
		if err != nil {
			loadercore.FillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = kinds[id]
			}
		}

		return values, errs
	}
}

var documentTemplateKindByTemplateIDKey = loadercore.NewKey(
	newDocumentTemplateKindByTemplateIDLoader,
)

func DocumentTemplateKindByTemplateID(
	ctx context.Context,
	key string,
) (documenttemplate.Kind, error) {
	return loadercore.Load(ctx, documentTemplateKindByTemplateIDKey, key)
}

func (f *DocumentTemplateKindByTemplateIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
