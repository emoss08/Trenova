package caseownerservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/recordcaseservice/casereading"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowerservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/caseowner"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	EventCaseAssigned  = "desk.case.assigned"
	notificationSource = "desk"
	casePathPrefix     = "/desk/r/"
)

var errTakenMeanwhile = errortypes.NewConflictError(
	"Someone else just changed who owns this case. Reload to see who has it.",
)

/*
Claim makes the actor the case's owner. An unowned case is anyone's to claim
who may work cases; one that is someone else's is taken over only by a person
who may assign cases.
*/
func (s *Service) Claim(
	ctx context.Context,
	req *caseownerports.AssignRequest,
	actor *permissionports.RequestActor,
) (*caseowner.Owner, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewAuthorizationError("Only a person can own a case")
	}

	standing, err := s.Standing(ctx, caseownerports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.RecordID,
	}, actor)
	if err != nil {
		return nil, err
	}
	switch {
	case standing.Owner != nil && standing.Owner.OwnerID == actor.UserID:
		return standing.Owner, nil
	case standing.Owner == nil && !standing.CanClaim:
		return nil, errortypes.NewAuthorizationError("You may not work cases")
	case standing.Owner != nil && !standing.CanAssign:
		return nil, errortypes.NewAuthorizationError(
			"This case already has an owner. Someone who assigns cases can hand it to you.",
		)
	}

	return s.set(ctx, &caseownerports.SetOwnerRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		RecordID:   req.RecordID,
		OwnerID:    actor.UserID,
		Source:     caseowner.SourceClaimed,
		ByID:       actor.UserID,
		Label:      req.Label,
		Expected:   standing.Owner,
	}, actor)
}

func (s *Service) Assign(
	ctx context.Context,
	req *caseownerports.AssignRequest,
	actor *permissionports.RequestActor,
) (*caseowner.Owner, error) {
	if req.OwnerID.IsNil() {
		return nil, errortypes.NewValidationError(
			"ownerId", errortypes.ErrRequired, "Choose who the case goes to",
		)
	}
	if req.OwnerID == actor.UserID {
		return s.Claim(ctx, req, actor)
	}

	standing, err := s.Standing(ctx, caseownerports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.RecordID,
	}, actor)
	if err != nil {
		return nil, err
	}
	if !standing.CanAssign {
		return nil, errortypes.NewAuthorizationError("You may not assign cases")
	}
	if standing.Owner != nil && standing.Owner.OwnerID == req.OwnerID {
		return standing.Owner, nil
	}

	able, err := s.allowed(
		ctx,
		permissionports.UserActor(tenantOf(actor, req.OwnerID)),
		permission.OpUpdate,
	)
	if err != nil {
		return nil, err
	}
	if !able {
		return nil, errortypes.NewValidationError(
			"ownerId", errortypes.ErrInvalid, "That person may not work cases",
		)
	}

	return s.set(ctx, &caseownerports.SetOwnerRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		RecordID:   req.RecordID,
		OwnerID:    req.OwnerID,
		Source:     caseowner.SourceAssigned,
		ByID:       actor.UserID,
		Label:      req.Label,
		Expected:   standing.Owner,
	}, actor)
}

func (s *Service) set(
	ctx context.Context,
	req *caseownerports.SetOwnerRequest,
	actor *permissionports.RequestActor,
) (*caseowner.Owner, error) {
	req.Now = s.now()
	owner, ok, err := s.repo.Set(ctx, req)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errTakenMeanwhile
	}

	s.recordChange(ctx, actor, req.Expected, owner)
	s.announce(ctx, owner.OrganizationID, owner.BusinessUnitID, owner.RecordID)
	if owner.OwnerID != actor.UserID {
		s.notify(ctx, owner, req.Label)
	}

	return owner, nil
}

/*
Release leaves the case unowned, for the next person to claim. Its owner may
let it go, and so may someone who assigns cases.
*/
func (s *Service) Release(
	ctx context.Context,
	req *caseownerports.ReleaseRequest,
	actor *permissionports.RequestActor,
) error {
	record := caseownerports.RecordRequest{TenantInfo: req.TenantInfo, RecordID: req.RecordID}
	standing, err := s.Standing(ctx, record, actor)
	if err != nil {
		return err
	}
	if standing.Owner == nil {
		return nil
	}
	if standing.Owner.OwnerID != actor.UserID && !standing.CanAssign {
		return errortypes.NewAuthorizationError("Only its owner can let this case go")
	}

	released, err := s.repo.Release(ctx, record, standing.Owner)
	if err != nil {
		return err
	}
	if !released {
		return errTakenMeanwhile
	}

	s.recordChange(ctx, actor, standing.Owner, nil)
	s.announce(ctx, standing.Owner.OrganizationID, standing.Owner.BusinessUnitID, req.RecordID)

	return nil
}

