package tenantlifecycle

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/maypok86/otter/v2"
)

const (
	instanceTagTTL     = 10 * time.Minute
	instanceTagEntries = 2000
)

type InstanceTags struct {
	tenants cloudtenantports.Repository
	cache   *otter.Cache[pulid.ID, string]
}

var _ emailports.InstanceTags = (*InstanceTags)(nil)

func NewInstanceTags(tenants cloudtenantports.Repository) *InstanceTags {
	return &InstanceTags{
		tenants: tenants,
		cache: otter.Must(&otter.Options[pulid.ID, string]{
			MaximumSize:      instanceTagEntries,
			ExpiryCalculator: otter.ExpiryWriting[pulid.ID, string](instanceTagTTL),
		}),
	}
}

func (t *InstanceTags) InstanceTag(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	if tenantInfo.OrgID.IsNil() {
		return "", nil
	}
	tag, err := t.cache.Get(ctx, tenantInfo.OrgID, otter.LoaderFunc[pulid.ID, string](
		func(ctx context.Context, orgID pulid.ID) (string, error) {
			return t.tenants.InstanceOf(ctx, orgID)
		},
	))
	if errors.Is(err, cloudtenantports.ErrNotHeld) {
		return "", nil
	}
	return tag, err
}
