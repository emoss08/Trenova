package inboundmessageservice

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type assigneeDirectory interface {
	ListUsersWithPermission(
		ctx context.Context,
		req permissionports.ListUsersWithPermissionRequest,
	) ([]permissionports.PermittedUser, error)
}

type organizationReader interface {
	GetByID(
		ctx context.Context,
		req tenantports.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
}

func listWorkersRequest(
	tenantInfo pagination.TenantInfo,
	now int64,
) permissionports.ListUsersWithPermissionRequest {
	return permissionports.ListUsersWithPermissionRequest{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		Resource:       workerPermission.resource,
		Operation:      workerPermission.operation,
		Now:            now,
	}
}

func (s *Service) Targets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (triage.Targets, error) {
	rows, err := s.targetRepo.List(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return triage.ResolveTargets(rows), nil
}

func (s *Service) ResponseTargets(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*inboundmessageports.ResponseTargetView, error) {
	rows, err := s.targetRepo.List(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return responseTargetViews(rows), nil
}

func responseTargetViews(rows []*triage.ResponseTarget) []*inboundmessageports.ResponseTargetView {
	targets := triage.ResolveTargets(rows)
	own := make(map[inboundmessage.Classification]struct{}, len(rows))
	for _, row := range rows {
		own[row.Classification] = struct{}{}
	}

	kinds := inboundmessage.AllClassifications()
	views := make([]*inboundmessageports.ResponseTargetView, 0, len(kinds))
	for _, kind := range kinds {
		_, custom := own[kind]
		views = append(views, &inboundmessageports.ResponseTargetView{
			Classification: kind,
			Minutes:        targets.For(kind),
			IsDefault:      !custom,
		})
	}

	return views
}

type TargetChange struct {
	Classification inboundmessage.Classification
	Minutes        *int
}

type UpdateTargetsRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	Changes    []TargetChange
}

func targetsToSave(req *UpdateTargetsRequest) (*inboundmessageports.SaveTargetsRequest, error) {
	multiErr := errortypes.NewMultiError()
	save := &inboundmessageports.SaveTargetsRequest{
		TenantInfo: req.TenantInfo,
		Upsert:     make([]*triage.ResponseTarget, 0, len(req.Changes)),
		Reset:      make([]inboundmessage.Classification, 0, len(req.Changes)),
	}
	seen := make(map[inboundmessage.Classification]struct{}, len(req.Changes))
	for idx, change := range req.Changes {
		field := fmt.Sprintf("input[%d]", idx)
		if _, dup := seen[change.Classification]; dup {
			multiErr.Add(field+".classification", errortypes.ErrDuplicate,
				"Each kind may be set once")
			continue
		}
		seen[change.Classification] = struct{}{}
		if change.Minutes == nil {
			if !change.Classification.IsValid() {
				multiErr.Add(field+".classification", errortypes.ErrInvalid,
					"Kind is not a known one")
				continue
			}
			save.Reset = append(save.Reset, change.Classification)
			continue
		}

		target := &triage.ResponseTarget{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			Classification: change.Classification,
			Minutes:        *change.Minutes,
			UpdatedByID:    req.ActorID,
		}
		target.Validate(multiErr.WithPrefix(field))
		save.Upsert = append(save.Upsert, target)
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return save, nil
}

func (s *Service) UpdateResponseTargets(
	ctx context.Context,
	req *UpdateTargetsRequest,
) ([]*inboundmessageports.ResponseTargetView, error) {
	if len(req.Changes) == 0 {
		return s.ResponseTargets(ctx, req.TenantInfo)
	}

	save, err := targetsToSave(req)
	if err != nil {
		return nil, err
	}
	if err = s.targetRepo.Save(ctx, *save); err != nil {
		return nil, err
	}

	changed := make([]inboundmessage.Classification, 0, len(req.Changes))
	for _, change := range req.Changes {
		changed = append(changed, change.Classification)
	}
	s.scheduleRecomputeDue(ctx, req.TenantInfo, changed)

	return s.ResponseTargets(ctx, req.TenantInfo)
}

func (s *Service) scheduleRecomputeDue(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	changed []inboundmessage.Classification,
) {
	if s.wakes != nil && s.wakes.Enabled() {
		err := s.wakes.RecomputeDue(ctx, tenantInfo, changed)
		if err == nil {
			return
		}
		s.l.Warn("could not start recomputing due times; recomputing them now",
			zap.Error(err))
	}

	if _, err := s.RecomputeDueFor(ctx, tenantInfo, changed); err != nil {
		s.l.Error("could not recompute due times after a target changed", zap.Error(err))
	}
}

func (s *Service) RecomputeDueFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	classifications []inboundmessage.Classification,
) (int64, error) {
	targets, err := s.Targets(ctx, tenantInfo)
	if err != nil {
		return 0, err
	}

	var total int64
	for _, class := range classifications {
		changed, rErr := s.queueRepo.RecomputeDue(ctx, inboundmessageports.RecomputeDueRequest{
			TenantInfo:     tenantInfo,
			Classification: class,
			Minutes:        targets.For(class),
		})
		if rErr != nil {
			return total, rErr
		}
		total += changed
	}
	if total > 0 {
		s.notify.TenantChanged(ctx, tenantInfo)
	}

	return total, nil
}

func (s *Service) Settings(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*triage.Settings, error) {
	return s.settingsRepo.Get(ctx, tenantInfo)
}

type UpdateSettingsRequest struct {
	TenantInfo     pagination.TenantInfo
	OnCallUserID   pulid.ID
	EndOfDayMinute int
	Version        int64
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	req *UpdateSettingsRequest,
) (*triage.Settings, error) {
	settings := &triage.Settings{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		OnCallUserID:   req.OnCallUserID,
		EndOfDayMinute: req.EndOfDayMinute,
		Version:        req.Version,
	}
	multiErr := errortypes.NewMultiError()
	settings.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if req.OnCallUserID.IsNotNil() {
		people, err := s.assignablePeople(ctx, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		if _, ok := people[req.OnCallUserID]; !ok {
			return nil, errortypes.NewValidationError("onCallUserId", errortypes.ErrInvalid,
				"The on-call person must be someone who can work messages in Triage")
		}
	}

	saved, err := s.settingsRepo.Save(ctx, settings)
	if err != nil {
		return nil, err
	}
	s.notify.TenantChanged(ctx, req.TenantInfo)

	return saved, nil
}

func (s *Service) OrganizationTimezone(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (string, error) {
	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}

	return org.Timezone, nil
}

func (s *Service) Assignees(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*inboundmessageports.AssigneeView, error) {
	users, err := s.directory.ListUsersWithPermission(ctx, listWorkersRequest(tenantInfo, s.now()))
	if err != nil {
		return nil, err
	}

	ids := make([]pulid.ID, 0, len(users))
	for _, user := range users {
		ids = append(ids, user.UserID)
	}
	counts, err := s.queueRepo.WaitingByOwners(ctx, inboundmessageports.WaitingByOwnersRequest{
		TenantInfo: tenantInfo,
		OwnerIDs:   ids,
	})
	if err != nil {
		return nil, err
	}

	slices.SortStableFunc(users, func(a, b permissionports.PermittedUser) int {
		return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name))
	})
	views := make([]*inboundmessageports.AssigneeView, 0, len(users))
	for _, user := range users {
		views = append(views, &inboundmessageports.AssigneeView{
			UserID:  user.UserID,
			Name:    user.Name,
			Waiting: counts[user.UserID],
		})
	}

	return views, nil
}

type HandOffRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	ToUserID   pulid.ID
}

