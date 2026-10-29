package workforceloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
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

func newPTOPolicyOpenAssignmentCountLoader(f *PTOPolicyOpenAssignmentCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *PTOPolicyOpenAssignmentCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[int] {
	return loadercore.BatchCountFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
			return f.counter.CountOpenAssignmentsByIDs(ctx, tenantInfo, ids)
		},
	)
}

var ptoPolicyOpenAssignmentCountKey = loadercore.NewKey(newPTOPolicyOpenAssignmentCountLoader)

func PTOPolicyOpenAssignmentCount(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, ptoPolicyOpenAssignmentCountKey, key)
}

func (f *PTOPolicyOpenAssignmentCountLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
