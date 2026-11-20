package driversettlementservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type settlementControlStub struct {
	tenantports.SettlementControlRepository
	control *settlementcontrol.SettlementControl
}

func (s *settlementControlStub) GetOrCreate(
	context.Context,
	pagination.TenantInfo,
) (*settlementcontrol.SettlementControl, error) {
	return s.control, nil
}

func TestCurrentPeriodResolvesInTheOrganizationsTimezone(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	control := settlementcontrol.NewDefault(tenantInfo.OrgID, tenantInfo.BuID)

	organizations := tenantportsmock.NewMockOrganizationRepository(t)
	organizations.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(&tenant.Organization{ID: tenantInfo.OrgID, Timezone: "America/Chicago"}, nil).
		Once()

	svc := &Service{
		settlementControl: &settlementControlStub{control: control},
		organizations:     organizations,
	}

	got, err := svc.CurrentPeriod(t.Context(), tenantInfo)
	require.NoError(t, err)

	loc, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	assert.Equal(t, ResolveCurrentPeriod(control, loc, timeutils.NowUnix()), got)

	end := time.Unix(got.PeriodEnd, 0).In(loc)
	assert.Equal(t, time.Sunday, end.Weekday())
	assert.Equal(t, 0, end.Hour())
	assert.Equal(t, 0, end.Minute())
}

func TestCurrentPeriodFailsWhenTheOrganizationCannotBeRead(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	readErr := errors.New("organization unavailable")

	organizations := tenantportsmock.NewMockOrganizationRepository(t)
	organizations.EXPECT().
		GetByID(mock.Anything, tenantports.GetOrganizationByIDRequest{TenantInfo: tenantInfo}).
		Return(nil, readErr).
		Once()

	svc := &Service{
		settlementControl: &settlementControlStub{
			control: settlementcontrol.NewDefault(tenantInfo.OrgID, tenantInfo.BuID),
		},
		organizations: organizations,
	}

	got, err := svc.CurrentPeriod(t.Context(), tenantInfo)
	require.ErrorIs(t, err, readErr)
	assert.Equal(t, PeriodBounds{}, got)
}
