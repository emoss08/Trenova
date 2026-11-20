package temporaljobs

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/entitlementports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

const KeepEntitledActivityName = "KeepEntitledOrganizationsActivity"

var ErrItemWithoutTenant = errors.New(
	"a fan-out item must carry the organization and business unit of the tenant it is for",
)

var keepEntitledOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 30 * time.Second,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    time.Second,
		BackoffCoefficient: 2,
		MaximumInterval:    30 * time.Second,
		MaximumAttempts:    5,
	},
}

type KeepEntitledRequest struct {
	Need          entitlement.Need `json:"need"`
	Organizations []pulid.ID       `json:"organizations"`
}

func KeepEntitled[T any](ctx workflow.Context, need entitlement.Need, items []T) ([]T, error) {
	if err := need.Validate(); err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return items, nil
	}

	owners := make([]pulid.ID, len(items))
	seen := make(map[pulid.ID]struct{}, len(items))
	organizations := make([]pulid.ID, 0, len(items))
	for i := range items {
		tenant, ok := dbscope.TenantOf(items[i])
		if !ok || tenant.OrganizationID.IsNil() {
			return nil, fmt.Errorf("%w: item %d of %T", ErrItemWithoutTenant, i, items[i])
		}
		owners[i] = tenant.OrganizationID
		if _, dup := seen[tenant.OrganizationID]; !dup {
			seen[tenant.OrganizationID] = struct{}{}
			organizations = append(organizations, tenant.OrganizationID)
		}
	}

	actx := workflow.WithActivityOptions(ctx, keepEntitledOptions)
	var entitled []pulid.ID
	if err := workflow.ExecuteActivity(
		actx,
		KeepEntitledActivityName,
		KeepEntitledRequest{Need: need, Organizations: organizations},
	).Get(actx, &entitled); err != nil {
		return nil, fmt.Errorf("check which tenants are entitled to %s: %w", need, err)
	}

	allowed := make(map[pulid.ID]struct{}, len(entitled))
	for _, organization := range entitled {
		allowed[organization] = struct{}{}
	}
	kept := make([]T, 0, len(items))
	for i := range items {
		if _, ok := allowed[owners[i]]; ok {
			kept = append(kept, items[i])
		}
	}
	return kept, nil
}

type EntitledActivities struct {
	filter entitlementports.Filter
}

func NewEntitledActivities(filter entitlementports.Filter) *EntitledActivities {
	return &EntitledActivities{filter: filter}
}

func (a *EntitledActivities) KeepEntitledOrganizationsActivity(
	ctx context.Context,
	req KeepEntitledRequest,
) ([]pulid.ID, error) {
	if err := req.Need.Validate(); err != nil {
		return nil, temporal.NewNonRetryableApplicationError(err.Error(), "InvalidNeed", err)
	}
	return a.filter.Keep(ctx, req.Need, req.Organizations)
}
