package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edisync"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editenderchange"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editenderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTenderChangeList struct {
	changes []*editenderchange.TenderChange
	request *editenderports.ListEDITenderChangesRequest
}

func (f *fakeTenderChangeList) ListTenderChanges(
	_ context.Context,
	req *editenderports.ListEDITenderChangesRequest,
) (*pagination.ListResult[*editenderchange.TenderChange], error) {
	f.request = req

	return &pagination.ListResult[*editenderchange.TenderChange]{Items: f.changes}, nil
}

func TestListEDITenderChanges_SaysWhatChangedAndWhichWaitOnThisOrganization(t *testing.T) {
	t.Parallel()

	params := agentParams(map[string]any{paramStatus: "PendingReview"}, "")
	change := &editenderchange.TenderChange{
		ID:     pulid.MustNew("editcg_"),
		Status: editenderchange.TenderChangeStatusPendingReview,
		Recipient: &editenderchange.TenderRecipient{
			RecipientKind:           editenderchange.TenderRecipientKindInternal,
			RecipientOrganizationID: params.OrganizationID,
		},
		NewTenderPayload: editransfer.LoadTenderPayload{BOL: "BOL-5"},
		DiffSummary: map[string]any{
			"weight": map[string]any{"previous": 20000, "next": 42000},
			"moves":  map[string]any{"previous": []any{}, "next": []any{map[string]any{}}},
		},
	}
	fake := &fakeTenderChangeList{changes: []*editenderchange.TenderChange{change}}
	tool := newListEDITenderChangesTool(fake)

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, editenderchange.TenderChangeStatusPendingReview, fake.request.Status)
	rows := result.(*gatedOutcome).Items.([]ediTenderChangeRow)
	require.Len(t, rows, 1)
	assert.True(t, rows[0].AwaitsThisOrganization)
	assert.Equal(t, "BOL-5", rows[0].BOL)
	require.Len(t, rows[0].Changes, 2)
	assert.Equal(t, "moves", rows[0].Changes[0].Field)
	assert.Equal(t, "weight", rows[0].Changes[1].Field)
	assert.Equal(t, "42000", rows[0].Changes[1].After)
}

type fakeTransferChangeList struct {
	changes []*edisync.TransferChange
	request *editenderports.ListEDITransferChangesRequest
}

func (f *fakeTransferChangeList) ListTransferChanges(
	_ context.Context,
	req *editenderports.ListEDITransferChangesRequest,
) (*pagination.ListResult[*edisync.TransferChange], error) {
	f.request = req

	return &pagination.ListResult[*edisync.TransferChange]{Items: f.changes}, nil
}

func TestListEDITransferChanges_GivesTheReportedStatus(t *testing.T) {
	t.Parallel()

	change := &edisync.TransferChange{
		ID:         pulid.MustNew("editc_"),
		Status:     edisync.TransferChangeStatusPendingReview,
		ChangeType: edisync.TransferChangeTypeShipmentStatus214,
		Payload:    map[string]any{"newStatus": "Completed"},
	}
	fake := &fakeTransferChangeList{changes: []*edisync.TransferChange{change}}
	tool := newListEDITransferChangesTool(fake)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		paramStatus: "PendingReview",
	}, ""))
	require.NoError(t, err)

	require.Len(t, fake.request.Filter.FieldFilters, 1)
	rows := result.(*gatedOutcome).Items.([]ediTransferChangeRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "Completed", rows[0].ReportedStatus)
}
