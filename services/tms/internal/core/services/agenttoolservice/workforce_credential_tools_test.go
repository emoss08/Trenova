package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeCredentials struct {
	credentialKeeper

	guard      *writeGuard
	credential *worker.WorkerCredential

	created  *workercredentialservice.CreateRequest
	updated  *worker.WorkerCredential
	attached *workercredentialservice.AttachDocumentRequest
	archived *workercredentialservice.StatusRequest
}

func newFakeCredentials() *fakeCredentials {
	verifiedAt := int64(1_780_000_000)
	expires := int64(1_800_000_000)
	return &fakeCredentials{
		guard: &writeGuard{},
		credential: &worker.WorkerCredential{
			ID:               pulid.MustNew("wcred_"),
			WorkerID:         pulid.MustNew("wrk_"),
			CredentialTypeID: pulid.MustNew("wct_"),
			CredentialType:   &worker.WorkerCredentialType{Name: "Medical card"},
			Status:           worker.CredentialStatusActive,
			Number:           "MC-1001",
			ExpiresAt:        &expires,
			VerifiedAt:       &verifiedAt,
			Version:          4,
		},
	}
}

func (f *fakeCredentials) Get(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) (*worker.WorkerCredential, error) {
	copied := *f.credential
	return &copied, nil
}

func (f *fakeCredentials) PlanCreate(
	_ context.Context,
	req *workercredentialservice.CreateRequest,
) (*workercredentialservice.CreatePlan, error) {
	if req.Entity.CredentialTypeID == f.credential.CredentialTypeID && !req.Renew {
		return nil, errortypes.NewValidationError("credentialTypeId", errortypes.ErrInvalid,
			"This worker already holds an active credential of this type; renew it instead")
	}
	planned := *req.Entity
	planned.CredentialType = f.credential.CredentialType
	plan := &workercredentialservice.CreatePlan{Credential: &planned}
	if req.Renew {
		plan.Superseded = f.credential
	}
	return plan, nil
}

func (f *fakeCredentials) Create(
	_ context.Context,
	req *workercredentialservice.CreateRequest,
) (*worker.WorkerCredential, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.created = req
	return f.credential, nil
}

func (f *fakeCredentials) PlanUpdate(
	_ context.Context,
	entity *worker.WorkerCredential,
) (*workercredentialservice.CredentialChange, error) {
	after := *entity
	if after.Number != f.credential.Number {
		after.VerifiedAt = nil
	}
	return &workercredentialservice.CredentialChange{Before: f.credential, After: &after}, nil
}

func (f *fakeCredentials) Update(
	_ context.Context,
	entity *worker.WorkerCredential,
	_ pulid.ID,
) (*worker.WorkerCredential, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.updated = entity
	return entity, nil
}

func (f *fakeCredentials) PlanAttachDocument(
	_ context.Context,
	req *workercredentialservice.AttachDocumentRequest,
) (*workercredentialservice.CredentialChange, error) {
	after := *f.credential
	after.DocumentID = req.DocumentID
	return &workercredentialservice.CredentialChange{Before: f.credential, After: &after}, nil
}

func (f *fakeCredentials) AttachDocument(
	_ context.Context,
	req *workercredentialservice.AttachDocumentRequest,
) (*worker.WorkerCredential, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.attached = req
	return f.credential, nil
}

func (f *fakeCredentials) PlanArchive(
	context.Context,
	*workercredentialservice.StatusRequest,
) (*workercredentialservice.CredentialChange, error) {
	after := *f.credential
	after.Status = worker.CredentialStatusArchived
	return &workercredentialservice.CredentialChange{Before: f.credential, After: &after}, nil
}

func (f *fakeCredentials) Archive(
	_ context.Context,
	req *workercredentialservice.StatusRequest,
) (*worker.WorkerCredential, error) {
	if err := f.guard.write(); err != nil {
		return nil, err
	}
	f.archived = req
	return f.credential, nil
}

func TestRecordWorkerCredential_RenewalShowsWhatItSupersedes(t *testing.T) {
	t.Parallel()

	credentials := newFakeCredentials()
	tool := newRecordWorkerCredentialTool(credentials)
	raw := map[string]any{
		paramWorkerID:         credentials.credential.WorkerID.String(),
		paramCredentialTypeID: credentials.credential.CredentialTypeID.String(),
		paramCredentialNumber: "MC-2002",
		paramExpiresOn:        "2028-09-30",
	}

	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(raw)),
		"a second active credential of a type is refused without renew")

	raw[paramRenew] = true
	params := executeParams(raw)
	preview := previewWithoutWrites(t, credentials.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	require.Len(t, preview.Changes, 2)
	assert.Contains(t, preview.Summary, "archived as superseded")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, credentials.created)
	assert.True(t, credentials.created.Renew)
	assert.Equal(t, "MC-2002", credentials.created.Entity.Number)
	require.NotNil(t, credentials.created.Entity.ExpiresAt)
}

func TestUpdateWorkerCredential_SaysWhenVerificationClears(t *testing.T) {
	t.Parallel()

	credentials := newFakeCredentials()
	tool := newUpdateWorkerCredentialTool(credentials)
	params := executeParams(map[string]any{
		paramCredentialID:     credentials.credential.ID.String(),
		paramCredentialNumber: "MC-1002",
	})

	preview := previewWithoutWrites(t, credentials.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Contains(t, preview.Summary, "verification is cleared")
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, credentials.updated)
	assert.Equal(t, "MC-1002", credentials.updated.Number)
	assert.Equal(t, credentials.credential.ExpiresAt, credentials.updated.ExpiresAt)
	assert.Nil(t, credentials.updated.CredentialType)
}

func TestAttachWorkerCredentialDocument_NamesTheDocument(t *testing.T) {
	t.Parallel()

	credentials := newFakeCredentials()
	tool := newAttachWorkerCredentialDocumentTool(credentials)
	documentID := pulid.MustNew("doc_")
	require.NoError(t, tool.Execute(t.Context(), executeParams(map[string]any{
		paramCredentialID: credentials.credential.ID.String(),
		wfParamDocument:   documentID.String(),
	})))
	require.NotNil(t, credentials.attached)
	assert.Equal(t, documentID, credentials.attached.DocumentID)
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(),
		executeParams(map[string]any{paramCredentialID: credentials.credential.ID.String()})))
}

func TestArchiveWorkerCredential_IsOnlyProposed(t *testing.T) {
	t.Parallel()

	credentials := newFakeCredentials()
	tool := newArchiveWorkerCredentialTool(credentials)
	params := executeParams(map[string]any{
		paramCredentialID: credentials.credential.ID.String(),
		fieldReason:       "Endorsement surrendered",
	})

	preview := previewWithoutWrites(t, credentials.guard, func() (*agent.ToolPreview, error) {
		return tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	})
	assert.Equal(t, "Would archive Medical card.", preview.Summary)
	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, credentials.archived)
	assert.Equal(t, "Endorsement surrendered", credentials.archived.Reason)
	assert.Equal(t, agent.TierPropose, tool.Policy().MaxTier)
	assert.Equal(t, permission.OpArchive, tool.Policy().Operation)
}
