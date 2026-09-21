package aiprovider

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
	TaskGeneral       = Task("General")
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
		TaskGeneral,
	}
}

// WritesToLedger reports whether a task's output can drive a financial mutation.
// Such a task is refused a provider that has not been marked trusted, so a
// misconfigured 7B model cannot quietly propose ledger corrections.
func (t Task) WritesToLedger() bool {
	return t == TaskBillingDiagnosis
}

// ReasoningEffort is how hard a model is asked to think before it answers, and
// whether it is asked at all.
//
// Off sends no reasoning parameter and is the default, because a provider
// that does not reason rejects the parameter outright: OpenAI returns 400 for
// reasoning_effort on a model without it. An operator turns this on for a
// model they know reasons. Whatever is chosen, thinking a provider volunteers
// unasked — DeepSeek-style reasoning_content — is still read and shown.
type ReasoningEffort string

const (
	ReasoningOff    = ReasoningEffort("Off")
	ReasoningLow    = ReasoningEffort("Low")
	ReasoningMedium = ReasoningEffort("Medium")
	ReasoningHigh   = ReasoningEffort("High")
)

func (e ReasoningEffort) IsValid() bool {
	switch e {
	case ReasoningOff, ReasoningLow, ReasoningMedium, ReasoningHigh:
		return true
	default:
		return false
	}
}

// Enabled reports whether the provider should be asked to reason.
func (e ReasoningEffort) Enabled() bool {
	return e.IsValid() && e != ReasoningOff
}

// Wire is the lower-case word the OpenAI-shaped protocols take.
func (e ReasoningEffort) Wire() string {
	switch e {
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
	case ReasoningLow:
		return 1024
	case ReasoningMedium:
		return 4096
	case ReasoningHigh:
		return 16384
	default:
		return 0
	}
}
