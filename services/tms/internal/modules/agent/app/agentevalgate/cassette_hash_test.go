package agentevalgate

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A filing result names the proposal it minted, a new id every run, so the
// hash reads minted ids by kind alone; a record id the case fixed still counts.
func TestHashRequest_IgnoresMintedIDsButNotRecordIDs(t *testing.T) {
	t.Parallel()

	request := func(proposal, shipment string) *llmports.ChatCompletionRequest {
		return &llmports.ChatCompletionRequest{
			System: "sys",
			Messages: []llmports.Message{{
				Role:    llmports.RoleTool,
				Content: "Recorded a proposal (proposal " + proposal + ") on " + shipment,
			}},
		}
	}

	first, err := HashRequest(
		request("ap_01M4JD9N4W3A5STCJ5XBHM5BSF", "shp_01JREDTEAMSH1PMENT00000000"),
	)
	require.NoError(t, err)
	second, err := HashRequest(
		request("ap_01M4JD9NNF856GR4NPNB0FCD46", "shp_01JREDTEAMSH1PMENT00000000"),
	)
	require.NoError(t, err)
	other, err := HashRequest(
		request("ap_01M4JD9NNF856GR4NPNB0FCD46", "shp_01JREDTEAMEDISH1PMENT00000"),
	)
	require.NoError(t, err)

	assert.Equal(t, first, second)
	assert.NotEqual(t, first, other)
}
