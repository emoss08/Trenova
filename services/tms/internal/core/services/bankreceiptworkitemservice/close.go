package bankreceiptworkitemservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/bankreceiptworkitem"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func ResolveWorkItem(
	entity *bankreceiptworkitem.WorkItem,
	req *serviceports.ResolveBankReceiptWorkItemRequest,
	userID pulid.ID,
	now int64,
) error {
	if !entity.Status.IsActive() {
		return errortypes.NewBusinessError(
			"Only active bank receipt work items can be resolved",
		)
	}
	if strings.TrimSpace(req.ResolutionNote) == "" {
		return errortypes.NewValidationError(
			"resolutionNote",
			errortypes.ErrRequired,
			"Resolution note is required",
		)
	}

	closeWorkItem(entity, &workItemClosure{
		status:     bankreceiptworkitem.StatusResolved,
		resolution: req.ResolutionType,
		note:       req.ResolutionNote,
		userID:     userID,
		at:         now,
	})

	return nil
}

func DismissWorkItem(
	entity *bankreceiptworkitem.WorkItem,
	req *serviceports.DismissBankReceiptWorkItemRequest,
	userID pulid.ID,
	now int64,
) error {
	if !entity.Status.IsActive() {
		return errortypes.NewBusinessError(
			"Only active bank receipt work items can be dismissed",
		)
	}

	closeWorkItem(entity, &workItemClosure{
		status:     bankreceiptworkitem.StatusDismissed,
		resolution: bankreceiptworkitem.ResolutionMarkedFalsePositive,
		note:       req.ResolutionNote,
		userID:     userID,
		at:         now,
	})

	return nil
}

type workItemClosure struct {
	status     bankreceiptworkitem.Status
	resolution bankreceiptworkitem.ResolutionType
	note       string
	userID     pulid.ID
	at         int64
}

func closeWorkItem(entity *bankreceiptworkitem.WorkItem, closure *workItemClosure) {
	at := closure.at
	entity.Status = closure.status
	entity.ResolutionType = closure.resolution
	entity.ResolutionNote = strings.TrimSpace(closure.note)
	entity.ResolvedByUserID = closure.userID
	entity.ResolvedAt = &at
	entity.UpdatedByID = closure.userID
}
