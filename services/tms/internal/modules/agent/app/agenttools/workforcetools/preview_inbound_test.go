package workforcetools

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/documents/app/inboundmessageservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type savingInboundDesk struct {
	message *inboundmessage.InboundMessage
	linkErr error
	guard   agenttoolstest.WriteGuard
}

func (d *savingInboundDesk) GetByID(
	context.Context,
	inboundmessageports.GetInboundMessageByIDRequest,
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
	if err := d.guard.Write(); err != nil {
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
	if err := d.guard.Write(); err != nil {
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
	tool := NewLinkInboundMessageTool(desk).(*LinkInboundMessageTool)
	shipmentID := pulid.MustNew("shp_")
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId":  message.ID.String(),
		"shipmentId": shipmentID.String(),
		"reason":     "The PRO in the subject is this load's.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, permission.ResourceInboundMessage, change.Resource)
	shipment := agenttoolstest.FieldByPath(t, change, "matchedShipmentId")
	assert.Equal(t, "Shipment", shipment.Label)
	assert.Nil(t, shipment.Before)
	require.NotNil(t, shipment.AfterRef)
	assert.Equal(t, permission.ResourceShipment, shipment.AfterRef.Resource)
	assert.Equal(t, shipmentID, shipment.AfterRef.ID)
	assert.Equal(t, "Sender domain", agenttoolstest.FieldByPath(t, change, "matchReason").Before)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, desk.message, inboundLinkOptions()...)
}

func TestLinkInboundMessage_PreviewWarnsOnARecordTheServiceCannotFind(t *testing.T) {
	t.Parallel()

	desk := &savingInboundDesk{
		message: inboundMessage(),
		linkErr: errortypes.NewValidationError("shipmentId", errortypes.ErrInvalid, "No shipment"),
	}
	tool := NewLinkInboundMessageTool(desk).(*LinkInboundMessageTool)

	preview := agenttoolstest.PreviewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.DeskParams(map[string]any{
			"messageId":  desk.message.ID.String(),
			"shipmentId": pulid.MustNew("shp_").String(),
			"reason":     "Guessing.",
		}))
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}

func TestMarkInboundMessage_PreviewMatchesWhatIsSaved(t *testing.T) {
	t.Parallel()

	message := inboundMessage()
	before := *message
	desk := &savingInboundDesk{message: message}
	tool := NewMarkInboundMessageTool(desk).(*MarkInboundMessageTool)
	params := agenttoolstest.DeskParams(map[string]any{
		"messageId": message.ID.String(),
		"status":    "Ignored",
		"note":      "A newsletter; nothing to do.",
	})

	preview := agenttoolstest.PreviewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), params)
	})

	change := agenttoolstest.PreviewChange(t, preview, 0)
	assert.Equal(t, agent.PreviewOperationArchive, change.Operation)
	assert.Equal(t, "InReview", agenttoolstest.FieldByPath(t, change, "status").Before)
	assert.Equal(t, "Ignored", agenttoolstest.FieldByPath(t, change, "status").After)
	assert.Equal(t, "A newsletter; nothing to do.", agenttoolstest.FieldByPath(t, change, "reviewNote").After)

	require.NoError(t, tool.Execute(t.Context(), params))
	agenttoolstest.RequireUpdateParity(t, change, &before, desk.message,
		toolpreview.Only(inboundReviewFields...), toolpreview.Volatile("reviewedAt"))
}

func TestMarkInboundMessage_PreviewWarnsOnAQuarantinedMessage(t *testing.T) {
	t.Parallel()

	message := inboundMessage()
	message.Status = inboundmessage.StatusQuarantined
	desk := &savingInboundDesk{message: message}
	tool := NewMarkInboundMessageTool(desk).(*MarkInboundMessageTool)

	preview := agenttoolstest.PreviewWithoutWrites(t, &desk.guard, func() (*agent.ToolPreview, error) {
		return tool.Preview(t.Context(), agenttoolstest.DeskParams(map[string]any{
			"messageId": message.ID.String(),
			"status":    "Actioned",
			"note":      "done",
		}))
	})

	assert.Empty(t, preview.Changes)
	agenttoolstest.RequireWarning(t, preview, agent.PreviewWarningWouldFail)
}
