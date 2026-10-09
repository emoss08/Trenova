package bulkeditservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/bulkedit"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/tableinsightservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/bulkeditjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const RealtimeResource = "bulk_edit"

var (
	ErrNotYours      = errors.New("this bulk edit belongs to someone else")
	ErrCannotUndo    = errors.New("this bulk edit can no longer be undone")
	ErrNotRunnable   = errors.New("this bulk edit is not waiting to run")
	ErrTooManyTarget = errors.New("too many rows match")
)

type Params struct {
	fx.In

	Logger    *zap.Logger
	Repo      repositories.BulkEditRepository
	Editors   []services.BulkEditor `group:"bulk_editors"`
	Insight   *tableinsightservice.Service
	Realtime  services.RealtimeService
	Workflows services.WorkflowStarter
}

type Service struct {
	l         *zap.Logger
	repo      repositories.BulkEditRepository
	editors   map[permission.Resource]services.BulkEditor
	insight   *tableinsightservice.Service
	realtime  services.RealtimeService
	workflows services.WorkflowStarter
}

func New(p Params) *Service { //nolint:gocritic // fx.In parameter structs are passed by value
	editors := make(map[permission.Resource]services.BulkEditor, len(p.Editors))
	for _, editor := range p.Editors {
		if editor != nil {
			editors[editor.Resource()] = editor
		}
	}
	return &Service{
		l:         p.Logger.Named("service.bulkedit"),
		repo:      p.Repo,
		editors:   editors,
		insight:   p.Insight,
		realtime:  p.Realtime,
		workflows: p.Workflows,
	}
}

func (s *Service) Editor(resource string) (permission.Resource, services.BulkEditor, error) {
	key := permission.Resource(resource)
	editor, ok := s.editors[key]
	if !ok {
		return "", nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			fmt.Sprintf("Rows of %q cannot be edited in bulk", resource),
		)
	}
	return key, editor, nil
}

type PreviewRequest struct {
	TenantInfo pagination.TenantInfo
	Resource   permission.Resource
	Selection  bulkedit.Selection
}

type Preview struct {
	Count     int
	TooMany   bool
	SampleIDs []string
}

const previewSample = 20

func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*Preview, error) {
	ids, total, err := s.resolveTargets(ctx, req.TenantInfo, req.Resource, &req.Selection)
	if err != nil && !errors.Is(err, ErrTooManyTarget) {
		return nil, err
	}
	return &Preview{
		Count:     total,
		TooMany:   total > bulkedit.MaxTargets,
		SampleIDs: ids[:min(len(ids), previewSample)],
	}, nil
}

type StartRequest struct {
	TenantInfo pagination.TenantInfo
	Resource   permission.Resource
	Field      string
	Value      string
	Selection  bulkedit.Selection
}

