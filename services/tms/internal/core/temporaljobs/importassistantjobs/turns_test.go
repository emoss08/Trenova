package importassistantjobs

import (
	"context"
	"errors"
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
)

type fakeRun struct {
	client.WorkflowRun
	id  string
	get func(result any) error
}

func (r fakeRun) GetID() string { return r.id }

func (r fakeRun) Get(_ context.Context, result any) error { return r.get(result) }

type fakeStarter struct {
	serviceports.WorkflowStarter
	options  []client.StartWorkflowOptions
	payloads []*TurnPayload
	err      error
	get      func(result any) error
}

func (f *fakeStarter) StartWorkflow(
	_ context.Context,
	options client.StartWorkflowOptions,
	_ any,
	args ...any,
) (client.WorkflowRun, error) {
	f.options = append(f.options, options)
	f.payloads = append(f.payloads, args[0].(*TurnPayload))
	if f.err != nil {
		return nil, f.err
	}

	return fakeRun{id: options.ID, get: f.get}, nil
}

type fakeReader struct {
	frames []serviceports.TurnStreamFrame
	ref    serviceports.TurnStreamRef
}

func (f *fakeReader) Read(_ context.Context, req serviceports.ReadTurnStreamRequest) error {
	f.ref = req.Ref
	for _, frame := range f.frames {
		if err := req.OnFrame(frame); err != nil {
			return err
		}
	}

	return nil
}

func chatRequest() *serviceports.ShipmentImportChatRequest {
	return &serviceports.ShipmentImportChatRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: pulid.MustNew("org_"),
			BuID:  pulid.MustNew("bu_"),
		},
		UserMessage: "Set the customer",
		DocumentID:  "doc_1",
	}
}

// A document answers one message at a time, on the chat queue, and the tenant
// the request's JSON leaves out travels beside it.
func TestTurns_StartsOneTurnPerDocumentOnTheChatQueue(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{get: func(result any) error {
		*result.(*serviceports.ShipmentImportChatResponse) = serviceports.ShipmentImportChatResponse{
			Message: "Done.",
		}

		return nil
	}}
	turns := &Turns{workflows: starter}
	req := chatRequest()

	response, err := turns.Chat(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, "Done.", response.Message)

	require.Len(t, starter.options, 1)
	options := starter.options[0]
	assert.Equal(t, "import-assistant/doc_1", options.ID)
	assert.Equal(t, temporaltype.TaskQueueAgentChat.String(), options.TaskQueue)
	assert.Equal(t, enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL, options.WorkflowIDConflictPolicy)
	assert.True(t, options.WorkflowExecutionErrorWhenAlreadyStarted)
	assert.Equal(t, req.TenantInfo, starter.payloads[0].TenantInfo)
	assert.False(t, starter.payloads[0].Stream)
}

func TestTurns_RefusesASecondMessageWhileTheFirstIsAnswered(t *testing.T) {
	t.Parallel()

	turns := &Turns{workflows: &fakeStarter{
		err: serviceerror.NewWorkflowExecutionAlreadyStarted("running", "", ""),
	}}

	_, err := turns.Chat(t.Context(), chatRequest())
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Contains(t, business.Error(), "still answering")
}

// A turn that failed is reported in the words it wrote for the person.
func TestTurns_ReportsAFailedTurnInItsOwnWords(t *testing.T) {
	t.Parallel()

	turns := &Turns{workflows: &fakeStarter{get: func(any) error {
		return temporal.NewNonRetryableApplicationError(
			"No AI provider is configured.",
			errTypeTurnFailed,
			nil,
		)
	}}}

	_, err := turns.Chat(t.Context(), chatRequest())
	var business *errortypes.BusinessError
	require.ErrorAs(t, err, &business)
	assert.Equal(t, "No AI provider is configured.", business.Error())

	plain := errors.New("temporal unreachable")
	turns = &Turns{workflows: &fakeStarter{err: plain}}
	_, err = turns.Chat(t.Context(), chatRequest())
	assert.ErrorIs(t, err, plain)
}

// The reply is relayed as it was written, frame for frame.
func TestTurns_RelaysTheStreamToTheReader(t *testing.T) {
	t.Parallel()

	reader := &fakeReader{frames: []serviceports.TurnStreamFrame{
		{ID: "0", Event: "text_delta", Data: []byte(`{"delta":"Hi"}`)},
		{ID: "1", Event: "done", Data: []byte(`{"conversationId":"sic_1","actions":[]}`)},
	}}
	starter := &fakeStarter{get: func(any) error {
		t.Fatal("a stream that reached its end needs no result")
		return nil
	}}
	turns := &Turns{workflows: starter, reader: reader}

	var events []string
	err := turns.ChatStream(t.Context(), chatRequest(), func(event serviceports.StreamEvent) {
		events = append(events, event.Event)
	})
	require.NoError(t, err)
	assert.Equal(t, []string{"text_delta", "done"}, events)
	assert.Equal(t, "import-assistant/doc_1", reader.ref.WorkflowID)
	assert.True(t, starter.payloads[0].Stream)
}

// A reader who arrives after the turn closed still hears how it ended.
func TestTurns_EndsAStreamTheTurnOutran(t *testing.T) {
	t.Parallel()

	starter := &fakeStarter{get: func(result any) error {
		*result.(*serviceports.ShipmentImportChatResponse) = serviceports.ShipmentImportChatResponse{
			ConversationID: "sic_1",
		}

		return nil
	}}
	turns := &Turns{workflows: starter, reader: &fakeReader{}}

	var events []serviceports.StreamEvent
	err := turns.ChatStream(t.Context(), chatRequest(), func(event serviceports.StreamEvent) {
		events = append(events, event)
	})
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, "done", events[0].Event)
}
