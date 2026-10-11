package agenttoolstest

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/operationstools"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/shared/pulid"
)

// PreviewResolve answers from the request, so the fake's record of the
// write stays empty unless the tool resolves.
func (f *FakeFailureDecider) PreviewResolve(
	_ context.Context,
	req *servicefailureports.ServiceFailureLifecycleRequest,
	actor *permissionports.RequestActor,
) (*servicefailureports.ServiceFailureLifecyclePreview, error) {
	after := *f.Existing
	after.Status = servicefailure.StatusResolved
	after.InternalNotes = req.Notes
	after.ResolvedByID = &actor.UserID
	if req.ReasonCodeID.IsNotNil() {
		reasonID := req.ReasonCodeID
		after.ReasonCodeID = &reasonID
		after.ReasonCode = &servicefailure.ReasonCode{
			Code:        "LATE_SHIPPER",
			Description: "Shipper closed early",
		}
	}

	return &servicefailureports.ServiceFailureLifecyclePreview{
		Before: f.Existing,
		After:  &after,
		EDI: &servicefailureports.ServiceFailure214LifecycleResult{
			Action:        ediports.ServiceFailureEDIActionSkipped,
			SkippedReason: operationstools.EdiReadyForGeneration,
			EDIPartnerID:  pulid.MustNew("edip_"),
		},
	}, nil
}
