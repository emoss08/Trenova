package rateconfirmationservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const invalidLinkCopy = "This rate confirmation link is no longer valid"

type publicTestDeps struct {
	repo                  *fakeRateConRepo
	carrierAssignmentRepo *mocks.MockCarrierAssignmentRepository
	svc                   *Service
}

func setupPublicTest(t *testing.T) *publicTestDeps {
	t.Helper()

	deps := &publicTestDeps{
		repo:                  newFakeRateConRepo(),
		carrierAssignmentRepo: mocks.NewMockCarrierAssignmentRepository(t),
	}
	deps.svc = &Service{
		l:                     zap.NewNop(),
		db:                    testDBConnection{},
		repo:                  deps.repo,
		carrierAssignmentRepo: deps.carrierAssignmentRepo,
	}
	return deps
}

func signableRateCon(deps *publicTestDeps) *rateconfirmation.RateConfirmation {
	entity := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		OrganizationID:      pulid.MustNew("org_"),
		BusinessUnitID:      pulid.MustNew("bu_"),
		CarrierAssignmentID: pulid.MustNew("cass_"),
		CarrierID:           pulid.MustNew("car_"),
		ShipmentID:          pulid.MustNew("shp_"),
		ShipmentMoveID:      pulid.MustNew("smv_"),
		Status:              rateconfirmation.StatusSent,
		PayloadSnapshot: map[string]any{
			"CarrierName":       "Eastline Transport",
			"ShipmentProNumber": "PRO-1001",
			"RevisionLabel":     "Revision 1",
			"RateMethodLabel":   "Flat",
			"TotalCost":         "1850.00",
			"Currency":          "USD",
			"Stops": []map[string]any{
				{
					"Sequence": 1,
					"Type":     "Pickup",
					"Name":     "Lakeside Distribution",
					"Address":  "2100 Harbor Dr, Green Bay, WI",
					"Window":   "Mon 08:00-12:00",
				},
			},
		},
	}
	deps.repo.add(entity)
	return entity
}

func mintTestToken(
	deps *publicTestDeps,
	entity *rateconfirmation.RateConfirmation,
) (string, *rateconfirmation.RateConfirmationToken) {
	raw := pulid.MustNew("raw_").String()
	token := &rateconfirmation.RateConfirmationToken{
		ID:                 pulid.MustNew("rct_"),
		OrganizationID:     entity.OrganizationID,
		BusinessUnitID:     entity.BusinessUnitID,
		RateConfirmationID: entity.ID,
		TokenHash:          tokenutils.Hash(raw),
		Email:              "dispatch@eastline.test",
		ExpiresAt:          timeutils.NowUnix() + 3600,
	}
	deps.repo.tokensByHash[token.TokenHash] = token
	return raw, token
}

func TestPreviewByTokenServesFrozenSnapshot(t *testing.T) {
	deps := setupPublicTest(t)
	entity := signableRateCon(deps)
	raw, token := mintTestToken(deps, entity)

	view, err := deps.svc.PreviewByToken(t.Context(), raw)

	require.NoError(t, err)
	assert.Equal(t, "Eastline Transport", view.CarrierName)
	assert.Equal(t, "PRO-1001", view.ShipmentProNumber)
	assert.Equal(t, "Revision 1", view.RevisionLabel)
	assert.Equal(t, "1850.00", view.TotalCost)
	assert.Equal(t, token.ExpiresAt, view.ExpiresAt)
	assert.False(t, view.Confirmed)
	require.Len(t, view.Stops, 1)
	assert.Equal(t, "Lakeside Distribution", view.Stops[0].Name)

	assert.Empty(t, deps.repo.updated, "a GET preview must never mutate")
	assert.Empty(t, deps.repo.markedUsed, "a GET preview must never burn the token")
}

