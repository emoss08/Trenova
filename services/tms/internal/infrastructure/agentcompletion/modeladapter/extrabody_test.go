package modeladapter

import (
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeExtraBody_LetsAVendorFieldThroughWithoutTouchingOurs(t *testing.T) {
	t.Parallel()

	body := chatRequest{
		Model:     "nvidia/nemotron-3.5-lightning-30b-a3b",
		MaxTokens: 4096,
		Stream:    true,
	}
	// What NVIDIA's own example sends, minus the two fields this system now
	// owns itself: the ceiling goes out as max_tokens from the provider's
	// own setting, and sampling has a task default.
	provider := &aiprovider.Provider{ExtraBody: map[string]any{
		"chat_template_kwargs": map[string]any{"enable_thinking": true},
		"reasoning_budget":     16384,
	}}

	merged, err := mergeExtraBody(body, provider)
	require.NoError(t, err)

	encoded, err := sonic.Marshal(merged)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &out))

	assert.Equal(t, float64(16384), out["reasoning_budget"])
	assert.Equal(t,
		map[string]any{"enable_thinking": true},
		out["chat_template_kwargs"],
	)
	// Ours are untouched.
	assert.Equal(t, float64(4096), out["max_tokens"])
	assert.Equal(t, "nvidia/nemotron-3.5-lightning-30b-a3b", out["model"])
	assert.Equal(t, true, out["stream"])
}

func TestMergeExtraBody_CannotOverrideWhatTheAdapterSet(t *testing.T) {
	t.Parallel()

	// Validation refuses these keys at save time; the merge refuses them
	// again, because a row written before that rule, or by hand, must not
	// be able to point a call at another model.
	body := chatRequest{Model: "ours", Stream: false}
	provider := &aiprovider.Provider{ExtraBody: map[string]any{
		"model":  "theirs",
		"stream": true,
	}}

	merged, err := mergeExtraBody(body, provider)
	require.NoError(t, err)

	encoded, err := sonic.Marshal(merged)
	require.NoError(t, err)
	var out map[string]any
	require.NoError(t, sonic.Unmarshal(encoded, &out))

	assert.Equal(t, "ours", out["model"])
	assert.Equal(t, false, out["stream"])
}

func TestMergeExtraBody_ReturnsTheBodyUntouchedWhenThereIsNothingToMerge(t *testing.T) {
	t.Parallel()

	body := chatRequest{Model: "ours"}

	merged, err := mergeExtraBody(body, &aiprovider.Provider{})
	require.NoError(t, err)
	assert.Equal(t, body, merged)

	merged, err = mergeExtraBody(body, nil)
	require.NoError(t, err)
	assert.Equal(t, body, merged)
}
