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
	case KindAnthropicMessages, KindOpenAIResponses, KindOpenAIChat:
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
	case KindOpenAIResponses, KindOpenAIChat:
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
	case KindOpenAIChat:
		return StructuredOutputPrompted
	default:
		return StructuredOutputPrompted
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
