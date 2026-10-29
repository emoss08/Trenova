package assistanttest

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantartifactports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/shared/pulid"
)

type BunchingRepo struct {
	StubArtifactRepo

	Deleted  []pulid.ID
	ListedBy []assistantartifactports.ListArtifactsByToolCallsRequest
}

func (r *BunchingRepo) Current() []*assistantartifact.Artifact {
	latest := make(map[pulid.ID]*assistantartifact.Artifact, len(r.Upserts))
	order := make([]pulid.ID, 0, len(r.Upserts))
	for _, artifact := range r.Upserts {
		if _, seen := latest[artifact.ID]; !seen {
			order = append(order, artifact.ID)
		}
		latest[artifact.ID] = artifact
	}

	out := make([]*assistantartifact.Artifact, 0, len(order))
	for _, id := range order {
		if !slices.Contains(r.Deleted, id) {
			out = append(out, latest[id])
		}
	}

	return out
}

func (r *BunchingRepo) ListByToolCalls(
	_ context.Context,
	req *assistantartifactports.ListArtifactsByToolCallsRequest,
) ([]*assistantartifact.Artifact, error) {
	r.ListedBy = append(r.ListedBy, *req)
	out := make([]*assistantartifact.Artifact, 0, len(req.CallIDs))
	for _, artifact := range r.Current() {
		if slices.Contains(req.CallIDs, artifact.SourceToolCallID) {
			out = append(out, artifact)
		}
	}

	return out, nil
}

func (r *BunchingRepo) Delete(
	_ context.Context,
	req *assistantartifactports.DeleteArtifactsRequest,
) error {
	r.Deleted = append(r.Deleted, req.IDs...)

	return nil
}

func InvoiceRead(
	callID string,
	number string,
	earlier ...string,
) agentruntimeports.ToolObservation {
	return agentruntimeports.ToolObservation{
		Call: llmports.ToolCall{ID: callID, Name: "invoice_get"},
		Data: map[string]any{
			"id":          pulid.MustNew("inv_").String(),
			"number":      number,
			"status":      "Draft",
			"billToName":  "Acme Foods",
			"totalAmount": "2100.00",
		},
		Earlier: earlier,
	}
}
