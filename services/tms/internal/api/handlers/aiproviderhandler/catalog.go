package aiproviderhandler

import "github.com/emoss08/trenova/internal/core/domain/aiprovider"

// KindDescriptor describes a wire protocol to the configuration UI.
type KindDescriptor struct {
	Kind                       aiprovider.Kind                 `json:"kind"`
	Label                      string                          `json:"label"`
	Description                string                          `json:"description"`
	DefaultBaseURL             string                          `json:"defaultBaseUrl"`
	RequiresAPIKey             bool                            `json:"requiresApiKey"`
	RequiresBaseURL            bool                            `json:"requiresBaseUrl"`
	DefaultStructuredOutput    aiprovider.StructuredOutputMode `json:"defaultStructuredOutputMode"`
	SupportsStructuredEnforced bool                            `json:"supportsStructuredEnforced"`
	SupportsEmbedding          bool                            `json:"supportsEmbedding"`
}

func kindDescriptors() []KindDescriptor {
	return []KindDescriptor{
		{
			Kind:                       aiprovider.KindAnthropicMessages,
			Label:                      "Anthropic Messages",
			Description:                "Anthropic's API, and any endpoint serving the same shape such as Amazon Bedrock.",
			DefaultBaseURL:             aiprovider.KindAnthropicMessages.DefaultBaseURL(),
			RequiresAPIKey:             true,
			DefaultStructuredOutput:    aiprovider.KindAnthropicMessages.DefaultStructuredOutputMode(),
			SupportsStructuredEnforced: true,
			SupportsEmbedding:          aiprovider.KindAnthropicMessages.SupportsEmbedding(),
		},
		{
			Kind:                       aiprovider.KindOpenAIResponses,
			Label:                      "OpenAI Responses",
			Description:                "OpenAI's Responses API, and endpoints serving the same shape.",
			DefaultBaseURL:             aiprovider.KindOpenAIResponses.DefaultBaseURL(),
			RequiresAPIKey:             true,
			DefaultStructuredOutput:    aiprovider.KindOpenAIResponses.DefaultStructuredOutputMode(),
			SupportsStructuredEnforced: true,
			SupportsEmbedding:          aiprovider.KindOpenAIResponses.SupportsEmbedding(),
		},
		{
			Kind:  aiprovider.KindOpenAIChat,
			Label: "OpenAI-compatible",
			Description: "Any server exposing /chat/completions — vLLM, SGLang, LM Studio, " +
				"OpenRouter, Groq, Together, Fireworks, DeepInfra, and others.",
			RequiresAPIKey:             false,
			RequiresBaseURL:            true,
			DefaultStructuredOutput:    aiprovider.KindOpenAIChat.DefaultStructuredOutputMode(),
			SupportsStructuredEnforced: true,
			SupportsEmbedding:          aiprovider.KindOpenAIChat.SupportsEmbedding(),
		},
		{
			Kind:  aiprovider.KindOllama,
			Label: "Ollama",
			Description: "Ollama's native API. Used instead of its OpenAI-compatible " +
				"endpoint, which ignores JSON schemas.",
			DefaultBaseURL:             aiprovider.KindOllama.DefaultBaseURL(),
			RequiresAPIKey:             false,
			DefaultStructuredOutput:    aiprovider.KindOllama.DefaultStructuredOutputMode(),
			SupportsStructuredEnforced: true,
			SupportsEmbedding:          aiprovider.KindOllama.SupportsEmbedding(),
		},
	}
}

// Preset is a known deployment an administrator can start from, so configuring a
// provider does not begin with hunting for the right base URL.
type Preset struct {
	Key                  string                          `json:"key"`
	Label                string                          `json:"label"`
	Kind                 aiprovider.Kind                 `json:"kind"`
	BaseURL              string                          `json:"baseUrl"`
	StructuredOutputMode aiprovider.StructuredOutputMode `json:"structuredOutputMode"`
	AllowPrivateNetwork  bool                            `json:"allowPrivateNetwork"`
	RequiresAPIKey       bool                            `json:"requiresApiKey"`
	SelfHosted           bool                            `json:"selfHosted"`
	ExampleModel         string                          `json:"exampleModel"`
	Notes                string                          `json:"notes,omitempty"`
	// Domain is the vendor's web domain, which is how the UI resolves a brand
	// logo without shipping one per vendor.
	Domain              string                         `json:"domain,omitempty"`
	Tasks               []aiprovider.Task              `json:"tasks,omitempty"`
	EmbeddingDimensions int                            `json:"embeddingDimensions,omitempty"`
	EmbeddingInputStyle aiprovider.EmbeddingInputStyle `json:"embeddingInputStyle,omitempty"`
}

