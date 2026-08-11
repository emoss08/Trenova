package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type offerReadingTenderRepo struct {
	repositories.TenderRepository

	offer *tender.TenderOffer
}

func (f *offerReadingTenderRepo) GetOfferByID(
	_ context.Context,
	_ repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	return f.offer, nil
}

type stubRateConIssuer struct {
	req    *portservices.IssueTenderRateConfirmationRequest
	result *portservices.IssueTenderRateConfirmationResult
	err    error
}

func (f *stubRateConIssuer) IssueForTenderAcceptance(
	_ context.Context,
	req *portservices.IssueTenderRateConfirmationRequest,
) (*portservices.IssueTenderRateConfirmationResult, error) {
	f.req = req
	return f.result, f.err
}

func acceptedOfferFixture() *tender.TenderOffer {
	respondedAt := int64(1_760_000_000)
	return &tender.TenderOffer{
		ID:             pulid.MustNew("tof_"),
		TenderID:       pulid.MustNew("ten_"),
		CarrierID:      pulid.MustNew("car_"),
		Status:         tender.OfferStatusAccepted,
		ResponseSource: tender.ResponseSourceEmail,
		RespondedAt:    &respondedAt,
		Carrier:        &carrier.Carrier{Name: "Eastline Transport"},
		Tender: &tender.Tender{
			ID:             pulid.MustNew("ten_"),
			ShipmentID:     pulid.MustNew("shp_"),
			ShipmentMoveID: pulid.MustNew("smv_"),
		},
	}
}

type issueLifecycleDeps struct {
	issuer    *stubRateConIssuer
	notifRepo *mocks.MockNotificationRepository
	svc       *Service
}

func setupIssueLifecycleTest(t *testing.T, offer *tender.TenderOffer) *issueLifecycleDeps {
	t.Helper()

	deps := &issueLifecycleDeps{
		issuer:    &stubRateConIssuer{},
		notifRepo: mocks.NewMockNotificationRepository(t),
	}
	deps.svc = &Service{
		l:    zap.NewNop(),
		repo: &offerReadingTenderRepo{offer: offer},
		notifications: notificationservice.New(notificationservice.Params{
			Logger:   zap.NewNop(),
			Repo:     deps.notifRepo,
			Realtime: &mocks.NoopRealtimeService{},
		}),
		rateConIssuer: deps.issuer,
	}
	return deps
}

func issueLifecycleTenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
}

func TestIssueRateConfirmationBuildsRequestFromOffer(t *testing.T) {
	offer := acceptedOfferFixture()
	deps := setupIssueLifecycleTest(t, offer)
	deps.issuer.result = &portservices.IssueTenderRateConfirmationResult{
		RateConfirmationID: pulid.MustNew("ratecon_"),
		Issued:             true,
	}
	tenantInfo := issueLifecycleTenant()

	err := deps.svc.IssueRateConfirmation(t.Context(), tenantInfo, offer.ID)

	require.NoError(t, err)
	require.NotNil(t, deps.issuer.req)
	assert.Equal(t, offer.ID, deps.issuer.req.OfferID)
	assert.Equal(t, offer.CarrierID, deps.issuer.req.CarrierID)
	assert.Equal(t, offer.Tender.ShipmentMoveID, deps.issuer.req.ShipmentMoveID)
	assert.Equal(t, tender.ResponseSourceEmail, deps.issuer.req.Source)
	assert.Equal(t, *offer.RespondedAt, deps.issuer.req.AcceptedAt)
}

func TestIssueRateConfirmationDeterministicFailureNotifiesDispatch(t *testing.T) {
	offer := acceptedOfferFixture()
	deps := setupIssueLifecycleTest(t, offer)
	deps.issuer.err = errortypes.NewBusinessError(
		"Document templates are not configured; the rate confirmation cannot be rendered",
	)

	var notified *notification.Notification
	deps.notifRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entity, ok := args.Get(1).(*notification.Notification)
			require.True(t, ok)
			notified = entity
		}).
		Return(&notification.Notification{}, nil)

	err := deps.svc.IssueRateConfirmation(t.Context(), issueLifecycleTenant(), offer.ID)

	require.NoError(t, err, "a deterministic issuance failure must not fail the acceptance")
	require.NotNil(t, notified)
	assert.Equal(t, "rate_confirmation_issue_failed", notified.EventType)
	assert.Equal(t, notification.PriorityHigh, notified.Priority)
	assert.Contains(t, notified.Message, "Eastline Transport")
	assert.Contains(t, notified.Message, "Generate it manually")
	assert.Equal(t, offer.TenderID.String(), *notified.CorrelationID)
}

func TestIssueRateConfirmationNotIssuedNotifiesDispatch(t *testing.T) {
	offer := acceptedOfferFixture()
	deps := setupIssueLifecycleTest(t, offer)
	deps.issuer.result = &portservices.IssueTenderRateConfirmationResult{
		Reason: "A dispatcher-managed rate confirmation already covers this move",
	}

	var notified *notification.Notification
	deps.notifRepo.On("Create", mock.Anything, mock.Anything).
		Run(func(args mock.Arguments) {
			entity, ok := args.Get(1).(*notification.Notification)
			require.True(t, ok)
			notified = entity
		}).
		Return(&notification.Notification{}, nil)

	err := deps.svc.IssueRateConfirmation(t.Context(), issueLifecycleTenant(), offer.ID)

	require.NoError(t, err)
	require.NotNil(t, notified)
	assert.Contains(t, notified.Message, "dispatcher-managed")
}

func TestIssueRateConfirmationInfraFailurePropagates(t *testing.T) {
	offer := acceptedOfferFixture()
	deps := setupIssueLifecycleTest(t, offer)
	deps.issuer.err = errortypes.NewConflictError("Rate confirmation is busy. Retry the request.")

	err := deps.svc.IssueRateConfirmation(t.Context(), issueLifecycleTenant(), offer.ID)

	require.Error(t, err, "transient failures must propagate so the activity retries")
}

func TestIssueRateConfirmationNoIssuerWiredNoOps(t *testing.T) {
	offer := acceptedOfferFixture()
	deps := setupIssueLifecycleTest(t, offer)
	deps.svc.rateConIssuer = nil

	err := deps.svc.IssueRateConfirmation(t.Context(), issueLifecycleTenant(), offer.ID)

	require.NoError(t, err)
	assert.Nil(t, deps.issuer.req)
}
