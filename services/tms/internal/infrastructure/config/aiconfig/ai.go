package aiconfig

import (
	"time"
)

// AIConfig is everything model-backed, in one section.
//
// The assistant, the agents, insight narration, the formula assistant and
// document extraction all run through the same completion router, so they
// share a master switch, the same timeouts and the same retry budget.
// Splitting them across a second section named after one feature is what
// produced the original confusion: the gate used to be
// documentIntelligence.enableAI, which ships false, so configuring a
// provider under AI Control produced "AI features are disabled" from a key
// named after a feature the operator was not using.
type Config struct {
	// Enabled is a pointer so an absent key means enabled rather than
	// disabled. Leaving it out is the common case and must not turn the
	// assistant off; the real gate is per-organization anyway, since the
	// router can only reach a provider row somebody configured and enabled.
	// This switch is here for an operator who wants to stop all of it at once.
	Enabled *bool `mapstructure:"enabled"`
	// VerdictCacheTTL is how long a scope verdict is shared between replicas.
	// A verdict is a pure function of the question, so it can be long; it is
	// bounded so a change to the classifier's prompt reaches every replica
	// within a day rather than never.
	VerdictCacheTTL time.Duration `mapstructure:"verdictCacheTtl" validate:"omitempty,min=0"`

	// Timeout bounds a plain reachability check against an endpoint.
	Timeout time.Duration `mapstructure:"timeout"`
	// ProbeTimeout bounds the Test Connection probe, which waits on a real
	// generation rather than only on a connection.
	ProbeTimeout time.Duration `mapstructure:"probeTimeout"`
	// CompletionTimeout bounds one blocking call to a model, including the
	// wait before it says anything.
	CompletionTimeout time.Duration `mapstructure:"completionTimeout"`
	// StreamIdleTimeout is how long a streaming reply may go silent.
	StreamIdleTimeout time.Duration `mapstructure:"streamIdleTimeout"`
	// PrivateNetworkProviders lets an organization point a provider at a
	// private or loopback address, which a self-hosted model needs. It is a
	// pointer so an absent key keeps that working; a deployment that hosts
	// many organizations sets it false, because otherwise any of their
	// administrators can make the server call into its own network.
	PrivateNetworkProviders *bool `mapstructure:"privateNetworkProviders"`
	// ServerCloudCredentials lets a provider on Amazon Bedrock, Vertex AI or
	// Claude Platform on AWS or Google Cloud sign in with this server's own
	// cloud identity (an IAM role, application default credentials) instead
	// of a key stored for it. Off unless set: on a server that hosts many
	// organizations it would let any of their administrators spend the
	// operator's cloud account.
	ServerCloudCredentials bool `mapstructure:"serverCloudCredentials"`
	// CacheDiagnostics asks the Claude API, on each agent turn's model call,
	// where the prompt stopped matching the call before it, and logs and
	// traces the answer. Off unless set: it is for an operator chasing cache
	// misses, and the API keeps a fingerprint of each request it is asked
	// about for a short while.
	CacheDiagnostics bool `mapstructure:"cacheDiagnostics"`
	MaxRetries       int  `mapstructure:"maxRetries"       validate:"omitempty,min=0,max=10"`

	ToolDisclosureThreshold *int `mapstructure:"toolDisclosureThreshold" validate:"omitempty,min=0"`

	// ModelPriceReference fills in a model's price, when the provider's own
	// model list gives none, from OpenRouter's public catalog, marked as an
	// estimate. It is a pointer so an absent key keeps it on; a deployment that
	// must make no outside call sets it false.
	ModelPriceReference *bool `mapstructure:"modelPriceReference"`
	// ModelPriceReferenceTTL is how long the catalog is kept before it is read
	// again; one read serves every replica and organization for that long.
	ModelPriceReferenceTTL time.Duration `mapstructure:"modelPriceReferenceTtl" validate:"omitempty,min=0"`

	// DocumentExtraction lets a model classify and extract uploaded
	// documents. Off by default, and Docling reads nothing without it.
	DocumentExtraction  bool `mapstructure:"documentExtraction"`
	MaxInputChars       int  `mapstructure:"maxInputChars"       validate:"omitempty,min=1000,max=500000"`
	ExtractionMaxTokens int  `mapstructure:"extractionMaxTokens" validate:"omitempty,min=256,max=32768"`

	MaxExtractedChars       int `mapstructure:"maxExtractedChars"       validate:"omitempty,min=1000,max=1000000"`
	ReconcileBatchSize      int `mapstructure:"reconcileBatchSize"      validate:"omitempty,min=1,max=1000"`
	MaxConcurrentActivities int `mapstructure:"maxConcurrentActivities" validate:"omitempty,min=1,max=64"`
}

const defaultVerdictCacheTTL = 24 * time.Hour

const defaultModelPriceReferenceTTL = 24 * time.Hour

// ModelPriceReferenceEnabled is nil-safe on the receiver and the field, so an
// absent section or key keeps estimated prices on.
func (c *Config) ModelPriceReferenceEnabled() bool {
	return c == nil || c.ModelPriceReference == nil || *c.ModelPriceReference
}

func (c *Config) GetModelPriceReferenceTTL() time.Duration {
	if c == nil || c.ModelPriceReferenceTTL <= 0 {
		return defaultModelPriceReferenceTTL
	}

	return c.ModelPriceReferenceTTL
}