// Presets covers the deployments organizations actually reach for. The
// structured-output mode differs per entry because support genuinely differs:
// vLLM and SGLang enforce schemas server-side, while a bare llama.cpp server
// does not reliably honour response_format on its OpenAI-compatible route.
func Presets() []Preset {
	return []Preset{
		{
			Key:                  "anthropic",
			Domain:               "anthropic.com",
			Label:                "Anthropic",
			Kind:                 aiprovider.KindAnthropicMessages,
			BaseURL:              "https://api.anthropic.com",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "claude-opus-5",
		},
		{
			Key:                  "openai",
			Domain:               "openai.com",
			Label:                "OpenAI",
			Kind:                 aiprovider.KindOpenAIResponses,
			BaseURL:              "https://api.openai.com",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "gpt-5-mini",
		},
		{
			Key:                  "openai-embeddings",
			Domain:               "openai.com",
			Label:                "OpenAI embeddings",
			Kind:                 aiprovider.KindOpenAIResponses,
			BaseURL:              "https://api.openai.com",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "text-embedding-3-small",
			Notes:                "Semantic retrieval through OpenAI's embeddings endpoint.",
			Tasks:                []aiprovider.Task{aiprovider.TaskEmbedding},
			EmbeddingDimensions:  1536,
			EmbeddingInputStyle:  aiprovider.EmbeddingInputStyleNone,
		},
		{
			Key:                  "voyage",
			Domain:               "voyageai.com",
			Label:                "Voyage AI embeddings",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://api.voyageai.com/v1",
			StructuredOutputMode: aiprovider.StructuredOutputPrompted,
			RequiresAPIKey:       true,
			ExampleModel:         "voyage-3.5",
			Notes: "Retrieval-tuned embeddings. Documents and queries are sent with " +
				"Voyage's input_type so each is embedded for its side of a search.",
			Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
			EmbeddingDimensions: 1024,
			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleVoyageInputType,
		},
		{
			Key:                  "gemini-embeddings",
			Domain:               "ai.google.dev",
			Label:                "Gemini embeddings",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://generativelanguage.googleapis.com/v1beta/openai",
			StructuredOutputMode: aiprovider.StructuredOutputPrompted,
			RequiresAPIKey:       true,
			ExampleModel:         "gemini-embedding-001",
			Notes: "Gemini's OpenAI-compatible endpoint. The model's native size is " +
				"reduced to the dimension chosen here.",
			Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
			EmbeddingDimensions: 1536,
			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,
		},
		{
			Key:                  "openrouter",
			Domain:               "openrouter.ai",
			Label:                "OpenRouter",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://openrouter.ai/api/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "qwen/qwen3-max",
			Notes:                "Routes to hundreds of models across many upstream providers.",
		},
		{
			Key:                  "groq",
			Domain:               "groq.com",
			Label:                "Groq",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://api.groq.com/openai/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "llama-3.3-70b-versatile",
			Notes:                "Lowest time to first token; useful for high-volume classification.",
		},
		{
			Key:                  "together",
			Domain:               "together.ai",
			Label:                "Together AI",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://api.together.xyz/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "Qwen/Qwen3-235B-A22B",
		},
		{
			Key:                  "fireworks",
			Domain:               "fireworks.ai",
			Label:                "Fireworks AI",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://api.fireworks.ai/inference/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "accounts/fireworks/models/qwen3-235b-a22b",
		},
		{
			Key:                  "deepinfra",
			Domain:               "deepinfra.com",
			Label:                "DeepInfra",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "https://api.deepinfra.com/v1/openai",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			ExampleModel:         "Qwen/Qwen3-235B-A22B",
		},
		{
			Key:                  "bedrock",
			Domain:               "aws.amazon.com",
			Label:                "Amazon Bedrock",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			RequiresAPIKey:       true,
			Notes: "Use the bedrock-mantle endpoint for your region and a Bedrock " +
				"API key, for example https://bedrock-mantle.us-east-1.amazonaws.com/openai/v1.",
		},
		{
			Key:                  "ollama",
			Domain:               "ollama.com",
			Label:                "Ollama (self-hosted)",
			Kind:                 aiprovider.KindOllama,
			BaseURL:              "http://localhost:11434",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			AllowPrivateNetwork:  true,
			SelfHosted:           true,
			ExampleModel:         "qwen3:8b",
			Notes:                "Runs open-weight models on your own hardware; no data leaves your network.",
		},
		{
			Key:                  "ollama-embeddings",
			Domain:               "ollama.com",
			Label:                "Ollama embeddings (self-hosted)",
			Kind:                 aiprovider.KindOllama,
			BaseURL:              "http://localhost:11434",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			AllowPrivateNetwork:  true,
			SelfHosted:           true,
			ExampleModel:         "nomic-embed-text",
			Notes: "Semantic retrieval on your own hardware. nomic-embed-text returns 768 " +
				"dimensions and expects search_document and search_query prefixes.",
			Tasks:               []aiprovider.Task{aiprovider.TaskEmbedding},
			EmbeddingDimensions: 768,
			EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNomicPrefix,
		},
		{
			Key:                  "vllm",
			Domain:               "vllm.ai",
			Label:                "vLLM (self-hosted)",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:8000/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			AllowPrivateNetwork:  true,
			SelfHosted:           true,
			ExampleModel:         "Qwen/Qwen3-32B",
			Notes:                "Enforces JSON schemas server-side; the strongest self-hosted option for throughput.",
		},
		{
			Key:                  "sglang",
			Domain:               "sglang.ai",
			Label:                "SGLang (self-hosted)",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:30000/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			AllowPrivateNetwork:  true,
			SelfHosted:           true,
			ExampleModel:         "Qwen/Qwen3-32B",
		},
		{
			Key:                  "lmstudio",
			Domain:               "lmstudio.ai",
			Label:                "LM Studio (self-hosted)",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:1234/v1",
			StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
			AllowPrivateNetwork:  true,
			SelfHosted:           true,
			ExampleModel:         "qwen3-32b",
		},
		{
			Key:                  "llamacpp",
			Domain:               "ggml.ai",
			Label:                "llama.cpp (self-hosted)",
			Kind:                 aiprovider.KindOpenAIChat,
			BaseURL:              "http://localhost:8080/v1",
			StructuredOutputMode: aiprovider.StructuredOutputPrompted,
			AllowPrivateNetwork:  true,
			SelfHosted:           true,
			ExampleModel:         "qwen3-32b",
			Notes: "llama-server does not reliably honour response_format on its " +
				"OpenAI-compatible route, so schemas are sent in the prompt instead.",
		},
	}
}

