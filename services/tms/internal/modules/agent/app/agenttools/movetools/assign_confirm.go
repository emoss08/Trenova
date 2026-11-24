package movetools

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchconsoleservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// moveReader reads moves back as the board draws them.
type moveReader interface {
	GetMoves(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		moveIDs []pulid.ID,
	) ([]*dispatchconsoleservice.BoardMove, error)
}

// assignedStates reads the moves an assignment ran on back, in one read, and
// says for each who is on it now, with what equipment and where it stands, so
// the agent tells the person what happened without reading every shipment
// again. A move it cannot read back is said so; the write still went through.
func assignedStates(
	ctx context.Context,
	console moveReader,
	tenant pagination.TenantInfo,
	moveIDs []pulid.ID,
) []string {
	if console == nil || len(moveIDs) == 0 {
		return nil
	}

	moves, err := console.GetMoves(ctx, tenant, moveIDs)
	if err != nil {
		return []string{
			"assigned, but the moves could not be read back to confirm who is on them",
		}
	}

	byID := make(map[pulid.ID]*dispatchconsoleservice.BoardMove, len(moves))
	for _, move := range moves {
		if move != nil && move.BoardMove != nil {
			byID[move.MoveID] = move
		}
	}

	states := make([]string, 0, len(moveIDs))
	for _, id := range moveIDs {
		move, ok := byID[id]
		if !ok {
			states = append(states, "a move that could not be read back")

			continue
		}
		states = append(states, assignedState(move))
	}

	return states
}

// assignedState is one move as it stands after the assignment: "S-1: Adam
// Stephens with tractor 104 and trailer 5301, move Assigned, the driver has not
// acknowledged it yet".
func assignedState(move *dispatchconsoleservice.BoardMove) string {
	who := move.AssignedWorkerName
	if who == "" {
		who = "no driver shows on it"
	}
	equipment := make([]string, 0, 2)
	if move.AssignedTractorCode != "" {
		equipment = append(equipment, "tractor "+move.AssignedTractorCode)
	}
	if move.AssignedTrailerCode != "" {
		equipment = append(equipment, "trailer "+move.AssignedTrailerCode)
	}
	if len(equipment) > 0 {
		who += " with " + strings.Join(equipment, " and ")
	}

	state := fmt.Sprintf("%s: %s, move %s", move.ProNumber, who, move.MoveStatus)
	if ack := strings.TrimSpace(string(move.AssignmentAckStatus)); ack != "" {
		state += ", driver acknowledgement " + ack
	}

	return state
}

// confirmAssignments adds what the moves now hold to a result, so the person
// approving and the agent after them read the outcome rather than a count.
func confirmAssignments(
	ctx context.Context,
	console moveReader,
	tenant pagination.TenantInfo,
	result *agentrecord.ToolExecutionResult,
	moveIDs []pulid.ID,
) {
	if result == nil {
		return
	}
	if states := assignedStates(ctx, console, tenant, moveIDs); len(states) > 0 {
		result.State = strings.Join(states, "; ")
	}
}
