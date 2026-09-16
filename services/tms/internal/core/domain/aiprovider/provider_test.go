package aiprovider_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validProvider() *aiprovider.Provider {
	return &aiprovider.Provider{
		OrganizationID:       pulid.MustNew("org_"),
		BusinessUnitID:       pulid.MustNew("bu_"),
		Name:                 "Hosted Claude",
		Kind:                 aiprovider.KindAnthropicMessages,
		Model:                "claude-opus-5",
		APIKey:               "sk-ant-test",
		StructuredOutputMode: aiprovider.StructuredOutputJSONSchema,
		MaxTokens:            8192,
		Tasks:                []aiprovider.Task{aiprovider.TaskGeneral},
		Enabled:              true,
	}
}

func fieldErrors(t *testing.T, p *aiprovider.Provider) map[string]bool {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)

	fields := make(map[string]bool)
	for _, e := range multiErr.Errors {
		fields[e.Field] = true
	}

	return fields
}

func TestValidate_AcceptsHostedProvider(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	validProvider().Validate(multiErr)
	require.False(t, multiErr.HasErrors(), "expected no errors, got %v", multiErr.Errors)
}

func TestValidate_OpenAIChatRequiresExplicitBaseURL(t *testing.T) {
	t.Parallel()

	// An OpenAI-compatible provider is self-hosted or third-party by definition,
	// so there is no sensible default endpoint to fall back to.
	p := validProvider()
	p.Kind = aiprovider.KindOpenAIChat
	p.Model = "qwen3:32b"
	p.APIKey = ""

	assert.True(t, fieldErrors(t, p)["baseUrl"])
}

func TestValidate_RejectsPrivateEndpointWithoutOptIn(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Kind = aiprovider.KindOpenAIChat
	p.BaseURL = "http://192.168.1.50:8000/v1"
	p.APIKey = ""

	assert.True(t, fieldErrors(t, p)["baseUrl"])
}

func TestValidate_AcceptsPrivateEndpointWithOptIn(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Kind = aiprovider.KindOpenAIChat
	p.BaseURL = "http://192.168.1.50:8000/v1"
	p.APIKey = ""
	p.AllowPrivateNetwork = true
	p.StructuredOutputMode = aiprovider.StructuredOutputPrompted

	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), "expected no errors, got %v", multiErr.Errors)
}

func TestValidate_RejectsMetadataEndpointEvenWithOptIn(t *testing.T) {
	t.Parallel()

	// Opting a provider into private networking must never become a way to reach
	// the cloud metadata service.
	p := validProvider()
	p.Kind = aiprovider.KindOpenAIChat
	p.BaseURL = "http://169.254.169.254/latest/meta-data/"
	p.APIKey = ""
	p.AllowPrivateNetwork = true

	assert.True(t, fieldErrors(t, p)["baseUrl"])
}

func TestValidate_RejectsNonHTTPScheme(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Kind = aiprovider.KindOpenAIChat
	p.BaseURL = "file:///etc/passwd"
	p.APIKey = ""
	p.AllowPrivateNetwork = true

	assert.True(t, fieldErrors(t, p)["baseUrl"])
}

func TestValidate_HostedProviderRequiresAPIKey(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.APIKey = ""

	assert.True(t, fieldErrors(t, p)["apiKey"])
}

func TestValidate_SelfHostedProviderDoesNotRequireAPIKey(t *testing.T) {
	t.Parallel()

	// A model server on a trusted network commonly has no auth at all.
	p := validProvider()
	p.Kind = aiprovider.KindOllama
	p.Model = "qwen3:8b"
	p.APIKey = ""
	p.AllowPrivateNetwork = true

	assert.False(t, fieldErrors(t, p)["apiKey"])
}

func TestValidate_LedgerTaskRequiresTrustedProvider(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Tasks = []aiprovider.Task{aiprovider.TaskBillingDiagnosis}
	p.Trusted = false

	assert.True(t, fieldErrors(t, p)["tasks[0]"])

	p.Trusted = true
	multiErr := errortypes.NewMultiError()
	p.Validate(multiErr)
	require.False(t, multiErr.HasErrors(), "expected no errors, got %v", multiErr.Errors)
}

