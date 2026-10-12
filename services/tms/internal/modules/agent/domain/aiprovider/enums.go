package aiprovider

import (
	"slices"

	"github.com/shopspring/decimal"
)

// Kind identifies the wire protocol a provider speaks, not the vendor behind it.
// Modelling the protocol is what makes the long tail reachable: every runtime that
// exposes an OpenAI-compatible /chat/completions endpoint — vLLM, SGLang, LM
// Studio, OpenRouter, Groq, Together, Fireworks, DeepInfra, and Bedrock's
// chat-completions endpoint — is served by KindOpenAIChat alone.
type Kind string

const (
	KindAnthropicMessages = Kind("AnthropicMessages")
	KindOpenAIResponses   = Kind("OpenAIResponses")
	KindOpenAIChat        = Kind("OpenAIChat")
	// KindOllama exists despite Ollama exposing an OpenAI-compatible endpoint,
	// because that endpoint drops response_format.json_schema on the floor
	// (ollama/ollama#10001) — the request succeeds and returns unconstrained prose,
	// which is the worst way for a schema to fail. Ollama's native /api/chat takes
	// a `format` object and does real grammar-constrained decoding, so routing it
	// here buys constrained output instead of a silent downgrade to prompting.
	KindOllama = Kind("Ollama")
)

// CachesOutsideInput reports whether the protocol counts cached prompt tokens
// apart from its input tokens. Anthropic reports cache reads and writes beside
// input_tokens; the OpenAI protocols count cached tokens inside them.
func (k Kind) CachesOutsideInput() bool {
	return k == KindAnthropicMessages
}

// CacheReadMultiple is the share of the input price a cached prompt token
// costs when no price was entered for it. Anthropic bills a cache read at a
// tenth of input; OpenAI's discount runs from half to a tenth by model, and a
// tenth is its rate for the current models, so an older model's operator
// enters its own.
func (k Kind) CacheReadMultiple() decimal.Decimal {
	return cacheReadMultiple
}

// CacheWriteMultiple is the multiple of the input price a prompt token written
// to the cache costs when no price was entered for it: Anthropic bills a
// five-minute cache write at one and a quarter times input; the other
// protocols report no writes, and a write they did report is input.
func (k Kind) CacheWriteMultiple() decimal.Decimal {
	if k == KindAnthropicMessages {
		return anthropicCacheWriteMultiple
	}

	return decimal.NewFromInt(1)
}

var (
	cacheReadMultiple           = decimal.RequireFromString("0.1")
	anthropicCacheWriteMultiple = decimal.RequireFromString("1.25")
)

func (k Kind) IsValid() bool {
	switch k {
	case KindAnthropicMessages, KindOpenAIResponses, KindOpenAIChat, KindOllama:
		return true
	default:
		return false
	}
}

// DefaultBaseURL is the endpoint used when a provider leaves BaseURL empty. Only
// the hosted first-party protocols have one; an OpenAI-compatible provider is
// self-hosted or third-party by definition and must name its own endpoint.
func (k Kind) DefaultBaseURL() string {
	switch k {
	case KindAnthropicMessages:
		return "https://api.anthropic.com"
	case KindOpenAIResponses:
		return "https://api.openai.com"
	case KindOllama:
		return "http://localhost:11434"
	case KindOpenAIChat:
		return ""
	default:
		return ""
	}
}

// RequiresAPIKey reports whether a credential must be present. A self-hosted
// runtime on a trusted network commonly has no auth at all, so KindOpenAIChat
// treats the key as optional rather than forcing operators to invent one.
func (k Kind) RequiresAPIKey() bool {
	return k == KindAnthropicMessages || k == KindOpenAIResponses
}

func (k Kind) SupportsEmbedding() bool {
	switch k {
	case KindOpenAIResponses, KindOpenAIChat, KindOllama:
		return true
	case KindAnthropicMessages:
		return false
	default:
		return false
	}
}

type EmbeddingInputStyle string

const (
	EmbeddingInputStyleNone            = EmbeddingInputStyle("None")
	EmbeddingInputStyleVoyageInputType = EmbeddingInputStyle("VoyageInputType")
	EmbeddingInputStyleNomicPrefix     = EmbeddingInputStyle("NomicPrefix")
)

func (s EmbeddingInputStyle) IsValid() bool {
	switch s {
	case EmbeddingInputStyleNone,
		EmbeddingInputStyleVoyageInputType,
		EmbeddingInputStyleNomicPrefix:
		return true
	default:
		return false
	}
}

func AllEmbeddingInputStyles() []EmbeddingInputStyle {
	return []EmbeddingInputStyle{
		EmbeddingInputStyleNone,
		EmbeddingInputStyleVoyageInputType,
		EmbeddingInputStyleNomicPrefix,
	}
}

var allowedEmbeddingDimensions = [...]int{768, 1024, 1536}

func AllowedEmbeddingDimensions() []int {
	return slices.Clone(allowedEmbeddingDimensions[:])
}

func IsAllowedEmbeddingDimension(dimensions int) bool {
	return slices.Contains(allowedEmbeddingDimensions[:], dimensions)
}

