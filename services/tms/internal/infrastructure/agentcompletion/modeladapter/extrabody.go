package modeladapter

import (
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
)

// mergeExtraBody folds a provider's own request fields into the body an
// adapter built.
//
// Serving stacks each add fields the protocol does not have — NVIDIA's NIM
// wants chat_template_kwargs and reasoning_budget to steer a Nemotron
// model's thinking, vLLM wants top_k, OpenRouter wants provider routing —
// and a model that needs one of them is simply unusable without a way to
// send it. Rather than grow the typed request for each vendor in turn,
// the provider carries them and they are merged here.
//
// The merge is one-directional on purpose: a key the adapter already set
// wins, so a provider cannot point the call at another model, disable the
// tools a caller passed, or flip streaming under a reader. What is left to
// it is sampling, vendor switches, and the fields this system has no
// opinion about.
//
// The body is returned unchanged when there is nothing to merge, so the
// common case neither allocates nor risks a round trip through JSON.
func mergeExtraBody(body any, provider *aiprovider.Provider) (any, error) {
	if provider == nil || len(provider.ExtraBody) == 0 {
		return body, nil
	}

	encoded, err := sonic.Marshal(body)
	if err != nil {
		return nil, err
	}

	merged := make(map[string]any, len(provider.ExtraBody)+16)
	for key, value := range provider.ExtraBody {
		merged[key] = value
	}

	// Decoding over the copy lets the adapter's own fields land last and
	// therefore win, without comparing key by key.
	if err = sonic.Unmarshal(encoded, &merged); err != nil {
		return nil, err
	}

	return merged, nil
}
