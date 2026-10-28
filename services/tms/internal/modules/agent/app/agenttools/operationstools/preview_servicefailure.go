package operationstools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/common"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
)

// serviceFailureResolver is what resolving a failure previews through: the
// lifecycle's own checks, and the EDI 214 it would send.
type serviceFailureResolver interface {
	ServiceFailureDecider
	PreviewResolve(
		ctx context.Context,
		req *servicefailureports.ServiceFailureLifecycleRequest,
		actor *permissionports.RequestActor,
	) (*servicefailureports.ServiceFailureLifecyclePreview, error)
}

// ediReadyForGeneration is the 214 preflight's word for a 214 that would be
// generated when the change is saved.
const EdiReadyForGeneration = "ready_for_generation"

func (t *ResolveServiceFailureTool) Preview(
	ctx context.Context,
	params agenttoolports.ToolExecuteParams, //nolint:gocritic // the ToolPreviewer interface passes params by value
) (*agent.ToolPreview, error) {
	request, existing, err := t.request(ctx, &params)
	if err != nil {
		return nil, err
	}

	resolved, err := t.failures.PreviewResolve(ctx, request, params.Actor)
	if err != nil {
		return nil, err
	}

	change, err := toolpreview.Changed(
		toolpreview.Record{
			Resource: permission.ResourceServiceFailure,
			ID:       existing.ID,
			Label:    "Service failure " + existing.Number,
			Version:  common.PinnedVersion(existing.Version),
		},
		resolved.Before,
		resolved.After,
		toolpreview.Only("status", "reasonCodeId", "internalNotes", "resolvedAt", "resolvedById"),
		toolpreview.WithRefs(map[string]permission.Resource{
			"reasonCodeId": permission.ResourceServiceFailureReasonCode,
			"resolvedById": permission.ResourceUser,
		}),
		toolpreview.Volatile("resolvedAt"),
	)
	if err != nil {
		return nil, err
	}

	summary := fmt.Sprintf("Would resolve service failure %s as %s.",
		existing.Number, reasonCodeWords(resolved.After))

	return ServiceFailureLifecyclePreview(summary, change, resolved, "resolved"), nil
}

func ServiceFailureLifecyclePreview(
	summary string,
	change *agent.RecordChange,
	preview *servicefailureports.ServiceFailureLifecyclePreview,
	transition string,
) *agent.ToolPreview {
	changes := []*agent.RecordChange{change}

	switch edi := preview.EDI; {
	case edi == nil:
	case edi.Action == ediports.ServiceFailureEDIActionSkipped &&
		edi.SkippedReason == EdiReadyForGeneration:
		body := "Reason: " + reasonCodeWords(preview.After) + ". " +
			strings.TrimSpace(preview.After.InternalNotes)
		if preview.After.Status == servicefailure.StatusVoided {
			body = "Voided: " + strings.TrimSpace(preview.After.VoidReason)
		}
		changes = append(changes, toolpreview.Send(
			toolpreview.Record{
				Resource: permission.ResourceEDI,
				ID:       edi.EDIPartnerID,
				Label:    "Customer's EDI trading partner",
			},
			&agent.MessagePreview{
				Channel: agent.MessageChannelEDI,
				To:      []string{"Customer's EDI trading partner"},
				Subject: "EDI 214 service failure " + transition,
				Body:    body,
			},
		))
		summary += " The customer's trading partner is sent an EDI 214 with the reason."
	case edi.Action == ediports.ServiceFailureEDIActionDuplicate:
		summary += " The EDI 214 for this was already sent and is not sent again."
	case edi.Action == ediports.ServiceFailureEDIActionBlocked:
		summary += " The EDI 214 would not be sent: " + edi.SkippedReason + "."
	}

	return toolpreview.Build(summary, changes...)
}

func reasonCodeWords(failure *servicefailure.ServiceFailure) string {
	if failure.ReasonCode == nil {
		return "the reason code on file"
	}
	if description := strings.TrimSpace(failure.ReasonCode.Description); description != "" {
		return failure.ReasonCode.Code + " (" + description + ")"
	}

	return failure.ReasonCode.Code
}
