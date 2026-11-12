package completioncontract

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
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
	WriteCaseReadingWorkflowName     = "WriteCaseReadingWorkflow"
)

const (
	// caseReadingTimeout bounds writing one case's reading, which nobody
	// waits on: the case shows what was computed until it lands.
	caseReadingTimeout = 3 * time.Minute

	// DefaultWait is how long a caller without a deadline of its own waits,
	// the API's own request timeout.
	DefaultWait = 55 * time.Second

	// minWait is the least a call is given, so a request that arrives with
	// its deadline all but spent still gets one honest attempt.
	minWait = 5 * time.Second

	// answerMargin is kept back from the request's deadline so the handler
	// can still answer before the request itself times out.
	answerMargin = 2 * time.Second

	// abandonTimeout bounds cancelling a call nobody is waiting for any
	// more.
	abandonTimeout = 5 * time.Second
)

type StructuredCompletionPayload struct {
	Request *llmports.StructuredCompletionRequest `json:"request"`
}

type TestAIProviderPayload struct {
	Request aiproviderports.GetAIProviderByIDRequest `json:"request"`
}

type ListAIProviderModelsPayload struct {
	Request *aiproviderports.ProbeAIProviderModelsRequest `json:"request"`
}

type TestAIProviderDraftPayload struct {
	Request *aiproviderports.ProbeAIProviderDraftRequest `json:"request"`
}

type WriteBriefingPayload struct {
	Request briefingports.WriteBriefingRequest `json:"request"`
}

type WriteCaseReadingPayload struct {
	Request *recordcaseports.WriteReadingRequest `json:"request"`
}
