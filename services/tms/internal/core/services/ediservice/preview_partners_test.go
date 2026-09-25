package ediservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// The tender change repository has no expectation while the preview runs,
// so a change it recorded would fail the test.
func TestPreviewTenderChanges_NamesTheRecipientsAfterShipmentUpdateReaches(t *testing.T) {
	t.Parallel()

	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	original := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		BOL:            "BOL-1",
		Version:        3,
	}
	updated := *original
	updated.BOL = "BOL-2"
	baseline := buildTenderPayload(original)
	baseline.PurposeCode = edi.LoadTenderPurposeChange
	internal := &edi.TenderRecipient{
		ID:                    pulid.MustNew("etr_"),
		BusinessUnitID:        buID,
		SourceOrganizationID:  orgID,
		SourceBusinessUnitID:  buID,
		SourceShipmentID:      original.ID,
		RecipientKind:         edi.TenderRecipientKindInternal,
		LatestBaselinePayload: baseline,
		LatestBaselineHash:    tenderPayloadHash(&baseline),
	}
	current := buildTenderPayload(&updated)
	current.PurposeCode = edi.LoadTenderPurposeChange
	upToDate := *internal
	upToDate.ID = pulid.MustNew("etr_")
	upToDate.LatestBaselineHash = tenderPayloadHash(&current)

	recipients := mocks.NewMockEDITenderRecipientRepository(t)
	recipients.EXPECT().
		ListActiveTenderRecipientsForSourceShipment(mock.Anything, mock.Anything).
		Return([]*edi.TenderRecipient{internal, &upToDate}, nil)
	changes := mocks.NewMockEDITenderChangeRepository(t)
	svc := &Service{
		l:                   zap.NewNop(),
		tenderRecipientRepo: recipients,
		tenderChangeRepo:    changes,
	}
	actor := &services.RequestActor{UserID: pulid.MustNew("usr_")}

	notices, err := svc.PreviewTenderChanges(t.Context(), original, &updated, actor)
	require.NoError(t, err)
	require.Len(t, notices, 1, "a recipient already on the new baseline is sent nothing")

	var created []*edi.TenderChange
	changes.EXPECT().
		CreateTenderChangeIdempotent(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			change *edi.TenderChange,
		) (*repositories.CreateEDITenderChangeIdempotentResult, error) {
			created = append(created, change)
			return &repositories.CreateEDITenderChangeIdempotentResult{
				TenderChange: change,
				Created:      true,
			}, nil
		})
	changes.EXPECT().
		SupersedeActionableTenderChanges(mock.Anything, mock.Anything).
		Return(nil)

	require.NoError(t, svc.AfterShipmentUpdate(t.Context(), original, &updated, actor))
	require.Len(t, created, 1)

	notice := notices[0]
	assert.Equal(t, created[0].RecipientID, notice.RecordID)
	assert.Equal(t, edi.TransactionSet204, notice.TransactionSet)
	assert.True(t, notice.HeldForReview)
	keys := make([]string, 0, len(created[0].DiffSummary))
	for key := range created[0].DiffSummary {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	assert.Equal(t, keys, notice.Changed)
	assert.Contains(t, notice.Changed, "bol")
}

func TestPreviewCancelNotices_NamesTheOtherSideOfEachActiveLink(t *testing.T) {
	t.Parallel()

	orgID, partnerOrg := pulid.MustNew("org_"), pulid.MustNew("org_")
	shipmentID := pulid.MustNew("shp_")
	links := mocks.NewMockEDIShipmentLinkRepository(t)
	links.EXPECT().
		GetShipmentLinksByShipmentID(mock.Anything, mock.Anything).
		Return([]*edi.ShipmentLink{
			{
				ID:                   pulid.MustNew("esl_"),
				SourceOrganizationID: orgID,
				SourceShipmentID:     shipmentID,
				TargetOrganizationID: partnerOrg,
				Status:               edi.ShipmentLinkStatusActive,
			},
			{
				ID:                   pulid.MustNew("esl_"),
				SourceOrganizationID: orgID,
				SourceShipmentID:     shipmentID,
				TargetOrganizationID: pulid.MustNew("org_"),
				Status:               edi.ShipmentLinkStatusClosed,
			},
		}, nil)
	svc := &Service{l: zap.NewNop(), shipmentLinkRepo: links}

	notices, err := svc.PreviewCancelNotices(
		t.Context(),
		pagination.TenantInfo{OrgID: orgID, BuID: pulid.MustNew("bu_")},
		shipmentID,
	)
	require.NoError(t, err)
	require.Len(t, notices, 1)
	assert.Equal(t, partnerOrg, notices[0].OrganizationID)
	assert.Equal(t, edi.TransactionSet214, notices[0].TransactionSet)
	assert.Equal(t, "A7", notices[0].StatusCode)
}
