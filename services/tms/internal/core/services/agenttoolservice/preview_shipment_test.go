package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tender"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/ediservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakePartnerNotices struct {
	original, updated *shipment.Shipment
}

func (f *fakePartnerNotices) PreviewTenderChanges(
	_ context.Context,
	original, updated *shipment.Shipment,
	_ *serviceports.RequestActor,
) ([]*ediservice.PartnerNotice, error) {
	f.original, f.updated = original, updated

	return []*ediservice.PartnerNotice{{
		RecordID:       pulid.MustNew("etr_"),
		PartnerName:    "Halstead Grocery EDI",
		TransactionSet: edi.TransactionSet204,
		Purpose:        "Change",
		Changed:        []string{"bol"},
	}}, nil
}

func (f *fakePartnerNotices) PreviewCancelNotices(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) ([]*ediservice.PartnerNotice, error) {
	return []*ediservice.PartnerNotice{{
		RecordID:       pulid.MustNew("esl_"),
		PartnerName:    "Linked trading partner",
		TransactionSet: edi.TransactionSet214,
		Purpose:        "Cancel",
		StatusCode:     "A7",
	}}, nil
}

type fakeLiveTenders struct{}

func (fakeLiveTenders) PreviewLiveTenders(
	context.Context,
	pagination.TenantInfo,
	pulid.ID,
) ([]*tender.Tender, error) {
	return []*tender.Tender{{
		ID:     pulid.MustNew("tnd_"),
		Mode:   tender.ModeWaterfall,
		Status: tender.StatusActive,
		Offers: []*tender.TenderOffer{
			{
				CarrierID:      pulid.MustNew("carr_"),
				Status:         tender.OfferStatusSent,
				Channel:        tender.ChannelEmail,
				RecipientEmail: "dispatch@acme.example",
			},
			{CarrierID: pulid.MustNew("carr_"), Status: tender.OfferStatusPending},
		},
	}}, nil
}

func (f *fakeShipmentService) PreviewCancel(
	_ context.Context,
	req *repositories.CancelShipmentRequest,
	_ *serviceports.RequestActor,
) (*serviceports.ShipmentCancelPreview, error) {
	before := &shipment.Shipment{
		ID:        req.ShipmentID,
		ProNumber: "SHP-3001",
		Status:    shipment.StatusAssigned,
		Version:   9,
	}
	after := *before
	after.ApplyCancel(req.CanceledByID, req.CanceledAt, req.CancelReason)

	return &serviceports.ShipmentCancelPreview{Before: before, After: &after}, nil
}

func TestUpdateShipmentPreview_IsThePatchExecuteSaves(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	weight := int64(40000)
	writer := &fakeShipmentWriter{existing: &shipment.Shipment{
		ID:        shipmentID,
		ProNumber: "SHP-2001",
		BOL:       "BOL-1",
		Weight:    &weight,
		Version:   5,
	}}
	partners := &fakePartnerNotices{}
	tool := newUpdateShipmentTool(writer, partners)
	customerID := pulid.MustNew("cus_")
	params := executeParams(map[string]any{
		"shipmentId": shipmentID.String(),
		"bol":        "BOL-2",
		"weight":     42000,
		"customerId": customerID.String(),
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, writer.updated, "a preview must not save")
	assert.Equal(t, "BOL-1", writer.existing.BOL, "a preview must not touch what it read")

	change := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, permission.ResourceShipment, change.Resource)
	assert.Equal(t, int64(5), *change.Version)
	bol := findField(t, change, "bol")
	assert.Equal(t, "BOL-1", bol.Before)
	assert.Equal(t, "BOL-2", bol.After)
	customer := findField(t, change, "customerId")
	require.NotNil(t, customer.AfterRef)
	assert.Equal(t, permission.ResourceCustomer, customer.AfterRef.Resource)

	send := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, agent.MessageChannelEDI, send.Message.Channel)
	assert.Equal(t, []string{"Halstead Grocery EDI"}, send.Message.To)
	assert.Equal(t, "Changed: bol.", send.Message.Body)
	assert.Equal(t, "BOL-2", partners.updated.BOL)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, writer.updated)
	assert.Equal(t, bol.After, writer.updated.BOL)
	assert.Equal(t, customerID.String(), customer.After)
	assert.Equal(t, int64(42000), *writer.updated.Weight)
}

func TestCancelShipmentPreview_ShowsWhoIsToldAndWhatIsWithdrawn(t *testing.T) {
	t.Parallel()

	shipments := &fakeShipmentService{}
	tool := newCancelShipmentTool(shipments, &fakePartnerNotices{}, fakeLiveTenders{})
	params := executeParams(map[string]any{
		"shipmentId":   pulid.MustNew("shp_").String(),
		"cancelReason": "Customer pulled the load",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, shipments.canceled, "a preview must not cancel")
	assert.Contains(t, preview.Summary, "SHP-3001")
	assert.Contains(t, preview.Summary, "1 live tender would be withdrawn")

	status := findChange(t, preview, agent.PreviewOperationUpdate)
	assert.Equal(t, "Canceled", findField(t, status, "status").After)
	assert.Equal(t, "Customer pulled the load", findField(t, status, "cancelReason").After)
	assert.True(t, findField(t, status, "canceledAt").Volatile)

	var channels []agent.MessageChannel
	for i := range preview.Changes {
		if message := preview.Changes[i].Message; message != nil {
			channels = append(channels, message.Channel)
		}
	}
	assert.Equal(t, []agent.MessageChannel{agent.MessageChannelEDI, agent.MessageChannelEmail}, channels,
		"the linked partner is sent the 214, and only the carrier holding an offer is told")
	withdrawn := findChange(t, preview, agent.PreviewOperationArchive)
	assert.Equal(t, permission.ResourceTender, withdrawn.Resource)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, shipments.canceled)
	assert.Equal(t, "Customer pulled the load", shipments.canceled.CancelReason)
}

func TestAddShipmentCommentPreview_ShowsTheNoteAndWhoReadsIt(t *testing.T) {
	t.Parallel()

	comments := &fakeCommentService{}
	tool := newAddShipmentCommentTool(comments)
	params := executeParams(map[string]any{
		"shipmentId": pulid.MustNew("shp_").String(),
		"comment":    "Delivery moved to 3 PM at the receiver's request.",
		"visibility": "Customer",
	})

	preview, err := tool.(serviceports.ToolPreviewer).Preview(t.Context(), params)
	require.NoError(t, err)
	require.Nil(t, comments.created, "a preview must not comment")

	created := findChange(t, preview, agent.PreviewOperationCreate)
	assert.Equal(t, permission.ResourceShipmentComment, created.Resource)
	assert.Equal(t, "Customer", findField(t, created, "visibility").After)
	shared := findChange(t, preview, agent.PreviewOperationSend)
	assert.Equal(t, agent.MessageChannelComment, shared.Message.Channel)
	assert.Equal(t, "Customer", shared.Message.Visibility)
	assert.Equal(t, []string{"The customer"}, shared.Message.To)

	require.NoError(t, tool.Execute(t.Context(), params))
	require.NotNil(t, comments.created)
	assert.Equal(t, comments.created.Comment, shared.Message.Body)
	assert.Equal(t, string(comments.created.Type), findField(t, created, "type").After)
}
