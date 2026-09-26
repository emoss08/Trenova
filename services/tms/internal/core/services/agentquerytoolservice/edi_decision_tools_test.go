package agentquerytoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTransferDetail struct {
	transfer   *edi.EDITransfer
	unresolved []edi.MappingResolution
	mapped     bool
}

func (f *fakeTransferDetail) GetTransfer(
	context.Context,
	repositories.GetEDITransferByIDRequest,
) (*edi.EDITransfer, error) {
	return f.transfer, nil
}

func (f *fakeTransferDetail) MappingPreview(
	context.Context,
	repositories.GetEDITransferByIDRequest,
) (*ediservice.MappingPreview, error) {
	f.mapped = true

	return &ediservice.MappingPreview{Unresolved: f.unresolved}, nil
}

func receivedTender(orgID pulid.ID) *edi.EDITransfer {
	start, end := int64(1_790_000_000), int64(1_790_007_200)

	return &edi.EDITransfer{
		ID:                   pulid.MustNew("edilt_"),
		SourceOrganizationID: orgID,
		TargetOrganizationID: orgID,
		InboundMessageID:     pulid.MustNew("edimsg_"),
		Status:               edi.TransferStatusPendingApproval,
		TenderPayload: edi.LoadTenderPayload{
			BOL:               "BOL-5",
			CustomerLabel:     "Acme Foods",
			TotalChargeAmount: decimal.NewNullDecimal(decimal.RequireFromString("1850.00")),
			Moves: []edi.LoadTenderMove{{Stops: []edi.LoadTenderStop{
				{
					Type:                 "Pickup",
					LocationName:         "Acme DC",
					LocationCity:         "Dallas",
					ScheduledWindowStart: start,
					ScheduledWindowEnd:   &end,
				},
				{Type: "Delivery", LocationName: "Store 12", LocationCity: "Tulsa"},
			}}},
			Commodities: []edi.LoadTenderCommodity{{CommodityLabel: "Produce", Weight: 40000}},
		},
	}
}

func TestGetEDITransfer_GivesTheTenderItsMappingsAndWhetherItCanBeAccepted(t *testing.T) {
	t.Parallel()

	params := agentParams(nil, permission.SensitivityRestricted)
	fake := &fakeTransferDetail{
		transfer: receivedTender(params.OrganizationID),
		unresolved: []edi.MappingResolution{
			{EntityType: edi.MappingEntityTypeCustomer, SourceLabel: "ACME"},
		},
	}
	tool := newGetEDITransferTool(fake, &fakePermissions{})
	params.Params = map[string]any{paramEDITransferID: fake.transfer.ID.String()}

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	view := result.(*ediTransferView)
	assert.Equal(t, transferInbound, view.Direction)
	require.Len(t, view.Stops, 2)
	assert.Equal(t, "Acme DC", view.Stops[0].Location)
	assert.Equal(t, "Dallas", view.Stops[0].City)
	require.Len(t, view.Commodities, 1)
	require.NotNil(t, view.Charges)
	assert.Equal(t, "1850.00", view.Charges.Total)
	assert.True(t, fake.mapped)
	require.Len(t, view.UnresolvedMappings, 1)
	assert.True(t, view.CanDecide)
	assert.False(t, view.CanAccept, "a tender with a missing mapping cannot be accepted yet")
	assert.Equal(t,
		[]agent.RecordRef{{EntityType: ediTransferEntity, ID: fake.transfer.ID.String()}},
		view.TaintedRecords())

	policy := tool.Policy()
	assert.Equal(t, agent.ExternalReadAlways, policy.ReadsExternal)
	assert.Equal(t, agent.TaintSourceEDI, policy.Source)
}

func TestGetEDITransfer_WithholdsTheRateBelowRestrictedAndNeverReadsTheOtherSidesMappings(
	t *testing.T,
) {
	t.Parallel()

	params := agentParams(nil, permission.SensitivityInternal)
	transfer := receivedTender(pulid.MustNew("org_"))
	transfer.InboundMessageID = pulid.Nil
	fake := &fakeTransferDetail{transfer: transfer}
	tool := newGetEDITransferTool(fake, &fakePermissions{})
	params.Params = map[string]any{paramEDITransferID: transfer.ID.String()}

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	view := result.(*ediTransferView)
	assert.Equal(t, transferOutbound, view.Direction)
	assert.False(t, fake.mapped, "the receiving organization's mappings are its own")
	assert.Nil(t, view.Charges)
	assert.Contains(t, view.Withheld, "charges")
}

type fakeMessageList struct {
	messages []*edi.EDIMessage
	request  *repositories.ListEDIMessagesRequest
}

func (f *fakeMessageList) ListMessagesCursor(
	_ context.Context,
	req *repositories.ListEDIMessagesRequest,
) (*pagination.CursorListResult[*edi.EDIMessage], error) {
	f.request = req

	return &pagination.CursorListResult[*edi.EDIMessage]{Items: f.messages}, nil
}

