package workercredentialservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
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

// fakeRepo is an in-memory WorkerCredentialRepository: enough state to drive
// the service's branching without a database.
type fakeRepo struct {
	types       map[pulid.ID]*worker.WorkerCredentialType
	credentials map[pulid.ID]*worker.WorkerCredential
	createReqs  []*repositories.CreateWorkerCredentialRequest
	ensured     int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		types:       map[pulid.ID]*worker.WorkerCredentialType{},
		credentials: map[pulid.ID]*worker.WorkerCredential{},
	}
}

func (f *fakeRepo) ListTypes(
	context.Context,
	*repositories.ListCredentialTypesRequest,
) (*pagination.CursorListResult[*worker.WorkerCredentialType], error) {
	return &pagination.CursorListResult[*worker.WorkerCredentialType]{}, nil
}

func (f *fakeRepo) ListActiveTypes(
	context.Context,
	pagination.TenantInfo,
) ([]*worker.WorkerCredentialType, error) {
	out := make([]*worker.WorkerCredentialType, 0, len(f.types))
	for _, typ := range f.types {
		if typ.Status == domaintypes.StatusActive {
			out = append(out, typ)
		}
	}
	return out, nil
}

func (f *fakeRepo) TypeSelectOptions(
	_ context.Context,
	_ *repositories.WorkerCredentialTypeSelectOptionsRequest,
) (*pagination.ListResult[*worker.WorkerCredentialType], error) {
	out := make([]*worker.WorkerCredentialType, 0, len(f.types))
	for _, typ := range f.types {
		if typ.Status == domaintypes.StatusActive {
			out = append(out, typ)
		}
	}
	return &pagination.ListResult[*worker.WorkerCredentialType]{Items: out, Total: len(out)}, nil
}

func (f *fakeRepo) GetTypeByID(
	_ context.Context,
	req *repositories.GetCredentialTypeByIDRequest,
) (*worker.WorkerCredentialType, error) {
	typ, ok := f.types[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("type")
	}
	return typ, nil
}

func (f *fakeRepo) TypeCodeExists(
	context.Context,
	*repositories.CredentialTypeCodeExistsRequest,
) (bool, error) {
	return false, nil
}

