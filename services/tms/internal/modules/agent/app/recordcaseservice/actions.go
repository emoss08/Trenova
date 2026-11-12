package recordcaseservice

import (
	"context"
	"errors"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordcaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	maxAcceptedItems = 25
	minReasonLength  = 3
	maxReasonLength  = 500
)

func (s *Service) ChooseOption(
	ctx context.Context,
	req *recordcaseports.ChooseOptionRequest,
	actor *permissionports.RequestActor,
) error {
	built, err := s.compose(ctx, &recordcaseports.CaseRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		ID:         req.ID,
	}, actor)
	if err != nil {
		return err
	}
	if !built.standing.CanWork {
		return errNotTheOwner
	}
	if built.decision != nil && built.decision.Phase != recordcase.PhaseAwaiting &&
		built.decision.Phase != recordcase.PhaseDeclined {
		return errortypes.NewBusinessError(
			"The plan has already been decided, so another option can no longer be chosen",
		)
	}

	chosen, err := s.repo.Choose(ctx, recordcaseports.ChooseRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
		OptionID:   req.OptionID,
	})
	if err != nil {
		return err
	}
	if chosen.Kind == recordcase.OptionKindLeave || chosen.AgentDefinitionID.IsNil() {
		return nil
	}
	if chosen.DraftRunID.IsNotNil() {
		drafted, dErr := s.drafted(ctx, req, chosen)
		if dErr != nil || drafted {
			return dErr
		}
	}

	return s.draft(ctx, req, chosen, actor)
}

func (s *Service) drafted(
	ctx context.Context,
	req *recordcaseports.ChooseOptionRequest,
	option *recordcase.Option,
) (bool, error) {
	now := s.now()
	found, err := s.repo.Decisions(ctx, &recordcaseports.DecisionsRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
		RunID:      option.DraftRunID,
		Since:      now - decidedLookback,
	})
	if err != nil {
		return false, err
	}
	for _, plan := range found.Plans {
		if phase, ok := planPhase(plan.Status); ok && phase != recordcase.PhaseDeclined {
			return true, nil
		}
	}
	for _, proposal := range found.Proposals {
		if phase, ok := proposalPhase(proposal.Status); ok && phase != recordcase.PhaseDeclined {
			return true, nil
		}
	}
	if option.DraftRequestedAt == nil {
		return false, nil
	}

	drafting, _ := s.draftState(
		ctx,
		&recordcaseports.CaseRequest{TenantInfo: req.TenantInfo},
		&composed{
			chosen: option,
			now:    now,
		},
	)

	return drafting, nil
}

func (s *Service) draft(
	ctx context.Context,
	req *recordcaseports.ChooseOptionRequest,
	option *recordcase.Option,
	actor *permissionports.RequestActor,
) error {
	subjectType, ok := agent.SubjectTypeOfID(req.ID)
	if !ok {
		return nil
	}
	run, err := s.runs.StartForDefinition(ctx, &agentruntimeports.StartAgentRunForDefinitionRequest{
		DefinitionID: option.AgentDefinitionID,
		SubjectType:  subjectType,
		SubjectID:    req.ID,
		Trigger:      agent.RunTriggerManual,
		CaseOptionID: option.ID,
		Watched:      true,
		TenantInfo:   req.TenantInfo,
	}, actor)
	if err != nil {
		if errors.Is(err, agentruntimeports.ErrAgentRunAlreadyOpen) {
			return errortypes.NewConflictError(
				"The agent is already working on this record. Try again when it finishes",
			)
		}

		return err
	}
	now := s.now()

	return s.repo.SetDraft(ctx, recordcaseports.SetDraftRequest{
		TenantInfo:  req.TenantInfo,
		OptionID:    option.ID,
		RunID:       run.ID,
		RequestedAt: now,
	})
}

func (s *Service) AcceptRisk(
	ctx context.Context,
	req *recordcaseports.AcceptRiskRequest,
	actor *permissionports.RequestActor,
) (int, error) {
	if len(req.Reason) < minReasonLength || len(req.Reason) > maxReasonLength {
		return 0, errortypes.NewValidationError(
			"reason",
			errortypes.ErrInvalid,
			"Say why the risk is accepted, in a sentence the team can read",
		)
	}
	if err := s.owners.AssertMayWork(ctx, caseownerports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
	}, actor); err != nil {
		return 0, err
	}
	page, err := s.watchtower.List(ctx, watchtowerports.ListWatchtowerItemsRequest{
		TenantInfo:     req.TenantInfo,
		UnresolvedOnly: true,
		Statuses:       openStatuses,
		AboutIDs:       []pulid.ID{req.ID},
		OrderByDue:     true,
		First:          maxSignals,
	}, actor)
	if err != nil {
		return 0, err
	}

	accepted := 0
	for _, item := range page.Items {
		if item.SourceKind.DeskLens() != watchtower.DeskLensWatchtower {
			continue
		}
		if accepted == maxAcceptedItems {
			break
		}
		if _, err = s.exceptions.ResolveException(ctx, &watchtowerports.ResolveExceptionRequest{
			ExceptionTarget: watchtowerports.ExceptionTarget{
				TenantInfo:      req.TenantInfo,
				ItemID:          item.ID,
				ExpectedVersion: item.Version,
			},
			Code: watchtower.ResolutionNoActionNeeded,
			Note: req.Reason,
		}, actor); err != nil {
			if accepted == 0 {
				return 0, err
			}
			s.l.Warn("an accepted risk left a condition open",
				zap.String("item", item.ID.String()), zap.Error(err))

			continue
		}
		accepted++
	}

	return accepted, nil
}