func TestValidate_EnabledProviderMustServeATask(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Tasks = nil

	assert.True(t, fieldErrors(t, p)["tasks"])
}

func TestValidate_DisabledProviderMayServeNoTask(t *testing.T) {
	t.Parallel()

	p := validProvider()
	p.Tasks = nil
	p.Enabled = false

	assert.False(t, fieldErrors(t, p)["tasks"])
}

func TestCanServeTask(t *testing.T) {
	t.Parallel()

	t.Run("rejects disabled provider", func(t *testing.T) {
		t.Parallel()
		p := validProvider()
		p.Enabled = false

		ok, reason := p.CanServeTask(aiprovider.TaskGeneral)
		assert.False(t, ok)
		assert.Contains(t, reason, "disabled")
	})

	t.Run("rejects unassigned task", func(t *testing.T) {
		t.Parallel()
		ok, reason := validProvider().CanServeTask(aiprovider.TaskDocumentExtraction)
		assert.False(t, ok)
		assert.Contains(t, reason, "not assigned")
	})

	t.Run("rejects untrusted provider for ledger task", func(t *testing.T) {
		t.Parallel()
		p := validProvider()
		p.Tasks = []aiprovider.Task{aiprovider.TaskBillingDiagnosis}
		p.Trusted = false

		ok, reason := p.CanServeTask(aiprovider.TaskBillingDiagnosis)
		assert.False(t, ok)
		assert.Contains(t, reason, "trusted")
	})

	t.Run("accepts trusted provider for ledger task", func(t *testing.T) {
		t.Parallel()
		p := validProvider()
		p.Tasks = []aiprovider.Task{aiprovider.TaskBillingDiagnosis}
		p.Trusted = true

		ok, _ := p.CanServeTask(aiprovider.TaskBillingDiagnosis)
		assert.True(t, ok)
	})
}

func TestResolvedBaseURL(t *testing.T) {
	t.Parallel()

	t.Run("falls back to kind default", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "https://api.anthropic.com", validProvider().ResolvedBaseURL())
	})

	t.Run("prefers explicit base url and trims trailing slash", func(t *testing.T) {
		t.Parallel()
		p := validProvider()
		p.BaseURL = "https://openrouter.ai/api/v1/"

		assert.Equal(t, "https://openrouter.ai/api/v1", p.ResolvedBaseURL())
	})

	t.Run("openai chat has no default", func(t *testing.T) {
		t.Parallel()
		p := validProvider()
		p.Kind = aiprovider.KindOpenAIChat
		p.BaseURL = ""

		assert.Empty(t, p.ResolvedBaseURL())
	})
}

func TestKindDefaults(t *testing.T) {
	t.Parallel()

	// Ollama constrains decoding on its native endpoint, so it is not downgraded
	// to prompted output the way a generic compatible endpoint is.
	assert.Equal(t,
		aiprovider.StructuredOutputJSONSchema,
		aiprovider.KindOllama.DefaultStructuredOutputMode(),
	)
	assert.Equal(t,
		aiprovider.StructuredOutputPrompted,
		aiprovider.KindOpenAIChat.DefaultStructuredOutputMode(),
	)

	assert.True(t, aiprovider.KindAnthropicMessages.RequiresAPIKey())
	assert.True(t, aiprovider.KindOpenAIResponses.RequiresAPIKey())
	assert.False(t, aiprovider.KindOllama.RequiresAPIKey())
	assert.False(t, aiprovider.KindOpenAIChat.RequiresAPIKey())
}

func TestTaskWritesToLedger(t *testing.T) {
	t.Parallel()

	assert.True(t, aiprovider.TaskBillingDiagnosis.WritesToLedger())
	assert.False(t, aiprovider.TaskDocumentClassification.WritesToLedger())
	assert.False(t, aiprovider.TaskGeneral.WritesToLedger())

	for _, task := range aiprovider.AllTasks() {
		assert.True(t, task.IsValid(), "task %q should be valid", task)
	}
}
