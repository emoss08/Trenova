package completionrouter

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

// One agent at one version in one organization shares a key, whoever is
// asking; anything else that changes the prompt's rules gets its own.
func TestPromptCacheKey(t *testing.T) {
	t.Parallel()

	agent := pulid.MustNew("agd_")
	one, two := int64(1), int64(2)
	request := func(user pulid.ID, version *int64) *serviceports.ChatCompletionRequest {
		req := chatRequest(pulid.Nil)
		req.Attribution = serviceports.AIUsageAttribution{
			UserID:            user,
			AgentDefinitionID: agent,
			DefinitionVersion: version,
		}
		return req
	}

	first := request(pulid.MustNew("usr_"), &one)
	second := request(pulid.MustNew("usr_"), &one)
	second.TenantInfo = first.TenantInfo

	key := promptCacheKey(first)
	assert.Len(t, key, 32)
	assert.Equal(t, key, promptCacheKey(second), "people asking the same agent share it")
	assert.NotContains(t, key, agent.String(), "no identifier of ours reaches the provider")

	newer := request(pulid.MustNew("usr_"), &two)
	newer.TenantInfo = first.TenantInfo
	assert.NotEqual(t, key, promptCacheKey(newer), "a new version of the agent is a new prompt")

	elsewhere := request(pulid.MustNew("usr_"), &one)
	assert.NotEqual(t, key, promptCacheKey(elsewhere), "another organization is another prompt")
}
