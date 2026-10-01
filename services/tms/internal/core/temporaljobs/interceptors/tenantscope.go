package interceptors

import (
	"context"

	"github.com/emoss08/trenova/pkg/dbscope"
	"go.temporal.io/sdk/interceptor"
)

type TenantScopeInterceptor struct {
	interceptor.InterceptorBase
}

func NewTenantScopeInterceptor() *TenantScopeInterceptor {
	return &TenantScopeInterceptor{}
}

func (i *TenantScopeInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &tenantScopeActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
	}
}

type tenantScopeActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
}

func (a *tenantScopeActivityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	if tenant, ok := dbscope.TenantOf(in.Args...); ok {
		ctx = dbscope.WithTenant(ctx, tenant)
	}

	return a.Next.ExecuteActivity(ctx, in)
}
