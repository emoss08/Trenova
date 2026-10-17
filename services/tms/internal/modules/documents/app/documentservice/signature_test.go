package documentservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/documentportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/permissionportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type signatureHarness struct {
	svc         *Service
	repo        *documentportsmock.MockDocumentRepository
	permissions *permissionportsmock.MockPermissionEngine
	audit       *auditportsmock.MockAuditService
	doc         *document.Document
	actor       permissionports.RequestActor
	tenant      pagination.TenantInfo
}

func newSignatureHarness(t *testing.T) *signatureHarness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	repo := documentportsmock.NewMockDocumentRepository(t)
	permissions := permissionportsmock.NewMockPermissionEngine(t)
	audit := auditportsmock.NewMockAuditService(t)

	return &signatureHarness{
		svc: &Service{
			l:            zap.NewNop(),
			repo:         repo,
			permissions:  permissions,
			auditService: audit,
		},
		repo:        repo,
		permissions: permissions,
		audit:       audit,
		doc: &document.Document{
			ID:                pulid.MustNew("doc_"),
			OrganizationID:    orgID,
			BusinessUnitID:    buID,
			LineageID:         pulid.MustNew("doc_"),
			VersionNumber:     1,
			IsCurrentVersion:  true,
			FileName:          "pod.pdf",
			OriginalName:      "pod.pdf",
			FileSize:          2048,
			FileType:          "application/pdf",
			StoragePath:       "org/bu/pod.pdf",
			CryptoMode:        "envelope_v1",
			CryptoVersion:     1,
			Status:            document.StatusActive,
			ResourceID:        pulid.MustNew("shp_").String(),
			ResourceType:      "shipment",
			ProcessingProfile: document.ProcessingProfileNone,
			UploadedByID:      userID,
			SignatureStatus:   document.SignatureStatusUnknown,
			Version:           4,
		},
		actor: permissionports.RequestActor{
			PrincipalType:  permissionports.PrincipalTypeUser,
			PrincipalID:    userID,
			UserID:         userID,
			OrganizationID: orgID,
			BusinessUnitID: buID,
		},
		tenant: pagination.TenantInfo{OrgID: orgID, BuID: buID, UserID: userID},
	}
}

func (h *signatureHarness) allow(allowed bool) {
	h.permissions.EXPECT().
		Check(mock.Anything, mock.MatchedBy(func(req *permissionports.PermissionCheckRequest) bool {
			return req.Resource == permission.ResourceDocument.String() &&
				req.Operation == permission.OpUpdate &&
				req.UserID == h.actor.UserID &&
				req.OrganizationID == h.tenant.OrgID &&
				req.ResourceID != nil && *req.ResourceID == h.doc.ID
		})).
		Return(&permissionports.PermissionCheckResult{Allowed: allowed}, nil)
}

func (h *signatureHarness) load() {
	h.repo.EXPECT().
		GetByID(mock.Anything, documentports.GetDocumentByIDRequest{
			ID:         h.doc.ID,
			TenantInfo: h.tenant,
		}).
		Return(h.doc, nil)
}

func (h *signatureHarness) request(status document.SignatureStatus, signedAt *int64) *SetSignatureRequest {
	return &SetSignatureRequest{
		DocumentID: h.doc.ID,
		TenantInfo: h.tenant,
		Status:     status,
		SignedAt:   signedAt,
		Version:    h.doc.Version,
		Actor:      h.actor,
	}
}

func TestSetSignatureRecordsAPersonsAnswerAndAuditsIt(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.allow(true)
	h.load()
	signedAt := time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC).Unix()

	h.repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(doc *document.Document) bool {
			return doc.ID == h.doc.ID &&
				doc.Version == h.doc.Version &&
				doc.SignatureStatus == document.SignatureStatusSigned &&
				doc.SignedAt != nil && *doc.SignedAt == signedAt &&
				doc.SignatureSource == document.SignatureSourcePerson &&
				doc.SignatureUpdatedByID == h.actor.UserID &&
				doc.SignatureUpdatedAt != nil
		})).
		RunAndReturn(func(_ context.Context, doc *document.Document) (*document.Document, error) {
			saved := *doc
			saved.Version++
			return &saved, nil
		})
	h.audit.EXPECT().
		LogAction(mock.MatchedBy(func(params *auditports.LogActionParams) bool {
			return params.Resource == permission.ResourceDocument &&
				params.ResourceID == h.doc.ID.String() &&
				params.Operation == permission.OpUpdate &&
				params.UserID == h.actor.UserID &&
				params.OrganizationID == h.doc.OrganizationID
		}), mock.Anything).
		Return(nil)

	updated, err := h.svc.SetSignature(
		t.Context(),
		h.request(document.SignatureStatusSigned, &signedAt),
	)

	require.NoError(t, err)
	assert.Equal(t, document.SignatureStatusSigned, updated.SignatureStatus)
	assert.Equal(t, document.SignatureSourcePerson, updated.SignatureSource)
	assert.Equal(t, h.doc.Version+1, updated.Version)
	assert.Equal(t, document.SignatureStatusUnknown, h.doc.SignatureStatus)
}