// TaskDescriptor describes a routable unit of work to the UI.
type TaskDescriptor struct {
	Task           aiprovider.Task `json:"task"`
	Label          string          `json:"label"`
	Description    string          `json:"description"`
	RequiresTrust  bool            `json:"requiresTrust"`
	VolumeGuidance string          `json:"volumeGuidance"`
}

func taskDescriptors() []TaskDescriptor {
	return []TaskDescriptor{
		{
			Task:           aiprovider.TaskDocumentClassification,
			Label:          "Document classification",
			Description:    "Decide what an uploaded document is.",
			VolumeGuidance: "High volume, low complexity — a small local model handles this well.",
		},
		{
			Task:           aiprovider.TaskDocumentExtraction,
			Label:          "Document extraction",
			Description:    "Pull structured fields out of a document.",
			VolumeGuidance: "High volume; needs reliable structured output.",
		},
		{
			Task:          aiprovider.TaskBillingDiagnosis,
			Label:         "Billing diagnosis",
			Description:   "Diagnose a blocked billing item and propose resolutions.",
			RequiresTrust: true,
			VolumeGuidance: "Low volume, high stakes — this reaches the ledger, so it " +
				"requires a provider marked trusted.",
		},
		{
			Task:           aiprovider.TaskFormulaAssistant,
			Label:          "Formula assistant",
			Description:    "Generate and explain rating formulas.",
			VolumeGuidance: "Interactive; favours a capable model.",
		},
		{
			Task:        aiprovider.TaskScopeClassification,
			Label:       "Scope classification",
			Description: "Decide whether a chat message is about this organization's freight at all.",
			VolumeGuidance: "Runs on every assistant turn and needs only a yes/no — the " +
				"cheapest thing to route to a small local model.",
		},
		{
			Task:        aiprovider.TaskAssistantChat,
			Label:       "Assistant chat",
			Description: "Answer questions and drive the tool loop in the assistant.",
			VolumeGuidance: "Interactive and multi-step; needs reliable tool calling and " +
				"favours a capable model.",
		},
		{
			Task:        aiprovider.TaskOperationalInsights,
			Label:       "Operational insights",
			Description: "Write the explanation on a finding whose numbers were already computed.",
			VolumeGuidance: "A batch every few hours. It never produces a number or a " +
				"severity, so it is safe to route to whatever is cheapest.",
		},
		{
			Task:        aiprovider.TaskDailyBriefing,
			Label:       "Daily briefing",
			Description: "Write the morning page from figures that were already gathered.",
			VolumeGuidance: "A handful of calls once a day. Like insights it writes only " +
				"wording, so the cheapest model that writes clean English will do.",
		},
		{
			Task:        aiprovider.TaskQueryCompose,
			Label:       "Table questions",
			Description: "Turn a sentence into a table's filters and sort.",
			VolumeGuidance: "One short call each time somebody asks a table a question. " +
				"It names fields from a catalogue it is shown and writes no SQL, and " +
				"everything it names is compiled against that catalogue afterwards, so a " +
				"cheap model that guesses wrong produces an unresolved line rather than a " +
				"wrong answer.",
		},
		{
			Task:        aiprovider.TaskInboundClassification,
			Label:       "Inbound mail",
			Description: "Decide what an email that arrived on a monitored address is.",
			VolumeGuidance: "One call per message, so it follows how much mail is " +
				"forwarded. It answers with a label from a fixed set that is re-checked " +
				"afterwards, so the cheapest model that reads English reliably will do.",
		},
		{
			Task:        aiprovider.TaskAccountingMapping,
			Label:       "Accounting mapping",
			Description: "Choose which accounting-system record matches a Trenova record when the name rules are unsure.",
			VolumeGuidance: "Runs only for records the deterministic matcher could not settle, at most " +
				"a hundred per refresh and usually a handful. It picks from a supplied list, so a " +
				"small, inexpensive model is enough.",
		},
		{
			Task:  aiprovider.TaskEvaluationJudge,
			Label: "Evaluation judge",
			Description: "Score a sample of agents' answers to their evaluation cases " +
				"against each case's rubric.",
			VolumeGuidance: "A few calls a night, only when judging is on in AI Control. " +
				"Its reading counts toward the evaluation budget and never overrides a " +
				"check that failed outright, so a capable model that follows a rubric " +
				"closely is worth more here than a cheap one.",
		},
		{
			Task:  aiprovider.TaskEmbedding,
			Label: "Embedding",
			Description: "Turn memories, documents and mail into vectors so agents can " +
				"find them by meaning rather than by exact words.",
			VolumeGuidance: "High volume and cheap: every indexed record is embedded once " +
				"and every search embeds its question. Needs an embedding model, not a " +
				"chat model, and a fixed dimension; changing the model re-indexes everything.",
		},
		{
			Task:           aiprovider.TaskGeneral,
			Label:          "General",
			Description:    "Anything not routed to a more specific task.",
			VolumeGuidance: "Fallback pool.",
		},
	}
}

type EmbeddingInputStyleDescriptor struct {
	Style       aiprovider.EmbeddingInputStyle `json:"style"`
	Label       string                         `json:"label"`
	Description string                         `json:"description"`
}

func embeddingInputStyleDescriptors() []EmbeddingInputStyleDescriptor {
	return []EmbeddingInputStyleDescriptor{
		{
			Style:       aiprovider.EmbeddingInputStyleNone,
			Label:       "Same for documents and queries",
			Description: "Documents and search queries are sent as they are.",
		},
		{
			Style: aiprovider.EmbeddingInputStyleVoyageInputType,
			Label: "Voyage input type",
			Description: "Sends input_type document or query, and the dimension as " +
				"output_dimension, the way Voyage's API expects.",
		},
		{
			Style: aiprovider.EmbeddingInputStyleNomicPrefix,
			Label: "Nomic prefixes",
			Description: "Prefixes each text with search_document: or search_query:, " +
				"which nomic-embed-text needs to embed each side of a search correctly.",
		},
	}
}
