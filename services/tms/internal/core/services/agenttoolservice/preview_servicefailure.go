package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
)

var _ serviceports.ToolPreviewer = (*resolveServiceFailureTool)(nil)

// serviceFailureResolver is what resolving a failure previews through: the
// lifecycle's own checks, and the EDI 214 it would send.
type serviceFailureResolver interface {
	serviceFailureDecider
	PreviewResolve(
		ctx context.Context,
		req *serviceports.ServiceFailureLifecycleRequest,
		actor *serviceports.RequestActor,
	) (*serviceports.ServiceFailureLifecyclePreview, error)
}

// ediReadyForGeneration is the 214 preflight's word for a 214 that would be
// generated when the change is saved.
const ediReadyForGeneration = "ready_for_generation"

func (t *resolveServiceFailureTool) Preview(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolPreview, error) {
	request, existing, err := t.request(ctx, params)
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
			Version:  pinnedVersion(existing.Version),
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
	changes := []*agent.RecordChange{change}

	switch edi := resolved.EDI; {
	case edi == nil:
	case edi.Action == serviceports.ServiceFailureEDIActionSkipped &&
		edi.SkippedReason == ediReadyForGeneration:
		changes = append(changes, toolpreview.Send(
			toolpreview.Record{
				Resource: permission.ResourceEDI,
				ID:       edi.EDIPartnerID,
				Label:    "Customer's EDI trading partner",
			},
			&agent.MessagePreview{
				Channel: agent.MessageChannelEDI,
				To:      []string{"Customer's EDI trading partner"},
				Subject: "EDI 214 service failure resolved",
				Body: "Reason: " + reasonCodeWords(resolved.After) + ". " +
					strings.TrimSpace(resolved.After.InternalNotes),
			},
		))
		summary += " The customer's trading partner is sent an EDI 214 with the reason."
	case edi.Action == serviceports.ServiceFailureEDIActionDuplicate:
		summary += " The EDI 214 for this was already sent and is not sent again."
	case edi.Action == serviceports.ServiceFailureEDIActionBlocked:
		summary += " The EDI 214 would not be sent: " + edi.SkippedReason + "."
	}

	return toolpreview.Build(summary, changes...), nil
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
