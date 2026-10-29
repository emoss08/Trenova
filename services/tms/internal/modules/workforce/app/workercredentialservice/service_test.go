package workercredentialservice_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/protectedfield"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/workerportsmock"
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
	types       map[pulid.ID]*workercredential.WorkerCredentialType
	credentials map[pulid.ID]*workercredential.WorkerCredential
	createReqs  []*workercredentialports.CreateWorkerCredentialRequest
	expiring    *workercredentialports.ListExpiringWorkerCredentialsRequest
	ensured     int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		types:       map[pulid.ID]*workercredential.WorkerCredentialType{},
		credentials: map[pulid.ID]*workercredential.WorkerCredential{},
	}
}

func (f *fakeRepo) ListTypes(
	context.Context,
	*workercredentialports.ListCredentialTypesRequest,
) (*pagination.CursorListResult[*workercredential.WorkerCredentialType], error) {
	return &pagination.CursorListResult[*workercredential.WorkerCredentialType]{}, nil
}

func (f *fakeRepo) ListActiveTypes(
	context.Context,
	pagination.TenantInfo,
) ([]*workercredential.WorkerCredentialType, error) {
	out := make([]*workercredential.WorkerCredentialType, 0, len(f.types))
	for _, typ := range f.types {
		if typ.Status == domaintypes.StatusActive {
			out = append(out, typ)
		}
	}
	return out, nil
}

func (f *fakeRepo) TypeSelectOptions(
	_ context.Context,
	_ *workercredentialports.WorkerCredentialTypeSelectOptionsRequest,
) (*pagination.ListResult[*workercredential.WorkerCredentialType], error) {
	out := make([]*workercredential.WorkerCredentialType, 0, len(f.types))
	for _, typ := range f.types {
		if typ.Status == domaintypes.StatusActive {
			out = append(out, typ)
		}
	}
	return &pagination.ListResult[*workercredential.WorkerCredentialType]{
		Items:      out,
		Total:      len(out),
		TotalExact: true,
	}, nil
}

func (f *fakeRepo) GetTypeByID(
	_ context.Context,
	req *workercredentialports.GetCredentialTypeByIDRequest,
) (*workercredential.WorkerCredentialType, error) {
	typ, ok := f.types[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("type")
	}
	return typ, nil
}

func (f *fakeRepo) GetTypesByIDs(
	_ context.Context,
	req *workercredentialports.GetCredentialTypesByIDsRequest,
) ([]*workercredential.WorkerCredentialType, error) {
	out := make([]*workercredential.WorkerCredentialType, 0, len(req.TypeIDs))
	for _, id := range req.TypeIDs {
		if typ, ok := f.types[id]; ok {
			out = append(out, typ)
		}
	}
	return out, nil
}

func (f *fakeRepo) TypeCodeExists(
	context.Context,
	*workercredentialports.CredentialTypeCodeExistsRequest,
) (bool, error) {
	return false, nil
}

