package rateconfirmationservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// planTestDeps wires the service with readers only. The email service and
// the template resolver are strict mocks told to expect nothing but what a
// preview reads, so a preview that sent, rendered a PDF or filed a document
// fails the test that runs it.
type planTestDeps struct {
	repo                  *fakeRateConRepo
	carrierAssignmentRepo *mocks.MockCarrierAssignmentRepository
	carrierRepo           *mocks.MockCarrierRepository
	shipmentRepo          *mocks.MockShipmentRepository
	templates             *mocks.MockDocumentTemplateResolver
	email                 *mocks.MockEmailService
	svc                   *Service
}

func setupPlanTest(t *testing.T) *planTestDeps {
	t.Helper()

	deps := &planTestDeps{
		repo:                  newFakeRateConRepo(),
		carrierAssignmentRepo: mocks.NewMockCarrierAssignmentRepository(t),
		carrierRepo:           mocks.NewMockCarrierRepository(t),
		shipmentRepo:          mocks.NewMockShipmentRepository(t),
		templates:             mocks.NewMockDocumentTemplateResolver(t),
		email:                 mocks.NewMockEmailService(t),
	}
	deps.svc = &Service{
		l:                     zap.NewNop(),
		db:                    testDBConnection{},
		repo:                  deps.repo,
		carrierAssignmentRepo: deps.carrierAssignmentRepo,
		carrierRepo:           deps.carrierRepo,
		shipmentRepo:          deps.shipmentRepo,
		templates:             deps.templates,
		emailService:          deps.email,
	}

	return deps
}

func carrierWithRateConContacts(id pulid.ID) *carrier.Carrier {
	return &carrier.Carrier{
		ID:    id,
		Name:  "Eastline Transport",
		Email: "office@eastline.test",
		Contacts: []*carrier.CarrierContact{
			{Email: "dispatch@eastline.test", ReceivesRateConfirmations: true},
			{Email: "billing@eastline.test"},
		},
	}
}

func TestPreviewGenerate_IsTheNextRevisionAndVoidsTheStandingOne(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	moveID := pulid.MustNew("smv_")
	shipmentID := pulid.MustNew("shp_")
	assignment := &shipment.CarrierAssignment{
		ID:             pulid.MustNew("cass_"),
		CarrierID:      pulid.MustNew("car_"),
		ShipmentMoveID: moveID,
		Status:         shipment.CarrierAssignmentStatusPending,
		ShipmentMove:   &shipment.ShipmentMove{ID: moveID, ShipmentID: shipmentID},
	}
	standing := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		CarrierAssignmentID: assignment.ID,
		Revision:            2,
		Status:              rateconfirmation.StatusSent,
	}
	deps.repo.add(standing)
	deps.repo.maxRevision = 2

	deps.carrierAssignmentRepo.On("GetActiveByMoveID", mock.Anything, tenantInfo, moveID).
		Return(assignment, nil)
	deps.carrierRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(carrierWithRateConContacts(assignment.CarrierID), nil).
		Once()
	deps.shipmentRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&shipment.Shipment{
			ID:        shipmentID,
			ProNumber: "PRO-1001",
			Moves:     []*shipment.ShipmentMove{{ID: moveID, ShipmentID: shipmentID}},
		}, nil).
		Once()

	plan, err := deps.svc.PreviewGenerate(t.Context(), tenantInfo, moveID, nil)
	require.NoError(t, err)

	assert.Equal(t, int64(3), plan.Created.Revision)
	assert.Equal(t, rateconfirmation.StatusGenerated, plan.Created.Status)
	assert.Equal(t, assignment.ID, plan.Created.CarrierAssignmentID)
	assert.Equal(t, shipmentID, plan.Created.ShipmentID)
	require.NotNil(t, plan.SupersededAfter)
	assert.Equal(t, rateconfirmation.StatusSent, plan.SupersededBefore.Status)
	assert.Equal(t, rateconfirmation.StatusVoided, plan.SupersededAfter.Status)
	assert.Equal(t, "Eastline Transport", plan.Carrier.Name)
	assert.Equal(t, "PRO-1001", plan.Shipment.ProNumber)
	assert.Empty(t, deps.repo.updated, "a preview voids nothing")
}

func TestPreviewGenerate_RefusesAMoveWithNoCarrier(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	moveID := pulid.MustNew("smv_")
	deps.carrierAssignmentRepo.On("GetActiveByMoveID", mock.Anything, tenantInfo, moveID).
		Return(nil, nil)

	_, err := deps.svc.PreviewGenerate(t.Context(), tenantInfo, moveID, nil)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
}

func TestPreviewSend_IsTheEmailSendWouldSend(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	entity := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		CarrierAssignmentID: pulid.MustNew("cass_"),
		CarrierID:           pulid.MustNew("car_"),
		Revision:            1,
		Status:              rateconfirmation.StatusGenerated,
		PayloadSnapshot:     map[string]any{"CarrierName": "Eastline Transport"},
	}
	deps.repo.add(entity)
	deps.svc.cfg = &config.Config{
		Tendering: config.TenderingConfig{PublicBaseURL: "https://tender.example"},
	}
	deps.carrierRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(carrierWithRateConContacts(entity.CarrierID), nil).
		Once()
	deps.templates.EXPECT().
		RenderMessage(mock.Anything, mock.Anything).
		Return(&services.RenderedMessage{
			Subject: "Rate confirmation PRO-1001",
			Text:    "Please sign the attached rate confirmation.",
		}, nil).
		Once()

	plan, err := deps.svc.PreviewSend(t.Context(), tenantInfo, entity.ID)
	require.NoError(t, err)

	assert.Equal(t, []string{"dispatch@eastline.test"}, plan.Recipients)
	assert.Equal(t, "Rate confirmation PRO-1001", plan.Subject)
	assert.Equal(t, "Please sign the attached rate confirmation.", plan.Body)
	assert.Equal(t, fileName(entity), plan.Attachment)
	assert.True(t, plan.SignLink, "an unexecuted revision travels with a sign link")
	assert.Equal(t, rateconfirmation.StatusSent, plan.After.Status)
	assert.Equal(t, "dispatch@eastline.test", plan.After.SentToEmails)
	assert.Equal(t, rateconfirmation.StatusGenerated, entity.Status)
	assert.Empty(t, deps.repo.createdTokens, "a preview mints no sign link")
}

