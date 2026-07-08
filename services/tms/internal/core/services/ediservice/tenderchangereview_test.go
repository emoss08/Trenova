package ediservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestService_ApplyTenderChange_ApplyFailureSurfacesErrorAndKeepsPendingReview(t *testing.T) {
	t.Parallel()

	fixture := newTenderChangeReviewFixture(t)
	change := fixture.pendingChange()
	applyErr := errors.New("boom")

	fixture.tenderChangeRepo.EXPECT().
		GetTenderChangeByID(
			mock.Anything,
			repositories.GetEDITenderChangeByIDRequest{
				ID:         change.ID,
				TenantInfo: fixture.tenantInfo(),
			},
		).
		Return(change, nil).
		Once()
	fixture.tenderRecipientRepo.EXPECT().
		GetTenderRecipientByID(
			mock.Anything,
			repositories.GetEDITenderRecipientByIDRequest{
				ID:         change.RecipientID,
				TenantInfo: fixture.tenantInfo(),
			},
		).
		Return(nil, applyErr).
		Once()
	fixture.tenderChangeRepo.EXPECT().
		UpdateTenderChange(mock.Anything, mock.MatchedBy(func(updated *edi.TenderChange) bool {
			return updated.Status == edi.TenderChangeStatusPendingReview &&
				updated.FailureReason == applyErr.Error() &&
				updated.ConflictMetadata["reason"] == applyErr.Error()
		})).
		RunAndReturn(func(_ context.Context, updated *edi.TenderChange) (*edi.TenderChange, error) {
			return updated, nil
		}).
		Once()

	result, err := fixture.service.ApplyTenderChange(
		t.Context(),
		&TenderChangeActionRequest{
			TenantInfo: fixture.tenantInfo(),
			ChangeID:   change.ID,
		},
		fixture.actor,
	)

	require.ErrorIs(t, err, applyErr)
	require.Nil(t, result)
}

func TestService_RejectTenderChange_UpdatesStatusWithoutApplying(t *testing.T) {
	t.Parallel()

	fixture := newTenderChangeReviewFixture(t)
	change := fixture.pendingChange()

	fixture.tenderChangeRepo.EXPECT().
		GetTenderChangeByID(
			mock.Anything,
			repositories.GetEDITenderChangeByIDRequest{
				ID:         change.ID,
				TenantInfo: fixture.tenantInfo(),
			},
		).
		Return(change, nil).
		Once()
	fixture.tenderChangeRepo.EXPECT().
		UpdateTenderChange(mock.Anything, mock.MatchedBy(func(updated *edi.TenderChange) bool {
			return updated.Status == edi.TenderChangeStatusRejected &&
				updated.ReviewedByID == fixture.actor.UserID &&
				updated.ReviewedAt != nil &&
				updated.FailureReason == "Not applicable"
		})).
		RunAndReturn(func(_ context.Context, updated *edi.TenderChange) (*edi.TenderChange, error) {
			return updated, nil
		}).
		Once()

	result, err := fixture.service.RejectTenderChange(
		t.Context(),
		&TenderChangeActionRequest{
			TenantInfo: fixture.tenantInfo(),
			ChangeID:   change.ID,
			Reason:     "Not applicable",
		},
		fixture.actor,
	)

	require.NoError(t, err)
	require.Equal(t, edi.TenderChangeStatusRejected, result.Status)
}

type tenderChangeReviewFixture struct {
	service             *Service
	actor               *services.RequestActor
	recipientOrgID      pulid.ID
	buID                pulid.ID
	tenderChangeRepo    *mocks.MockEDITenderChangeRepository
	tenderRecipientRepo *mocks.MockEDITenderRecipientRepository
}

func newTenderChangeReviewFixture(t *testing.T) *tenderChangeReviewFixture {
	t.Helper()

	fixture := &tenderChangeReviewFixture{
		recipientOrgID:      pulid.MustNew("org_"),
		buID:                pulid.MustNew("bu_"),
		tenderChangeRepo:    mocks.NewMockEDITenderChangeRepository(t),
		tenderRecipientRepo: mocks.NewMockEDITenderRecipientRepository(t),
	}
	fixture.actor = &services.RequestActor{
		UserID:         pulid.MustNew("usr_"),
		PrincipalType:  services.PrincipalTypeUser,
		OrganizationID: fixture.recipientOrgID,
		BusinessUnitID: fixture.buID,
	}
	fixture.service = &Service{
		l:                   zap.NewNop(),
		db:                  transferChangeApplyDB{},
		tenderChangeRepo:    fixture.tenderChangeRepo,
		tenderRecipientRepo: fixture.tenderRecipientRepo,
	}
	return fixture
}

func (f *tenderChangeReviewFixture) tenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: f.recipientOrgID, BuID: f.buID}
}

func (f *tenderChangeReviewFixture) pendingChange() *edi.TenderChange {
	recipientID := pulid.MustNew("editr_")
	return &edi.TenderChange{
		ID:             pulid.MustNew("editch_"),
		BusinessUnitID: f.buID,
		RecipientID:    recipientID,
		RecipientKind:  edi.TenderRecipientKindInternal,
		Status:         edi.TenderChangeStatusPendingReview,
		Recipient: &edi.TenderRecipient{
			ID:                      recipientID,
			RecipientKind:           edi.TenderRecipientKindInternal,
			RecipientOrganizationID: f.recipientOrgID,
			RecipientBusinessUnitID: f.buID,
		},
	}
}
