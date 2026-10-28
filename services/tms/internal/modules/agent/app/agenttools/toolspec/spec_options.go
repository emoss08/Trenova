package toolspec

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
)

// previewValidates is Validate for a tool whose Preview runs the service's
// own plan: the preview is computed and, when it says the write would be
// refused as it stands, that refusal is the validation error. A tool built
// this way checks a call with exactly the code that decides its write.
func PreviewValidates(
	ctx context.Context,
	previewer agenttoolports.ToolPreviewer,
	params *agenttoolports.ToolExecuteParams,
) error {
	preview, err := previewer.Preview(ctx, *params)
	if err != nil {
		return err
	}

	return toolpreview.Refused(preview)
}

func FuelInternalSpec(spec *ReceivableSpec) *ReceivableSpec {
	spec.Egress = agent.EgressInternal
	if spec.DefaultTier == "" {
		spec.DefaultTier = agent.TierPropose
	}
	if spec.MaxTier == "" {
		spec.MaxTier = agent.TierActWithApproval
	}

	return spec
}
