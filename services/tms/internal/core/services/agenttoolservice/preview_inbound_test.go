package agenttoolservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingInboundDesk struct {
	message *inboundmessage.InboundMessage
	linkErr error
	guard   writeGuard
}

func (d *savingInboundDesk) GetByID(
	context.Context,
	repositories.GetInboundMessageByIDRequest,
) (*inboundmessage.InboundMessage, error) {
	copied := *d.message

	return &copied, nil
}

func (d *savingInboundDesk) CheckLink(context.Context, inboundmessageservice.LinkRequest) error {
	return d.linkErr
}

func (d *savingInboundDesk) Link(
	_ context.Context,
	req inboundmessageservice.LinkRequest,
) (*inboundmessage.InboundMessage, error) {
	if err := d.guard.write(); err != nil {
		return nil, err
	}
	if d.linkErr != nil {
		return nil, d.linkErr
	}
	inboundmessageservice.ApplyLink(d.message, &req)

	return d.message, nil
}

func (d *savingInboundDesk) Review(
	_ context.Context,
	req inboundmessageservice.ReviewRequest,
) (*inboundmessage.InboundMessage, error) {
	if err := d.guard.write(); err != nil {
		return nil, err
	}
	if err := inboundmessageservice.ApplyReview(d.message, &req, timeutils.NowUnix()); err != nil {
		return nil, err
	}

	return d.message, nil
}

func TestLinkInboundMessage_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	message := inboundMessage()
	message.MatchReason = "Sender domain"
	before := *message
	desk := &savingInboundDesk{message: message}
	tool := newLinkInboundMessageTool(desk).(*linkInboundMessageTool)
	shipmentID := pulid.MustNew("shp_")
	params := deskParams(map[string]any{
		"messageId":  message.ID.String(),
		"shipmentId": shipmentID.String(),
		"reason":     "The PRO in the subject is this load's.",
	})

	preview := previewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInboundMessage, change.Resource)
	shipment := fieldByPath(t, change, "matchedShipmentId")
	assert.Equal(t, "Shipment", shipment.Label)
	assert.Nil(t, shipment.Before)
	require.NotNil(t, shipment.AfterRef)
	assert.Equal(t, permission.ResourceShipment, shipment.AfterRef.Resource)
	assert.Equal(t, shipmentID, shipment.AfterRef.ID)
	assert.Equal(t, "Sender domain", fieldByPath(t, change, "matchReason").Before)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, desk.message, inboundLinkOptions()...)
}

func TestLinkInboundMessage_PreviewWarnsOnARecordTheServiceCannotFind(t *testing.T) {
	t.Parallel()

	desk := &savingInboundDesk{
		message: inboundMessage(),
		linkErr: errortypes.NewValidationError("shipmentId", errortypes.ErrInvalid, "No shipment"),
	}
	tool := newLinkInboundMessageTool(desk).(*linkInboundMessageTool)

	preview := previewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), deskParams(map[string]any{
			"messageId":  desk.message.ID.String(),
			"shipmentId": pulid.MustNew("shp_").String(),
			"reason":     "Guessing.",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestMarkInboundMessage_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	message := inboundMessage()
	before := *message
	desk := &savingInboundDesk{message: message}
	tool := newMarkInboundMessageTool(desk).(*markInboundMessageTool)
	params := deskParams(map[string]any{
		"messageId": message.ID.String(),
		"status":    "Ignored",
		"note":      "A newsletter; nothing to do.",
	})

	preview := previewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := previewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
	assert.Equal(t, "InReview", fieldByPath(t, change, "status").Before)
	assert.Equal(t, "Ignored", fieldByPath(t, change, "status").After)
	assert.Equal(t, "A newsletter; nothing to do.", fieldByPath(t, change, "reviewNote").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	requireUpdateParity(t, change, &before, desk.message,
		toolpreview.Only(inboundReviewFields...), toolpreview.Volatile("reviewedAt"))
}

func TestMarkInboundMessage_PreviewWarnsOnAQuarantinedMessage(t *testing.T) {
	t.Parallel()

	message := inboundMessage()
	message.Status = inboundmessage.StatusQuarantined
	desk := &savingInboundDesk{message: message}
	tool := newMarkInboundMessageTool(desk).(*markInboundMessageTool)

	preview := previewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), deskParams(map[string]any{
			"messageId": message.ID.String(),
			"status":    "Actioned",
			"note":      "done",
		}))
	})

	assert.Empty(t, preview.Changes)
	requireWarning(t, preview, agent.PreviewWarningWouldFail)
}