/*
Route gives an unowned case to the first person its record names who may work
cases and is there to work it: someone away with nobody covering is passed
over, so the case does not wait unseen until they are back. It is called as a record first gets something open about it, by a job
no person is waiting on, so a failure is logged and the case stays unowned
for someone to claim.
*/
func (s *Service) Route(
	ctx context.Context,
	req caseownerports.CandidatesRequest,
	first ...pulid.ID,
) {
	if req.RecordID.IsNil() {
		return
	}
	record := caseownerports.RecordRequest{TenantInfo: req.TenantInfo, RecordID: req.RecordID}
	owner, err := s.repo.Get(ctx, record)
	if err != nil || owner != nil {
		return
	}

	named, err := s.repo.Candidates(ctx, req)
	if err != nil {
		s.l.Warn("could not read who a record names for its case",
			zap.String("recordId", req.RecordID.String()), zap.Error(err))

		return
	}

	people := append(first, named...)
	uncovered, err := s.repo.Uncovered(ctx, caseownerports.AwayRequest{
		TenantInfo: req.TenantInfo,
		Now:        s.now(),
		UserIDs:    people,
	})
	if err != nil {
		s.l.Warn("could not read who is away",
			zap.String("recordId", req.RecordID.String()), zap.Error(err))

		return
	}

	tried := make(map[pulid.ID]struct{}, len(people))
	for _, candidate := range people {
		if candidate.IsNil() || slices.Contains(uncovered, candidate) {
			continue
		}
		if _, seen := tried[candidate]; seen {
			continue
		}
		tried[candidate] = struct{}{}

		tenant := req.TenantInfo
		tenant.UserID = candidate
		able, checkErr := s.allowed(ctx, permissionports.UserActor(tenant), permission.OpUpdate)
		if checkErr != nil || !able {
			continue
		}
		if err = s.repo.SetIfUnowned(ctx, &caseownerports.SetOwnerRequest{
			TenantInfo: req.TenantInfo,
			Kind:       req.Kind,
			RecordID:   req.RecordID,
			OwnerID:    candidate,
			Source:     caseowner.SourceAuto,
			Now:        s.now(),
		}); err != nil {
			s.l.Warn("could not route a case to its owner",
				zap.String("recordId", req.RecordID.String()), zap.Error(err))
		}

		return
	}
}

func ownerState(owner *caseowner.Owner) map[string]any {
	if owner == nil {
		return map[string]any{"ownerId": ""}
	}

	return map[string]any{
		"ownerId":    owner.OwnerID.String(),
		"source":     string(owner.Source),
		"recordKind": string(owner.RecordKind),
	}
}

func (s *Service) recordChange(
	ctx context.Context,
	actor *permissionports.RequestActor,
	before, after *caseowner.Owner,
) {
	subject := after
	if subject == nil {
		subject = before
	}
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceWatchtower,
		ResourceID:     subject.RecordID.String(),
		Operation:      permission.OpAssign,
		Action:         "case owner changed",
		Actor:          actor.AuditActor(),
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		Before:         ownerState(before),
		After:          ownerState(after),
	})
}

func (s *Service) announce(ctx context.Context, orgID, buID, recordID pulid.ID) {
	for _, resource := range []string{
		casereading.RealtimeResource,
		watchtowerservice.RealtimeResource,
	} {
		if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
			OrganizationID: orgID,
			BusinessUnitID: buID,
			Resource:       resource,
			Action:         "updated",
			RecordID:       recordID,
		}); err != nil {
			s.l.Warn("a change of case owner could not be announced",
				zap.String("recordId", recordID.String()), zap.Error(err))
		}
	}
}

func (s *Service) notify(ctx context.Context, owner *caseowner.Owner, label string) {
	buID, target := owner.BusinessUnitID, owner.OwnerID
	correlation := owner.RecordID.String()
	link := casePathPrefix + string(owner.RecordKind) + "/" + owner.RecordID.String()
	if _, err := s.notifications.Create(ctx, &notification.Notification{
		OrganizationID:  owner.OrganizationID,
		BusinessUnitID:  &buID,
		TargetUserID:    &target,
		EventType:       EventCaseAssigned,
		Priority:        notification.PriorityMedium,
		Channel:         notification.ChannelUser,
		Title:           caseTitle(label),
		Message:         "Open it on the Desk to see what it needs.",
		Source:          notificationSource,
		CorrelationID:   &correlation,
		Data:            map[string]any{"link": link},
		RelatedEntities: map[string]any{"recordId": owner.RecordID.String()},
	}); err != nil {
		s.l.Warn("could not tell a case's new owner",
			zap.String("recordId", owner.RecordID.String()), zap.Error(err))
	}
}

func caseTitle(label string) string {
	if label == "" {
		return "A case was assigned to you"
	}

	return "The case on " + label + " was assigned to you"
}
