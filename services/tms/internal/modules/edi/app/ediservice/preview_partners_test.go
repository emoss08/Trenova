package ediservice

import (
	"context"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/testutil/mocks/editenderportsmock"
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
	baseline.PurposeCode = editransfer.LoadTenderPurposeChange
	internal := &editenderchange.TenderRecipient{
		ID:                    pulid.MustNew("etr_"),
		BusinessUnitID:        buID,
		SourceOrganizationID:  orgID,
		SourceShipmentID:      original.ID,
		RecipientKind:         editenderchange.TenderRecipientKindInternal,
		LatestBaselinePayload: baseline,
		LatestBaselineHash:    tenderPayloadHash(&baseline),
	}
	current := buildTenderPayload(&updated)
	current.PurposeCode = editransfer.LoadTenderPurposeChange
	upToDate := *internal
	upToDate.ID = pulid.MustNew("etr_")
	upToDate.LatestBaselineHash = tenderPayloadHash(&current)

	recipients := editenderportsmock.NewMockEDITenderRecipientRepository(t)
	recipients.EXPECT().
		ListActiveTenderRecipientsForSourceShipment(mock.Anything, mock.Anything).
		Return([]*editenderchange.TenderRecipient{internal, &upToDate}, nil)
	changes := editenderportsmock.NewMockEDITenderChangeRepository(t)
	svc := &Service{
		l:                   zap.NewNop(),
		tenderRecipientRepo: recipients,
		tenderChangeRepo:    changes,
	}
	actor := &permissionports.RequestActor{UserID: pulid.MustNew("usr_")}

	notices, err := svc.PreviewTenderChanges(t.Context(), original, &updated, actor)
	require.NoError(t, err)
	require.Len(t, notices, 1, "a recipient already on the new baseline is sent nothing")

	var created []*editenderchange.TenderChange
	changes.EXPECT().
		CreateTenderChangeIdempotent(mock.Anything, mock.Anything).
		RunAndReturn(func(
			_ context.Context,
			change *editenderchange.TenderChange,
		) (*editenderports.CreateEDITenderChangeIdempotentResult, error) {
			created = append(created, change)
			return &editenderports.CreateEDITenderChangeIdempotentResult{
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
	assert.Equal(t, edidocument.TransactionSet204, notice.TransactionSet)
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
	links := editenderportsmock.NewMockEDIShipmentLinkRepository(t)
	links.EXPECT().
		GetShipmentLinksByShipmentID(mock.Anything, mock.Anything).
		Return([]*edisync.ShipmentLink{
			{
				ID:                   pulid.MustNew("esl_"),
				SourceOrganizationID: orgID,
				SourceShipmentID:     shipmentID,
				TargetOrganizationID: partnerOrg,
				Status:               edisync.ShipmentLinkStatusActive,
			},
			{
				ID:                   pulid.MustNew("esl_"),
				SourceOrganizationID: orgID,
				SourceShipmentID:     shipmentID,
				TargetOrganizationID: pulid.MustNew("org_"),
				Status:               edisync.ShipmentLinkStatusClosed,
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
	assert.Equal(t, edidocument.TransactionSet214, notices[0].TransactionSet)
	assert.Equal(t, "A7", notices[0].StatusCode)
}
