package worker_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func checklistTemplate() *worker.WorkerChecklistTemplate {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	credType := pulid.MustNew("wct_")
	docType := pulid.MustNew("dt_")
	return &worker.WorkerChecklistTemplate{
		ID:             pulid.MustNew("wclt_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Code:           "ONB",
		Name:           "Driver onboarding",
		Kind:           worker.ChecklistKindOnboarding,
		Trigger:        worker.ChecklistTriggerHired,
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		Items: []*worker.WorkerChecklistTemplateItem{
			{ID: pulid.MustNew("wclti_"), Label: "CDL on file", Kind: worker.ChecklistItemCredential, Required: true, Owner: worker.ChecklistOwnerSafety, CredentialTypeID: credType, DueOffsetDays: 3},
			{ID: pulid.MustNew("wclti_"), Label: "Drug test result", Kind: worker.ChecklistItemDocument, Required: true, Owner: worker.ChecklistOwnerSafety, DocumentTypeID: docType, DueOffsetDays: 7},
			{ID: pulid.MustNew("wclti_"), Label: "Dash invite", Kind: worker.ChecklistItemPortalAccess, Required: false, Owner: worker.ChecklistOwnerDispatch},
			{ID: pulid.MustNew("wclti_"), Label: "Fuel card issued", Kind: worker.ChecklistItemEquipment, Required: true, Owner: worker.ChecklistOwnerFleet, DueOffsetDays: 1},
		},
	}
}

func TestChecklistTemplate_Validate(t *testing.T) {
	template := checklistTemplate()
	multiErr := errortypes.NewMultiError()
	template.Validate(multiErr)
	assert.False(t, multiErr.HasErrors())

	template.Trigger = worker.ChecklistTriggerManual
	template.Items[0].CredentialTypeID = pulid.Nil
	template.Items[1].DocumentTypeID = pulid.Nil
	multiErr = errortypes.NewMultiError()
	template.Validate(multiErr)
	fields := map[string]bool{}
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = true
	}
	assert.True(t, fields["isDefault"], "a manual template cannot be the default")
	assert.True(t, fields["items[0].credentialTypeId"])
	assert.True(t, fields["items[1].documentTypeId"])

	empty := checklistTemplate()
	empty.Items = nil
	multiErr = errortypes.NewMultiError()
	empty.Validate(multiErr)
	assert.Equal(t, "items", multiErr.Errors[0].Field)
}

func TestChecklistTriggerForEvent(t *testing.T) {
	trigger, ok := worker.ChecklistTriggerForEvent(worker.EmploymentEventHired)
	assert.True(t, ok)
	assert.Equal(t, worker.ChecklistTriggerHired, trigger)
	trigger, ok = worker.ChecklistTriggerForEvent(worker.EmploymentEventTerminated)
	assert.True(t, ok)
	assert.Equal(t, worker.ChecklistTriggerTerminated, trigger)
	_, ok = worker.ChecklistTriggerForEvent(worker.EmploymentEventPromoted)
	assert.False(t, ok)
}

func TestChecklistTemplate_Instantiate(t *testing.T) {
	template := checklistTemplate()
	workerID := pulid.MustNew("wrk_")
	start := int64(1_800_000_000)
	checklist := template.Instantiate(workerID, start, pulid.MustNew("usr_"), pulid.MustNew("wee_"))

	assert.Equal(t, worker.ChecklistStatusOpen, checklist.Status)
	assert.Equal(t, template.Kind, checklist.Kind)
	assert.Equal(t, template.Name, checklist.Name)
	require.Len(t, checklist.Items, 4)
	assert.Equal(t, start+3*day, *checklist.Items[0].DueAt)
	assert.Nil(t, checklist.Items[2].DueAt, "items without an offset have no due date")
	assert.Equal(t, template.Items[0].ID, checklist.Items[0].TemplateItemID)
	assert.Equal(t, int32(3), checklist.Items[3].SortOrder)
	require.NotNil(t, checklist.DueAt)
	assert.Equal(t, start+7*day, *checklist.DueAt, "the checklist is due when its last item is")
}

