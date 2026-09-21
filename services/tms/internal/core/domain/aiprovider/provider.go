package aiprovider

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxProviderNameLength = 100
	maxModelLength        = 200
	minMaxTokens          = 256
	maxMaxTokens          = 200000
	defaultMaxTokens      = 8192
)

// Provider is one configured model endpoint belonging to an organization. An
// organization may hold several — a local model for classification, a hosted one
// for anything that reaches the ledger — which is why this is a row per endpoint
// rather than a credential keyed by vendor.
type Provider struct {
	bun.BaseModel `bun:"table:ai_providers,alias:aiprv" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Name        string `json:"name"        bun:"name,type:VARCHAR(100),notnull"`
	Description string `json:"description" bun:"description,type:TEXT,nullzero"`
	Kind        Kind   `json:"kind"        bun:"kind,type:VARCHAR(50),notnull"`

	// BaseURL is the endpoint root, without the protocol-specific path. Empty
	// falls back to Kind.DefaultBaseURL.
	BaseURL string `json:"baseUrl" bun:"base_url,type:TEXT,nullzero"`
	// Model is free text on purpose. Self-hosted deployments name models whatever
	// they like ("llama3.3:70b", "qwen2.5-coder:32b"), so no closed set can hold.
	Model string `json:"model" bun:"model,type:VARCHAR(200),notnull"`

	// APIKey holds the credential encrypted at rest. It is cleared before the
	// entity leaves the service layer.
	APIKey string `json:"-" bun:"api_key,type:TEXT,nullzero"`

	// AllowPrivateNetwork lets this provider reach a loopback or RFC 1918 address.
	// It exists because a self-hosted model is normally unreachable otherwise, and
	// it is deliberately per-provider so enabling it for an internal GPU box does
	// not widen egress for every other provider.
	AllowPrivateNetwork bool `json:"allowPrivateNetwork" bun:"allow_private_network,type:BOOLEAN,notnull"`

	StructuredOutputMode StructuredOutputMode `json:"structuredOutputMode" bun:"structured_output_mode,type:VARCHAR(50),notnull"`
	MaxTokens            int                  `json:"maxTokens"            bun:"max_tokens,type:INTEGER,notnull"`
	// ReasoningEffort asks a model that can think to do so before answering.
	// Off is the default: the parameter is refused by models without it.
	ReasoningEffort ReasoningEffort `json:"reasoningEffort" bun:"reasoning_effort,type:VARCHAR(50),notnull,nullzero,default:'Off'"`

	// Tasks are the units of work this provider may serve. Priority orders the
	// candidates for a task, lowest first, which gives fallback chains without a
	// second table.
	Tasks    []Task `json:"tasks"    bun:"tasks,type:TEXT[],array,nullzero"`
	Priority int    `json:"priority" bun:"priority,type:INTEGER,notnull"`

	// Trusted marks a provider an administrator vouches for. A task that can drive
	// a financial mutation will not route to an untrusted provider.
	Trusted bool `json:"trusted" bun:"trusted,type:BOOLEAN,notnull"`
	Enabled bool `json:"enabled" bun:"enabled,type:BOOLEAN,notnull"`

	// LastTest is the outcome of the most recent live probe, kept so the
	// configuration UI can show whether an endpoint was ever reachable without
	// probing it again on every page load.
	LastTest *TestOutcome `json:"lastTest" bun:"last_test,type:JSONB,nullzero"`

	// HasAPIKey is set on the redacted copy so a client can tell a credential is
	// stored without the secret making the trip.
	HasAPIKey bool `json:"hasApiKey" bun:"-"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (p *Provider) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("aiprv_")
		}
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}

// TestOutcome is what a live probe of the endpoint revealed, recorded on the
// provider when it ran.
type TestOutcome struct {
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	ModelIdentifier string `json:"modelIdentifier,omitempty"`
	SchemaHonoured  bool   `json:"schemaHonoured"`
	LatencyMS       int64  `json:"latencyMs"`
	Detail          string `json:"detail,omitempty"`
	TestedAt        int64  `json:"testedAt"`
}

func (p *Provider) GetID() pulid.ID { return p.ID }

func (p *Provider) GetCreatedAt() int64 { return p.CreatedAt }

func (p *Provider) GetTableName() string { return "ai_providers" }

// Redacted returns a copy safe to hand to a client. The credential never leaves
// the service layer; callers are told only whether one is set, so the UI can show
// a configured state without the secret making the trip.
func (p *Provider) Redacted() *Provider {
	clone := *p
	clone.HasAPIKey = p.HasStoredAPIKey()
	clone.APIKey = ""

	return &clone
}

// HasStoredAPIKey reports whether a credential is stored.
func (p *Provider) HasStoredAPIKey() bool {
	return strings.TrimSpace(p.APIKey) != ""
}

func (p *Provider) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "aiprv",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText},
			{Name: "model", Type: domaintypes.FieldTypeText},
			{Name: "kind", Type: domaintypes.FieldTypeEnum},
		},
	}
}

// ResolvedBaseURL is the endpoint this provider actually calls.
func (p *Provider) ResolvedBaseURL() string {
	if trimmed := strings.TrimRight(strings.TrimSpace(p.BaseURL), "/"); trimmed != "" {
		return trimmed
	}

	return p.Kind.DefaultBaseURL()
}

