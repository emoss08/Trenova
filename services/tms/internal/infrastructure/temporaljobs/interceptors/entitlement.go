package interceptors

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs/workfeature"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/interceptor"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const NotEntitledErrorType = "TenantNotEntitled"

const refusalLogEvery = time.Minute

type RefusalRecorder interface {
	RecordEntitlementRefusal(activityType string, feature platformcatalog.FeatureKey)
}

type EntitlementInterceptor struct {
	interceptor.InterceptorBase
	gate      platformcatalogports.FeatureGate
	refusals  RefusalRecorder
	logger    *zap.Logger
	sometimes *rate.Sometimes
}

func NewEntitlementInterceptor(
	gate platformcatalogports.FeatureGate,
	refusals RefusalRecorder,
	logger *zap.Logger,
) *EntitlementInterceptor {
	return &EntitlementInterceptor{
		gate:      gate,
		refusals:  refusals,
		logger:    logger.Named("temporal.entitlement"),
		sometimes: &rate.Sometimes{First: 1, Interval: refusalLogEvery},
	}
}

func (i *EntitlementInterceptor) InterceptActivity(
	_ context.Context,
	next interceptor.ActivityInboundInterceptor,
) interceptor.ActivityInboundInterceptor {
	return &entitlementActivityInbound{
		ActivityInboundInterceptorBase: interceptor.ActivityInboundInterceptorBase{Next: next},
		root:                           i,
	}
}

type entitlementActivityInbound struct {
	interceptor.ActivityInboundInterceptorBase
	root *EntitlementInterceptor
}

func (a *entitlementActivityInbound) ExecuteActivity(
	ctx context.Context,
	in *interceptor.ExecuteActivityInput,
) (any, error) {
	if err := a.root.check(ctx); err != nil {
		return nil, err
	}
	return a.Next.ExecuteActivity(ctx, in)
}

func (i *EntitlementInterceptor) check(ctx context.Context) error {
	feature, scheduled := workfeature.From(ctx)
	if !scheduled {
		return nil
	}
	tenant, ok := dbscope.TenantFrom(ctx)
	if !ok || tenant.OrganizationID.IsNil() {
		return nil
	}

	entitled, err := i.gate.Enabled(ctx, feature)
	if err != nil {
		return fmt.Errorf("check the plan of the tenant an activity runs for: %w", err)
	}
	if entitled {
		return nil
	}

	activityType := activityTypeOf(ctx)
	if i.refusals != nil {
		i.refusals.RecordEntitlementRefusal(activityType, feature)
	}
	i.sometimes.Do(func() {
		i.logger.Warn(
			"refused an activity for a tenant whose plan does not include the feature its schedule belongs to; the fan-out or sweep that scheduled it should have skipped the tenant",
			zap.String("activity", activityType),
			zap.String("feature", string(feature)),
			zap.String("organizationId", tenant.OrganizationID.String()),
		)
	})
	return temporal.NewNonRetryableApplicationError(
		fmt.Sprintf(
			"the tenant's plan does not include %s, or the tenant is suspended",
			feature,
		),
		NotEntitledErrorType,
		nil,
	)
}

func activityTypeOf(ctx context.Context) string {
	if !activity.IsActivity(ctx) {
		return "unknown"
	}
	return activity.GetInfo(ctx).ActivityType.Name
}
