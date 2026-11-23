package workerchecklistservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
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

const workerResourceType = "worker"

// ListForWorker returns the worker's checklists with the auto-satisfy pass
// applied, so what the office sees already reflects the credentials and
// documents on file.
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
	})
	if err != nil {
		return nil, err
	}

	open := make([]*workerchecklist.WorkerChecklist, 0, len(checklists))
	for _, checklist := range checklists {
		if checklist.IsOpen() {
			open = append(open, checklist)
		}
	}
	if len(open) > 0 {
		if err = s.refreshMany(ctx, tenantInfo, workerID, open); err != nil {
			s.l.Warn(
				"failed to refresh checklists",
				zap.String("workerId", workerID.String()),
				zap.Error(err),
			)
		}
	}
	return checklists, nil
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*workerchecklist.WorkerChecklist, error) {
	checklist, err := s.repo.GetByID(ctx, &workerchecklistports.GetWorkerChecklistByIDRequest{
		ID:           id,
		TenantInfo:   tenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if checklist.IsOpen() {
		if err = s.refreshMany(
			ctx,
			tenantInfo,
			checklist.WorkerID,
			[]*workerchecklist.WorkerChecklist{checklist},
		); err != nil {
			s.l.Warn("failed to refresh checklist", zap.String("id", id.String()), zap.Error(err))
		}
	}
	return checklist, nil
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

	if err = s.refreshMany(
		ctx,
		req.TenantInfo,
		req.WorkerID,
		[]*workerchecklist.WorkerChecklist{created},
	); err != nil {
		log.Warn("failed to auto-satisfy new checklist", zap.Error(err))
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

// refreshMany runs the auto-satisfy pass over open checklists for one worker,
// persists what changed, closes checklists whose required items are all
// settled, and flips DQF readiness when an onboarding checklist closes.
func (s *Service) refreshMany(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	checklists []*workerchecklist.WorkerChecklist,
) error {
	if len(checklists) == 0 {
		return nil
	}
	lockKey := workerID.String()
	if _, busy := s.refreshLocks.LoadOrStore(lockKey, struct{}{}); busy {
		return nil
	}
	defer s.refreshLocks.Delete(lockKey)

	evidence, err := s.collectEvidence(ctx, tenantInfo, workerID)
	if err != nil {
		return err
	}

	now := timeutils.NowUnix()
	for _, checklist := range checklists {
		changed := checklist.AutoSatisfy(evidence, now)
		if len(changed) > 0 {
			if err = s.repo.UpdateItems(ctx, changed); err != nil {
				return err
			}
		}
		if err = s.closeIfComplete(ctx, tenantInfo, checklist, now); err != nil {
			return err
		}
		if len(changed) > 0 {
			s.publish(
				ctx,
				tenantInfo,
				realtimeResource,
				permission.OpUpdate,
				checklist.ID,
				pulid.Nil,
			)
		}
	}
	return nil
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
			ResourceType: workerResourceType,
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