func TestChecklist_ProgressAndAutoSatisfy(t *testing.T) {
	template := checklistTemplate()
	start := int64(1_800_000_000)
	checklist := template.Instantiate(pulid.MustNew("wrk_"), start, pulid.Nil, pulid.Nil)
	now := start + 5*day

	progress := checklist.Progress(now)
	assert.Equal(t, 4, progress.Total)
	assert.Equal(t, 0, progress.Settled)
	assert.Equal(t, 3, progress.RequiredTotal)
	assert.Equal(t, 2, progress.Overdue, "CDL (3d) and fuel card (1d) are past due, drug test (7d) is not")
	assert.False(t, progress.Complete())

	credID := pulid.MustNew("wcred_")
	expiry := now + 400*day
	docID := pulid.MustNew("doc_")
	changed := checklist.AutoSatisfy(worker.ChecklistEvidence{
		CredentialsByType: map[pulid.ID]*worker.WorkerCredential{
			template.Items[0].CredentialTypeID: {ID: credID, Status: worker.CredentialStatusActive, ExpiresAt: &expiry},
		},
		DocumentsByType: map[pulid.ID]*document.Document{
			template.Items[1].DocumentTypeID: {ID: docID},
		},
		HasPortalAccess: true,
	}, now)
	require.Len(t, changed, 3)
	assert.Equal(t, credID, checklist.Items[0].EvidenceCredentialID)
	assert.Equal(t, docID, checklist.Items[1].EvidenceDocumentID)
	assert.True(t, checklist.Items[2].AutoCompleted)
	assert.Equal(t, worker.ChecklistItemPending, checklist.Items[3].Status, "equipment is never auto-completed")

	progress = checklist.Progress(now)
	assert.Equal(t, 3, progress.Settled)
	assert.Equal(t, 2, progress.RequiredDone)
	assert.Equal(t, 75, progress.Percent)
	assert.False(t, progress.Complete())

	checklist.Items[3].Status = worker.ChecklistItemSkipped
	progress = checklist.Progress(now)
	assert.True(t, progress.Complete(), "a skipped required item still settles the checklist")
	assert.Equal(t, 0, progress.Overdue)

	again := checklist.AutoSatisfy(worker.ChecklistEvidence{HasPortalAccess: true}, now)
	assert.Empty(t, again, "settled items are not re-evaluated")
}

func TestChecklist_AutoSatisfyIgnoresExpiredCredentialAndFlipsPortalRuleForOffboarding(t *testing.T) {
	template := checklistTemplate()
	template.Kind = worker.ChecklistKindOffboarding
	now := int64(1_800_000_000)
	checklist := template.Instantiate(pulid.MustNew("wrk_"), now, pulid.Nil, pulid.Nil)
	expired := now - day

	changed := checklist.AutoSatisfy(worker.ChecklistEvidence{
		CredentialsByType: map[pulid.ID]*worker.WorkerCredential{
			template.Items[0].CredentialTypeID: {ID: pulid.MustNew("wcred_"), Status: worker.CredentialStatusActive, ExpiresAt: &expired},
		},
		HasPortalAccess: true,
	}, now)
	assert.Empty(t, changed, "an expired credential is not evidence, and portal access still granted does not settle offboarding")

	changed = checklist.AutoSatisfy(worker.ChecklistEvidence{HasPortalAccess: false}, now)
	require.Len(t, changed, 1)
	assert.Equal(t, worker.ChecklistItemPortalAccess, changed[0].Kind)
}

func TestChecklistKindClosedByEvent(t *testing.T) {
	kind, ok := worker.ChecklistKindClosedByEvent(worker.EmploymentEventTerminated)
	require.True(t, ok)
	assert.Equal(t, worker.ChecklistKindOnboarding, kind)

	kind, ok = worker.ChecklistKindClosedByEvent(worker.EmploymentEventRehired)
	require.True(t, ok)
	assert.Equal(t, worker.ChecklistKindOffboarding, kind)

	_, ok = worker.ChecklistKindClosedByEvent(worker.EmploymentEventPromoted)
	assert.False(t, ok)
}
