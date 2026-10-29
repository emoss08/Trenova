package workercredentialservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) hold(
	typ *workercredential.WorkerCredentialType,
	expiry int64,
) *workercredential.WorkerCredential {
	credential := h.credential(typ, expiry)
	credential.ID = pulid.MustNew("wcr_")
	credential.Status = workercredential.CredentialStatusActive
	h.repo.credentials[credential.ID] = credential

	return credential
}

func TestList_OneWorkerReadsWhatTheyHoldNarrowedByTypeAndRequired(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	cdl := h.hold(h.cdl, 1_900_000_000)
	h.hold(h.forklift, 1_900_000_000)

	all, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{h.wrk.ID},
	})
	require.NoError(t, err)
	assert.Len(t, all, 2)
	assert.Nil(t, h.repo.expiring, "one worker's record is not an expiry walk")

	byCode, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{h.wrk.ID}, CredentialTypeCode: " cdl ",
	})
	require.NoError(t, err)
	require.Len(t, byCode, 1, "a code matches whatever its case")
	assert.Equal(t, cdl.ID, byCode[0].ID)

	required, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{h.wrk.ID}, RequiredOnly: true,
	})
	require.NoError(t, err)
	require.Len(t, required, 1)
	assert.Equal(t, cdl.ID, required[0].ID)

	limited, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{h.wrk.ID}, Limit: 1,
	})
	require.NoError(t, err)
	assert.Len(t, limited, 1)

	rest, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{h.wrk.ID}, Limit: 1, Offset: 1,
	})
	require.NoError(t, err)
	require.Len(t, rest, 1)
	assert.NotEqual(t, limited[0].ID, rest[0].ID, "the next page starts after the first")

	past, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{h.wrk.ID}, Offset: 5,
	})
	require.NoError(t, err)
	assert.Empty(t, past)
}

func TestList_WhatFallsDueIsWalkedInTheQuery(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		req     workercredentialservice.ListRequest
		horizon int
		grace   int
		codes   []string
	}{
		"no worker reads what falls due within 30 days": {
			req:     workercredentialservice.ListRequest{Limit: 20},
			horizon: workercredentialservice.DefaultExpiryHorizonDays,
		},
		"a horizon past the most is clamped": {
			req:     workercredentialservice.ListRequest{ExpiringWithinDays: 900},
			horizon: workercredentialservice.MaxExpiryHorizonDays,
		},
		"expired ones reach back the grace period": {
			req:     workercredentialservice.ListRequest{IncludeExpired: true},
			horizon: workercredentialservice.DefaultExpiryHorizonDays,
			grace:   workercredentialservice.ExpiredGraceDays,
		},
		"the next page is walked from its offset": {
			req:     workercredentialservice.ListRequest{Limit: 26, Offset: 25},
			horizon: workercredentialservice.DefaultExpiryHorizonDays,
		},
		"a window with a type still walks only that window": {
			req: workercredentialservice.ListRequest{
				ExpiringWithinDays: 30, CredentialTypeCode: "ROAD_TEST",
			},
			horizon: 30,
			codes:   []string{"ROAD_TEST"},
		},
		"a code narrows the walk itself, upper-cased": {
			req: workercredentialservice.ListRequest{
				ExpiringWithinDays: 7, CredentialTypeCode: "med_card", RequiredOnly: true,
			},
			horizon: 7,
			codes:   []string{"MED_CARD"},
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			req := tc.req
			req.TenantInfo = h.tenant
			_, err := h.svc.List(t.Context(), &req)
			require.NoError(t, err)

			got := h.repo.expiring
			require.NotNil(t, got)
			assert.Equal(t, h.tenant, got.TenantInfo)
			assert.Equal(t, tc.horizon, got.HorizonDays)
			assert.Equal(t, tc.grace, got.GraceDays)
			assert.Equal(t, tc.codes, got.CredentialTypeCodes)
			assert.Equal(t, req.RequiredOnly, got.RequiredOnly)
			assert.Equal(t, req.Limit, got.Limit)
			assert.Equal(t, req.Offset, got.Offset)
			assert.True(t, got.SoonestFirst)
			assert.False(t, got.AnyExpiry, "what falls due is walked inside its window")
		})
	}
}

// A type named across the organization with no window is who holds it: a road
// test never expires, so no window ever found one, and asked who had one the
// agent heard that nobody did.
func TestList_ATypeAcrossTheOrganizationIsWhoHoldsIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, CredentialTypeCode: "road_test", Limit: 25,
	})
	require.NoError(t, err)

	got := h.repo.expiring
	require.NotNil(t, got)
	assert.True(t, got.AnyExpiry)
	assert.Equal(t, []string{"ROAD_TEST"}, got.CredentialTypeCodes)
	assert.Empty(t, got.WorkerIDs)
	assert.True(t, got.SoonestFirst)
}

// Several workers are read in one call, each one's credentials as one
// worker's call reads them, and a call past the most is refused.
func TestList_ReadsSeveralWorkersAtOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.hold(h.cdl, 1_900_000_000)
	other := pulid.MustNew("wrk_")

	held, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: []pulid.ID{other, h.wrk.ID},
	})
	require.NoError(t, err)
	assert.Len(t, held, 1)
	assert.Nil(t, h.repo.expiring, "the workers' own records are not an expiry walk")

	many := make([]pulid.ID, workercredentialservice.MaxListedWorkers+1)
	for idx := range many {
		many[idx] = pulid.MustNew("wrk_")
	}
	_, err = h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: h.tenant, WorkerIDs: many,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "workerIds", verr.Field)
}

// ListExpiring walks every tenant when handed none, which the nightly sweep
// wants and a read on behalf of a person or an agent must never ask for.
func TestList_NeverWalksWithoutATenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.List(t.Context(), &workercredentialservice.ListRequest{
		TenantInfo: pagination.TenantInfo{OrgID: h.tenant.OrgID},
	})

	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Nil(t, h.repo.expiring)
}
