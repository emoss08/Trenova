package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type PayProfileActiveAssignmentCountLoaderFactoryParams struct {
	fx.In

	PayProfileRepo repositories.PayProfileRepository
}

type PayProfileActiveAssignmentCountLoaderFactory struct {
	payProfileRepo repositories.PayProfileRepository
}

func NewPayProfileActiveAssignmentCountLoaderFactory(
	p PayProfileActiveAssignmentCountLoaderFactoryParams,
) *PayProfileActiveAssignmentCountLoaderFactory {
	return &PayProfileActiveAssignmentCountLoaderFactory{payProfileRepo: p.PayProfileRepo}
}

func (f *PayProfileActiveAssignmentCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *PayProfileActiveAssignmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.payProfileRepo.CountActiveAssignmentsByIDs(
			ctx,
			repositories.CountActivePayAssignmentsRequest{
				TenantInfo: tenantInfo,
				ProfileIDs: ids,
			},
		)
	})
}
