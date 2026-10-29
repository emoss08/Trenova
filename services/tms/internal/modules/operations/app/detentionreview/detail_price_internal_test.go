package detentionreview

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/app/detentioneffects"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type detailOccurrences struct {
	detentionports.DetentionOccurrenceRepository

	occurrence *detention.DetentionOccurrence
}

func (d *detailOccurrences) GetByID(
	_ context.Context,
	_ *detentionports.GetDetentionOccurrenceByIDRequest,
) (*detention.DetentionOccurrence, error) {
	return d.occurrence, nil
}

func (d *detailOccurrences) GetByShipment(
	_ context.Context,
	_ *detentionports.GetOccurrencesByShipmentRequest,
) ([]*detention.DetentionOccurrence, error) {
	return []*detention.DetentionOccurrence{d.occurrence}, nil
}

func detailService(t *testing.T, occurrence *detention.DetentionOccurrence, now int64) *Service {
	t.Helper()

	organizations := tenantportsmock.NewMockOrganizationRepository(t)
	organizations.EXPECT().GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Trenova Freight", Timezone: "UTC"}, nil).Maybe()

	return &Service{
		occurrenceRepo: &detailOccurrences{occurrence: occurrence},
		now:            func() int64 { return now },
		effects: detentioneffects.New(detentioneffects.Params{
			Logger:        zap.NewNop(),
			Organizations: organizations,
		}),
	}
}

func TestGetOccurrenceDetail_PricesARunningClockAsOfNow(t *testing.T) {
	t.Parallel()

	arrived := int64(1767225600)
	now := arrived + 10*3600
	running := &detention.DetentionOccurrence{
		ID:              pulid.MustNew("dto_"),
		ShipmentID:      pulid.MustNew("shp_"),
		PolicySnapshot:  hourlySnapshot(),
		StopType:        shipmentvocab.StopTypeDelivery,
		ScheduleType:    shipmentvocab.StopScheduleTypeAppointment,
		ArrivedAt:       &arrived,
		ClockStartAt:    arrived,
		IsOpen:          true,
		Status:          detention.OccurrenceStatusAccruing,
		BillableMinutes: 15,
		BillableAmount:  decimal.RequireFromString("18.75"),
	}

	detail, err := detailService(t, running, now).GetOccurrenceDetail(
		t.Context(),
		&detentionports.GetDetentionOccurrenceByIDRequest{
			OccurrenceID: running.ID,
			TenantInfo: pagination.TenantInfo{
				OrgID: pulid.MustNew("org_"),
				BuID:  pulid.MustNew("bu_"),
			},
		},
	)

	require.NoError(t, err)
	require.NotNil(t, detail.CurrentlyBillable)
	assert.Equal(t, int32(480), detail.CurrentlyBillable.BillableMinutes)
	assert.True(t,
		detail.CurrentlyBillable.BillableAmount.Equal(decimal.RequireFromString("600")),
		detail.CurrentlyBillable.BillableAmount.String(),
	)
	assert.Equal(t, now, detail.CurrentlyBillable.AsOf)
}

func TestGetOccurrenceDetail_LeavesAStoppedClockWithItsOwnFigure(t *testing.T) {
	t.Parallel()

	arrived := int64(1767225600)
	departed := arrived + 5*3600
	stopped := &detention.DetentionOccurrence{
		ID:             pulid.MustNew("dto_"),
		ShipmentID:     pulid.MustNew("shp_"),
		PolicySnapshot: hourlySnapshot(),
		ArrivedAt:      &arrived,
		DepartedAt:     &departed,
		ClockStartAt:   arrived,
		Status:         detention.OccurrenceStatusPending,
		BillableAmount: decimal.RequireFromString("225"),
	}

	detail, err := detailService(t, stopped, departed+3600).GetOccurrenceDetail(
		t.Context(),
		&detentionports.GetDetentionOccurrenceByIDRequest{OccurrenceID: stopped.ID},
	)

	require.NoError(t, err)
	assert.Nil(t, detail.CurrentlyBillable)
}