func (s *Service) Create(ctx context.Context, req *StartRequest) (*bulkedit.BulkEdit, error) {
	editor, ok := s.editors[req.Resource]
	if !ok {
		return nil, errortypes.NewValidationError(
			"resource",
			errortypes.ErrInvalid,
			"This table has no bulk edit",
		)
	}
	if !hasField(editor, req.Field) {
		return nil, errortypes.NewValidationError(
			"field",
			errortypes.ErrInvalid,
			"This field cannot be changed in bulk",
		)
	}

	entity := &bulkedit.BulkEdit{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		UserID:         req.TenantInfo.UserID,
		Resource:       string(req.Resource),
		Field:          req.Field,
		Value:          req.Value,
		Selection:      req.Selection,
		Targets:        []bulkedit.Target{},
		Status:         bulkedit.StatusQueued,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.Create(ctx, entity)
}

func (s *Service) Start(ctx context.Context, req *StartRequest) (*bulkedit.BulkEdit, error) {
	if _, total, err := s.resolveTargets(
		ctx,
		req.TenantInfo,
		req.Resource,
		&req.Selection,
	); err != nil {
		if errors.Is(err, ErrTooManyTarget) {
			return nil, errortypes.NewBusinessError(err.Error())
		}
		return nil, err
	} else if total == 0 {
		return nil, errortypes.NewBusinessError("No rows match, so there is nothing to change")
	}

	entity, err := s.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.run(ctx, entity)
}

func (s *Service) Undo(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*bulkedit.BulkEdit, error) {
	entity, err := s.RequestUndo(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}
	return s.run(ctx, entity)
}

func (s *Service) run(ctx context.Context, entity *bulkedit.BulkEdit) (*bulkedit.BulkEdit, error) {
	_, err := s.workflows.StartWorkflow(ctx,
		client.StartWorkflowOptions{
			ID: bulkeditjobs.WorkflowIDPrefix + entity.ID.String() + "-" + string(
				entity.Status,
			),
			TaskQueue:             temporaltype.TaskQueueSystem.String(),
			WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		},
		bulkeditjobs.BulkEditWorkflow,
		&bulkeditjobs.EditPayload{
			BasePayload: temporaltype.BasePayload{
				OrganizationID: entity.OrganizationID,
				BusinessUnitID: entity.BusinessUnitID,
				UserID:         entity.UserID,
				Timestamp:      timeutils.NowUnix(),
			},
			EditID: entity.ID,
		},
	)
	if err == nil {
		return entity, nil
	}

	s.l.Error("failed to start bulk edit", zap.String("editId", entity.ID.String()), zap.Error(err))
	failed, finalizeErr := s.Finalize(
		ctx,
		pagination.TenantInfo{
			OrgID:  entity.OrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: entity.UserID,
		},
		entity.ID,
		"The bulk edit could not be queued; try again shortly",
	)
	if finalizeErr != nil {
		return nil, finalizeErr
	}
	return failed, nil
}

func (s *Service) RequestUndo(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*bulkedit.BulkEdit, error) {
	entity, err := s.mine(ctx, tenantInfo, id)
	if err != nil {
		return nil, err
	}
	if !entity.CanUndo(timeutils.NowUnix()) {
		return nil, errortypes.NewBusinessError(ErrCannotUndo.Error())
	}

	entity.Status = bulkedit.StatusUndoing
	entity.FailureMessage = ""
	return s.repo.Update(ctx, entity)
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*bulkedit.BulkEdit, error) {
	return s.mine(ctx, tenantInfo, id)
}

func (s *Service) ListMine(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource string,
	limit int,
) ([]*bulkedit.BulkEdit, error) {
	return s.repo.ListForUser(ctx, &repositories.ListBulkEditsRequest{
		TenantInfo: tenantInfo,
		Resource:   resource,
		Limit:      limit,
	})
}

func (s *Service) Prepare(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*bulkedit.BulkEdit, error) {
	entity, err := s.repo.GetByID(
		ctx,
		&repositories.GetBulkEditRequest{TenantInfo: tenantInfo, ID: id},
	)
	if err != nil {
		return nil, err
	}

	switch entity.Status {
	case bulkedit.StatusQueued:
		ids, total, resolveErr := s.resolveTargets(
			ctx,
			tenantInfo,
			permission.Resource(entity.Resource),
			&entity.Selection,
		)
		if resolveErr != nil {
			return nil, resolveErr
		}
		entity.Targets = make([]bulkedit.Target, 0, len(ids))
		for _, target := range ids {
			entity.Targets = append(entity.Targets, bulkedit.Target{ID: target})
		}
		entity.TotalCount = total
		entity.Status = bulkedit.StatusRunning
	case bulkedit.StatusUndoing:
		undo := make([]bulkedit.Target, 0, entity.ChangedCount)
		for _, target := range entity.Targets {
			if target.Changed {
				undo = append(undo, bulkedit.Target{ID: target.ID, Previous: target.Previous})
			}
		}
		entity.Targets = undo
		entity.TotalCount = len(undo)
	case bulkedit.StatusRunning:
		return entity, nil
	case bulkedit.StatusDone, bulkedit.StatusFailed, bulkedit.StatusUndone:
		return nil, ErrNotRunnable
	default:
		return nil, ErrNotRunnable
	}

	entity.ProcessedCount = 0
	entity.ChangedCount = 0
	entity.FailedCount = 0
	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.announce(ctx, updated)
	return updated, nil
}

func (s *Service) ProcessBatch(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	limit int,
) (int, error) {
	entity, err := s.repo.GetByID(
		ctx,
		&repositories.GetBulkEditRequest{TenantInfo: tenantInfo, ID: id},
	)
	if err != nil {
		return 0, err
	}
	editor, ok := s.editors[permission.Resource(entity.Resource)]
	if !ok {
		return 0, ErrNotRunnable
	}

	start := entity.ProcessedCount
	end := min(start+limit, len(entity.Targets))
	if start >= end {
		return 0, nil
	}
	batch := entity.Targets[start:end]
	actor := services.UserActor(tenantInfo)

	if entity.Status == bulkedit.StatusUndoing {
		s.undoBatch(ctx, editor, entity, batch, tenantInfo, actor)
	} else {
		s.applyBatch(ctx, editor, entity, batch, tenantInfo, actor)
	}

	entity.ProcessedCount = end
	if _, err = s.repo.Update(ctx, entity); err != nil {
		return 0, err
	}
	s.announce(ctx, entity)
	return end - start, nil
}

func (s *Service) applyBatch(
	ctx context.Context,
	editor services.BulkEditor,
	entity *bulkedit.BulkEdit,
	batch []bulkedit.Target,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) {
	ids := targetIDs(batch)
	outcomes, err := editor.Apply(ctx, &services.BulkEditApplyRequest{
		TenantInfo: tenantInfo,
		Actor:      actor,
		IDs:        ids,
		Field:      entity.Field,
		Value:      entity.Value,
	})
	recordOutcomes(entity, batch, outcomes, err, true)
}

func (s *Service) undoBatch(
	ctx context.Context,
	editor services.BulkEditor,
	entity *bulkedit.BulkEdit,
	batch []bulkedit.Target,
	tenantInfo pagination.TenantInfo,
	actor *services.RequestActor,
) {
	groups := make(map[string][]int, 4)
	order := make([]string, 0, 4)
	for idx, target := range batch {
		if _, seen := groups[target.Previous]; !seen {
			order = append(order, target.Previous)
		}
		groups[target.Previous] = append(groups[target.Previous], idx)
	}

	for _, previous := range order {
		indexes := groups[previous]
		group := make([]bulkedit.Target, 0, len(indexes))
		for _, idx := range indexes {
			group = append(group, batch[idx])
		}
		outcomes, err := editor.Apply(ctx, &services.BulkEditApplyRequest{
			TenantInfo: tenantInfo,
			Actor:      actor,
			IDs:        targetIDs(group),
			Field:      entity.Field,
			Value:      previous,
		})
		recordOutcomes(entity, group, outcomes, err, false)
		for position, idx := range indexes {
			batch[idx] = group[position]
		}
	}
}

func recordOutcomes(
	entity *bulkedit.BulkEdit,
	batch []bulkedit.Target,
	outcomes []services.BulkEditOutcome,
	batchErr error,
	keepPrevious bool,
) {
	byID := make(map[string]services.BulkEditOutcome, len(outcomes))
	for _, outcome := range outcomes {
		byID[outcome.ID.String()] = outcome
	}

	for idx := range batch {
		target := &batch[idx]
		target.Done = true
		outcome, ok := byID[target.ID]
		switch {
		case batchErr != nil:
			target.Error = failureText(batchErr)
		case !ok:
			target.Error = "The record was not changed"
		case outcome.Err != nil:
			target.Error = failureText(outcome.Err)
		default:
			target.Changed = outcome.Changed
			if keepPrevious {
				target.Previous = outcome.Previous
			}
		}

		if target.Error != "" {
			entity.FailedCount++
		} else if target.Changed {
			entity.ChangedCount++
		}
	}
}

func failureText(err error) string {
	text := err.Error()
	if len(text) > bulkedit.MaxFailureMessage {
		return text[:bulkedit.MaxFailureMessage]
	}
	return text
}

func targetIDs(targets []bulkedit.Target) []pulid.ID {
	ids := make([]pulid.ID, 0, len(targets))
	for _, target := range targets {
		ids = append(ids, pulid.ID(target.ID))
	}
	return ids
}

func (s *Service) Finalize(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	failure string,
) (*bulkedit.BulkEdit, error) {
	entity, err := s.repo.GetByID(
		ctx,
		&repositories.GetBulkEditRequest{TenantInfo: tenantInfo, ID: id},
	)
	if err != nil {
		return nil, err
	}
	if entity.Status.IsTerminal() {
		return entity, nil
	}

	now := timeutils.NowUnix()
	switch {
	case failure != "":
		entity.Status = bulkedit.StatusFailed
		entity.FailureMessage = failure
		entity.CompletedAt = &now
	case entity.Status == bulkedit.StatusUndoing:
		entity.Status = bulkedit.StatusUndone
		entity.UndoneAt = &now
	default:
		entity.Status = bulkedit.StatusDone
		entity.CompletedAt = &now
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.announce(ctx, updated)
	return updated, nil
}

func (s *Service) resolveTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	resource permission.Resource,
	selection *bulkedit.Selection,
) (ids []string, total int, err error) {
	if len(selection.IDs) > 0 {
		return selection.IDs, len(selection.IDs), nil
	}
	if selection.Filter == nil {
		return nil, 0, errortypes.NewValidationError(
			"selection",
			errortypes.ErrRequired,
			"Choose the rows to change",
		)
	}

	result, err := s.insight.MatchingIDs(ctx, resource, &repositories.TableMatchRequest{
		Scope: repositories.TableInsightScope{
			Filter: &pagination.QueryOptions{
				TenantInfo:   tenantInfo,
				Query:        selection.Filter.Query,
				FieldFilters: selection.Filter.FieldFilters,
				FilterGroups: selection.Filter.FilterGroups,
			},
			Options: selection.Filter.Options,
		},
		Limit: bulkedit.MaxTargets + 1,
	})
	if err != nil {
		return nil, 0, err
	}
	if result.Total > bulkedit.MaxTargets {
		return result.IDs, result.Total, fmt.Errorf(
			"%w: %d rows match, more than the %d a bulk edit can change — narrow the filters",
			ErrTooManyTarget,
			result.Total,
			bulkedit.MaxTargets,
		)
	}
	return result.IDs, result.Total, nil
}

func (s *Service) mine(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*bulkedit.BulkEdit, error) {
	entity, err := s.repo.GetByID(
		ctx,
		&repositories.GetBulkEditRequest{TenantInfo: tenantInfo, ID: id},
	)
	if err != nil {
		return nil, err
	}
	if entity.UserID != tenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(ErrNotYours.Error())
	}
	return entity, nil
}

func (s *Service) announce(ctx context.Context, entity *bulkedit.BulkEdit) {
	if s.realtime == nil {
		return
	}
	if err := s.realtime.PublishResourceInvalidation(
		ctx,
		&services.PublishResourceInvalidationRequest{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			AudienceUserID: entity.UserID,
			Resource:       RealtimeResource,
			Action:         "updated",
			RecordID:       entity.ID,
		},
	); err != nil {
		s.l.Debug("failed to announce bulk edit progress", zap.Error(err))
	}
}

func hasField(editor services.BulkEditor, name string) bool {
	for _, field := range editor.Fields() {
		if field.Name == name {
			return true
		}
	}
	return false
}
