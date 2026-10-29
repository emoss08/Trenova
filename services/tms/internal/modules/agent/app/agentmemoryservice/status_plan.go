package agentmemoryservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type StatusChange struct {
	Status   agentmemory.MemoryStatus
	ByUserID pulid.ID
	At       int64
}

func StatusActor(actor *permissionports.RequestActor) pulid.ID {
	if actor != nil && actor.IsUser() {
		return actor.UserID
	}

	return pulid.Nil
}

func PlanStatus(entity *agentmemory.Memory, change StatusChange) error {
	if err := checkStatusTarget(change.Status); err != nil {
		return err
	}
	if entity.Status.IsSuggestion() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A suggested memory is approved or dismissed, not retired or restored",
		)
	}
	if entity.Status == change.Status {
		return errortypes.NewBusinessError(
			"This memory is already {0}",
			strings.ToLower(string(change.Status)),
		)
	}

	entity.Status = change.Status
	if change.Status != agentmemory.MemoryStatusRetired {
		entity.RetiredAt = nil
		entity.RetiredByUserID = nil

		return nil
	}

	at := change.At
	entity.RetiredAt = &at
	entity.RetiredByUserID = nil
	if change.ByUserID.IsNotNil() {
		byUser := change.ByUserID
		entity.RetiredByUserID = &byUser
	}

	return nil
}

func checkStatusTarget(status agentmemory.MemoryStatus) error {
	if !status.IsValid() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a memory status", status),
		)
	}
	if status.IsSuggestion() {
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A memory is suggested only by feedback; approve or dismiss the suggestion instead",
		)
	}

	return nil
}

// statusComment is what the audit log says a status change did.
func statusComment(from, to agentmemory.MemoryStatus) string {
	switch {
	case to == agentmemory.MemoryStatusPaused:
		return "Agent memory paused"
	case to == agentmemory.MemoryStatusRetired:
		return "Agent memory retired"
	case from == agentmemory.MemoryStatusPaused:
		return "Agent memory resumed"
	default:
		return "Agent memory restored"
	}
}