func TestPreviewSend_RefusesWhatSendRefuses(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	entity := &rateconfirmation.RateConfirmation{
		ID:     pulid.MustNew("ratecon_"),
		Status: rateconfirmation.StatusVoided,
	}
	deps.repo.add(entity)

	_, previewErr := deps.svc.PreviewSend(t.Context(), tenantInfo, entity.ID)
	_, writeErr := deps.svc.Send(t.Context(), tenantInfo, entity.ID)

	require.Error(t, previewErr)
	assert.True(t, errortypes.IsBusinessError(previewErr))
	require.Error(t, writeErr)
	assert.Equal(t, writeErr.Error(), previewErr.Error())
}

// Voiding an executed agreement withdraws the confirmation, so its
// assignment goes back to pending; the preview shows both from the domain
// methods the write calls.
func TestPreviewVoid_RevertsTheAssignmentOfAConfirmedAgreement(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	confirmedAt := int64(1_790_000_000)
	entity := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		CarrierAssignmentID: pulid.MustNew("cass_"),
		Status:              rateconfirmation.StatusConfirmed,
		ConfirmedAt:         &confirmedAt,
	}
	deps.repo.add(entity)
	assignment := &shipment.CarrierAssignment{
		ID:          entity.CarrierAssignmentID,
		Status:      shipment.CarrierAssignmentStatusConfirmed,
		ConfirmedAt: &confirmedAt,
	}
	deps.carrierAssignmentRepo.On("GetByID", mock.Anything, mock.Anything).Return(assignment, nil)

	plan, err := deps.svc.PreviewVoid(t.Context(), tenantInfo, entity.ID, "Rate renegotiated")
	require.NoError(t, err)

	assert.Equal(t, rateconfirmation.StatusVoided, plan.After.Status)
	assert.Equal(t, "Rate renegotiated", plan.After.VoidReason)
	require.NotNil(t, plan.AssignmentAfter)
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, plan.AssignmentBefore.Status)
	assert.Equal(t, shipment.CarrierAssignmentStatusPending, plan.AssignmentAfter.Status)
	assert.Equal(t, rateconfirmation.StatusConfirmed, entity.Status)
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, assignment.Status)
	assert.Empty(t, deps.repo.revokedFor, "a preview revokes no sign link")
}

func TestPreviewVoid_RequiresAReason(t *testing.T) {
	deps := setupPlanTest(t)

	_, err := deps.svc.PreviewVoid(t.Context(), testTenant(), pulid.MustNew("ratecon_"), "")
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
}

func TestPreviewMarkConfirmed_ConfirmsTheAgreementAndItsAssignment(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	entity := &rateconfirmation.RateConfirmation{
		ID:                  pulid.MustNew("ratecon_"),
		CarrierAssignmentID: pulid.MustNew("cass_"),
		Status:              rateconfirmation.StatusSent,
	}
	deps.repo.add(entity)
	assignment := &shipment.CarrierAssignment{
		ID:     entity.CarrierAssignmentID,
		Status: shipment.CarrierAssignmentStatusPending,
	}
	deps.carrierAssignmentRepo.On("GetByID", mock.Anything, mock.Anything).Return(assignment, nil)

	plan, err := deps.svc.PreviewMarkConfirmed(t.Context(), tenantInfo, entity.ID, "Dana Ruiz")
	require.NoError(t, err)

	assert.Equal(t, rateconfirmation.StatusConfirmed, plan.After.Status)
	assert.Equal(t, "Dana Ruiz", plan.After.ConfirmedByName)
	assert.Equal(t, rateconfirmation.ViaDispatcher, plan.After.ConfirmedVia)
	assert.Equal(t, shipment.CarrierAssignmentStatusConfirmed, plan.AssignmentAfter.Status)
	assert.Equal(t, rateconfirmation.StatusSent, entity.Status)
	assert.Empty(t, deps.repo.updated)
}

func TestPreviewMarkConfirmed_RefusesWhatMarkConfirmedRefuses(t *testing.T) {
	deps := setupPlanTest(t)
	tenantInfo := testTenant()
	entity := &rateconfirmation.RateConfirmation{
		ID:     pulid.MustNew("ratecon_"),
		Status: rateconfirmation.StatusConfirmed,
	}
	deps.repo.add(entity)

	_, previewErr := deps.svc.PreviewMarkConfirmed(t.Context(), tenantInfo, entity.ID, "Dana Ruiz")
	_, writeErr := deps.svc.MarkConfirmed(t.Context(), tenantInfo, entity.ID, "Dana Ruiz")

	require.Error(t, previewErr)
	require.Error(t, writeErr)
	assert.Equal(t, writeErr.Error(), previewErr.Error())
}
