package loaders

import (
	"context"

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
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *GLAccountByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*glaccount.GLAccount] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*glaccount.GLAccount, error) {
		return f.glAccountRepo.GetByIDs(ctx, glaccountports.GetGLAccountsByIDsRequest{
			TenantInfo:   tenantInfo,
			GLAccountIDs: ids,
		})
	}, "GLAccount not found within your organization")
}
