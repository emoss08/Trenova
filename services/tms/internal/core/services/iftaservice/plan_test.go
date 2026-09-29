package iftaservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanGenerate_ComputesTheDraftWithoutSavingIt(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	req := &iftaservice.GenerateReturnRequest{
		TenantInfo: h.tenant, Period: h.period, UserID: h.userID,
	}
	planned, err := h.svc.PlanGenerate(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, ifta.ReturnStatusDraft, planned.Status)
	assert.Equal(t, "1000", planned.TotalMiles.String())
	assert.NotEmpty(t, planned.Lines)
	assert.Empty(t, h.repo.returns)

	h.generate(t)
	_, err = h.svc.PlanGenerate(t.Context(), req)
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
}

func TestPlanRecompute_ShowsTheNewFiguresAndKeepsTheStoredReturn(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ret := h.generate(t)

	h.repo.miles.RouteRows[0].Miles = dec("2000")
	h.repo.miles.RouteRows[0].LoadedMiles = dec("2000")
	change, err := h.svc.PlanRecompute(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: ret.ID, Version: ret.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, "1000", change.Before.TotalMiles.String())
	assert.Equal(t, "2000", change.After.TotalMiles.String())
	assert.Equal(t, ret.Version, h.repo.returns[ret.ID].Version)
	assert.Equal(t, "1000", h.repo.returns[ret.ID].TotalMiles.String())

	finalized := h.finalize(t, ret)
	_, err = h.svc.PlanRecompute(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
	})
	assertValidationField(t, err, "status")
}

func TestPlanAmend_OpensNothingUntilTheAmendmentIsMade(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	filed := h.file(t, h.finalize(t, h.generate(t)))

	plan, err := h.svc.PlanAmend(t.Context(), &iftaservice.AmendReturnRequest{
		TenantInfo: h.tenant, ID: filed.ID, Reason: "Missed a fuel purchase in Oklahoma",
		UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, filed.ID, plan.Filed.ID)
	assert.Equal(t, 1, plan.Draft.AmendmentNumber)
	assert.Equal(t, ifta.ReturnStatusDraft, plan.Draft.Status)
	assert.Len(t, h.repo.returns, 1)

	_, err = h.svc.PlanAmend(t.Context(), &iftaservice.AmendReturnRequest{
		TenantInfo: h.tenant, ID: filed.ID, Reason: "short", UserID: h.userID,
	})
	assertValidationField(t, err, "reason")
}

func TestPlanDelete_DraftOnly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	draft := h.generate(t)

	planned, err := h.svc.PlanDelete(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: draft.ID, Version: draft.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, draft.ID, planned.ID)
	assert.Len(t, h.repo.returns, 1)

	finalized := h.finalize(t, draft)
	_, err = h.svc.PlanDelete(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
	})
	assertValidationField(t, err, "status")
}

func TestPlanMileageEntries_CheckWhatTheWriteChecksAndSaveNothing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entry := &ifta.JurisdictionMileageEntry{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		TractorID:      h.tractorID,
		JurisdictionID: h.tx.ID,
		TraveledAt:     h.now,
		Miles:          dec("212.345"),
		Loaded:         true,
	}
	planned, err := h.svc.PlanCreateMileageEntry(t.Context(), entry, h.userID)
	require.NoError(t, err)
	assert.Equal(t, "212.35", planned.Miles.StringFixed(2))
	assert.Equal(t, ifta.MileageSourceManual, planned.Source)
	assert.Equal(t, ifta.NewPeriod(2026, 2), planned.Period())
	assert.Empty(t, h.repo.entries)

	created, err := h.svc.CreateMileageEntry(t.Context(), entry, h.userID)
	require.NoError(t, err)

	edited := *created
	edited.Miles = dec("300")
	change, err := h.svc.PlanUpdateMileageEntry(t.Context(), &edited, h.userID)
	require.NoError(t, err)
	assert.Equal(t, "212.35", change.Before.Miles.StringFixed(2))
	assert.Equal(t, "300.00", change.After.Miles.StringFixed(2))
	assert.Equal(t, "212.35", h.repo.entries[created.ID].Miles.StringFixed(2))

	stale := *created
	stale.Version++
	_, err = h.svc.PlanUpdateMileageEntry(t.Context(), &stale, h.userID)
	assertValidationField(t, err, "version")

	toDelete, err := h.svc.PlanDeleteMileageEntry(t.Context(), &iftaservice.DeleteMileageEntryRequest{
		TenantInfo: h.tenant, ID: created.ID, Version: created.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, created.ID, toDelete.ID)
	assert.Len(t, h.repo.entries, 1)
}
