package triagesupport

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	DaySeconds        = 24 * 60 * 60
	PreviewWindowDays = 30
	ProfileWindowDays = 90
)

type OrganizationReader interface {
	GetByID(
		ctx context.Context,
		req tenantports.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
}

type Clock struct {
	orgs OrganizationReader
	now  func() int64
}

func NewClock(orgs OrganizationReader, now func() int64) *Clock {
	if now == nil {
		now = timeutils.NowUnix
	}

	return &Clock{orgs: orgs, now: now}
}

func (c *Clock) Now() int64 {
	return c.now()
}

func (c *Clock) DaysAgo(days int) int64 {
	return c.now() - int64(days)*DaySeconds
}

func (c *Clock) Location(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*time.Location, error) {
	if c.orgs == nil {
		return time.UTC, nil
	}

	org, err := c.orgs.GetByID(ctx, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo})
	if err != nil {
		return nil, fmt.Errorf("read the organization's time zone: %w", err)
	}

	loc, _ := timeutils.ResolveZone(org.Timezone)

	return loc, nil
}