func (f *fakeRepo) CreateType(
	_ context.Context,
	entity *worker.WorkerCredentialType,
) (*worker.WorkerCredentialType, error) {
	entity.ID = pulid.MustNew("wct_")
	f.types[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateType(
	_ context.Context,
	entity *worker.WorkerCredentialType,
) (*worker.WorkerCredentialType, error) {
	entity.Version++
	f.types[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) EnsureSystemTypes(
	context.Context,
	pagination.TenantInfo,
	[]*worker.WorkerCredentialType,
) (int, error) {
	f.ensured++
	return 0, nil
}

func (f *fakeRepo) CountCredentialsByType(
	_ context.Context,
	req *repositories.CountCredentialsByTypeRequest,
) (int, error) {
	count := 0
	for _, cred := range f.credentials {
		if cred.CredentialTypeID == req.TypeID && (!req.ActiveOnly || cred.IsActive()) {
			count++
		}
	}
	return count, nil
}

func (f *fakeRepo) CountCredentialsByTypeIDs(
	_ context.Context,
	req *repositories.CountCredentialsByTypeIDsRequest,
) (map[pulid.ID]int, error) {
	counts := make(map[pulid.ID]int, len(req.TypeIDs))
	for _, typeID := range req.TypeIDs {
		for _, cred := range f.credentials {
			if cred.CredentialTypeID == typeID && (!req.ActiveOnly || cred.IsActive()) {
				counts[typeID]++
			}
		}
	}
	return counts, nil
}

func (f *fakeRepo) ListForWorker(
	_ context.Context,
	req *repositories.ListWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	out := make([]*worker.WorkerCredential, 0, len(f.credentials))
	for _, cred := range f.credentials {
		if cred.WorkerID != req.WorkerID {
			continue
		}
		if !req.IncludeArchived && !cred.IsActive() {
			continue
		}
		if req.IncludeType {
			cred.CredentialType = f.types[cred.CredentialTypeID]
		}
		out = append(out, cred)
	}
	return out, nil
}

func (f *fakeRepo) GetByID(
	_ context.Context,
	req *repositories.GetWorkerCredentialByIDRequest,
) (*worker.WorkerCredential, error) {
	cred, ok := f.credentials[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("credential")
	}
	copied := *cred
	if req.IncludeType {
		copied.CredentialType = f.types[cred.CredentialTypeID]
	}
	return &copied, nil
}

func (f *fakeRepo) Create(
	_ context.Context,
	req *repositories.CreateWorkerCredentialRequest,
) (*worker.WorkerCredential, error) {
	f.createReqs = append(f.createReqs, req)
	if req.SupersedeReason != "" {
		for _, cred := range f.credentials {
			if cred.WorkerID == req.Entity.WorkerID &&
				cred.CredentialTypeID == req.Entity.CredentialTypeID && cred.IsActive() {
				cred.Status = worker.CredentialStatusArchived
				cred.ArchiveReason = req.SupersedeReason
			}
		}
	}
	for _, cred := range f.credentials {
		if cred.WorkerID == req.Entity.WorkerID &&
			cred.CredentialTypeID == req.Entity.CredentialTypeID && cred.IsActive() {
			return nil, errortypes.NewValidationError(
				"credentialTypeId",
				errortypes.ErrDuplicate,
				"duplicate",
			)
		}
	}
	req.Entity.ID = pulid.MustNew("wcred_")
	f.credentials[req.Entity.ID] = req.Entity
	return req.Entity, nil
}

func (f *fakeRepo) Update(
	_ context.Context,
	entity *worker.WorkerCredential,
) (*worker.WorkerCredential, error) {
	entity.Version++
	f.credentials[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListExpiring(
	context.Context,
	*repositories.ListExpiringWorkerCredentialsRequest,
) ([]*worker.WorkerCredential, error) {
	return nil, nil
}

type harness struct {
	svc        *workercredentialservice.Service
	repo       *fakeRepo
	workerRepo *mocks.MockWorkerRepository
	docRepo    *mocks.MockDocumentRepository
	tenant     pagination.TenantInfo
	wrk        *worker.Worker
	cdl        *worker.WorkerCredentialType
	medCard    *worker.WorkerCredentialType
	forklift   *worker.WorkerCredentialType
	userID     pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	repo := newFakeRepo()
	workerRepo := mocks.NewMockWorkerRepository(t)
	docRepo := mocks.NewMockDocumentRepository(t)
	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil).Maybe()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	wrk := &worker.Worker{
		ID:             pulid.MustNew("wrk_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		DriverType:     worker.DriverTypeOTR,
		Profile: &worker.WorkerProfile{
			LicenseNumber:    "D123",
			LicenseExpiry:    1_900_000_000,
			ComplianceStatus: worker.ComplianceStatusPending,
		},
	}
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(wrk, nil).
		Maybe()

	addType := func(code string, required, requiresDoc bool, field worker.CredentialProfileField) *worker.WorkerCredentialType {
		typ := &worker.WorkerCredentialType{
			ID:                pulid.MustNew("wct_"),
			OrganizationID:    tenant.OrgID,
			BusinessUnitID:    tenant.BuID,
			Code:              code,
			Name:              code,
			Category:          worker.CredentialCategoryOther,
			Status:            domaintypes.StatusActive,
			IsRequired:        required,
			RequiresDocument:  requiresDoc,
			RenewalWindowDays: 30,
			ProfileField:      field,
			IsSystem:          field.IsSet(),
		}
		repo.types[typ.ID] = typ
		return typ
	}

	svc := workercredentialservice.New(workercredentialservice.Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		WorkerRepo:   workerRepo,
		DocumentRepo: docRepo,
		AuditService: audit,
	})

	return &harness{
		svc:        svc,
		repo:       repo,
		workerRepo: workerRepo,
		docRepo:    docRepo,
		tenant:     tenant,
		wrk:        wrk,
		cdl:        addType("CDL", true, true, worker.CredentialProfileFieldLicenseExpiry),
		medCard:    addType("MED_CARD", true, true, worker.CredentialProfileFieldMedicalCardExpiry),
		forklift:   addType("FORKLIFT", false, false, worker.CredentialProfileFieldNone),
		userID:     pulid.MustNew("usr_"),
	}
}

func (h *harness) credential(
	typ *worker.WorkerCredentialType,
	expiry int64,
) *worker.WorkerCredential {
	return &worker.WorkerCredential{
		OrganizationID:   h.tenant.OrgID,
		BusinessUnitID:   h.tenant.BuID,
		WorkerID:         h.wrk.ID,
		CredentialTypeID: typ.ID,
		Number:           "N-1",
		ExpiresAt:        &expiry,
	}
}

func TestCreate_MirrorsProfileAndRefreshesCompliance(t *testing.T) {
	h := newHarness(t)
	expiry := int64(1_950_000_000)

	var patched *repositories.PatchProfileCredentialFieldRequest
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.PatchProfileCredentialFieldRequest) error {
			patched = req
			return nil
		}).
		Once()
	var compliance *repositories.UpdateProfileComplianceStatusRequest
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *repositories.UpdateProfileComplianceStatusRequest) error {
			compliance = req
			return nil
		}).
		Once()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.medCard, expiry),
		UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.CredentialStatusActive, created.Status)

	require.NotNil(t, patched)
	assert.Equal(t, worker.CredentialProfileFieldMedicalCardExpiry, patched.Field)
	assert.Equal(t, expiry, *patched.ExpiresAt)
	assert.Empty(t, patched.Number, "only the CDL carries its number onto the profile")

	require.NotNil(t, compliance)
	assert.Equal(t, worker.ComplianceStatusNonCompliant, compliance.Status,
		"the CDL slot is still missing so the worker is not compliant")
}

