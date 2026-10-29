package agentevents_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentevents"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type recordingAttachments struct {
	seen []pulid.ID
}

func (r *recordingAttachments) AttachmentExtracted(
	_ context.Context,
	event *agentports.AgentEvent,
) {
	r.seen = append(r.seen, event.SubjectID)
}

/*
An attached file's card is shown whether or not the organization's plan has
automated agents: the person asked for it by attaching the file. Observers are
bound once the graph is built, and only an extraction reaches them.
*/
func TestPublish_TellsAttachmentObserversOfEachExtraction(t *testing.T) {
	t.Parallel()

	publisher := agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Definitions: &fakeDefinitions{},
		RunService:  &fakeRunService{open: map[pulid.ID]bool{}},
	})
	observer := &recordingAttachments{}
	agentevents.RegisterAttachmentObservers(agentevents.ObserverParams{
		Publisher: publisher,
		Observers: []agentports.AttachmentExtractionObserver{observer, nil},
	})
	extracted := pulid.MustNew("doc_")

	publisher.Publish(t.Context(), agentports.AgentEvent{
		Kind:       agenttrigger.EventDocumentExtracted,
		SubjectID:  extracted,
		TenantInfo: tenantInfo,
	})
	publisher.Publish(t.Context(), agentports.AgentEvent{
		Kind:       agenttrigger.EventBillingQueueItemException,
		SubjectID:  pulid.MustNew("bqi_"),
		TenantInfo: tenantInfo,
	})

	require.Len(t, observer.seen, 1)
	assert.Equal(t, extracted, observer.seen[0])
}

func TestPublish_WithNoAttachmentObserversStillPublishes(t *testing.T) {
	t.Parallel()

	runs := &fakeRunService{open: map[pulid.ID]bool{}}
	publisher := agentevents.New(agentevents.Params{
		Logger:      zap.NewNop(),
		Definitions: &fakeDefinitions{},
		RunService:  runs,
	})

	assert.NotPanics(t, func() {
		publisher.Publish(t.Context(), agentports.AgentEvent{
			Kind:       agenttrigger.EventDocumentExtracted,
			SubjectID:  pulid.MustNew("doc_"),
			TenantInfo: tenantInfo,
		})
	})
}