func (f *fakeRepo) CreateType(
	_ context.Context,
	entity *workercredential.WorkerCredentialType,
) (*workercredential.WorkerCredentialType, error) {
	entity.ID = pulid.MustNew("wct_")
	f.types[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateType(
	_ context.Context,
	entity *workercredential.WorkerCredentialType,
) (*workercredential.WorkerCredentialType, error) {
	entity.Version++
	f.types[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) EnsureSystemTypes(
	context.Context,
	pagination.TenantInfo,
	[]*workercredential.WorkerCredentialType,
) (int, error) {
	f.ensured++
	return 0, nil
}

func (f *fakeRepo) CountCredentialsByType(
	_ context.Context,
	req *workercredentialports.CountCredentialsByTypeRequest,
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
	req *workercredentialports.CountCredentialsByTypeIDsRequest,
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
	req *workercredentialports.ListWorkerCredentialsRequest,
) ([]*workercredential.WorkerCredential, error) {
	out := make([]*workercredential.WorkerCredential, 0, len(f.credentials))
	for _, cred := range f.credentials {
		if !slices.Contains(req.WorkerIDs, cred.WorkerID) {
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
	slices.SortFunc(out, func(a, b *workercredential.WorkerCredential) int {
		return strings.Compare(a.ID.String(), b.ID.String())
	})
	return out, nil
}

func (f *fakeRepo) GetByID(
	_ context.Context,
	req *workercredentialports.GetWorkerCredentialByIDRequest,
) (*workercredential.WorkerCredential, error) {
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
	req *workercredentialports.CreateWorkerCredentialRequest,
) (*workercredential.WorkerCredential, error) {
	f.createReqs = append(f.createReqs, req)
	if req.SupersedeReason != "" {
		for _, cred := range f.credentials {
			if cred.WorkerID == req.Entity.WorkerID &&
				cred.CredentialTypeID == req.Entity.CredentialTypeID && cred.IsActive() {
				cred.Status = workercredential.CredentialStatusArchived
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
	entity *workercredential.WorkerCredential,
) (*workercredential.WorkerCredential, error) {
	entity.Version++
	f.credentials[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListExpiring(
	_ context.Context,
	req *workercredentialports.ListExpiringWorkerCredentialsRequest,
) ([]*workercredential.WorkerCredential, error) {
	f.expiring = req
	return nil, nil
}

type harness struct {
	svc        *workercredentialservice.Service
	repo       *fakeRepo
	workerRepo *workerportsmock.MockWorkerRepository
	docRepo    *documentportsmock.MockDocumentRepository
	tenant     pagination.TenantInfo
	wrk        *worker.Worker
	cdl        *workercredential.WorkerCredentialType
	medCard    *workercredential.WorkerCredentialType
	forklift   *workercredential.WorkerCredentialType
	userID     pulid.ID
}

func (h *harness) actor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    h.userID,
		UserID:         h.userID,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
	}
}

type fixedFieldAccess protectedfield.GroupSet

func (f fixedFieldAccess) Visible(
	context.Context,
	*permissionports.RequestActor,
) protectedfield.GroupSet {
	return protectedfield.GroupSet(f)
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	return newHarnessSeeing(t, nil)
}

func newHarnessSeeing(t *testing.T, fields permissionports.FieldAccess) *harness {
	t.Helper()
	repo := newFakeRepo()
	workerRepo := workerportsmock.NewMockWorkerRepository(t)
	docRepo := documentportsmock.NewMockDocumentRepository(t)
	audit := auditportsmock.NewMockAuditService(t)
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

	addType := func(code string, required, requiresDoc bool, field workercredential.CredentialProfileField) *workercredential.WorkerCredentialType {
		typ := &workercredential.WorkerCredentialType{
			ID:                pulid.MustNew("wct_"),
			OrganizationID:    tenant.OrgID,
			BusinessUnitID:    tenant.BuID,
			Code:              code,
			Name:              code,
			Category:          workercredential.CredentialCategoryOther,
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
		Fields:       fields,
	})

	return &harness{
		svc:        svc,
		repo:       repo,
		workerRepo: workerRepo,
		docRepo:    docRepo,
		tenant:     tenant,
		wrk:        wrk,
		cdl: addType(
			"CDL",
			true,
			true,
			workercredential.CredentialProfileFieldLicenseExpiry,
		),
		medCard: addType(
			"MED_CARD",
			true,
			true,
			workercredential.CredentialProfileFieldMedicalCardExpiry,
		),
		forklift: addType("FORKLIFT", false, false, workercredential.CredentialProfileFieldNone),
		userID:   pulid.MustNew("usr_"),
	}
}

func (h *harness) credential(
	typ *workercredential.WorkerCredentialType,
	expiry int64,
) *workercredential.WorkerCredential {
	return &workercredential.WorkerCredential{
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

	var patched *workerports.PatchProfileCredentialFieldRequest
	h.workerRepo.EXPECT().
		PatchProfileCredentialField(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *workerports.PatchProfileCredentialFieldRequest) error {
			patched = req
			return nil
		}).
		Once()
	var compliance *workerports.UpdateProfileComplianceStatusRequest
	h.workerRepo.EXPECT().
		UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, req *workerports.UpdateProfileComplianceStatusRequest) error {
			compliance = req
			return nil
		}).
		Once()

	created, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.medCard, expiry),
		UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, workercredential.CredentialStatusActive, created.Status)

	require.NotNil(t, patched)
	assert.Equal(t, workercredential.CredentialProfileFieldMedicalCardExpiry, patched.Field)
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
	assert.Equal(t, workercredential.CredentialStatusArchived, h.repo.credentials[first.ID].Status)
	assert.Equal(t, workercredential.CredentialStatusActive, h.repo.credentials[second.ID].Status)
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
	updated, err := h.svc.Update(t.Context(), &notesOnly, h.actor())
	require.NoError(t, err)
	assert.True(t, updated.IsVerified(), "notes do not invalidate a verification")

	newExpiry := int64(1_960_000_000)
	moved := *updated
	moved.ExpiresAt = &newExpiry
	updated, err = h.svc.Update(t.Context(), &moved, h.actor())
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

	byType := func(typ *workercredential.WorkerCredentialType) *workercredential.WorkerCredential {
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
	assert.Equal(t, int64(7), workercredential.DaysUntil(now+7*day, now))
}