func TestCreate_RenewSupersedesActiveCredential(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	first, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.Error(t, err, "a second active credential of the same type is refused without renew")

	second, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		Renew:  true,
		UserID: h.userID,
	})
	require.NoError(t, err)

	require.Len(t, h.repo.createReqs, 3)
	assert.Equal(t, "Superseded by renewal", h.repo.createReqs[2].SupersedeReason)
	assert.Equal(t, h.userID, h.repo.createReqs[2].SupersededByID)
	assert.Equal(t, worker.CredentialStatusArchived, h.repo.credentials[first.ID].Status)
	assert.Equal(t, worker.CredentialStatusActive, h.repo.credentials[second.ID].Status)
}

func TestCreate_TypeRequirements(t *testing.T) {
	h := newHarness(t)
	h.cdl.RequiresNumber = true

	entity := h.credential(h.cdl, 0)
	entity.Number = ""
	entity.ExpiresAt = nil

	_, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: entity,
		UserID: h.userID,
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"number", "expiresAt"}, fields)
}

func TestVerify_RequiresDocumentWhenTypeDemandsIt(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.medCard, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.Verify(t.Context(), &workercredentialservice.StatusRequest{
		ID:         created.ID,
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "documentId", verr.Field)

	docID := pulid.MustNew("doc_")
	h.docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&document.Document{
			ID:           docID,
			ResourceType: "worker",
			ResourceID:   h.wrk.ID.String(),
		}, nil).
		Once()
	attached, err := h.svc.AttachDocument(
		t.Context(),
		&workercredentialservice.AttachDocumentRequest{
			ID:         created.ID,
			DocumentID: docID,
			TenantInfo: h.tenant,
			UserID:     h.userID,
		},
	)
	require.NoError(t, err)
	assert.Equal(t, docID, attached.DocumentID)

	verified, err := h.svc.Verify(t.Context(), &workercredentialservice.StatusRequest{
		ID:         created.ID,
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.NoError(t, err)
	assert.True(t, verified.IsVerified())
	assert.Equal(t, h.userID, verified.VerifiedByID)
}

func TestAttachDocument_RejectsForeignDocument(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	h.docRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&document.Document{
			ID:           pulid.MustNew("doc_"),
			ResourceType: "worker",
			ResourceID:   pulid.MustNew("wrk_").String(),
		}, nil).
		Once()
	_, err = h.svc.AttachDocument(t.Context(), &workercredentialservice.AttachDocumentRequest{
		ID:         created.ID,
		DocumentID: pulid.MustNew("doc_"),
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "documentId", verr.Field)
}

func TestUpdate_ClearsVerificationWhenFactsChange(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)
	verified, err := h.svc.Verify(t.Context(), &workercredentialservice.StatusRequest{
		ID: created.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	require.NoError(t, err)
	require.True(t, verified.IsVerified())

	notesOnly := *verified
	notesOnly.Notes = "renewed at the depot"
	updated, err := h.svc.Update(t.Context(), &notesOnly, h.userID)
	require.NoError(t, err)
	assert.True(t, updated.IsVerified(), "notes do not invalidate a verification")

	newExpiry := int64(1_960_000_000)
	moved := *updated
	moved.ExpiresAt = &newExpiry
	updated, err = h.svc.Update(t.Context(), &moved, h.userID)
	require.NoError(t, err)
	assert.False(t, updated.IsVerified(), "a new expiry needs a fresh verification")
}

func TestSyncFromProfile(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	medExpiry := int64(1_940_000_000)
	h.wrk.Profile.MedicalCardExpiry = &medExpiry

	require.NoError(t, h.svc.SyncFromProfile(t.Context(), h.wrk, h.userID))

	byType := func(typ *worker.WorkerCredentialType) *worker.WorkerCredential {
		for _, cred := range h.repo.credentials {
			if cred.CredentialTypeID == typ.ID && cred.IsActive() {
				return cred
			}
		}
		return nil
	}
	cdl := byType(h.cdl)
	require.NotNil(t, cdl, "the licence column creates a CDL credential")
	assert.Equal(t, "D123", cdl.Number)
	assert.Equal(t, int64(1_900_000_000), *cdl.ExpiresAt)
	med := byType(h.medCard)
	require.NotNil(t, med)
	assert.Equal(t, medExpiry, *med.ExpiresAt)
	assert.Nil(t, byType(h.forklift), "non-mirrored types are untouched")

	h.wrk.Profile.LicenseExpiry = 1_910_000_000
	h.wrk.Profile.MedicalCardExpiry = nil
	require.NoError(t, h.svc.SyncFromProfile(t.Context(), h.wrk, h.userID))

	assert.Equal(
		t,
		int64(1_910_000_000),
		*byType(h.cdl).ExpiresAt,
		"changed expiry updates in place",
	)
	assert.Nil(t, byType(h.medCard), "a cleared column archives the mirrored credential")
	assert.Equal(t, "Cleared on worker profile", h.repo.credentials[med.ID].ArchiveReason)

	require.NoError(t, h.svc.SyncFromProfile(t.Context(), h.wrk, h.userID))
	assert.Len(t, h.repo.createReqs, 2, "an unchanged profile creates nothing")
}

func TestArchiveType_GuardsMirroredAndHeldTypes(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	_, err := h.svc.ArchiveType(t.Context(), &workercredentialservice.TypeStatusChangeRequest{
		ID: h.medCard.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	var verr *errortypes.Error
	require.ErrorAs(t, err, &verr)
	assert.Equal(t, "status", verr.Field)

	_, err = h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)
	_, err = h.svc.ArchiveType(t.Context(), &workercredentialservice.TypeStatusChangeRequest{
		ID: h.forklift.ID, TenantInfo: h.tenant, UserID: h.userID,
	})
	require.ErrorAs(t, err, &verr)
	assert.Contains(t, verr.Error(), "still hold")
}

func TestSummary_SeedsSystemTypesOnce(t *testing.T) {
	h := newHarness(t)

	_, err := h.svc.Summary(t.Context(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	_, err = h.svc.Summary(t.Context(), h.tenant, h.wrk.ID)
	require.NoError(t, err)
	assert.Equal(t, 1, h.repo.ensured)
}

func TestDaysUntilAgreesWithForecastWindow(t *testing.T) {
	now := int64(1_800_000_000)
	assert.Equal(t, int64(7), worker.DaysUntil(now+7*day, now))
}
