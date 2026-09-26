package tenderservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// readingTenderRepo answers the reads a preview makes. Every write reaches
// the nil repository it embeds and panics, so a preview that wrote fails.
type readingTenderRepo struct {
	repositories.TenderRepository

	entity *tender.Tender
}

func (r *readingTenderRepo) GetByID(
	_ context.Context,
	_ repositories.GetTenderByIDRequest,
) (*tender.Tender, error) {
	return r.entity, nil
}

func (r *readingTenderRepo) GetOfferByID(
	_ context.Context,
	req repositories.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	offer := r.entity.FindOffer(req.OfferID)
	if offer == nil {
		return nil, errortypes.NewNotFoundError("Tender offer not found")
	}
	answered := *offer
	answered.Tender = r.entity

	return &answered, nil
}

func sequentialTender(status tender.Status, offers ...tender.OfferStatus) *tender.Tender {
	entity := &tender.Tender{
		ID:     pulid.MustNew("ten_"),
		Mode:   tender.ModeWaterfall,
		Status: status,
	}
	for idx, offerStatus := range offers {
		entity.Offers = append(entity.Offers, &tender.TenderOffer{
			ID:        pulid.MustNew("tof_"),
			TenderID:  entity.ID,
			CarrierID: pulid.MustNew("car_"),
			Rank:      int16(idx + 1),
			Rate:      decimal.NewFromInt(int64(1500 + 100*idx)),
			Channel:   tender.ChannelEmail,
			Status:    offerStatus,
		})
	}

	return entity
}

func managedService(entity *tender.Tender, workflows *fakeWorkflowStarter) *Service {
	return &Service{
		l:            zap.NewNop(),
		repo:         &readingTenderRepo{entity: entity},
		workflows:    workflows,
		eventService: &recordingEventService{},
		auditService: &recordingAuditService{},
	}
}

func TestPreviewCancel_WithdrawsTheTenderAndTheOffersCarriersHold(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive,
		tender.OfferStatusDeclined, tender.OfferStatusSent, tender.OfferStatusPending)
	svc := managedService(entity, &fakeWorkflowStarter{})

	plan, err := svc.PreviewCancel(t.Context(), &CancelTenderRequest{
		TenantInfo: createTestTenant(),
		TenderID:   entity.ID,
		Reason:     "Covered by our own driver",
	})
	require.NoError(t, err)

	assert.Equal(t, tender.StatusActive, plan.Before.Status)
	assert.Equal(t, tender.StatusCanceled, plan.After.Status)
	assert.Equal(t, "Covered by our own driver", plan.After.CancellationReason)
	require.Len(t, plan.After.Offers, 3)
	assert.Equal(t, tender.OfferStatusDeclined, plan.After.Offers[0].Status)
	assert.Equal(t, tender.OfferStatusWithdrawn, plan.After.Offers[1].Status)
	assert.Equal(t, tender.OfferStatusSkipped, plan.After.Offers[2].Status)
	assert.Equal(t, tender.OfferStatusSent, entity.Offers[1].Status, "the stored offer is untouched")
}

// A tender that already ended cannot be canceled, and the preview says so in
// the words the write would use.
func TestPreviewCancel_RefusesWhatCancelRefuses(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusAccepted, tender.OfferStatusAccepted)
	svc := managedService(entity, &fakeWorkflowStarter{})
	req := &CancelTenderRequest{
		TenantInfo: createTestTenant(),
		TenderID:   entity.ID,
		Reason:     "No longer needed",
	}

	_, previewErr := svc.PreviewCancel(t.Context(), req)
	writeErr := svc.Cancel(t.Context(), req)

	require.Error(t, previewErr)
	assert.True(t, errortypes.IsBusinessError(previewErr))
	require.Error(t, writeErr)
	assert.Equal(t, writeErr.Error(), previewErr.Error())
}

func TestPreviewResponse_AcceptCloseOutTheRestOfThePlan(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive,
		tender.OfferStatusSent, tender.OfferStatusPending)
	svc := managedService(entity, &fakeWorkflowStarter{})

	plan, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[0].ID,
		Action:     tender.ResponseActionAccept,
		Source:     tender.ResponseSourceManual,
	})
	require.NoError(t, err)

	assert.Equal(t, entity.Offers[0].ID, plan.Offer.ID)
	assert.Equal(t, tender.StatusAccepted, plan.TenderAfter.Status)
	require.NotNil(t, plan.TenderAfter.AcceptedOfferID)
	assert.Equal(t, entity.Offers[0].ID, *plan.TenderAfter.AcceptedOfferID)
	assert.Equal(t, tender.OfferStatusAccepted, plan.TenderAfter.Offers[0].Status)
	assert.Equal(t, tender.ResponseSourceManual, plan.TenderAfter.Offers[0].ResponseSource)
	assert.Equal(t, tender.OfferStatusSkipped, plan.TenderAfter.Offers[1].Status)
	assert.Nil(t, plan.NextOffer)
	assert.False(t, plan.Exhausted)
}

func TestPreviewResponse_DeclineOffersTheNextCarrierInAWaterfall(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive,
		tender.OfferStatusSent, tender.OfferStatusPending)
	svc := managedService(entity, &fakeWorkflowStarter{})

	plan, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo:    createTestTenant(),
		OfferID:       entity.Offers[0].ID,
		Action:        tender.ResponseActionDecline,
		Source:        tender.ResponseSourceManual,
		DeclineReason: "No truck in the area",
	})
	require.NoError(t, err)

	assert.Equal(t, tender.StatusActive, plan.TenderAfter.Status)
	assert.Equal(t, tender.OfferStatusDeclined, plan.TenderAfter.Offers[0].Status)
	assert.Equal(t, "No truck in the area", plan.TenderAfter.Offers[0].DeclineReason)
	require.NotNil(t, plan.NextOffer)
	assert.Equal(t, entity.Offers[1].ID, plan.NextOffer.ID)
	assert.False(t, plan.Exhausted)
}

func TestPreviewResponse_DeclineOfTheLastCarrierExhaustsTheTender(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive,
		tender.OfferStatusDeclined, tender.OfferStatusSent)
	svc := managedService(entity, &fakeWorkflowStarter{})

	plan, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[1].ID,
		Action:     tender.ResponseActionDecline,
		Source:     tender.ResponseSourceManual,
	})
	require.NoError(t, err)

	assert.Nil(t, plan.NextOffer)
	assert.True(t, plan.Exhausted)
	assert.Equal(t, tender.StatusExhausted, plan.TenderAfter.Status)
}

// An offer the carrier can no longer answer is refused by the preview with
// the error the write returns; the write also files the answer as late, which
// the preview must not.
func TestPreviewResponse_RefusesAnOfferNoLongerOpen(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive,
		tender.OfferStatusExpired, tender.OfferStatusSent)
	svc := managedService(entity, &fakeWorkflowStarter{})

	_, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[0].ID,
		Action:     tender.ResponseActionAccept,
		Source:     tender.ResponseSourceManual,
	})
	require.ErrorIs(t, err, ErrOfferNoLongerAvailable)
}

func TestPreviewResponse_RefusesAnUnknownAction(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive, tender.OfferStatusSent)
	svc := managedService(entity, &fakeWorkflowStarter{})

	_, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[0].ID,
		Action:     tender.ResponseAction("Maybe"),
		Source:     tender.ResponseSourceManual,
	})
	require.Error(t, err)
	assert.False(t, errors.Is(err, ErrOfferNoLongerAvailable))
}
