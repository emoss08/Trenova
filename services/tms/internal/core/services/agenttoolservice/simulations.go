package agenttoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

// The previews below say what a write would change by reading the record
// as it stands. A tool whose current state cannot be read still answers,
// with the change it would make and no "from".

type tractorReader interface {
	Get(ctx context.Context, req repositories.GetTractorByIDRequest) (*tractor.Tractor, error)
}

type trailerReader interface {
	Get(ctx context.Context, req repositories.GetTrailerByIDRequest) (*trailer.Trailer, error)
}

func (t *updateTractorStatusTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	ids, status, err := equipmentStatusArgs(params.Params, "tractorIds")
	if err != nil {
		return nil, err
	}

	reader, _ := t.tractors.(tractorReader)
	changes := make([]agent.FieldChange, 0, len(ids))
	for _, id := range ids {
		change := agent.FieldChange{Field: "tractor " + id.String(), To: string(status)}
		if reader != nil {
			if entity, gErr := reader.Get(ctx, repositories.GetTractorByIDRequest{
				ID:         id,
				TenantInfo: tenantFrom(params),
			}); gErr == nil {
				change.Field = "tractor " + entity.Code
				change.From = string(entity.Status)
			}
		}
		changes = append(changes, change)
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf("Would set %s to %s.", countOf(len(ids), "tractor"), status),
		Changes: changes,
	}, nil
}

func (t *updateTrailerStatusTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	ids, status, err := equipmentStatusArgs(params.Params, "trailerIds")
	if err != nil {
		return nil, err
	}

	reader, _ := t.trailers.(trailerReader)
	changes := make([]agent.FieldChange, 0, len(ids))
	for _, id := range ids {
		change := agent.FieldChange{Field: "trailer " + id.String(), To: string(status)}
		if reader != nil {
			if entity, gErr := reader.Get(ctx, repositories.GetTrailerByIDRequest{
				ID:         id,
				TenantInfo: tenantFrom(params),
			}); gErr == nil {
				change.Field = "trailer " + entity.Code
				change.From = string(entity.Status)
			}
		}
		changes = append(changes, change)
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf("Would set %s to %s.", countOf(len(ids), "trailer"), status),
		Changes: changes,
	}, nil
}

func (t *cancelShipmentTool) Simulate(
	ctx context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	shipmentID, err := requirePulid(params.Params, "shipmentId")
	if err != nil {
		return nil, err
	}
	reason, err := requireString(params.Params, "cancelReason")
	if err != nil {
		return nil, err
	}

	entity, err := t.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenantFrom(params),
	})
	if err != nil {
		return nil, err
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf(
			"Would cancel shipment %s, releasing its assignments and stopping it being billed. Reason: %s",
			entity.ProNumber, reason,
		),
		Changes: []agent.FieldChange{
			{Field: "status", From: string(entity.Status), To: "Canceled"},
			{Field: "cancelReason", To: reason},
		},
	}, nil
}

func (t *assignMoveTool) Simulate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	moveID, err := requirePulid(params.Params, "shipmentMoveId")
	if err != nil {
		return nil, err
	}
	workerID, err := requirePulid(params.Params, "primaryWorkerId")
	if err != nil {
		return nil, err
	}
	tractorID, err := requirePulid(params.Params, "tractorId")
	if err != nil {
		return nil, err
	}

	changes := []agent.FieldChange{
		{Field: "primaryWorkerId", From: "unassigned", To: workerID.String()},
		{Field: "tractorId", From: "unassigned", To: tractorID.String()},
	}
	if trailerID, ok, pErr := optionalPulid(params.Params, "trailerId"); pErr != nil {
		return nil, pErr
	} else if ok {
		changes = append(changes, agent.FieldChange{Field: "trailerId", From: "none", To: trailerID.String()})
	}
	if secondID, ok, pErr := optionalPulid(params.Params, "secondaryWorkerId"); pErr != nil {
		return nil, pErr
	} else if ok {
		changes = append(changes, agent.FieldChange{Field: "secondaryWorkerId", From: "none", To: secondID.String()})
	}

	return &agent.ToolSimulation{
		Summary: fmt.Sprintf("Would put driver %s on move %s with tractor %s.", workerID, moveID, tractorID),
		Changes: changes,
	}, nil
}

func (t *rememberTool) Simulate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	content, err := requireString(params.Params, "content")
	if err != nil {
		return nil, err
	}
	kind := optionalString(params.Params, "kind")
	if kind == "" {
		kind = string(agent.MemoryKindFact)
	}

	changes := []agent.FieldChange{
		{Field: "kind", To: kind},
		{Field: "content", To: content},
	}
	if subject := strings.TrimSpace(optionalString(params.Params, "subjectType")); subject != "" {
		changes = append(changes, agent.FieldChange{
			Field: "about",
			To:    strings.ToLower(subject) + " " + optionalString(params.Params, "subjectId"),
		})
	}

	return &agent.ToolSimulation{
		Summary: "Would record a memory every later run of every agent reads.",
		Changes: changes,
	}, nil
}

func (t *forgetMemoryTool) Simulate(
	_ context.Context,
	params serviceports.ToolExecuteParams,
) (*agent.ToolSimulation, error) {
	memoryID, err := requirePulid(params.Params, "memoryId")
	if err != nil {
		return nil, err
	}

	return &agent.ToolSimulation{
		Summary: "Would retire memory " + memoryID.String() + " so no later run reads it.",
		Changes: []agent.FieldChange{{Field: "status", From: "Active", To: "Retired"}},
	}, nil
}

func countOf(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}

	return fmt.Sprintf("%d %ss", n, noun)
}
