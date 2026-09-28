package workercredentialservice_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/workercredentialservice"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestPlanCreate_NamesWhatARenewalArchives(t *testing.T) {
	h := newHarness(t)
	h.workerRepo.EXPECT().PatchProfileCredentialField(mock.Anything, mock.Anything).
		Return(nil).Maybe()
	h.workerRepo.EXPECT().UpdateProfileComplianceStatus(mock.Anything, mock.Anything).
		Return(nil).Maybe()

	planned, err := h.svc.PlanCreate(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.CredentialStatusActive, planned.Credential.Status)
	assert.Equal(t, "FORKLIFT", planned.Credential.CredentialType.Code)
	assert.Nil(t, planned.Superseded)
	assert.Empty(t, h.repo.createReqs)

	first, err := h.svc.Create(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_900_000_000),
		UserID: h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.PlanCreate(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		UserID: h.userID,
	})
	require.Error(t, err, "a second active credential needs renew")

	renewal, err := h.svc.PlanCreate(t.Context(), &workercredentialservice.CreateRequest{
		Entity: h.credential(h.forklift, 1_950_000_000),
		Renew:  true,
		UserID: h.userID,
	})
	require.NoError(t, err)
	require.NotNil(t, renewal.Superseded)
	assert.Equal(t, first.ID, renewal.Superseded.ID)
	assert.Len(t, h.repo.createReqs, 1)

	edit := *first
	edit.Number = "N-2"
	change, err := h.svc.PlanUpdate(t.Context(), &edit)
	require.NoError(t, err)
	assert.Equal(t, "N-2", change.After.Number)
	assert.Equal(t, "N-1", h.repo.credentials[first.ID].Number)

	archive, err := h.svc.PlanArchive(t.Context(), &workercredentialservice.StatusRequest{
		ID: first.ID, TenantInfo: h.tenant, Reason: "Left the company", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, worker.CredentialStatusArchived, archive.After.Status)
	assert.Equal(t, worker.CredentialStatusActive, h.repo.credentials[first.ID].Status)
}