// GetVerdictCacheTTL is nil-safe: a guard built without this section shares
// verdicts for a day.
func (c *Config) GetVerdictCacheTTL() time.Duration {
	if c == nil || c.VerdictCacheTTL <= 0 {
		return defaultVerdictCacheTTL
	}

	return c.VerdictCacheTTL
}

// AIEnabledKey names the configuration key in error messages, so a disabled
// install says which switch to look at rather than leaving it to be guessed.
const EnabledKey = "ai.enabled"

// AIEnabled is nil-safe on the receiver as well as the field. A service built
// without this section — which a test does, and which a future caller that
// forgets to wire it would — must fall to the same "available" default as an
// absent key, rather than panicking inside a gate whose whole job is to answer
// a yes-or-no question.
// PrivateNetworkProvidersAllowed reports whether a provider may be reached
// on a private or loopback address.
func (c *Config) PrivateNetworkProvidersAllowed() bool {
	return c == nil || c.PrivateNetworkProviders == nil || *c.PrivateNetworkProviders
}

// ServerCloudCredentialsAllowed reports whether a provider may sign in with
// the server's own cloud credentials.
func (c *Config) ServerCloudCredentialsAllowed() bool {
	return c != nil && c.ServerCloudCredentials
}

const DefaultToolDisclosureThreshold = 12

func (c *Config) GetToolDisclosureThreshold() int {
	if c == nil || c.ToolDisclosureThreshold == nil || *c.ToolDisclosureThreshold < 0 {
		return DefaultToolDisclosureThreshold
	}

	return *c.ToolDisclosureThreshold
}

// CacheDiagnosticsEnabled reports whether model calls ask the Claude API why
// their prompt cache missed.
func (c *Config) CacheDiagnosticsEnabled() bool {
	return c != nil && c.CacheDiagnostics
}

func (c *Config) AIEnabled() bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

// DocumentExtractionEnabled says whether a model may read uploaded
// documents. It is separate from the master switch on purpose: reading a
// customer's paperwork with a model is a decision an operator makes on its
// own terms, and it ships off, while the assistant an organization has
// configured a provider for ships on.
func (c *Config) DocumentExtractionEnabled() bool {
	return c != nil && c.DocumentExtraction
}

func (c *Config) GetTimeout() time.Duration {
	if c.Timeout <= 0 {
		return 20 * time.Second
	}

	return c.Timeout
}

// GetProbeTimeout bounds the Test Connection call against a provider.
//
// A probe is not a reachability check: it asks the model to emit a two-field
// JSON object, because what the test is really for is finding out whether the
// endpoint honours a schema. So it has to wait for a real generation, and on
// a queued free tier — NVIDIA NIM, a shared router, a cold self-hosted model —
// the first response header can be thirty seconds away. Twenty seconds failed
// those providers with "could not reach the endpoint" when the endpoint was
// fine and merely busy. The ceiling is kept under the server's own request
// timeout so a slow probe returns a verdict rather than a 504.
func (c *Config) GetProbeTimeout() time.Duration {
	if c.ProbeTimeout <= 0 {
		return 45 * time.Second
	}

	return c.ProbeTimeout
}

// GetCompletionTimeout bounds one blocking call to a model.
//
// Separate from GetTimeout, which is sized for a reachability probe: asking
// whether an endpoint answers is a second or two of work, while asking a model
// to read a prompt and write an answer is minutes on a loaded or self-hosted
// one. Sharing the probe's budget made a long answer fail outright on providers
// that do not stream, which looks to the reader exactly like an outage.
func (c *Config) GetCompletionTimeout() time.Duration {
	if c.CompletionTimeout <= 0 {
		return 5 * time.Minute
	}

	return c.CompletionTimeout
}

// GetStreamIdleTimeout is how long a streaming reply may go silent.
//
// A stream cannot use GetTimeout: that one bounds a whole request, and
// http.Client applies it to reading the body, so a long answer arriving
// perfectly well would be severed partway through. What a stream needs bounded
// is silence, measured between reads, which is long by default because the gap
// before the first token of a considered answer is ordinary rather than a
// fault.
func (c *Config) GetStreamIdleTimeout() time.Duration {
	if c.StreamIdleTimeout <= 0 {
		return 5 * time.Minute
	}

	return c.StreamIdleTimeout
}

func (c *Config) GetMaxInputChars() int {
	if c.MaxInputChars <= 0 {
		return 24000
	}

	return c.MaxInputChars
}

func (c *Config) GetExtractionMaxTokens() int {
	if c.ExtractionMaxTokens <= 0 {
		return 5000
	}

	return c.ExtractionMaxTokens
}

func (c *Config) GetMaxRetries() int {
	if c.MaxRetries <= 0 {
		return 2
	}

	return c.MaxRetries
}

func (c *Config) GetMaxConcurrentActivities() int {
	if c.MaxConcurrentActivities <= 0 {
		return 2
	}

	return c.MaxConcurrentActivities
}

func (c *Config) GetMaxExtractedChars() int {
	if c.MaxExtractedChars <= 0 {
		return 200000
	}

	return c.MaxExtractedChars
}

func (c *Config) GetReconcileBatchSize() int {
	if c.ReconcileBatchSize <= 0 {
		return 100
	}

	return c.ReconcileBatchSize
}

func Defaults() map[string]any {
	return map[string]any{
		"timeout":                 "20s",
		"probeTimeout":            "45s",
		"completionTimeout":       "5m",
		"streamIdleTimeout":       "5m",
		"maxRetries":              2,
		"maxInputChars":           24000,
		"maxExtractedChars":       200000,
		"reconcileBatchSize":      100,
		"maxConcurrentActivities": 2,
	}
}
