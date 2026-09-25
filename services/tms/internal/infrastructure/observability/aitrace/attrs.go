package aitrace

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/shared/pulid"
	"go.opentelemetry.io/otel/attribute"
)

const (
	GenAIOperationName            = attribute.Key("gen_ai.operation.name")
	GenAIAgentID                  = attribute.Key("gen_ai.agent.id")
	GenAIAgentName                = attribute.Key("gen_ai.agent.name")
	GenAIConversationID           = attribute.Key("gen_ai.conversation.id")
	GenAIProviderName             = attribute.Key("gen_ai.provider.name")
	GenAIRequestModel             = attribute.Key("gen_ai.request.model")
	GenAIRequestMaxTokens         = attribute.Key("gen_ai.request.max_tokens")
	GenAIResponseModel            = attribute.Key("gen_ai.response.model")
	GenAIResponseFinishReasons    = attribute.Key("gen_ai.response.finish_reasons")
	GenAIUsageInputTokens         = attribute.Key("gen_ai.usage.input_tokens")
	GenAIUsageOutputTokens        = attribute.Key("gen_ai.usage.output_tokens")
	GenAIUsageCacheReadTokens     = attribute.Key("gen_ai.usage.cache_read.input_tokens")
	GenAIUsageCacheCreationTokens = attribute.Key("gen_ai.usage.cache_creation.input_tokens")
	GenAIToolName                 = attribute.Key("gen_ai.tool.name")
	GenAIToolCallID               = attribute.Key("gen_ai.tool.call.id")
	GenAIToolType                 = attribute.Key("gen_ai.tool.type")
	ServerAddress                 = attribute.Key("server.address")
	ErrorType                     = attribute.Key("error.type")
	UserID                        = attribute.Key("user.id")
)

const (
	OperationInvokeAgent = "invoke_agent"
	OperationChat        = "chat"
	OperationEmbeddings  = "embeddings"
	OperationExecuteTool = "execute_tool"
	ToolTypeFunction     = "function"
)

