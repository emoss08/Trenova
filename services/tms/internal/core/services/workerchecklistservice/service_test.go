package workerchecklistservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerchecklistservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const day = int64(86400)

type fakeRepo struct {
	templates  map[pulid.ID]*worker.WorkerChecklistTemplate
	checklists map[pulid.ID]*worker.WorkerChecklist
	items      map[pulid.ID]*worker.WorkerChecklistItem
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		templates:  map[pulid.ID]*worker.WorkerChecklistTemplate{},
		checklists: map[pulid.ID]*worker.WorkerChecklist{},
		items:      map[pulid.ID]*worker.WorkerChecklistItem{},
	}
}

func (f *fakeRepo) ListTemplates(
	context.Context,
	*repositories.ListChecklistTemplatesRequest,
) (*pagination.CursorListResult[*worker.WorkerChecklistTemplate], error) {
	return &pagination.CursorListResult[*worker.WorkerChecklistTemplate]{}, nil
}

func (f *fakeRepo) ListActiveTemplates(
	context.Context,
	pagination.TenantInfo,
) ([]*worker.WorkerChecklistTemplate, error) {
	out := make([]*worker.WorkerChecklistTemplate, 0, len(f.templates))
	for _, t := range f.templates {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeRepo) GetTemplateByID(
	_ context.Context,
	req *repositories.GetChecklistTemplateByIDRequest,
) (*worker.WorkerChecklistTemplate, error) {
	t, ok := f.templates[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("template")
	}
	return t, nil
}

func (f *fakeRepo) GetDefaultTemplate(
	_ context.Context,
	req *repositories.GetDefaultChecklistTemplateRequest,
) (*worker.WorkerChecklistTemplate, error) {
	for _, t := range f.templates {
		if t.IsDefault && t.Trigger == req.Trigger && t.Status == domaintypes.StatusActive {
			return t, nil
		}
	}
	return nil, errortypes.NewNotFoundError("template")
}

func (f *fakeRepo) TemplateCodeExists(
	context.Context,
	*repositories.ChecklistTemplateCodeExistsRequest,
) (bool, error) {
	return false, nil
}

func (f *fakeRepo) CreateTemplate(
	_ context.Context,
	entity *worker.WorkerChecklistTemplate,
) (*worker.WorkerChecklistTemplate, error) {
	entity.ID = pulid.MustNew("wclt_")
	for _, item := range entity.Items {
		item.ID = pulid.MustNew("wclti_")
	}
	f.templates[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateTemplate(
	_ context.Context,
	entity *worker.WorkerChecklistTemplate,
) (*worker.WorkerChecklistTemplate, error) {
	entity.Version++
	f.templates[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ClearDefaultTemplate(
	context.Context,
	*repositories.ClearDefaultChecklistTemplateRequest,
) error {
	return nil
}

func (f *fakeRepo) CountOpenChecklists(
	context.Context,
	*repositories.CountOpenChecklistsRequest,
) (int, error) {
	return 0, nil
}

func (f *fakeRepo) CountOpenChecklistsByTemplateIDs(
	context.Context,
	*repositories.CountOpenChecklistsByTemplateIDsRequest,
) (map[pulid.ID]int, error) {
	return map[pulid.ID]int{}, nil
}

func (f *fakeRepo) ListForWorker(
	_ context.Context,
	req *repositories.ListWorkerChecklistsRequest,
) ([]*worker.WorkerChecklist, error) {
	out := make([]*worker.WorkerChecklist, 0, len(f.checklists))
	for _, c := range f.checklists {
		if c.WorkerID != req.WorkerID {
			continue
		}
		if !req.IncludeClosed && !c.IsOpen() {
			continue
		}
		out = append(out, c)
	}
	return out, nil
}

func (f *fakeRepo) GetByID(
	_ context.Context,
	req *repositories.GetWorkerChecklistByIDRequest,
) (*worker.WorkerChecklist, error) {
	c, ok := f.checklists[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("checklist")
	}
	return c, nil
}

func (f *fakeRepo) GetItemByID(
	_ context.Context,
	req *repositories.GetWorkerChecklistItemByIDRequest,
) (*worker.WorkerChecklistItem, error) {
	item, ok := f.items[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("item")
	}
	return item, nil
}

func (f *fakeRepo) Create(
	_ context.Context,
	entity *worker.WorkerChecklist,
) (*worker.WorkerChecklist, error) {
	entity.ID = pulid.MustNew("wcl_")
	for _, item := range entity.Items {
		item.ID = pulid.MustNew("wcli_")
		item.ChecklistID = entity.ID
		f.items[item.ID] = item
	}
	f.checklists[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) Update(
	_ context.Context,
	entity *worker.WorkerChecklist,
) (*worker.WorkerChecklist, error) {
	entity.Version++
	f.checklists[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateItems(_ context.Context, items []*worker.WorkerChecklistItem) error {
	for _, item := range items {
		item.Version++
		f.items[item.ID] = item
	}
	return nil
}

type fakeCredentialRepo struct {
	repositories.WorkerCredentialRepository
	credentials []*worker.WorkerCredential
}

func (f *fakeCredentialRepo) ListForWorker(
	context.Context,
	*repositories.ListWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	return f.credentials, nil
}

type harness struct {
	svc        *workerchecklistservice.Service
	repo       *fakeRepo
	creds      *fakeCredentialRepo
	docs       *mocks.MockDocumentRepository
	workerRepo *mocks.MockWorkerRepository
	tenant     pagination.TenantInfo
	wrk        *worker.Worker
	template   *worker.WorkerChecklistTemplate
	credType   pulid.ID
	docType    pulid.ID
	userID     pulid.ID
	qualified  []bool
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Status:         domaintypes.StatusActive,
		Profile:        &worker.WorkerProfile{HireDate: 1_700_000_000},
	}
	h := &harness{
		repo:     newFakeRepo(),
		creds:    &fakeCredentialRepo{},
		tenant:   tenant,
		wrk:      wrk,
		credType: pulid.MustNew("wct_"),
		docType:  pulid.MustNew("dt_"),
		userID:   pulid.MustNew("usr_"),
	}

	h.workerRepo = mocks.NewMockWorkerRepository(t)
	h.workerRepo.EXPECT().GetByID(mock.Anything, mock.Anything).Return(wrk, nil).Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileQualification(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateProfileQualificationRequest) error {
			h.qualified = append(h.qualified, req.Qualified)
			return nil
		}).
		Maybe()
	h.docs = mocks.NewMockDocumentRepository(t)
	h.docs.EXPECT().GetByResourceID(mock.Anything, mock.Anything).Return(nil, nil).Maybe()
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	h.template = &worker.WorkerChecklistTemplate{
		ID:             pulid.MustNew("wclt_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Code:           "ONB",
		Name:           "Driver onboarding",
		Kind:           worker.ChecklistKindOnboarding,
		Trigger:        worker.ChecklistTriggerHired,
		Status:         domaintypes.StatusActive,
		IsDefault:      true,
		Items: []*worker.WorkerChecklistTemplateItem{
			{
				ID:               pulid.MustNew("wclti_"),
				Label:            "CDL on file",
				Kind:             worker.ChecklistItemCredential,
				Required:         true,
				Owner:            worker.ChecklistOwnerSafety,
				CredentialTypeID: h.credType,
				DueOffsetDays:    3,
			},
			{
				ID:            pulid.MustNew("wclti_"),
				Label:         "Fuel card",
				Kind:          worker.ChecklistItemEquipment,
				Required:      true,
				Owner:         worker.ChecklistOwnerFleet,
				DueOffsetDays: 2,
			},
			{
				ID:       pulid.MustNew("wclti_"),
				Label:    "Dash invite",
				Kind:     worker.ChecklistItemPortalAccess,
				Required: false,
				Owner:    worker.ChecklistOwnerDispatch,
			},
		},
	}
	h.repo.templates[h.template.ID] = h.template

	h.svc = workerchecklistservice.New(workerchecklistservice.Params{
		Logger:         zap.NewNop(),
		Repo:           h.repo,
		WorkerRepo:     h.workerRepo,
		CredentialRepo: h.creds,
		DocumentRepo:   h.docs,
		AuditService:   audit,
	})
	return h
}

func (h *harness) itemByLabel(
	checklist *worker.WorkerChecklist,
	label string,
) *worker.WorkerChecklistItem {
	for _, item := range checklist.Items {
		if item.Label == label {
			return item
		}
	}
	return nil
}

func TestSpawnForEvent_UsesDefaultTemplateAndAutoSatisfies(t *testing.T) {
	h := newHarness(t)
	h.wrk.UserID = pulid.MustNew("usr_")
	expiry := int64(2_000_000_000)
	h.creds.credentials = []*worker.WorkerCredential{
		{
			ID:               pulid.MustNew("wcred_"),
			CredentialTypeID: h.credType,
			Status:           worker.CredentialStatusActive,
			ExpiresAt:        &expiry,
		},
	}
	event := &worker.WorkerEmploymentEvent{
		ID:          pulid.MustNew("wee_"),
		Kind:        worker.EmploymentEventHired,
		EffectiveAt: 1_800_000_000,
	}

	checklist, err := h.svc.SpawnForEvent(context.Background(), event, h.wrk, h.userID)
	require.NoError(t, err)
	require.NotNil(t, checklist)
	assert.Equal(t, worker.ChecklistStatusOpen, checklist.Status)
	assert.Equal(t, event.ID, checklist.SourceEventID)
	assert.Equal(t, int64(1_800_000_000), checklist.StartedAt)

	cdl := h.itemByLabel(checklist, "CDL on file")
	assert.Equal(t, worker.ChecklistItemDone, cdl.Status)
	assert.True(t, cdl.AutoCompleted)
	assert.Equal(t, h.creds.credentials[0].ID, cdl.EvidenceCredentialID)
	assert.Equal(t, worker.ChecklistItemDone, h.itemByLabel(checklist, "Dash invite").Status)
	assert.Equal(t, worker.ChecklistItemPending, h.itemByLabel(checklist, "Fuel card").Status)
	assert.Empty(t, h.qualified, "the required equipment item keeps onboarding open")

	again, err := h.svc.SpawnForEvent(context.Background(), event, h.wrk, h.userID)
	require.NoError(t, err)
	assert.Equal(
		t,
		checklist.ID,
		again.ID,
		"a second spawn from the same template returns the open checklist",
	)

	none, err := h.svc.SpawnForEvent(
		context.Background(),
		&worker.WorkerEmploymentEvent{Kind: worker.EmploymentEventPromoted},
		h.wrk,
		h.userID,
	)
	require.NoError(t, err)
	assert.Nil(t, none)
}

func TestCompleteItem_ClosesChecklistAndFlagsQualification(t *testing.T) {
	h := newHarness(t)
	expiry := int64(2_000_000_000)
	h.creds.credentials = []*worker.WorkerCredential{
		{
			ID:               pulid.MustNew("wcred_"),
			CredentialTypeID: h.credType,
			Status:           worker.CredentialStatusActive,
			ExpiresAt:        &expiry,
		},
	}
	checklist, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo:    h.tenant,
		WorkerID:      h.wrk.ID,
		TemplateID:    h.template.ID,
		SourceEventID: pulid.MustNew("wee_"),
		UserID:        h.userID,
	})
	require.NoError(t, err)
	fuel := h.itemByLabel(checklist, "Fuel card")

	updated, err := h.svc.CompleteItem(context.Background(), &workerchecklistservice.ItemRequest{
		ID:         fuel.ID,
		TenantInfo: h.tenant,
		Note:       "Card 4471 issued",
		Version:    fuel.Version,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ChecklistStatusCompleted, updated.Status)
	require.NotNil(t, updated.CompletedAt)
	assert.Equal(t, []bool{true}, h.qualified, "closing onboarding marks the worker DQF-ready")
	done := h.itemByLabel(updated, "Fuel card")
	assert.Equal(t, h.userID, done.CompletedByID)
	assert.Equal(t, "Card 4471 issued", done.Note)
	assert.False(t, done.AutoCompleted)

	_, err = h.svc.CompleteItem(context.Background(), &workerchecklistservice.ItemRequest{
		ID: fuel.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr, "a settled item cannot be completed twice")
}

func TestSkipItem_NeedsNoteAndEvidenceMustBelongToWorker(t *testing.T) {
	h := newHarness(t)
	checklist, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID, UserID: h.userID, SourceEventID: pulid.MustNew("wee_"),
	})
	require.NoError(t, err)
	fuel := h.itemByLabel(checklist, "Fuel card")

	_, err = h.svc.SkipItem(context.Background(), &workerchecklistservice.ItemRequest{
		ID: fuel.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "note", verr.Field)

	h.docs.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&document.Document{ID: pulid.MustNew("doc_"), ResourceType: "worker", ResourceID: pulid.MustNew("wrk_").String()}, nil).
		Once()
	_, err = h.svc.CompleteItem(context.Background(), &workerchecklistservice.ItemRequest{
		ID: fuel.ID, TenantInfo: h.tenant, EvidenceDocumentID: pulid.MustNew("doc_"), UserID: h.userID,
	})
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "evidenceDocumentId", verr.Field)
}

func TestReopenItem_ReopensCompletedChecklist(t *testing.T) {
	h := newHarness(t)
	expiry := int64(2_000_000_000)
	h.creds.credentials = []*worker.WorkerCredential{
		{
			ID:               pulid.MustNew("wcred_"),
			CredentialTypeID: h.credType,
			Status:           worker.CredentialStatusActive,
			ExpiresAt:        &expiry,
		},
	}
	checklist, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID, UserID: h.userID, SourceEventID: pulid.MustNew("wee_"),
	})
	require.NoError(t, err)
	fuel := h.itemByLabel(checklist, "Fuel card")
	completed, err := h.svc.CompleteItem(context.Background(), &workerchecklistservice.ItemRequest{
		ID: fuel.ID, TenantInfo: h.tenant, Version: fuel.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	require.Equal(t, worker.ChecklistStatusCompleted, completed.Status)

	reopened, err := h.svc.ReopenItem(
		context.Background(),
		&workerchecklistservice.ReopenItemRequest{
			ID: fuel.ID, TenantInfo: h.tenant, UserID: h.userID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, worker.ChecklistStatusOpen, reopened.Status)
	assert.Nil(t, reopened.CompletedAt)
	assert.Equal(t, worker.ChecklistItemPending, h.itemByLabel(reopened, "Fuel card").Status)
}

func TestCancel_LeavesItemsAndRefusesInactiveTemplate(t *testing.T) {
	h := newHarness(t)
	checklist, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID, UserID: h.userID, SourceEventID: pulid.MustNew("wee_"),
	})
	require.NoError(t, err)

	cancelled, err := h.svc.Cancel(context.Background(), &workerchecklistservice.CancelRequest{
		ID: checklist.ID, TenantInfo: h.tenant, Reason: "Hire fell through", Version: checklist.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ChecklistStatusCancelled, cancelled.Status)
	assert.Equal(t, "Hire fell through", cancelled.CancelReason)
	assert.Len(t, cancelled.Items, 3)

	h.template.Status = domaintypes.StatusInactive
	_, err = h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant, WorkerID: h.wrk.ID, TemplateID: h.template.ID, UserID: h.userID, SourceEventID: pulid.MustNew("wee_"),
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "templateId", verr.Field)
}

func TestProgressWindow(t *testing.T) {
	template := &worker.WorkerChecklistTemplate{
		Items: []*worker.WorkerChecklistTemplateItem{
			{Label: "x", Kind: worker.ChecklistItemTask, Required: true, DueOffsetDays: 1},
		},
	}
	checklist := template.Instantiate(pulid.MustNew("wrk_"), 100, pulid.Nil, pulid.Nil)
	assert.Equal(t, 1, checklist.Progress(100+2*day).Overdue)
}

// Onboarding is something that happens to a worker when they are hired, not
// a button. Starting the hired-trigger template by hand is refused.
func TestStart_RefusesTriggeredTemplateByHand(t *testing.T) {
	h := newHarness(t)

	_, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		TemplateID: h.template.ID,
		UserID:     h.userID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "started by the employment event")
	assert.Empty(t, h.repo.checklists)

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
			{ID: pulid.MustNew("wclti_"), Label: "Review file", Kind: worker.ChecklistItemTask, Required: true, Owner: worker.ChecklistOwnerHR},
		},
	}
	h.repo.templates[manual.ID] = manual

	first, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		TemplateID: manual.ID,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	second, err := h.svc.Start(context.Background(), &workerchecklistservice.StartRequest{
		TenantInfo: h.tenant,
		WorkerID:   h.wrk.ID,
		TemplateID: manual.ID,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, first.ID, second.ID, "a hand-started template never runs twice at once")
}

// A rehire is a new employment, so it gets a new onboarding even though the
// old one is on the record; the same event never spawns twice.
func TestSpawnForEvent_OncePerEvent(t *testing.T) {
	h := newHarness(t)
	hired := &worker.WorkerEmploymentEvent{
		ID:          pulid.MustNew("wee_"),
		Kind:        worker.EmploymentEventHired,
		EffectiveAt: 1_700_000_000,
	}
	first, err := h.svc.SpawnForEvent(context.Background(), hired, h.wrk, h.userID)
	require.NoError(t, err)
	require.NotNil(t, first)

	cancelled, err := h.svc.Cancel(context.Background(), &workerchecklistservice.CancelRequest{
		ID:         first.ID,
		TenantInfo: h.tenant,
		Reason:     "left before starting",
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.ChecklistStatusCancelled, cancelled.Status)

	again, err := h.svc.SpawnForEvent(context.Background(), hired, h.wrk, h.userID)
	require.NoError(t, err)
	assert.Equal(t, first.ID, again.ID, "the same event returns the checklist it already spawned, closed or not")

	rehired := &worker.WorkerEmploymentEvent{
		ID:          pulid.MustNew("wee_"),
		Kind:        worker.EmploymentEventRehired,
		EffectiveAt: 1_800_000_000,
	}
	h.template.Trigger = worker.ChecklistTriggerRehired
	fresh, err := h.svc.SpawnForEvent(context.Background(), rehired, h.wrk, h.userID)
	require.NoError(t, err)
	require.NotNil(t, fresh)
	assert.NotEqual(t, first.ID, fresh.ID, "a new employment gets its own onboarding")
	assert.Equal(t, rehired.ID, fresh.SourceEventID)
	assert.Equal(t, worker.ChecklistStatusOpen, fresh.Status)
}

func TestCloseForEvent_CancelsWhatTheEventMakesMoot(t *testing.T) {
	h := newHarness(t)
	hired := &worker.WorkerEmploymentEvent{
		ID:          pulid.MustNew("wee_"),
		Kind:        worker.EmploymentEventHired,
		EffectiveAt: 1_700_000_000,
	}
	onboarding, err := h.svc.SpawnForEvent(context.Background(), hired, h.wrk, h.userID)
	require.NoError(t, err)
	require.Equal(t, worker.ChecklistStatusOpen, onboarding.Status)

	promoted := &worker.WorkerEmploymentEvent{ID: pulid.MustNew("wee_"), Kind: worker.EmploymentEventPromoted}
	closed, err := h.svc.CloseForEvent(context.Background(), promoted, h.wrk, h.userID)
	require.NoError(t, err)
	assert.Equal(t, 0, closed, "a promotion leaves onboarding running")

	terminated := &worker.WorkerEmploymentEvent{
		ID:          pulid.MustNew("wee_"),
		Kind:        worker.EmploymentEventTerminated,
		EffectiveAt: 1_750_000_000,
	}
	closed, err = h.svc.CloseForEvent(context.Background(), terminated, h.wrk, h.userID)
	require.NoError(t, err)
	assert.Equal(t, 1, closed)
	stored := h.repo.checklists[onboarding.ID]
	assert.Equal(t, worker.ChecklistStatusCancelled, stored.Status)
	assert.Equal(t, "Superseded by terminated event", stored.CancelReason)

	closed, err = h.svc.CloseForEvent(context.Background(), terminated, h.wrk, h.userID)
	require.NoError(t, err)
	assert.Equal(t, 0, closed, "closing is idempotent")
}
