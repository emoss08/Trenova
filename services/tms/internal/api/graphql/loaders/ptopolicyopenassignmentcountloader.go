package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/ptopolicyservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type ptoPolicyOpenAssignmentCounter interface {
	CountOpenAssignmentsByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		policyIDs []pulid.ID,
	) (map[pulid.ID]int, error)
}

type PTOPolicyOpenAssignmentCountLoaderFactoryParams struct {
	fx.In

	PTOPolicyService *ptopolicyservice.Service
}

type PTOPolicyOpenAssignmentCountLoaderFactory struct {
	counter ptoPolicyOpenAssignmentCounter
}

func NewPTOPolicyOpenAssignmentCountLoaderFactory(
	p PTOPolicyOpenAssignmentCountLoaderFactoryParams,
) *PTOPolicyOpenAssignmentCountLoaderFactory {
	return &PTOPolicyOpenAssignmentCountLoaderFactory{counter: p.PTOPolicyService}
}

func (f *PTOPolicyOpenAssignmentCountLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *PTOPolicyOpenAssignmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.counter.CountOpenAssignmentsByIDs(ctx, tenantInfo, ids)
	})
}