// ResolvedMaxTokens clamps the configured ceiling into the supported range.
func (p *Provider) ResolvedMaxTokens() int {
	if p.MaxTokens <= 0 {
		return defaultMaxTokens
	}

	return p.MaxTokens
}

// NetworkPolicy is the egress policy this provider's HTTP client must enforce.
func (p *Provider) NetworkPolicy() httpsafe.Policy {
	return httpsafe.Policy{AllowPrivateNetworks: p.AllowPrivateNetwork}
}

// ServesTask reports whether this provider is a candidate for t.
func (p *Provider) ServesTask(t Task) bool {
	for _, task := range p.Tasks {
		if task == t {
			return true
		}
	}

	return false
}

// CanServeTask reports whether this provider may serve t right now, and why not
// when it may not. A disabled provider, or an untrusted one asked for work that
// reaches the ledger, is not a candidate.
func (p *Provider) CanServeTask(t Task) (bool, string) {
	switch {
	case !p.Enabled:
		return false, "provider is disabled"
	case !p.ServesTask(t):
		return false, "provider is not assigned to this task"
	case t.WritesToLedger() && !p.Trusted:
		return false, "provider is not marked trusted, and this task can change financial records"
	default:
		return true, ""
	}
}

func (p *Provider) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxProviderNameLength).
				Error("Name cannot be longer than 100 characters"),
		),
		validation.Field(&p.Kind,
			validation.Required.Error("Kind is required"),
			domainvalidation.ValidEnum[Kind]("Kind is not a protocol this system speaks"),
		),
		validation.Field(&p.Model,
			validation.Required.Error("Model is required"),
			validation.Length(1, maxModelLength).
				Error("Model cannot be longer than 200 characters"),
		),
		validation.Field(&p.StructuredOutputMode,
			validation.Required.Error("Structured output mode is required"),
			domainvalidation.ValidEnum[StructuredOutputMode](
				"Structured output mode is invalid",
			),
		),
		validation.Field(&p.ReasoningEffort,
			validation.Required.Error("Reasoning effort is required"),
			domainvalidation.ValidEnum[ReasoningEffort]("Reasoning effort is invalid"),
		),
		validation.Field(&p.MaxTokens,
			validation.Min(minMaxTokens).
				Error("Max tokens must be at least 256"),
			validation.Max(maxMaxTokens).
				Error("Max tokens cannot exceed 200000"),
		),
		validation.Field(&p.Priority,
			validation.Min(0).Error("Priority cannot be negative"),
		),
	))

	p.validateTasks(multiErr)
	p.validateEndpoint(multiErr)
}

func (p *Provider) validateTasks(multiErr *errortypes.MultiError) {
	for idx, task := range p.Tasks {
		if !task.IsValid() {
			multiErr.Add(
				fmt.Sprintf("tasks[%d]", idx),
				errortypes.ErrInvalid,
				"Task is not one this system routes",
			)
		}
	}

	if !p.Enabled {
		return
	}

	// An enabled provider serving nothing is almost always a half-finished setup,
	// and it fails silently at call time rather than at save time.
	if len(p.Tasks) == 0 {
		multiErr.Add(
			"tasks",
			errortypes.ErrInvalid,
			"An enabled provider must serve at least one task",
		)
	}

	for idx, task := range p.Tasks {
		if task.WritesToLedger() && !p.Trusted {
			multiErr.Add(
				fmt.Sprintf("tasks[%d]", idx),
				errortypes.ErrInvalid,
				"This task can change financial records, so it can only be served by a provider marked trusted",
			)
		}
	}
}

func (p *Provider) validateEndpoint(multiErr *errortypes.MultiError) {
	if !p.Kind.IsValid() {
		return
	}

	baseURL := p.ResolvedBaseURL()
	if baseURL == "" {
		multiErr.Add(
			"baseUrl",
			errortypes.ErrRequired,
			"Base URL is required for an OpenAI-compatible provider",
		)
		return
	}

	// Validated here so a bad or blocked endpoint is rejected while an
	// administrator is looking at the form, not on the first call at midnight.
	if _, err := httpsafe.ValidateURLWithPolicy(baseURL, p.NetworkPolicy()); err != nil {
		multiErr.Add("baseUrl", errortypes.ErrInvalid, endpointErrorMessage(err))
	}

	if p.Kind.RequiresAPIKey() && strings.TrimSpace(p.APIKey) == "" {
		multiErr.Add("apiKey", errortypes.ErrRequired, "API key is required for this provider")
	}
}

func endpointErrorMessage(err error) string {
	switch {
	case errors.Is(err, httpsafe.ErrBlockedScheme):
		return "Base URL must use http or https"
	case errors.Is(err, httpsafe.ErrMissingHost):
		return "Base URL must include a host"
	case errors.Is(err, httpsafe.ErrBlockedAddress):
		return "Base URL points at an address this server will not call. Enable \"Allow private network\" if this is a model server on your own network; link-local addresses are never allowed."
	default:
		return "Base URL is not a valid URL"
	}
}