func TestListEDIMessages_FiltersOnDeliveryAndNeverReturnsTheDocument(t *testing.T) {
	t.Parallel()

	message := &edi.EDIMessage{
		ID:                pulid.MustNew("edimsg_"),
		TransactionSet:    edi.TransactionSet214,
		Direction:         edi.DocumentDirectionOutbound,
		DeliveryStatus:    edi.MessageDeliveryStatusFailed,
		DeliveryLastError: "550 mailbox unavailable",
		RawX12:            "ISA*00*SECRET~",
	}
	fake := &fakeMessageList{messages: []*edi.EDIMessage{message}}
	tool := newListEDIMessagesTool(fake)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		paramDeliveryStatus: "Failed",
		paramDirection:      "Outbound",
	}, ""))
	require.NoError(t, err)

	assert.Equal(t, edi.DocumentDirectionOutbound, fake.request.Direction)
	require.Len(t, fake.request.Filter.FieldFilters, 1)
	outcome := result.(ediCursorOutcome)
	rows := outcome.Items.([]ediMessageListRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "550 mailbox unavailable", rows[0].DeliveryError)
	assert.Len(t, outcome.TaintedRecords(), 1)
	assert.NotContains(t, rows[0].ID, "SECRET")
}

type fakeTenderChangeList struct {
	changes []*edi.TenderChange
	request *repositories.ListEDITenderChangesRequest
}

func (f *fakeTenderChangeList) ListTenderChanges(
	_ context.Context,
	req *repositories.ListEDITenderChangesRequest,
) (*pagination.ListResult[*edi.TenderChange], error) {
	f.request = req

	return &pagination.ListResult[*edi.TenderChange]{Items: f.changes}, nil
}

func TestListEDITenderChanges_SaysWhatChangedAndWhichWaitOnThisOrganization(t *testing.T) {
	t.Parallel()

	params := agentParams(map[string]any{paramStatus: "PendingReview"}, "")
	change := &edi.TenderChange{
		ID:     pulid.MustNew("editcg_"),
		Status: edi.TenderChangeStatusPendingReview,
		Recipient: &edi.TenderRecipient{
			RecipientKind:           edi.TenderRecipientKindInternal,
			RecipientOrganizationID: params.OrganizationID,
		},
		NewTenderPayload: edi.LoadTenderPayload{BOL: "BOL-5"},
		DiffSummary: map[string]any{
			"weight": map[string]any{"previous": 20000, "next": 42000},
			"moves":  map[string]any{"previous": []any{}, "next": []any{map[string]any{}}},
		},
	}
	fake := &fakeTenderChangeList{changes: []*edi.TenderChange{change}}
	tool := newListEDITenderChangesTool(fake)

	result, err := tool.Query(t.Context(), params)
	require.NoError(t, err)

	assert.Equal(t, edi.TenderChangeStatusPendingReview, fake.request.Status)
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
	changes []*edi.TransferChange
	request *repositories.ListEDITransferChangesRequest
}

func (f *fakeTransferChangeList) ListTransferChanges(
	_ context.Context,
	req *repositories.ListEDITransferChangesRequest,
) (*pagination.ListResult[*edi.TransferChange], error) {
	f.request = req

	return &pagination.ListResult[*edi.TransferChange]{Items: f.changes}, nil
}

func TestListEDITransferChanges_GivesTheReportedStatus(t *testing.T) {
	t.Parallel()

	change := &edi.TransferChange{
		ID:         pulid.MustNew("editc_"),
		Status:     edi.TransferChangeStatusPendingReview,
		ChangeType: edi.TransferChangeTypeShipmentStatus214,
		Payload:    map[string]any{"newStatus": "Completed"},
	}
	fake := &fakeTransferChangeList{changes: []*edi.TransferChange{change}}
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

type fakePartnerList struct {
	partners []*edi.EDIPartner
	request  *repositories.ListEDIPartnersRequest
}

func (f *fakePartnerList) ListPartnersCursor(
	_ context.Context,
	req *repositories.ListEDIPartnersRequest,
) (*pagination.CursorListResult[*edi.EDIPartner], error) {
	f.request = req

	return &pagination.CursorListResult[*edi.EDIPartner]{Items: f.partners}, nil
}

func TestListEDIPartners_FindsACustomersPartner(t *testing.T) {
	t.Parallel()

	customerID := pulid.MustNew("cus_")
	fake := &fakePartnerList{partners: []*edi.EDIPartner{{
		ID:         pulid.MustNew("edip_"),
		Code:       "ACME",
		Name:       "Acme Foods",
		CustomerID: customerID,
		Settings:   map[string]any{"apiToken": "tok_live_secret"},
	}}}
	tool := newListEDIPartnersTool(fake)

	result, err := tool.Query(t.Context(), agentParams(map[string]any{
		paramCustomerID: customerID.String(),
	}, ""))
	require.NoError(t, err)

	assert.Equal(t, customerID, fake.request.CustomerID)
	rows := result.(ediCursorOutcome).Items.([]ediPartnerRow)
	require.Len(t, rows, 1)
	assert.Equal(t, "ACME", rows[0].Code)
	assert.Equal(t, agent.ExternalReadNever, tool.Policy().ReadsExternal)
}