func (s *Service) HandOff(
	ctx context.Context,
	req *HandOffRequest,
) (*inboundmessageports.HandOffResult, error) {
	to, err := s.handOffTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	affected := 0
	for affected < triage.DefaultHandOffBatchMax {
		ids, cErr := s.queueRepo.HandOffCandidates(
			ctx,
			inboundmessageports.HandOffCandidatesRequest{
				TenantInfo: req.TenantInfo,
				To:         req.ToUserID,
				Limit: min(
					inboundmessageports.HandOffBatchSize,
					triage.DefaultHandOffBatchMax-affected,
				),
			},
		)
		if cErr != nil {
			return nil, cErr
		}
		if len(ids) == 0 {
			break
		}

		result, tErr := s.runTransition(ctx, handOffTransition(req, to, ids))
		if tErr != nil {
			return nil, tErr
		}
		if len(result.changed) == 0 {
			break
		}
		affected += len(result.changed)
		if len(ids) < inboundmessageports.HandOffBatchSize {
			break
		}
	}

	return &inboundmessageports.HandOffResult{ToUserID: req.ToUserID, Affected: affected}, nil
}

func (s *Service) PlanHandOff(ctx context.Context, req *HandOffRequest) (*QueuePlan, error) {
	to, err := s.handOffTarget(ctx, req)
	if err != nil {
		return nil, err
	}

	limit := min(inboundmessageports.HandOffBatchSize, triage.DefaultHandOffBatchMax)
	ids, err := s.queueRepo.HandOffCandidates(ctx, inboundmessageports.HandOffCandidatesRequest{
		TenantInfo: req.TenantInfo,
		To:         req.ToUserID,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}

	plan, err := s.planTransition(ctx, handOffTransition(req, to, ids))
	if err != nil {
		return nil, err
	}
	plan.More = len(ids) == limit

	return plan, nil
}

func (s *Service) handOffTarget(ctx context.Context, req *HandOffRequest) (assignee, error) {
	people, err := s.assignablePeople(ctx, req.TenantInfo)
	if err != nil {
		return assignee{}, err
	}
	to, ok := people[req.ToUserID]
	if !ok {
		return assignee{}, errortypes.NewValidationError("toUserId", errortypes.ErrInvalid,
			"The queue can only be handed to someone who can work messages in Triage")
	}

	return to, nil
}

func handOffTransition(req *HandOffRequest, to assignee, ids []pulid.ID) *transition {
	target := req.ToUserID

	return &transition{
		tenant: req.TenantInfo,
		ids:    ids,
		eligible: func(state *inboundmessageports.QueueState) bool {
			return state.IsWaiting() && state.OwnerID != req.ToUserID
		},
		patch: inboundmessageports.QueuePatch{
			Owner:   &target,
			HandOff: &inboundmessageports.HandOffPatch{To: req.ToUserID},
		},
		activity: func(_, after *inboundmessageports.QueueState) *triage.Activity {
			activity := activityOf(after, triage.ActivityHandedOff).About(to.Name, "")
			activity.RecordID = req.ToUserID

			return byPerson(activity, req.ActorID)
		},
	}
}
