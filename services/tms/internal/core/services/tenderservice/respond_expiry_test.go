package tenderservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	portservices "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type lateFilingTenderRepo struct {
	readingTenderRepo

	late []*repositories.RecordLateOfferResponseRequest
}

func (r *lateFilingTenderRepo) RecordLateOfferResponse(
	_ context.Context,
	req *repositories.RecordLateOfferResponseRequest,
) error {
	r.late = append(r.late, req)
	return nil
}

func expiredSentTender() *tender.Tender {
	entity := sequentialTender(tender.StatusActive, tender.OfferStatusSent)
	expiredAt := timeutils.NowUnix() - 60
	entity.Offers[0].ExpiresAt = &expiredAt
	return entity
}

func TestRecordResponse_RefusesAnOfferPastItsExpiry(t *testing.T) {
	t.Parallel()

	entity := expiredSentTender()
	repo := &lateFilingTenderRepo{readingTenderRepo: readingTenderRepo{entity: entity}}
	events := &recordingEventService{}
	svc := &Service{
		repo:         repo,
		workflows:    &fakeWorkflowStarter{},
		eventService: events,
		auditService: &recordingAuditService{},
		l:            zap.NewNop(),
	}

	err := svc.RecordResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[0].ID,
		Action:     tender.ResponseActionAccept,
		Source:     tender.ResponseSourceEmail,
	})

	require.ErrorIs(t, err, ErrOfferNoLongerAvailable)
	require.Len(t, repo.late, 1, "the answer is kept as a late response")
	assert.Equal(t, entity.Offers[0].ID, repo.late[0].OfferID)
	assert.Equal(t, tender.ResponseActionAccept, repo.late[0].Action)
}

func TestPreviewResponse_RefusesAnOfferPastItsExpiry(t *testing.T) {
	t.Parallel()

	entity := expiredSentTender()
	svc := managedService(entity, &fakeWorkflowStarter{})

	_, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[0].ID,
		Action:     tender.ResponseActionDecline,
		Source:     tender.ResponseSourceManual,
	})
	require.ErrorIs(t, err, ErrOfferNoLongerAvailable)
}

func TestPreviewResponse_AcceptsAnOfferBeforeItsExpiry(t *testing.T) {
	t.Parallel()

	entity := sequentialTender(tender.StatusActive, tender.OfferStatusSent)
	expiresAt := timeutils.NowUnix() + 3600
	entity.Offers[0].ExpiresAt = &expiresAt
	svc := managedService(entity, &fakeWorkflowStarter{})

	preview, err := svc.PreviewResponse(t.Context(), &portservices.TenderResponseRequest{
		TenantInfo: createTestTenant(),
		OfferID:    entity.Offers[0].ID,
		Action:     tender.ResponseActionAccept,
		Source:     tender.ResponseSourceManual,
	})
	require.NoError(t, err)
	assert.Equal(t, tender.StatusAccepted, preview.TenderAfter.Status)
}

func TestDispatchOffer_DoesNotRedeliverAnOfferThatHasExpired(t *testing.T) {
	t.Parallel()

	entity := expiredSentTender()
	svc := managedService(entity, &fakeWorkflowStarter{})

	result, err := svc.DispatchOffer(t.Context(), createTestTenant(), entity.Offers[0].ID)

	require.NoError(t, err)
	assert.True(t, result.Delivered, "the workflow expires it at its deadline")
	assert.Equal(t, *entity.Offers[0].ExpiresAt, result.ExpiresAt)
}
