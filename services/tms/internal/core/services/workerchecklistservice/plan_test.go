package workerchecklistservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *harness) manualTemplate() *worker.WorkerChecklistTemplate {
	manual := &worker.WorkerChecklistTemplate{
		ID:             pulid.MustNew("wclt_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		Code:           "AUDIT",
		Name:           "Annual file audit",
		Kind:           worker.ChecklistKindCustom,
		Trigger:        worker.ChecklistTriggerManual,
		Status:         domaintypes.StatusActive,
		Items: []*worker.WorkerChecklistTemplateItem{
			{
				ID:       pulid.MustNew("wclti_"),
				Label:    "Review file",
				Kind:     worker.ChecklistItemTask,
				Required: true,
				Owner:    worker.ChecklistOwnerHR,
			},
		},
	}
	h.repo.templates[manual.ID] = manual
	return manual
}

func TestPlanStart_OpensNothingAndFindsTheOpenOne(t *testing.T) {
	h := newHarness(t)
	manual := h.manualTemplate()
	req := &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		TemplateID: manual.ID,
		UserID:     h.userID,
	}

	plan, err := h.svc.PlanStart(t.Context(), req)
	require.NoError(t, err)
	assert.False(t, plan.Existing)
	assert.Len(t, plan.Checklist.Items, 1)
	assert.Empty(t, h.repo.checklists)

	started, err := h.svc.Start(t.Context(), req)
	require.NoError(t, err)
	again, err := h.svc.PlanStart(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, again.Existing)
	assert.Equal(t, started.ID, again.Checklist.ID)

	_, err = h.svc.PlanStart(t.Context(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		TemplateID: h.template.ID,
	})
	require.Error(t, err, "an onboarding is started by the hire, not by hand")
}

func TestPlanItems_SettleAndReopenWithoutWriting(t *testing.T) {
	h := newHarness(t)
	manual := h.manualTemplate()
	started, err := h.svc.Start(t.Context(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		TemplateID: manual.ID,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	item := h.itemByLabel(started, "Review file")
	require.NotNil(t, item)

	_, err = h.svc.PlanSkipItem(t.Context(), &workerchecklistservice.ItemRequest{
		ID: item.ID, TenantInfo: h.tenant,
	})
	require.Error(t, err, "a skip needs a note")

	done, err := h.svc.PlanCompleteItem(t.Context(), &workerchecklistservice.ItemRequest{
		ID: item.ID, TenantInfo: h.tenant, Note: " Checked ", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ChecklistItemDone, done.Item.After.Status)
	assert.Equal(t, "Checked", done.Item.After.Note)
	assert.Equal(t, worker.ChecklistItemPending, h.repo.items[item.ID].Status)

	pending, err := h.svc.PlanReopenItem(t.Context(), &workerchecklistservice.ReopenItemRequest{
		ID: item.ID, TenantInfo: h.tenant,
	})
	require.NoError(t, err)
	assert.Equal(t, pending.Item.Before.Status, pending.Item.After.Status,
		"a pending item stays pending")

	cancel, err := h.svc.PlanCancel(t.Context(), &workerchecklistservice.CancelRequest{
		ID: started.ID, TenantInfo: h.tenant, Reason: "Started by mistake",
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ChecklistStatusCancelled, cancel.After.Status)
	assert.Equal(t, worker.ChecklistStatusOpen, h.repo.checklists[started.ID].Status)
}