func TestPublicTokenFailureModesShareOneCopy(t *testing.T) {
	deps := setupPublicTest(t)
	entity := signableRateCon(deps)

	expiredRaw, expired := mintTestToken(deps, entity)
	expired.ExpiresAt = timeutils.NowUnix() - 60

	usedRaw, used := mintTestToken(deps, entity)
	usedAt := timeutils.NowUnix()
	used.UsedAt = &usedAt

	revokedRaw, revoked := mintTestToken(deps, entity)
	revoked.RevokedAt = &usedAt

	voidedEntity := signableRateCon(deps)
	voidedEntity.Status = rateconfirmation.StatusVoided
	voidedRaw, _ := mintTestToken(deps, voidedEntity)

	for name, raw := range map[string]string{
		"unknown": "does-not-exist",
		"empty":   "",
		"expired": expiredRaw,
		"used":    usedRaw,
		"revoked": revokedRaw,
		"voided":  voidedRaw,
	} {
		_, err := deps.svc.PreviewByToken(t.Context(), raw)
		require.Error(t, err, name)
		assert.ErrorContains(t, err, invalidLinkCopy, name)

		err = deps.svc.ConfirmByToken(t.Context(), raw, "J. Doe", "Dispatcher")
		require.Error(t, err, name)
		assert.ErrorContains(t, err, invalidLinkCopy, name)
	}
}

func TestConfirmByTokenRequiresSignerName(t *testing.T) {
	deps := setupPublicTest(t)
	entity := signableRateCon(deps)
	raw, _ := mintTestToken(deps, entity)

	err := deps.svc.ConfirmByToken(t.Context(), raw, "", "Dispatcher")

	require.Error(t, err)
	assert.ErrorContains(t, err, "signer's name is required")
	assert.Empty(t, deps.repo.updated)
	assert.Empty(t, deps.repo.markedUsed)
}

func TestConfirmByTokenConfirmsThenBurns(t *testing.T) {
	deps := setupPublicTest(t)
	entity := signableRateCon(deps)
	raw, token := mintTestToken(deps, entity)

	assignment := &shipment.CarrierAssignment{
		ID:        entity.CarrierAssignmentID,
		CarrierID: entity.CarrierID,
		Status:    shipment.CarrierAssignmentStatusPending,
	}
	deps.carrierAssignmentRepo.On("GetByID", mock.Anything, mock.Anything).
		Return(assignment, nil)
	deps.carrierAssignmentRepo.On("Update", mock.Anything, assignment).
		Return(assignment, nil)

	err := deps.svc.ConfirmByToken(t.Context(), raw, "J. Doe", "Operations Manager")

	require.NoError(t, err)
	assert.Equal(t, rateconfirmation.StatusConfirmed, entity.Status)
	assert.Equal(t, rateconfirmation.ViaPublicSignature, entity.ConfirmedVia)
	assert.Equal(t, "J. Doe", entity.ConfirmedByName)
	assert.Equal(t, "Operations Manager", entity.ConfirmedByTitle)
	require.NotNil(t, entity.ConfirmedAt)
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, assignment.Status)
	require.Len(t, deps.repo.markedUsed, 1)
	assert.Equal(t, token.ID, deps.repo.markedUsed[0])
}

func TestConfirmByTokenAlreadyConfirmedIsIdempotent(t *testing.T) {
	deps := setupPublicTest(t)
	entity := signableRateCon(deps)
	entity.Status = rateconfirmation.StatusConfirmed
	raw, token := mintTestToken(deps, entity)

	err := deps.svc.ConfirmByToken(t.Context(), raw, "J. Doe", "")

	require.NoError(t, err)
	assert.Empty(t, deps.repo.updated, "an already-confirmed agreement is not rewritten")
	require.Len(t, deps.repo.markedUsed, 1)
	assert.Equal(t, token.ID, deps.repo.markedUsed[0])
}

func TestConfirmByTokenFailureLeavesTokenUsable(t *testing.T) {
	deps := setupPublicTest(t)
	entity := signableRateCon(deps)
	raw, _ := mintTestToken(deps, entity)
	deps.repo.updateErr = errors.New("connection reset")

	err := deps.svc.ConfirmByToken(t.Context(), raw, "J. Doe", "")

	require.Error(t, err)
	assert.Empty(t, deps.repo.markedUsed,
		"a failed confirmation must not burn the carrier's only link")
}