const (
	AIAnchor               = attribute.Key("trenova.ai.anchor")
	AIAgentVersion         = attribute.Key("trenova.ai.agent.version")
	AIOwnerKind            = attribute.Key("trenova.ai.owner.kind")
	AIOwnerID              = attribute.Key("trenova.ai.owner.id")
	AIRunID                = attribute.Key("trenova.ai.run.id")
	AITurnID               = attribute.Key("trenova.ai.turn.id")
	AITrigger              = attribute.Key("trenova.ai.trigger")
	AIStatus               = attribute.Key("trenova.ai.status")
	AITokensInput          = attribute.Key("trenova.ai.tokens.input")
	AITokensOutput         = attribute.Key("trenova.ai.tokens.output")
	AICostUSD              = attribute.Key("trenova.ai.cost_usd")
	AIToolCalls            = attribute.Key("trenova.ai.tool_calls")
	AITainted              = attribute.Key("trenova.ai.tainted")
	AITaintSources         = attribute.Key("trenova.ai.taint.sources")
	AISimulation           = attribute.Key("trenova.ai.simulation")
	AIPurpose              = attribute.Key("trenova.ai.purpose")
	AIFeature              = attribute.Key("trenova.ai.feature")
	AIDelegateCallID       = attribute.Key("trenova.ai.delegate.call_id")
	AIDelegateAgentID      = attribute.Key("trenova.ai.delegate.agent.id")
	AIDelegateAgentName    = attribute.Key("trenova.ai.delegate.agent.name")
	AIDelegateDeclined     = attribute.Key("trenova.ai.delegate.declined")
	AIActivityAttempt      = attribute.Key("trenova.ai.activity.attempt")
	AIStream               = attribute.Key("trenova.ai.stream")
	AILooped               = attribute.Key("trenova.ai.looped")
	AIAttempts             = attribute.Key("trenova.ai.attempts")
	AIFailover             = attribute.Key("trenova.ai.failover")
	AIMidReplyRestarts     = attribute.Key("trenova.ai.mid_reply_restarts")
	AIProviderID           = attribute.Key("trenova.ai.provider.id")
	AIAttempt              = attribute.Key("trenova.ai.attempt")
	AIReasoningTokens      = attribute.Key("trenova.ai.reasoning_tokens")
	AITruncated            = attribute.Key("trenova.ai.truncated")
	AIToolEffect           = attribute.Key("trenova.ai.tool.effect")
	AIToolKind             = attribute.Key("trenova.ai.tool.kind")
	AIStepKey              = attribute.Key("trenova.ai.step.key")
	AIStepState            = attribute.Key("trenova.ai.step.state")
	AITier                 = attribute.Key("trenova.ai.tier")
	AITierSource           = attribute.Key("trenova.ai.tier.source")
	AIHeldBy               = attribute.Key("trenova.ai.held_by")
	AIEgressClass          = attribute.Key("trenova.ai.egress_class")
	AIAfterExternalContent = attribute.Key("trenova.ai.after_external_content")
	AIOutcome              = attribute.Key("trenova.ai.outcome")
	AIProposalID           = attribute.Key("trenova.ai.proposal.id")
	AIEntityType           = attribute.Key("trenova.ai.entity.type")
	AIEntityID             = attribute.Key("trenova.ai.entity.id")
	AIVersionBefore        = attribute.Key("trenova.ai.version.before")
	AIVersionAfter         = attribute.Key("trenova.ai.version.after")
	AISimulated            = attribute.Key("trenova.ai.simulated")
	AIDecision             = attribute.Key("trenova.ai.decision")
	AIReasonCode           = attribute.Key("trenova.ai.reason_code")
	AIModificationCount    = attribute.Key("trenova.ai.modification_count")
	AIJobFeature           = attribute.Key("trenova.ai.job.feature")
	AIPreviewCoverage      = attribute.Key("trenova.ai.preview.coverage")
	AIPreviewStale         = attribute.Key("trenova.ai.preview.stale")
	AIPreviewRecords       = attribute.Key("trenova.ai.preview.records")
	AIPreviewWithheld      = attribute.Key("trenova.ai.preview.withheld")
	AIPreviewRecorded      = attribute.Key("trenova.ai.preview.recorded")
	AIPreviewPurpose       = attribute.Key("trenova.ai.preview.purpose")
	AIPreviewDigest        = attribute.Key("trenova.ai.preview.digest")
	AIPreviewReviewed      = attribute.Key("trenova.ai.preview.reviewed")
	TenantOrganizationID   = attribute.Key("trenova.tenant.organization_id")
	TenantBusinessUnitID   = attribute.Key("trenova.tenant.business_unit_id")
)

const (
	EventProviderBusyWait = "provider.busy_wait"
	EventProviderResting  = "provider.resting"
	EventWaitSeconds      = attribute.Key("wait_s")
)

const (
	StepStateFresh    = "fresh"
	StepStateReplayed = "replayed"
	StepStateUnknown  = "unknown"
)

const (
	OutcomeRan        = "ran"
	OutcomeProposed   = "proposed"
	OutcomeSimulated  = "simulated"
	OutcomeDenied     = "denied"
	OutcomeInvalid    = "invalid"
	OutcomeDuplicate  = "duplicate"
	OutcomeOverBudget = "over_budget"
	OutcomeFailed     = "failed"
	OutcomeUnknown    = "unknown"
)

const (
	PurposeLive       = "live"
	PurposeEvaluation = "evaluation"
)

const (
	SpanCompletion   = "trenova.ai.completion"
	SpanWrite        = "trenova.ai.write"
	SpanDelegateOpen = "trenova.ai.delegate.open"
	SpanJob          = "trenova.ai.job"
	SpanPreview      = "trenova.ai.preview"
)

const (
	providerAnthropic = "anthropic"
	providerOpenAI    = "openai"
	providerOllama    = "ollama"
)

func ProviderName(kind aiprovider.Kind) string {
	switch kind {
	case aiprovider.KindAnthropicMessages:
		return providerAnthropic
	case aiprovider.KindOpenAIResponses, aiprovider.KindOpenAIChat:
		return providerOpenAI
	case aiprovider.KindOllama:
		return providerOllama
	}

	return strings.ToLower(string(kind))
}

func Tenant(organizationID, businessUnitID pulid.ID) []attribute.KeyValue {
	return []attribute.KeyValue{
		TenantOrganizationID.String(organizationID.String()),
		TenantBusinessUnitID.String(businessUnitID.String()),
	}
}

func spanName(operation, subject string) string {
	if subject == "" {
		return operation
	}

	return operation + " " + subject
}
