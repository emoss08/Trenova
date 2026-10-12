package inboundmessageservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type BulkReviewRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	Selection  Selection
	Status     inboundmessage.Status
	Note       string
	Teach      inboundmessage.NovaAction
}

type ReopenRequest struct {
	TenantInfo pagination.TenantInfo
	ActorID    pulid.ID
	Selection  Selection
}

type AssignRequest struct {
	TenantInfo      pagination.TenantInfo
	ActorID         pulid.ID
	Selection       Selection
	OwnerID         pulid.ID
	CanAssignOthers bool
}

func (s *Service) ReviewMany(
	ctx context.Context,
	req *BulkReviewRequest,
) (*inboundmessageports.BulkResult, error) {
	note, err := reviewNote(&ReviewRequest{Status: req.Status, Note: req.Note})
	if err != nil {
		return nil, err
	}
	if req.Teach != "" {
		if !req.Teach.IsValid() {
			return nil, errortypes.NewValidationError("teach", errortypes.ErrInvalid,
				"Nova cannot learn that action")
		}
		if s.ruleTeacher == nil {
			return nil, errortypes.NewBusinessError("Rules are not available on this installation")
		}
	}

	ids, err := s.resolveSelection(ctx, req.TenantInfo, req.Selection)
	if err != nil {
		return nil, err
	}

	result, err := s.runTransition(ctx, settleTransition(settleParams{
		tenant:   req.TenantInfo,
		ids:      ids,
		status:   req.Status,
		note:     note,
		actorID:  req.ActorID,
		activity: settleActivityKind(req.Status),
	}))
	if err != nil {
		return nil, err
	}

	ruleID, err := s.teach(ctx, req, result)
	if err != nil {
		return nil, err
	}

	return s.bulkResult(ctx, req.TenantInfo, result, ruleID)
}

type settleParams struct {
	tenant   pagination.TenantInfo
	ids      []pulid.ID
	status   inboundmessage.Status
	note     string
	actorID  pulid.ID
	activity triage.ActivityKind
	subject  string
	detail   string
	recordID pulid.ID
	eligible func(state *inboundmessageports.QueueState) bool
}

func settleTransition(p settleParams) *transition {
	eligible := p.eligible
	if eligible == nil {
		eligible = func(state *inboundmessageports.QueueState) bool {
			return state.Status != p.status
		}
	}

	return &transition{
		tenant:   p.tenant,
		ids:      p.ids,
		eligible: eligible,
		patch: inboundmessageports.QueuePatch{
			Status: p.status,
			Review: &inboundmessageports.ReviewPatch{ReviewerID: p.actorID, Note: p.note},
			Unpark: true,
			Due:    inboundmessageports.DueClear,
		},
		activity: func(_, after *inboundmessageports.QueueState) *triage.Activity {
			activity := activityOf(after, p.activity).About(p.subject, p.detail)
			activity.RecordID = p.recordID

			return byPerson(activity, p.actorID)
		},
	}
}

func settleActivityKind(status inboundmessage.Status) triage.ActivityKind {
	if status == inboundmessage.StatusIgnored {
		return triage.ActivityIgnored
	}

	return triage.ActivityHandled
}

func (s *Service) teach(
	ctx context.Context,
	req *BulkReviewRequest,
	result *transitionResult,
) (*pulid.ID, error) {
	if req.Teach == "" {
		return nil, nil
	}

	var source *inboundmessageports.QueueState
	for _, state := range result.changed {
		if state.Classification != "" {
			source = state
			break
		}
	}
	if source == nil {
		return nil, errortypes.NewValidationError("teach", errortypes.ErrInvalid,
			"Nova can only learn from a message it has read")
	}

	rule, err := s.ruleTeacher.Teach(ctx, inboundmessageports.TeachRequest{
		TenantInfo:     req.TenantInfo,
		ActorID:        req.ActorID,
		MessageID:      source.ID,
		Classification: source.Classification,
		SenderDomain:   source.SenderDomain,
		MailboxID:      source.MailboxID,
		Action:         req.Teach,
	})
	if err != nil {
		return nil, err
	}

	if err = s.activityRepo.Insert(ctx, []*triage.Activity{
		byPerson(
			activityOf(
				source,
				triage.ActivityRuleCreated,
			).About(string(req.Teach), source.SenderDomain),
			req.ActorID,
		),
	}); err != nil {
		s.l.Warn("could not record the rule a review turned on",
			zap.String("messageId", source.ID.String()), zap.Error(err))
	}

	return &rule.ID, nil
}