// StructuredOutputMode describes how far a provider can be trusted to honour a
// JSON schema. This is the single biggest behavioural difference between a
// frontier API and a self-hosted model, and getting it wrong shows up as a parse
// failure at run time rather than at configuration time.
type StructuredOutputMode string

const (
	// StructuredOutputJSONSchema means the endpoint enforces the supplied schema
	// server-side and the response is guaranteed to conform.
	StructuredOutputJSONSchema = StructuredOutputMode("JSONSchema")
	// StructuredOutputJSONMode means the endpoint guarantees syntactically valid
	// JSON but not adherence to the schema, so the shape is validated on receipt.
	StructuredOutputJSONMode = StructuredOutputMode("JSONMode")
	// StructuredOutputPrompted means the schema is carried in the prompt and
	// nothing is guaranteed. The response is fenced, repaired, and validated.
	StructuredOutputPrompted = StructuredOutputMode("Prompted")
)

func (m StructuredOutputMode) IsValid() bool {
	switch m {
	case StructuredOutputJSONSchema, StructuredOutputJSONMode, StructuredOutputPrompted:
		return true
	default:
		return false
	}
}

func AllStructuredOutputModes() []StructuredOutputMode {
	return []StructuredOutputMode{
		StructuredOutputJSONSchema,
		StructuredOutputJSONMode,
		StructuredOutputPrompted,
	}
}

// DefaultStructuredOutputMode is the mode assumed for a kind when an operator has
// not chosen one. The hosted protocols enforce schemas; an arbitrary
// OpenAI-compatible endpoint is assumed to be the weakest case until proven
// otherwise by a test connection.
func (k Kind) DefaultStructuredOutputMode() StructuredOutputMode {
	switch k {
	case KindAnthropicMessages, KindOpenAIResponses:
		return StructuredOutputJSONSchema
	// Ollama constrains decoding to the schema on its native endpoint, so it is
	// as strong as the hosted APIs here despite being self-hosted.
	case KindOllama:
		return StructuredOutputJSONSchema
	case KindOpenAIChat:
		return StructuredOutputPrompted
	default:
		return StructuredOutputPrompted
	}
}

// Task names a unit of AI work that can be routed to a provider independently.
// Splitting them is the point of the abstraction: classification is high-volume
// and cheap enough for a local model, while a billing diagnosis writes to the
// ledger and warrants a frontier model.
type Task string

const (
	TaskDocumentClassification = Task("DocumentClassification")
	TaskDocumentExtraction     = Task("DocumentExtraction")
	TaskBillingDiagnosis       = Task("BillingDiagnosis")
	TaskFormulaAssistant       = Task("FormulaAssistant")
	// TaskScopeClassification decides whether a request belongs to this system at
	// all. It runs on every chat turn and needs only a yes/no with a category, so
	// it is the cheapest thing to route to a small local model.
	TaskScopeClassification = Task("ScopeClassification")
	// TaskAssistantChat answers a person's question and drives the tool loop.
	TaskAssistantChat = Task("AssistantChat")
	// TaskOperationalInsights writes the prose on a home-screen insight. It never
	// produces a number, a severity or a link — those are computed before it runs
	// and checked after it — so it is safe to route to whatever is cheapest.
	TaskOperationalInsights = Task("OperationalInsights")
	// TaskDailyBriefing writes the morning page from figures already
	// gathered. Like the insight narration it produces only wording, and
	// every number it writes is checked against those figures afterwards,
	// so it is safe to route wherever is cheapest.
	TaskDailyBriefing = Task("DailyBriefing")
	// TaskQueryCompose turns a sentence into a table's filters. It names
	// fields and values from a catalogue it is shown and writes no SQL;
	// everything it names is compiled against that catalogue afterwards, so a
	// cheap model that guesses wrong produces an unresolved line rather than a
	// wrong answer.
	TaskQueryCompose = Task("QueryCompose")
	// TaskInboundClassification decides what an email that arrived actually is.
	// Like the other classifiers it answers with a label from a fixed set, and
	// the label is re-checked against that set afterwards — a model that invents
	// a category produces a message for a person, not a new kind of work.
	TaskInboundClassification = Task("InboundClassification")
	TaskAccountingMapping     = Task("AccountingMapping")
	TaskEvaluationJudge       = Task("EvaluationJudge")
	TaskEmbedding             = Task("Embedding")
	TaskGeneral               = Task("General")
)

func (t Task) IsValid() bool {
	switch t {
	case TaskDocumentClassification,
		TaskDocumentExtraction,
		TaskBillingDiagnosis,
		TaskFormulaAssistant,
		TaskScopeClassification,
		TaskAssistantChat,
		TaskOperationalInsights,
		TaskDailyBriefing,
		TaskQueryCompose,
		TaskInboundClassification,
		TaskAccountingMapping,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral:
		return true
	default:
		return false
	}
}

