package tariff_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff/record"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/ratetablecache"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type holidayLister struct {
	workerports.OrgHolidayRepository

	mu       sync.Mutex
	requests []workerports.ListOrgHolidaysRequest
	rows     []*pto.OrgHoliday
}

func (h *holidayLister) List(
	_ context.Context,
	req *workerports.ListOrgHolidaysRequest,
) ([]*pto.OrgHoliday, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.requests = append(h.requests, *req)
	return h.rows, nil
}

func TestRecordSourceLoadsHolidaysOncePerOrganizationAndYearInABatch(t *testing.T) {
	t.Parallel()

	lister := &holidayLister{rows: []*pto.OrgHoliday{{
		Name:           "Christmas Day",
		HolidayDate:    time.Date(2020, time.December, 25, 0, 0, 0, 0, time.UTC).Unix(),
		Kind:           pto.HolidayKindHoliday,
		RecursAnnually: true,
	}}}
	source := tariff.NewRecordSource(tariff.RecordSourceParams{Holidays: lister})
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: tenant.BuID}
	ctx := ratetablecache.With(t.Context())

	first, err := source.Holidays(ctx, tenant, 2026)
	require.NoError(t, err)
	again, err := source.Holidays(ctx, tenant, 2026)
	require.NoError(t, err)
	_, err = source.Holidays(ctx, other, 2026)
	require.NoError(t, err)

	assert.Equal(t, first, again)
	assert.Len(t, first, 3)
	require.Len(t, lister.requests, 2, "one read per organisation in the batch")
	window := record.HolidayWindowFor(2026)
	assert.Equal(t, window.From, lister.requests[0].From)
	assert.Equal(t, window.To, lister.requests[0].To)
	assert.Equal(t, tenant, lister.requests[0].TenantInfo)
	assert.Equal(t, other, lister.requests[1].TenantInfo)
}
