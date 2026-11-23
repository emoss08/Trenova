package workerchecklistservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerchecklist"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

// ListForWorker returns the worker's checklists as stored. The auto-satisfy
// pass runs when the evidence changes (RefreshEvidence), never on a read.
func (s *Service) ListForWorker(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	includeClosed bool,
) ([]*workerchecklist.WorkerChecklist, error) {
	checklists, err := s.repo.ListForWorker(ctx, &workerchecklistports.ListWorkerChecklistsRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      workerID,
		IncludeClosed: includeClosed,
		IncludeItems:  true,
		Limit:         pagination.MaxLimit,
	})
	if err != nil {
		return nil, err
	}
	return checklists, nil
}

func (s *Service) CountClosed(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (int, error) {
	return s.repo.CountClosedForWorker(ctx, tenantInfo, workerID)
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*workerchecklist.WorkerChecklist, error) {
	return s.repo.GetByID(ctx, &workerchecklistports.GetWorkerChecklistByIDRequest{
		ID:           id,
		TenantInfo:   tenantInfo,
		IncludeItems: true,
	})
}

type StartRequest struct {
	TenantInfo    pagination.TenantInfo
	WorkerID      pulid.ID
	TemplateID    pulid.ID
	StartedAt     int64
	SourceEventID pulid.ID
	UserID        pulid.ID
}

// Start spawns a checklist from a template. Onboarding and offboarding
// templates are driven by the employment event that carries them: they can
// only be started with a source event, once per event, so a worker is
// onboarded once per hire rather than whenever somebody presses a button.
// Templates started by hand never run twice at once; the open one is returned.
func (s *Service) Start(
	ctx context.Context,
	req *StartRequest,
) (*workerchecklist.WorkerChecklist, error) {
	log := s.l.With(
		zap.String("operation", "Start"),
		zap.String("workerId", req.WorkerID.String()),
		zap.String("templateId", req.TemplateID.String()),
	)

	plan, err := s.PlanStart(ctx, req)
	if err != nil {
		return nil, err
	}
	if plan.Existing {
		return plan.Checklist, nil
	}
	checklist := plan.Checklist

	created, err := s.repo.Create(ctx, checklist)
	if err != nil {
		log.Error("failed to start checklist", zap.Error(err))
		return nil, err
	}

	if err = s.RefreshEvidence(ctx, req.TenantInfo, req.WorkerID); err != nil {
		log.Warn("failed to auto-satisfy new checklist", zap.Error(err))
	} else if refreshed, getErr := s.Get(ctx, req.TenantInfo, created.ID); getErr == nil {
		created = refreshed
	}

	s.audit(ctx, &auditParams{
		resource:   permission.ResourceWorkerChecklist,
		resourceID: created.GetResourceID(),
		operation:  permission.OpCreate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    created,
		comment:    "Checklist started: " + created.Name,
		log:        log,
	})
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpCreate, created.ID, req.UserID)

	return created, nil
}

func (s *Service) PlanForEvent(
	ctx context.Context,
	kind employment.EmploymentEventKind,
	wrk *worker.Worker,
) (*workerchecklistports.EventPlan, error) {
	plan := &workerchecklistports.EventPlan{Worker: wrk}
	if wrk == nil {
		return plan, nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}

	if closes, ok := workerchecklist.ChecklistKindClosedByEvent(kind); ok {
		checklists, err := s.repo.ListForWorker(
			ctx,
			&workerchecklistports.ListWorkerChecklistsRequest{
				TenantInfo: tenantInfo,
				WorkerID:   wrk.ID,
			},
		)
		if err != nil {
			return nil, err
		}
		for _, checklist := range checklists {
			if checklist.Kind == closes && checklist.IsOpen() {
				plan.Close = append(plan.Close, checklist)
			}
		}
	}

	if trigger, ok := workerchecklist.ChecklistTriggerForEvent(kind); ok {
		template, err := s.repo.GetDefaultTemplate(
			ctx,
			&workerchecklistports.GetDefaultChecklistTemplateRequest{
				TenantInfo: tenantInfo,
				Trigger:    trigger,
			},
		)
		switch {
		case err == nil:
			plan.Template = template
		case errortypes.IsNotFoundError(err):
		default:
			return nil, err
		}
	}
	return plan, nil
}

