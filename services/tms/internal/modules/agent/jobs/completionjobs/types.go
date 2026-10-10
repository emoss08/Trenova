// Package completionjobs runs the model calls a person waits on outside a
// conversation: composing a table's filters, writing or explaining a formula,
// rewriting today's briefing, and testing a provider. Each runs as a short
// workflow on the chat queue, and the request that asked waits for it.
package completionjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
)

const (
	jobFeatureStructured   = "structured_completion"
	jobFeatureProviderTest = "provider_test"
	jobFeatureModelList    = "provider_models"
	jobFeatureBriefing     = "briefing_write"
	jobFailureAbandoned    = "abandoned"
	jobFailureFailed       = "failed"
)

const (
	StructuredCompletionWorkflowName = "StructuredCompletionWorkflow"
	TestAIProviderWorkflowName       = "TestAIProviderWorkflow"
	ListAIProviderModelsWorkflowName = "ListAIProviderModelsWorkflow"
	TestAIProviderDraftWorkflowName  = "TestAIProviderDraftWorkflow"
	WriteBriefingWorkflowName        = "WriteBriefingWorkflow"
)

const (
	// defaultWait is how long a caller without a deadline of its own waits,
	// the API's own request timeout.
	defaultWait = 55 * time.Second

	// minWait is the least a call is given, so a request that arrives with
	// its deadline all but spent still gets one honest attempt.
	minWait = 5 * time.Second

	// answerMargin is kept back from the request's deadline so the handler
	// can still answer before the request itself times out.
	answerMargin = 2 * time.Second

	// callAttempts bounds Temporal's retries of a one-shot model call inside
	// the time the person is waiting.
	callAttempts = 3

	// writeAttempts bounds retries of a briefing write. Writing a day again
	// replaces it, so a retry is safe; two is enough for a lost worker.
	writeAttempts = 2

	// abandonTimeout bounds cancelling a call nobody is waiting for any
	// more.
	abandonTimeout = 5 * time.Second
)

type StructuredCompletionPayload struct {
	Request *agentports.StructuredCompletionRequest `json:"request"`
}

type TestAIProviderPayload struct {
	Request agentports.GetAIProviderByIDRequest `json:"request"`
}

type ListAIProviderModelsPayload struct {
	Request *agentports.ProbeAIProviderModelsRequest `json:"request"`
}

type TestAIProviderDraftPayload struct {
	Request *agentports.ProbeAIProviderDraftRequest `json:"request"`
}

type WriteBriefingPayload struct {
	Request briefingports.WriteBriefingRequest `json:"request"`
}