// awayOf says a case is put away when nothing about the record is in front
// of anyone and something is snoozed: every condition was put away. One open
// item, or a change waiting to be decided, and the case is in front again.
func awayOf(items []*watchtower.Item) *recordcase.Away {
	var until int64
	for _, item := range items {
		if item.Status != watchtower.StatusSnoozed {
			return nil
		}
		if item.SnoozedUntil != nil && (until == 0 || *item.SnoozedUntil < until) {
			until = *item.SnoozedUntil
		}
	}
	if until == 0 {
		return nil
	}

	return &recordcase.Away{Until: until}
}

/*
PutAway takes a case out of Needs you once the person has done what they can:
it snoozes every condition still open about the record until a time. Nothing
is closed. The source still clears an item when the problem goes away, a new
signal about the record is a new item and shows at once, and at the time the
snoozed ones wake by themselves. A change waiting to be approved is never put
away: it is decided.
*/
func (s *Service) PutAway(
	ctx context.Context,
	req *recordcaseports.PutAwayRequest,
	actor *permissionports.RequestActor,
) (int, error) {
	if err := s.owners.AssertMayWork(ctx, caseownerports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
	}, actor); err != nil {
		return 0, err
	}
	page, err := s.watchtower.List(ctx, watchtowerports.ListWatchtowerItemsRequest{
		TenantInfo:     req.TenantInfo,
		UnresolvedOnly: true,
		Statuses:       openStatuses,
		AboutIDs:       []pulid.ID{req.ID},
		OrderByDue:     true,
		First:          maxSignals,
	}, actor)
	if err != nil {
		return 0, err
	}
	for _, item := range page.Items {
		if item.SourceKind.DeskLens() == watchtower.DeskLensDecisions {
			return 0, errortypes.NewBusinessError(
				"A change is waiting to be approved or declined. Decide it before putting " +
					"the case away.",
			)
		}
	}
	if len(page.Items) == 0 {
		return 0, errortypes.NewBusinessError("Nothing is open about this record to put away")
	}

	put := 0
	for _, item := range page.Items {
		if put == maxAcceptedItems {
			break
		}
		if _, err = s.exceptions.SnoozeException(ctx, &watchtowerports.SnoozeExceptionRequest{
			ExceptionTarget: watchtowerports.ExceptionTarget{
				TenantInfo:      req.TenantInfo,
				ItemID:          item.ID,
				ExpectedVersion: item.Version,
			},
			Until: req.Until,
			Note:  "Put away from its case",
		}, actor); err != nil {
			if put == 0 {
				return 0, err
			}
			s.l.Warn("a case put away left a condition in front",
				zap.String("item", item.ID.String()), zap.Error(err))

			continue
		}
		put++
	}

	return put, nil
}

// BringBack wakes everything about the record that was put away, so the case
// is in front again before its time.
func (s *Service) BringBack(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
) (int, error) {
	if err := s.owners.AssertMayWork(ctx, caseownerports.RecordRequest{
		TenantInfo: req.TenantInfo,
		RecordID:   req.ID,
	}, actor); err != nil {
		return 0, err
	}
	page, err := s.watchtower.List(ctx, watchtowerports.ListWatchtowerItemsRequest{
		TenantInfo:     req.TenantInfo,
		UnresolvedOnly: true,
		Statuses:       []watchtower.Status{watchtower.StatusSnoozed},
		AboutIDs:       []pulid.ID{req.ID},
		OrderByDue:     true,
		First:          maxSignals,
	}, actor)
	if err != nil {
		return 0, err
	}

	woke := 0
	for _, item := range page.Items {
		if _, err = s.exceptions.UnsnoozeException(ctx, &watchtowerports.UnsnoozeExceptionRequest{
			ExceptionTarget: watchtowerports.ExceptionTarget{
				TenantInfo:      req.TenantInfo,
				ItemID:          item.ID,
				ExpectedVersion: item.Version,
			},
		}, actor); err != nil {
			if woke == 0 {
				return 0, err
			}
			s.l.Warn("a case brought back left a condition put away",
				zap.String("item", item.ID.String()), zap.Error(err))

			continue
		}
		woke++
	}

	return woke, nil
}

type Composer struct {
	service *Service
}

type Reading struct {
	Fingerprint string
	Label       string
	Noun        string
	Status      string
	Title       string
	Proposal    string
	Exhibits    []recordcase.Exhibit
	Signals     []string
}

func (c *Composer) Compose(
	ctx context.Context,
	req *recordcaseports.CaseRequest,
	actor *permissionports.RequestActor,
) (*Reading, error) {
	built, err := c.service.compose(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	head := headOf(built)
	reading := &Reading{
		Fingerprint: fingerprintOf(built),
		Label:       built.record.Label,
		Noun:        built.record.Kind.Noun(),
		Status:      head.Status,
		Title:       head.Title,
		Exhibits:    built.exhibits,
		Signals:     make([]string, 0, len(built.items)),
	}
	if built.decision != nil {
		reading.Proposal = built.decision.Title
	}
	for _, item := range conditions(built) {
		reading.Signals = append(reading.Signals, joined(item.Title, item.Summary))
	}

	return reading, nil
}
