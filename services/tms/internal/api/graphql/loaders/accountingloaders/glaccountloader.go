package accountingloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/glaccount"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/glaccountports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type GLAccountByIDLoaderFactoryParams struct {
	fx.In

	GLAccountRepo glaccountports.GLAccountRepository
}

type GLAccountByIDLoaderFactory struct {
	glAccountRepo glaccountports.GLAccountRepository
}

func NewGLAccountByIDLoaderFactory(
	p GLAccountByIDLoaderFactoryParams,
) *GLAccountByIDLoaderFactory {
	return &GLAccountByIDLoaderFactory{
		glAccountRepo: p.GLAccountRepo,
	}
}

func newGLAccountByIDLoader(f *GLAccountByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *glaccount.GLAccount] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *GLAccountByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*glaccount.GLAccount] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*glaccount.GLAccount, error) {
			return f.glAccountRepo.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
				TenantInfo:   tenantInfo,
				GLAccountIDs: ids,
			})
		},
		"GLAccount not found within your organization",
	)
}

var glAccountByIDKey = loadercore.NewKey(newGLAccountByIDLoader)

func GLAccountByID(ctx context.Context, key string) (*glaccount.GLAccount, error) {
	return loadercore.Load(ctx, glAccountByIDKey, key)
}

func (f *GLAccountByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
