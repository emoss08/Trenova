package workerchecklistservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
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
) ([]*worker.WorkerChecklist, error) {
	checklists, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerChecklistsRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      workerID,
		IncludeClosed: includeClosed,
		IncludeItems:  true,
	})
	if err != nil {
		return nil, err
	}

	open := make([]*worker.WorkerChecklist, 0, len(checklists))
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
) (*worker.WorkerChecklist, error) {
	checklist, err := s.repo.GetByID(ctx, &repositories.GetWorkerChecklistByIDRequest{
		ID:           id,
		TenantInfo:   tenantInfo,
		IncludeItems: true,
	})
	if err != nil {
		return nil, err
	}
	if checklist.IsOpen() {
		if err = s.refreshMany(ctx, tenantInfo, checklist.WorkerID, []*worker.WorkerChecklist{checklist}); err != nil {
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
func (s *Service) Start(ctx context.Context, req *StartRequest) (*worker.WorkerChecklist, error) {
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

	if err = s.refreshMany(ctx, req.TenantInfo, req.WorkerID, []*worker.WorkerChecklist{created}); err != nil {
		log.Warn("failed to auto-satisfy new checklist", zap.Error(err))
	}

	s.audit(&auditParams{
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

// SpawnForEvent implements services.ChecklistSpawner: it starts the default
// template for the trigger the event maps to, if the organisation has one.
func (s *Service) SpawnForEvent(
	ctx context.Context,
	event *worker.WorkerEmploymentEvent,
	wrk *worker.Worker,
	userID pulid.ID,
) (*worker.WorkerChecklist, error) {
	if event == nil || wrk == nil {
		return nil, nil //nolint:nilnil // nothing to spawn
	}
	trigger, ok := worker.ChecklistTriggerForEvent(event.Kind)
	if !ok {
		return nil, nil //nolint:nilnil // informational events spawn nothing
	}
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}
	template, err := s.repo.GetDefaultTemplate(
		ctx,
		&repositories.GetDefaultChecklistTemplateRequest{
			TenantInfo: tenantInfo,
			Trigger:    trigger,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil //nolint:nilnil // no default template configured
		}
		return nil, err
	}
	return s.Start(ctx, &StartRequest{
		TenantInfo:    tenantInfo,
		WorkerID:      wrk.ID,
		TemplateID:    template.ID,
		StartedAt:     event.EffectiveAt,
		SourceEventID: event.ID,
		UserID:        userID,
	})
}

// CloseForEvent implements services.ChecklistSpawner: it cancels the open
// checklists the event makes moot, so a driver who leaves mid-onboarding does
// not keep an onboarding open, and one who is rehired mid-offboarding does
// not keep being offboarded.
func (s *Service) CloseForEvent(
	ctx context.Context,
	event *worker.WorkerEmploymentEvent,
	wrk *worker.Worker,
	userID pulid.ID,
) (int, error) {
	if event == nil || wrk == nil {
		return 0, nil
	}
	kind, ok := worker.ChecklistKindClosedByEvent(event.Kind)
	if !ok {
		return 0, nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}
	open, err := s.repo.ListForWorker(ctx, &repositories.ListWorkerChecklistsRequest{
		TenantInfo:   tenantInfo,
		WorkerID:     wrk.ID,
		IncludeItems: true,
	})
	if err != nil {
		return 0, err
	}

	reason := "Superseded by " + strings.ToLower(string(event.Kind)) + " event"
	log := s.l.With(
		zap.String("operation", "CloseForEvent"),
		zap.String("eventId", event.ID.String()),
	)
	closed := 0
	for _, checklist := range open {
		if checklist.Kind != kind || !checklist.IsOpen() {
			continue
		}
		if _, err = s.cancelChecklist(ctx, checklist, reason, tenantInfo, userID, log); err != nil {
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
) (*worker.WorkerChecklist, error) {
	return s.settleItem(ctx, req, worker.ChecklistItemDone, "Checklist item completed")
}

func (s *Service) SkipItem(ctx context.Context, req *ItemRequest) (*worker.WorkerChecklist, error) {
	return s.settleItem(ctx, req, worker.ChecklistItemSkipped, "Checklist item skipped")
}

func (s *Service) MarkItemNotApplicable(
	ctx context.Context,
	req *ItemRequest,
) (*worker.WorkerChecklist, error) {
	return s.settleItem(
		ctx,
		req,
		worker.ChecklistItemNotApplicable,
		"Checklist item marked not applicable",
	)
}

func (s *Service) settleItem(
	ctx context.Context,
	req *ItemRequest,
	status worker.ChecklistItemStatus,
	comment string,
) (*worker.WorkerChecklist, error) {
	log := s.l.With(zap.String("operation", "SettleItem"), zap.String("itemId", req.ID.String()))

	plan, err := s.planSettleItem(ctx, req, status)
	if err != nil {
		return nil, err
	}
	*plan.Item.Before = *plan.Item.After
	item, checklist := plan.Item.Before, plan.Checklist

	if err = s.repo.UpdateItems(ctx, []*worker.WorkerChecklistItem{item}); err != nil {
		log.Error("failed to settle checklist item", zap.Error(err))
		return nil, err
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerChecklist,
		resourceID: checklist.GetResourceID(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    item,
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
) (*worker.WorkerChecklist, error) {
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

	if err = s.repo.UpdateItems(ctx, []*worker.WorkerChecklistItem{item}); err != nil {
		log.Error("failed to reopen checklist item", zap.Error(err))
		return nil, err
	}

	if checklist.Status == worker.ChecklistStatusCompleted {
		checklist.Status = worker.ChecklistStatusOpen
		checklist.CompletedAt = nil
		if _, err = s.repo.Update(ctx, checklist); err != nil {
			log.Error("failed to reopen checklist", zap.Error(err))
			return nil, err
		}
	}

	s.audit(&auditParams{
		resource:   permission.ResourceWorkerChecklist,
		resourceID: checklist.GetResourceID(),
		operation:  permission.OpUpdate,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    item,
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

func (s *Service) Cancel(ctx context.Context, req *CancelRequest) (*worker.WorkerChecklist, error) {
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
	checklist *worker.WorkerChecklist,
	reason string,
	tenantInfo pagination.TenantInfo,
	userID pulid.ID,
	log *zap.Logger,
) (*worker.WorkerChecklist, error) {
	now := timeutils.NowUnix()
	checklist.Status = worker.ChecklistStatusCancelled
	checklist.CancelledAt = &now
	checklist.CancelReason = strings.TrimSpace(reason)
	items := checklist.Items
	saved, err := s.repo.Update(ctx, checklist)
	if err != nil {
		log.Error("failed to cancel checklist", zap.Error(err))
		return nil, err
	}
	saved.Items = items

	s.audit(&auditParams{
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
	checklists []*worker.WorkerChecklist,
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
	checklist *worker.WorkerChecklist,
	now int64,
) error {
	if !checklist.IsOpen() || !checklist.Progress(now).Complete() {
		return nil
	}
	items := checklist.Items
	checklist.Status = worker.ChecklistStatusCompleted
	checklist.CompletedAt = &now
	saved, err := s.repo.Update(ctx, checklist)
	if err != nil {
		return err
	}
	saved.Items = items

	if saved.Kind == worker.ChecklistKindOnboarding {
		if err = s.workerRepo.UpdateProfileQualification(ctx, &repositories.UpdateProfileQualificationRequest{
			TenantInfo: tenantInfo,
			WorkerID:   saved.WorkerID,
			Qualified:  true,
		}); err != nil {
			s.l.Warn(
				"failed to flag worker as qualified",
				zap.String("workerId", saved.WorkerID.String()),
				zap.Error(err),
			)
		} else {
			s.publish(ctx, tenantInfo, realtimeWorkers, permission.OpUpdate, saved.WorkerID, pulid.Nil)
		}
	}
	s.publish(ctx, tenantInfo, realtimeResource, permission.OpUpdate, saved.ID, pulid.Nil)
	return nil
}

func (s *Service) collectEvidence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (worker.ChecklistEvidence, error) {
	evidence := worker.ChecklistEvidence{
		CredentialsByType: map[pulid.ID]*worker.WorkerCredential{},
		DocumentsByType:   map[pulid.ID]*document.Document{},
	}

	wrk, err := s.workerRepo.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:         workerID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return evidence, err
	}
	evidence.HasPortalAccess = !wrk.UserID.IsNil()

	credentials, err := s.credentialRepo.ListForWorker(
		ctx,
		&repositories.ListWorkerCredentialsRequest{
			TenantInfo:  tenantInfo,
			WorkerID:    workerID,
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
		&repositories.GetDocumentsByResourceRequest{
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
