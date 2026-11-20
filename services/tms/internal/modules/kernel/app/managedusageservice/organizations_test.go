package managedusageservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadingsForOrganizationsAskEachMeterOnceForAllOfThem(t *testing.T) {
	t.Parallel()

	current := now - now%3600
	since := current - 3600
	orgs := []pulid.ID{pulid.MustNew("org_"), pulid.MustNew("org_"), pulid.MustNew("org_")}
	shipments := &fakeCounter{
		meter: platformcatalog.MeterShipmentsTotal,
		byOrg: []managedusageports.OrganizationHourCount{
			{OrganizationID: orgs[0], Hour: since + 60, Quantity: 3},
			{OrganizationID: orgs[1], Hour: current, Quantity: 7},
		},
	}
	storage := &fakeLevel{
		meter:  platformcatalog.MeterDocumentStorageBytes,
		levels: map[string]int64{orgs[2].String(): 2048},
	}
	svc := newServiceIn(&fakeRepo{}, true,
		[]managedusageports.HourlyCounter{shipments},
		[]managedusageports.LevelReader{storage})

	readings, err := svc.ReadingsForOrganizations(t.Context(), orgs, since)
	require.NoError(t, err)
	assert.Equal(t, 1, shipments.orgAsked, "one statement for every organization")
	assert.Equal(t, 1, storage.orgAsked)
	assert.Equal(t, orgs, shipments.orgReq.OrganizationIDs)
	assert.Equal(t, current+3600, shipments.orgReq.Until)

	require.Len(t, readings, 3)
	assert.Equal(t, []managedusageports.MeterReading{
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: since, Quantity: 3},
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: current, Quantity: 0},
		{Meter: platformcatalog.MeterDocumentStorageBytes, Hour: current, Quantity: 0},
	}, readings[orgs[0]])
	assert.Equal(t, int64(7), readings[orgs[1]][1].Quantity)
	assert.Equal(t, int64(2048), readings[orgs[2]][2].Quantity)
}

func TestReadingsForOrganizationsAreOnlyAShareServersQuestion(t *testing.T) {
	t.Parallel()

	svc := newServiceIn(&fakeRepo{}, false, nil, nil)
	_, err := svc.ReadingsForOrganizations(t.Context(), []pulid.ID{pulid.MustNew("org_")}, now-now%3600)
	require.ErrorIs(t, err, managedscope.ErrWrongTenancy)
}

func TestReadingsForOrganizationsKeepTheWindowRules(t *testing.T) {
	t.Parallel()

	svc := newServiceIn(&fakeRepo{}, true, nil, nil)
	_, err := svc.ReadingsForOrganizations(t.Context(), []pulid.ID{pulid.MustNew("org_")}, now-now%3600+1)
	require.Error(t, err)
}