func TestSetSignatureRefusesAPersonWithoutUpdatePermission(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.allow(false)

	_, err := h.svc.SetSignature(t.Context(), h.request(document.SignatureStatusUnsigned, nil))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	h.repo.AssertNotCalled(t, "GetByID", mock.Anything, mock.Anything)
	h.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestSetSignatureRefusesWhenPermissionsCannotBeChecked(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.svc.permissions = nil

	_, err := h.svc.SetSignature(t.Context(), h.request(document.SignatureStatusUnsigned, nil))

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
}

func TestSetSignatureRefusesAnAgent(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	req := h.request(document.SignatureStatusSigned, nil)
	req.Actor = permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeAgent,
		PrincipalID:   pulid.MustNew("agdef_"),
		UserID:        h.actor.UserID,
	}

	_, err := h.svc.SetSignature(t.Context(), req)

	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))
	h.permissions.AssertNotCalled(t, "Check", mock.Anything, mock.Anything)
}

func TestSetSignatureRefusesAStaleVersion(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.allow(true)
	h.load()
	req := h.request(document.SignatureStatusUnsigned, nil)
	req.Version = h.doc.Version - 1

	_, err := h.svc.SetSignature(t.Context(), req)

	require.Error(t, err)
	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
	assert.Equal(t, errortypes.ErrVersionMismatch, validation.Code)
	assert.Equal(t, "version", validation.Field)
	h.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	h.audit.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestSetSignatureSurfacesARaceTheRepositoryCaught(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.allow(true)
	h.load()
	conflict := errortypes.NewValidationError(
		"version",
		errortypes.ErrVersionMismatch,
		"Document was changed by someone else",
	)
	h.repo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil, conflict)

	_, err := h.svc.SetSignature(t.Context(), h.request(document.SignatureStatusUnsigned, nil))

	require.ErrorIs(t, err, conflict)
	h.audit.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestSetSignatureRefusesADateOnAnUnsignedDocument(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.allow(true)
	h.load()
	signedAt := time.Date(2026, time.October, 14, 12, 0, 0, 0, time.UTC).Unix()

	_, err := h.svc.SetSignature(
		t.Context(),
		h.request(document.SignatureStatusUnsigned, &signedAt),
	)

	require.Error(t, err)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	h.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

func TestSetSignatureLeavesAnUnchangedAnswerAlone(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.doc.SignatureStatus = document.SignatureStatusUnsigned
	h.doc.SignatureSource = document.SignatureSourcePerson
	h.doc.SignatureUpdatedByID = h.actor.UserID
	h.allow(true)
	h.load()

	got, err := h.svc.SetSignature(t.Context(), h.request(document.SignatureStatusUnsigned, nil))

	require.NoError(t, err)
	assert.Same(t, h.doc, got)
	h.repo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
	h.audit.AssertNotCalled(t, "LogAction", mock.Anything, mock.Anything)
}

func TestSetSignatureOverridesWhatExtractionRead(t *testing.T) {
	t.Parallel()

	h := newSignatureHarness(t)
	h.doc.SignatureStatus = document.SignatureStatusSigned
	h.doc.SignatureSource = document.SignatureSourceExtraction
	h.allow(true)
	h.load()
	h.repo.EXPECT().
		Update(mock.Anything, mock.MatchedBy(func(doc *document.Document) bool {
			return doc.SignatureStatus == document.SignatureStatusUnsigned &&
				doc.SignedAt == nil &&
				doc.SignatureSource == document.SignatureSourcePerson
		})).
		RunAndReturn(func(_ context.Context, doc *document.Document) (*document.Document, error) {
			return doc, nil
		})
	h.audit.EXPECT().LogAction(mock.Anything, mock.Anything).Return(nil)

	updated, err := h.svc.SetSignature(
		t.Context(),
		h.request(document.SignatureStatusUnsigned, nil),
	)

	require.NoError(t, err)
	assert.True(t, updated.SignatureSetByPerson())
}
