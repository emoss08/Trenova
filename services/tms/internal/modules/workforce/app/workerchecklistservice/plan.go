package workerchecklistservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type (
	ChecklistChange = kernelports.RecordChange[workerchecklist.WorkerChecklist]
	ItemChange      = kernelports.RecordChange[workerchecklist.WorkerChecklistItem]
)

type StartPlan struct {
	Checklist *workerchecklist.WorkerChecklist
	Template  *workerchecklist.WorkerChecklistTemplate
	Existing  bool
}

type ItemPlan struct {
	Item      ItemChange
	Checklist *workerchecklist.WorkerChecklist
}

func (s *Service) PlanStart(ctx context.Context, req *StartRequest) (*StartPlan, error) {
	template, err := s.repo.GetTemplateByID(
		ctx,
		&workerchecklistports.GetChecklistTemplateByIDRequest{
			ID:           req.TemplateID,
			TenantInfo:   req.TenantInfo,
			IncludeItems: true,
		},
	)
	if err != nil {
		return nil, err
	}
	if template.Status != "Active" {
		return nil, errortypes.NewValidationError(
			"templateId",
			errortypes.ErrInvalid,
			"This checklist template is inactive",
		)
	}
	fromEvent := !req.SourceEventID.IsNil()
	if !fromEvent && template.Trigger != workerchecklist.ChecklistTriggerManual {
		return nil, errortypes.NewValidationError(
			"templateId",
			errortypes.ErrInvalidOperation,
			"This checklist is started by the employment event it belongs to, not by hand",
		)
	}

	existing, err := s.repo.ListForWorker(ctx, &workerchecklistports.ListWorkerChecklistsRequest{
		TenantInfo:    req.TenantInfo,
		WorkerID:      req.WorkerID,
		IncludeClosed: fromEvent,
		IncludeItems:  true,
	})
	if err != nil {
		return nil, err
	}
	for _, checklist := range existing {
		if checklist.TemplateID != template.ID {
			continue
		}
		if (fromEvent && checklist.SourceEventID == req.SourceEventID) ||
			(!fromEvent && checklist.IsOpen()) {
			return &StartPlan{Checklist: checklist, Template: template, Existing: true}, nil
		}
	}

	if _, err = s.workerRepo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:         req.WorkerID,
		TenantInfo: req.TenantInfo,
	}); err != nil {
		return nil, err
	}

	startedAt := req.StartedAt
	if startedAt <= 0 {
		startedAt = timeutils.NowUnix()
	}
	checklist := template.Instantiate(req.WorkerID, startedAt, req.UserID, req.SourceEventID)
	multiErr := errortypes.NewMultiError()
	checklist.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	return &StartPlan{Checklist: checklist, Template: template}, nil
}

func (s *Service) PlanCompleteItem(ctx context.Context, req *ItemRequest) (*ItemPlan, error) {
	return s.planSettleItem(ctx, req, workerchecklist.ChecklistItemDone)
}

func (s *Service) PlanSkipItem(ctx context.Context, req *ItemRequest) (*ItemPlan, error) {
	return s.planSettleItem(ctx, req, workerchecklist.ChecklistItemSkipped)
}

func (s *Service) PlanMarkItemNotApplicable(
	ctx context.Context,
	req *ItemRequest,
) (*ItemPlan, error) {
	return s.planSettleItem(ctx, req, workerchecklist.ChecklistItemNotApplicable)
}

func settleNoteRequired(status workerchecklist.ChecklistItemStatus, note string) error {
	if strings.TrimSpace(note) != "" {
		return nil
	}
	switch status {
	case workerchecklist.ChecklistItemSkipped:
		return errortypes.NewValidationError(
			"note",
			errortypes.ErrRequired,
			"Say why this item is being skipped",
		)
	case workerchecklist.ChecklistItemNotApplicable:
		return errortypes.NewValidationError(
			"note",
			errortypes.ErrRequired,
			"Say why this item does not apply",
		)
	case workerchecklist.ChecklistItemPending, workerchecklist.ChecklistItemDone:
		return nil
	}
	return nil
}

