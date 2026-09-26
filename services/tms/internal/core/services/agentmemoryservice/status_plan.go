package agentmemoryservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type StatusChange struct {
	Status   agent.MemoryStatus
	ByUserID pulid.ID
	At       int64
}

func StatusActor(actor *services.RequestActor) pulid.ID {
	if actor != nil && actor.IsUser() {
		return actor.UserID
	}

	return pulid.Nil
}

func PlanStatus(entity *agent.Memory, change StatusChange) error {
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
	if change.Status != agent.MemoryStatusRetired {
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

func checkStatusTarget(status agent.MemoryStatus) error {
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
