package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/glaccount"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type GLAccountByIDLoaderFactoryParams struct {
	fx.In

	GLAccountRepo repositories.GLAccountRepository
}

type GLAccountByIDLoaderFactory struct {
	glAccountRepo repositories.GLAccountRepository
}

func NewGLAccountByIDLoaderFactory(
	p GLAccountByIDLoaderFactoryParams,
) *GLAccountByIDLoaderFactory {
	return &GLAccountByIDLoaderFactory{
		glAccountRepo: p.GLAccountRepo,
	}
}

func (f *GLAccountByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *glaccount.GLAccount] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *GLAccountByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*glaccount.GLAccount] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*glaccount.GLAccount, error) {
		return f.glAccountRepo.GetByIDs(ctx, repositories.GetGLAccountsByIDsRequest{
			TenantInfo:   tenantInfo,
			GLAccountIDs: ids,
		})
	}, "GLAccount not found within your organization")
}