func (s *Service) loadItem(
	ctx context.Context,
	req *ReopenItemRequest,
) (*workerchecklist.WorkerChecklistItem, *workerchecklist.WorkerChecklist, error) {
	item, err := s.repo.GetItemByID(ctx, &workerchecklistports.GetWorkerChecklistItemByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	if req.Version > 0 && item.Version != req.Version {
		return nil, nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Checklist item was changed by someone else. Reload and try again",
		)
	}
	checklist, err := s.repo.GetByID(ctx, &workerchecklistports.GetWorkerChecklistByIDRequest{
		ID:         item.ChecklistID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, nil, err
	}
	return item, checklist, nil
}

func (s *Service) planSettleItem(
	ctx context.Context,
	req *ItemRequest,
	status workerchecklist.ChecklistItemStatus,
) (*ItemPlan, error) {
	if err := settleNoteRequired(status, req.Note); err != nil {
		return nil, err
	}
	original, checklist, err := s.loadItem(ctx, &ReopenItemRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Version:    req.Version,
	})
	if err != nil {
		return nil, err
	}
	if original.Status.Settled() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"This item is already settled. Reopen it first",
		)
	}
	if !checklist.IsOpen() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"This checklist is closed",
		)
	}

	item := *original
	if !req.EvidenceDocumentID.IsNil() {
		doc, docErr := s.documentRepo.GetByID(ctx, documentports.GetDocumentByIDRequest{
			ID:         req.EvidenceDocumentID,
			TenantInfo: req.TenantInfo,
		})
		if docErr != nil {
			return nil, docErr
		}
		if doc.ResourceType != workerResourceType || doc.ResourceID != checklist.WorkerID.String() {
			return nil, errortypes.NewValidationError(
				"evidenceDocumentId",
				errortypes.ErrInvalid,
				"Document does not belong to this worker",
			)
		}
		item.EvidenceDocumentID = doc.ID
	}

	now := timeutils.NowUnix()
	item.Status = status
	item.CompletedAt = &now
	item.CompletedByID = req.UserID
	item.AutoCompleted = false
	item.Note = strings.TrimSpace(req.Note)
	return &ItemPlan{
		Item:      ItemChange{Before: original, After: &item},
		Checklist: checklist,
	}, nil
}

func (s *Service) PlanReopenItem(ctx context.Context, req *ReopenItemRequest) (*ItemPlan, error) {
	original, checklist, err := s.loadItem(ctx, req)
	if err != nil {
		return nil, err
	}
	item := *original
	if original.Status.Settled() {
		item.Status = workerchecklist.ChecklistItemPending
		item.CompletedAt = nil
		item.CompletedByID = pulid.Nil
		item.AutoCompleted = false
		item.EvidenceCredentialID = pulid.Nil
	}
	return &ItemPlan{
		Item:      ItemChange{Before: original, After: &item},
		Checklist: checklist,
	}, nil
}

func (s *Service) PlanCancel(ctx context.Context, req *CancelRequest) (*ChecklistChange, error) {
	original, err := s.repo.GetByID(ctx, &workerchecklistports.GetWorkerChecklistByIDRequest{
		ID:           req.ID,
		TenantInfo:   req.TenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if req.Version > 0 && original.Version != req.Version {
		return nil, errortypes.NewValidationError(
			"version",
			errortypes.ErrVersionMismatch,
			"Checklist was changed by someone else. Reload and try again",
		)
	}
	checklist := *original
	if original.IsOpen() {
		now := timeutils.NowUnix()
		checklist.Status = workerchecklist.ChecklistStatusCancelled
		checklist.CancelledAt = &now
		checklist.CancelReason = strings.TrimSpace(req.Reason)
	}
	return &ChecklistChange{Before: original, After: &checklist}, nil
}