func (s *Service) SpawnForEvent(
	ctx context.Context,
	event *employment.WorkerEmploymentEvent,
	plan *workerchecklistports.EventPlan,
	userID pulid.ID,
) (*workerchecklist.WorkerChecklist, error) {
	if event == nil || plan == nil || plan.Worker == nil || plan.Template == nil {
		return nil, nil //nolint:nilnil // the plan starts nothing
	}
	wrk := plan.Worker
	return s.Start(ctx, &StartRequest{
		TenantInfo:    pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID},
		WorkerID:      wrk.ID,
		TemplateID:    plan.Template.ID,
		StartedAt:     event.EffectiveAt,
		SourceEventID: event.ID,
		UserID:        userID,
	})
}

func (s *Service) CloseForEvent(
	ctx context.Context,
	event *employment.WorkerEmploymentEvent,
	plan *workerchecklistports.EventPlan,
	userID pulid.ID,
) (int, error) {
	if event == nil || plan == nil || plan.Worker == nil || len(plan.Close) == 0 {
		return 0, nil
	}
	wrk := plan.Worker
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}
	reason := "Superseded by " + strings.ToLower(string(event.Kind)) + " event"
	log := s.l.With(
		zap.String("operation", "CloseForEvent"),
		zap.String("eventId", event.ID.String()),
	)
	closed := 0
	for _, checklist := range plan.Close {
		if !checklist.IsOpen() {
			continue
		}
		if _, err := s.cancelChecklist(ctx, checklist, reason, tenantInfo, userID, log); err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

type ItemRequest struct {
	ID                 pulid.ID
	TenantInfo         pagination.TenantInfo
	Note               string
	EvidenceDocumentID pulid.ID
	Version            int64
	UserID             pulid.ID
}

func (s *Service) CompleteItem(
	ctx context.Context,
	req *ItemRequest,
) (*workerchecklist.WorkerChecklist, error) {
	return s.settleItem(ctx, req, workerchecklist.ChecklistItemDone, "Checklist item completed")
}

func (s *Service) SkipItem(
	ctx context.Context,
	req *ItemRequest,
) (*workerchecklist.WorkerChecklist, error) {
	return s.settleItem(ctx, req, workerchecklist.ChecklistItemSkipped, "Checklist item skipped")
}

func (s *Service) MarkItemNotApplicable(
	ctx context.Context,
	req *ItemRequest,
) (*workerchecklist.WorkerChecklist, error) {
	return s.settleItem(
		ctx,
		req,
		workerchecklist.ChecklistItemNotApplicable,
		"Checklist item marked not applicable",
	)
}

func (s *Service) settleItem(
	ctx context.Context,
	req *ItemRequest,
	status workerchecklist.ChecklistItemStatus,
	comment string,
) (*workerchecklist.WorkerChecklist, error) {
	log := s.l.With(zap.String("operation", "SettleItem"), zap.String("itemId", req.ID.String()))

	plan, err := s.planSettleItem(ctx, req, status)
	if err != nil {
		return nil, err
	}
	*plan.Item.Before = *plan.Item.After
	item, checklist := plan.Item.Before, plan.Checklist

	if err = s.repo.UpdateItems(ctx, []*workerchecklist.WorkerChecklistItem{item}); err != nil {
		log.Error("failed to settle checklist item", zap.Error(err))
		return nil, err
	}

	s.audit(ctx, &auditParams{
		resource:   permission.ResourceWorkerChecklist,
		resourceID: checklist.GetResourceID(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    item,
		owner:      checklist,
		comment:    comment + ": " + item.Label,
		log:        log,
	})

	return s.Get(ctx, req.TenantInfo, checklist.ID)
}

type ReopenItemRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Version    int64
	UserID     pulid.ID
}

// ReopenItem puts a settled item back to Pending. Auto-completed items are
// reopened too, but the next refresh will tick them again if the evidence is
// still there, so reopening one only makes sense after the evidence changed.
func (s *Service) ReopenItem(
	ctx context.Context,
	req *ReopenItemRequest,
) (*workerchecklist.WorkerChecklist, error) {
	log := s.l.With(zap.String("operation", "ReopenItem"), zap.String("itemId", req.ID.String()))

	plan, err := s.PlanReopenItem(ctx, req)
	if err != nil {
		return nil, err
	}
	checklist := plan.Checklist
	if !plan.Item.Before.Status.Settled() {
		return s.Get(ctx, req.TenantInfo, checklist.ID)
	}
	*plan.Item.Before = *plan.Item.After
	item := plan.Item.Before

	if err = s.repo.UpdateItems(ctx, []*workerchecklist.WorkerChecklistItem{item}); err != nil {
		log.Error("failed to reopen checklist item", zap.Error(err))
		return nil, err
	}

	if checklist.Status == workerchecklist.ChecklistStatusCompleted {
		checklist.Status = workerchecklist.ChecklistStatusOpen
		checklist.CompletedAt = nil
		if _, err = s.repo.Update(ctx, checklist); err != nil {
			log.Error("failed to reopen checklist", zap.Error(err))
			return nil, err
		}
	}

	s.audit(ctx, &auditParams{
		resource:   permission.ResourceWorkerChecklist,
		resourceID: checklist.GetResourceID(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    item,
		owner:      checklist,
		comment:    "Checklist item reopened: " + item.Label,
		log:        log,
	})
	s.publish(ctx, req.TenantInfo, realtimeResource, permission.OpUpdate, checklist.ID, req.UserID)

	return s.Get(ctx, req.TenantInfo, checklist.ID)
}

type CancelRequest struct {
	ID         pulid.ID
	TenantInfo pagination.TenantInfo
	Reason     string
	Version    int64
	UserID     pulid.ID
}

func (s *Service) Cancel(
	ctx context.Context,
	req *CancelRequest,
) (*workerchecklist.WorkerChecklist, error) {
	log := s.l.With(zap.String("operation", "Cancel"), zap.String("id", req.ID.String()))

	change, err := s.PlanCancel(ctx, req)
	if err != nil {
		return nil, err
	}
	if !change.Before.IsOpen() {
		return change.Before, nil
	}
	checklist := change.Before

	return s.cancelChecklist(ctx, checklist, req.Reason, req.TenantInfo, req.UserID, log)
}

func (s *Service) cancelChecklist(
	ctx context.Context,
	checklist *workerchecklist.WorkerChecklist,
	reason string,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	log *zap.Logger,
) (*workerchecklist.WorkerChecklist, error) {
	now := timeutils.NowUnix()
	checklist.Status = workerchecklist.ChecklistStatusCancelled
	checklist.CancelledAt = &now
	checklist.CancelReason = strings.TrimSpace(reason)
	items := checklist.Items
	saved, err := s.repo.Update(ctx, checklist)
	if err != nil {
		log.Error("failed to cancel checklist", zap.Error(err))
		return nil, err
	}
	saved.Items = items

	s.audit(ctx, &auditParams{
		resource:   permission.ResourceWorkerChecklist,
		resourceID: saved.GetResourceID(),
		operation:  permission.OpCancel,
		userID:     userID,
		tenant:     tenantInfo,
		current:    saved,
		comment:    "Checklist cancelled",
		log:        log,
	})
	s.publish(ctx, tenantInfo, realtimeResource, permission.OpCancel, saved.ID, userID)

	return saved, nil
}

// RefreshEvidence runs the auto-satisfy pass over the worker's open
// checklists: it ticks the items whose evidence now exists, closes the
// checklists whose required items are settled, and flips DQF readiness when
// an onboarding checklist closes. The checklists are locked for the pass, so
// two passes for one worker on any replica run one after the other, and a
// second pass finds nothing left to tick.
func (s *Service) RefreshEvidence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) error {
	return s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		checklists, err := s.repo.LockOpenForWorker(
			txCtx,
			&workerchecklistports.LockOpenChecklistsRequest{
				TenantInfo: tenantInfo,
				WorkerID:   workerID,
			},
		)
		if err != nil || len(checklists) == 0 {
			return err
		}
		evidence, err := s.collectEvidence(txCtx, tenantInfo, workerID)
		if err != nil {
			return err
		}

		now := timeutils.NowUnix()
		for _, checklist := range checklists {
			changed := checklist.AutoSatisfy(evidence, now)
			if len(changed) > 0 {
				if err = s.repo.UpdateItems(txCtx, changed); err != nil {
					return err
				}
			}
			if err = s.closeIfComplete(txCtx, tenantInfo, checklist, now); err != nil {
				return err
			}
			if len(changed) > 0 {
				s.publish(
					txCtx,
					tenantInfo,
					realtimeResource,
					permission.OpUpdate,
					checklist.ID,
					pulid.Nil,
				)
			}
		}
		return nil
	})
}

