package interceptors

import (
	"context"

	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/interceptor"
)

// RefusalInterceptor makes an activity's final refusals non-retryable on the
// way out (temporaltype.ToActivityFailure), so a missing record or a failed
// validation fails once instead of spending every attempt of its retry policy.
type RefusalInterceptor struct {
	interceptor.InterceptorBase
}

func NewRefusalInterceptor() *RefusalInterceptor {
	return &RefusalInterceptor{}
}

func (i *RefusalInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &refusalActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
	}
}

type refusalActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
}

func (a *refusalActivityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	result, err := a.Next.ExecuteActivity(ctx, in)

	return result, temporaltype.ToActivityFailure(err)
}