func (s *Service) Reopen(
	ctx context.Context,
	req *ReopenRequest,
) (*inboundmessageports.BulkResult, error) {
	ids, err := s.resolveSelection(ctx, req.TenantInfo, req.Selection)
	if err != nil {
		return nil, err
	}

	targets, err := s.Targets(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	result, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    ids,
		eligible: func(state *inboundmessageports.QueueState) bool {
			return state.Settled()
		},
		patch: inboundmessageports.QueuePatch{
			Status:       inboundmessage.StatusInReview,
			Review:       &inboundmessageports.ReviewPatch{Clear: true},
			Unpark:       true,
			Due:          inboundmessageports.DueFromTargets,
			Targets:      targets,
			ClearHandled: true,
		},
		activity: func(before, after *inboundmessageports.QueueState) *triage.Activity {
			return byPerson(
				activityOf(after, triage.ActivityReopened).About(string(before.Status), ""),
				req.ActorID,
			)
		},
	})
	if err != nil {
		return nil, err
	}

	return s.bulkResult(ctx, req.TenantInfo, result, nil)
}

func (s *Service) Assign(
	ctx context.Context,
	req *AssignRequest,
) (*inboundmessageports.BulkResult, error) {
	if req.OwnerID.IsNotNil() && req.OwnerID != req.ActorID && !req.CanAssignOthers {
		return nil, errortypes.NewAuthorizationError(
			"You can take messages yourself, but assigning them to someone else needs the assign permission",
		)
	}

	people, err := s.assignablePeople(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	ownerName := ""
	if req.OwnerID.IsNotNil() {
		owner, ok := people[req.OwnerID]
		if !ok {
			return nil, errortypes.NewValidationError("ownerId", errortypes.ErrInvalid,
				"That person cannot work messages in Triage")
		}
		ownerName = owner.Name
	}

	ids, err := s.resolveSelection(ctx, req.TenantInfo, req.Selection)
	if err != nil {
		return nil, err
	}

	owner := req.OwnerID
	result, err := s.runTransition(ctx, &transition{
		tenant: req.TenantInfo,
		ids:    ids,
		check: func(states []*inboundmessageports.QueueState) error {
			if req.OwnerID.IsNotNil() || req.CanAssignOthers {
				return nil
			}
			for _, state := range states {
				if state.OwnerID.IsNotNil() && state.OwnerID != req.ActorID {
					return errortypes.NewAuthorizationError(
						"Unassigning someone else's message needs the assign permission",
					)
				}
			}

			return nil
		},
		eligible: func(state *inboundmessageports.QueueState) bool {
			return !state.Settled() && state.OwnerID != req.OwnerID
		},
		patch: inboundmessageports.QueuePatch{Owner: &owner},
		activity: func(before, after *inboundmessageports.QueueState) *triage.Activity {
			if req.OwnerID.IsNil() {
				activity := activityOf(after, triage.ActivityUnassigned).
					About(people[before.OwnerID].Name, "")
				activity.RecordID = before.OwnerID

				return byPerson(activity, req.ActorID)
			}
			activity := activityOf(after, triage.ActivityAssigned).About(ownerName, "")
			activity.RecordID = req.OwnerID

			return byPerson(activity, req.ActorID)
		},
	})
	if err != nil {
		return nil, err
	}

	return s.bulkResult(ctx, req.TenantInfo, result, nil)
}

type assignee struct {
	ID   pulid.ID
	Name string
}

func (s *Service) assignablePeople(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]assignee, error) {
	users, err := s.directory.ListUsersWithPermission(ctx, listWorkersRequest(tenantInfo, s.now()))
	if err != nil {
		return nil, err
	}

	people := make(map[pulid.ID]assignee, len(users))
	for _, user := range users {
		people[user.UserID] = assignee{ID: user.UserID, Name: user.Name}
	}

	return people, nil
}

var workerPermission = struct {
	resource  permission.Resource
	operation permission.Operation
}{
	resource:  permission.ResourceInboundMessage,
	operation: permission.OpUpdate,
}