// ResourceDocumentsChanged refreshes a worker's checklists when a document
// filed against the worker is added, changed or removed.
func (s *Service) ResourceDocumentsChanged(
	ctx context.Context,
	change documentports.ResourceDocumentsChange,
) {
	if change.ResourceType != permission.ResourceWorker.String() {
		return
	}
	workerID, err := pulid.Parse(change.ResourceID)
	if err != nil || workerID.IsNil() {
		return
	}
	if err = s.RefreshEvidence(ctx, change.TenantInfo, workerID); err != nil {
		s.l.Warn(
			"failed to refresh checklists after a document change",
			zap.String("workerId", workerID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) closeIfComplete(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	checklist *workerchecklist.WorkerChecklist,
	now int64,
) error {
	if !checklist.IsOpen() || !checklist.Progress(now).Complete() {
		return nil
	}
	items := checklist.Items
	checklist.Status = workerchecklist.ChecklistStatusCompleted
	checklist.CompletedAt = &now
	saved, err := s.repo.Update(ctx, checklist)
	if err != nil {
		return err
	}
	saved.Items = items

	if saved.Kind == workerchecklist.ChecklistKindOnboarding {
		if err = s.workerRepo.UpdateProfileQualification(
			ctx,
			&workerports.UpdateProfileQualificationRequest{
				TenantInfo: tenantInfo,
				WorkerID:   saved.WorkerID,
				Qualified:  true,
			},
		); err != nil {
			s.l.Warn(
				"failed to flag worker as qualified",
				zap.String("workerId", saved.WorkerID.String()),
				zap.Error(err),
			)
		} else {
			s.publish(
				ctx,
				tenantInfo,
				realtimeWorkers,
				permission.OpUpdate,
				saved.WorkerID,
				pulid.Nil,
			)
		}
	}
	s.publish(ctx, tenantInfo, realtimeResource, permission.OpUpdate, saved.ID, pulid.Nil)
	return nil
}

func (s *Service) collectEvidence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (workerchecklist.ChecklistEvidence, error) {
	evidence := workerchecklist.ChecklistEvidence{
		CredentialsByType: map[pulid.ID]*workercredential.WorkerCredential{},
		DocumentsByType:   map[pulid.ID]*document.Document{},
	}

	wrk, err := s.workerRepo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID:         workerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return evidence, err
	}
	evidence.HasPortalAccess = !wrk.UserID.IsNil()

	credentials, err := s.credentialRepo.ListForWorker(
		ctx,
		&workercredentialports.ListWorkerCredentialsRequest{
			TenantInfo:  tenantInfo,
			WorkerIDs:   []pulid.ID{workerID},
			IncludeType: true,
		},
	)
	if err != nil {
		return evidence, err
	}
	for _, cred := range credentials {
		if cred.IsActive() {
			evidence.CredentialsByType[cred.CredentialTypeID] = cred
		}
	}

	documents, err := s.documentRepo.GetByResourceID(
		ctx,
		&documentports.GetDocumentsByResourceRequest{
			TenantInfo:   tenantInfo,
			ResourceID:   workerID.String(),
			ResourceType: permission.ResourceWorker.String(),
		},
	)
	if err != nil {
		return evidence, err
	}
	for _, doc := range documents {
		if doc.DocumentTypeID == nil || doc.Status == document.StatusArchived {
			continue
		}
		if _, seen := evidence.DocumentsByType[*doc.DocumentTypeID]; !seen {
			evidence.DocumentsByType[*doc.DocumentTypeID] = doc
		}
	}

	return evidence, nil
}