// AllTasks lists every routable task, for building configuration UIs.
func AllTasks() []Task {
	return []Task{
		TaskDocumentClassification,
		TaskDocumentExtraction,
		TaskBillingDiagnosis,
		TaskFormulaAssistant,
		TaskScopeClassification,
		TaskAssistantChat,
		TaskOperationalInsights,
		TaskDailyBriefing,
		TaskQueryCompose,
		TaskInboundClassification,
		TaskAccountingMapping,
		TaskEvaluationJudge,
		TaskEmbedding,
		TaskGeneral,
	}
}

// WritesToLedger reports whether a task's output can drive a financial mutation.
// Such a task is refused a provider that has not been marked trusted, so a
// misconfigured 7B model cannot quietly propose ledger corrections.
func (t Task) WritesToLedger() bool {
	return t == TaskBillingDiagnosis
}

func (t Task) Generates() bool {
	return t != TaskEmbedding
}

// ReasoningEffort is how hard a model is asked to think before it answers, and
// whether it is asked at all.
//
// Off sends no reasoning parameter and is the default, because a provider
// that does not reason rejects the parameter outright: OpenAI returns 400 for
// reasoning_effort on a model without it. An operator turns this on for a
// model they know reasons. Whatever is chosen, thinking a provider volunteers
// unasked — DeepSeek-style reasoning_content — is still read and shown.
//
// Off is not "no thinking" on a model that reasons by default: the GPT-5
// family and Ollama's thinking models reason at their own default effort when
// nothing is sent, out of sight and before the first token. None tells such a
// model not to reason; Minimal asks for the least reasoning it allows.
type ReasoningEffort string

const (
	ReasoningOff     = ReasoningEffort("Off")
	ReasoningNone    = ReasoningEffort("None")
	ReasoningMinimal = ReasoningEffort("Minimal")
	ReasoningLow     = ReasoningEffort("Low")
	ReasoningMedium  = ReasoningEffort("Medium")
	ReasoningHigh    = ReasoningEffort("High")
)

func AllReasoningEfforts() []ReasoningEffort {
	return []ReasoningEffort{
		ReasoningOff,
		ReasoningNone,
		ReasoningMinimal,
		ReasoningLow,
		ReasoningMedium,
		ReasoningHigh,
	}
}

func (e ReasoningEffort) IsValid() bool {
	return slices.Contains(AllReasoningEfforts(), e)
}

// ThinkingStyle is how a Claude model is asked to think, for an Anthropic
// provider whose model id does not say.
//
// Claude models from Opus 4.6 and Sonnet 4.6 on think by effort and refuse a
// token budget with a 400; older ones take only the budget. The adapter reads
// which from the model id, which works for the ids the Claude API, Bedrock and
// Vertex use. A gateway's alias says nothing, and is read as an older model,
// so an operator who knows the model behind it says which here.
//
// Auto reads the model id and is the default. Effort asks by effort whatever
// the id reads; Budget asks with a token budget. Only the Anthropic protocol
// takes either.
type ThinkingStyle string

const (
	ThinkingStyleAuto   = ThinkingStyle("Auto")
	ThinkingStyleEffort = ThinkingStyle("Effort")
	ThinkingStyleBudget = ThinkingStyle("Budget")
)

func AllThinkingStyles() []ThinkingStyle {
	return []ThinkingStyle{
		ThinkingStyleAuto,
		ThinkingStyleEffort,
		ThinkingStyleBudget,
	}
}

func (s ThinkingStyle) IsValid() bool {
	return slices.Contains(AllThinkingStyles(), s)
}

// Enabled reports whether the provider should be asked to reason.
func (e ReasoningEffort) Enabled() bool {
	return e.IsValid() && e != ReasoningOff && e != ReasoningNone
}

// Disabled reports whether the provider should be told explicitly not to
// reason, which differs from Off: Off says nothing at all.
func (e ReasoningEffort) Disabled() bool {
	return e == ReasoningNone
}

// Wire is the lower-case word the OpenAI-shaped protocols take.
func (e ReasoningEffort) Wire() string {
	switch e {
	case ReasoningNone:
		return "none"
	case ReasoningMinimal:
		return "minimal"
	case ReasoningLow:
		return "low"
	case ReasoningMedium:
		return "medium"
	case ReasoningHigh:
		return "high"
	default:
		return ""
	}
}

// ThinkingBudget is the token budget the Anthropic protocol takes for the
// effort. The floor is the API's minimum; the ceiling is well under a reply's
// own token ceiling, which the adapter raises to fit.
func (e ReasoningEffort) ThinkingBudget() int {
	switch e {
	case ReasoningMinimal, ReasoningLow:
		return 1024
	case ReasoningMedium:
		return 4096
	case ReasoningHigh:
		return 16384
	default:
		return 0
	}
}

// CapAction is what happens to a task once a provider has spent its monthly
// cap: Next moves it to the next provider in line, Stop fails it there.
type CapAction string

const (
	CapActionNext = CapAction("Next")
	CapActionStop = CapAction("Stop")
)

func AllCapActions() []CapAction {
	return []CapAction{CapActionNext, CapActionStop}
}

func (a CapAction) IsValid() bool {
	return slices.Contains(AllCapActions(), a)
}
