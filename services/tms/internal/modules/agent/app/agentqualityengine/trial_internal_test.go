package agentqualityengine

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrialProvider_IsOneOfTheProvidersThatServeConversations(t *testing.T) {
	t.Parallel()

	fast := &aiprovider.Provider{ID: pulid.MustNew("aip_"), Name: "Anthropic fast"}
	deep := &aiprovider.Provider{ID: pulid.MustNew("aip_"), Name: "Anthropic deep"}
	serving := resolvedProvider{candidates: []*aiprovider.Provider{deep, fast}}

	picked, err := serving.trial(fast.ID)
	require.NoError(t, err)
	assert.Same(t, fast, picked)

	none, err := serving.trial(pulid.Nil)
	require.NoError(t, err)
	assert.Nil(t, none, "no provider asked for is a run on the agent's own")
}

func TestTrialProvider_RefusesAProviderThatDoesNotServeConversations(t *testing.T) {
	t.Parallel()

	serving := resolvedProvider{candidates: []*aiprovider.Provider{
		{ID: pulid.MustNew("aip_"), Name: "Anthropic deep"},
	}}

	_, err := serving.trial(pulid.MustNew("aip_"))
	require.Error(t, err)
	var refused *errortypes.Error
	require.ErrorAs(t, err, &refused,
		"a provider that is off, serves another task or belongs elsewhere is the caller's mistake")
	assert.Equal(t, "trialProviderId", refused.Field())

	_, err = resolvedProvider{}.trial(pulid.MustNew("aip_"))
	require.Error(t, err)
}

func TestTrialLabel_NamesTheProviderAndItsModel(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Anthropic fast · claude-haiku-5-5", trialLabel(&aiprovider.Provider{
		Name:  "Anthropic fast",
		Model: "claude-haiku-5-5",
	}))
	assert.Equal(t, "Local", trialLabel(&aiprovider.Provider{Name: "Local"}))
	assert.Equal(t, "claude-haiku-5-5", trialLabel(&aiprovider.Provider{
		Name:  "claude-haiku-5-5",
		Model: "claude-haiku-5-5",
	}), "a provider named for its model is not said twice")

	long := trialLabel(&aiprovider.Provider{
		Name:  strings.Repeat("n", 250),
		Model: strings.Repeat("m", 250),
	})
	assert.LessOrEqual(t, len([]rune(long)), agentqualitysuite.MaxTrialLabelChars)
}
