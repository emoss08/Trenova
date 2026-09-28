package extractionrolloutservice

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
)

const (
	haltAttempts      = 3
	haltNoticeLimit   = 25
	haltNoticeSource  = "extraction-rollout"
	haltNoticeLinkURL = "/admin/agent-control?tab=quality&quality=extraction&extraction=rollout"
)

func (s *Service) ObserveCorrection(
	ctx context.Context,
	correction *aicorrection.Correction,
) error {
	if correction == nil ||
		correction.Task != aicorrection.TaskShipmentDraftExtraction ||
		correction.ExtractionProviderID == nil {
		return nil
	}

	return s.enforceGuards(ctx, pagination.TenantInfo{
		OrgID: correction.OrganizationID,
		BuID:  correction.BusinessUnitID,
	})
}

func (s *Service) enforceGuards(ctx context.Context, tenant pagination.TenantInfo) error {
	rollout, err := s.Get(ctx, tenant)
	if err != nil {
		return err
	}
	if rollout.CandidateID().IsNil() || rollout.StartedAt == nil {
		return nil
	}

	input, err := s.guardInput(ctx, tenant, rollout)
	if err != nil {
		return err
	}
	breach, breached := rollout.Breach(input)
	if !breached {
		return nil
	}

	return s.halt(ctx, tenant, rollout, breach)
}

func (s *Service) guardInput(
	ctx context.Context,
	tenant pagination.TenantInfo,
	rollout *extractionrollout.ExtractionRollout,
) (*extractionrollout.GuardInput, error) {
	candidate := rollout.CandidateID()
	accuracy, err := s.accuracy(ctx, tenant, rollout)
	if err != nil {
		return nil, err
	}
	totals, err := s.assignments.Totals(ctx, repositories.TotalRolloutAssignmentsRequest{
		TenantInfo:          tenant,
		CandidateProviderID: candidate,
		Since:               *rollout.StartedAt,
	})
	if err != nil {
		return nil, err
	}
	t := tallyAssignments(totals)

	return &extractionrollout.GuardInput{
		CandidateAccuracy:  accuracy.candidate,
		ProductionAccuracy: accuracy.production,
		CandidateOutcomes:  t.candidateOutcomes,
		ControlOutcomes:    t.controlOutcomes,
	}, nil
}

type accuracySplit struct {
	candidate  extractionrollout.ArmAccuracy
	production extractionrollout.ArmAccuracy
}

func (s *Service) accuracy(
	ctx context.Context,
	tenant pagination.TenantInfo,
	rollout *extractionrollout.ExtractionRollout,
) (accuracySplit, error) {
	totals, err := s.corrections.TotalsByProvider(
		ctx,
		&repositories.TotalAICorrectionsByProviderRequest{
			TenantInfo: tenant,
			Task:       aicorrection.TaskShipmentDraftExtraction,
			ProviderID: *rollout.ProviderID,
			Since:      *rollout.StartedAt,
		},
	)
	if err != nil {
		return accuracySplit{}, err
	}

	var split accuracySplit
	for _, total := range totals {
		side := &split.production
		if total.Candidate {
			side = &split.candidate
		}
		side.Scored += total.Scored
		side.Correct += total.Correct
	}

	return split, nil
}

func (s *Service) halt(
	ctx context.Context,
	tenant pagination.TenantInfo,
	rollout *extractionrollout.ExtractionRollout,
	breach extractionrollout.Breach,
) error {
	startedAt := *rollout.StartedAt
	for range haltAttempts {
		previous := auditable(rollout)
		rollout.Halt(breach, s.now())
		saved, err := s.rollouts.Save(ctx, rollout)
		if err == nil {
			s.reportHalt(ctx, saved, previous)
			return nil
		}
		if !errortypes.IsVersionMismatchError(err) {
			return err
		}

		rollout, err = s.Get(ctx, tenant)
		if err != nil {
			return err
		}
		if rollout.CandidateID().IsNil() ||
			rollout.StartedAt == nil ||
			*rollout.StartedAt != startedAt {
			return nil
		}
	}

	return fmt.Errorf("halt extraction rollout %s: changed on every attempt", rollout.ID)
}

func (s *Service) reportHalt(
	ctx context.Context,
	rollout *extractionrollout.ExtractionRollout,
	previous map[string]any,
) {
	s.l.Warn("extraction rollout stopped by a guard",
		zap.String("rolloutId", rollout.ID.String()),
		zap.String("reason", rollout.HaltReason.String()),
		zap.Float64("candidateRate", rollout.HaltCandidateRate),
		zap.Float64("baselineRate", rollout.HaltBaselineRate),
	)

	s.logAction(nil, &services.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     rollout.ID.String(),
		Operation:      permission.OpUpdate,
		CurrentState:   jsonutils.MustToJSON(auditable(rollout)),
		PreviousState:  jsonutils.MustToJSON(previous),
		OrganizationID: rollout.OrganizationID,
		BusinessUnitID: rollout.BusinessUnitID,
		Critical:       true,
	}, "Extraction rollout stopped by a guard")

	if s.notifier == nil {
		return
	}

	title, message := haltNotice(rollout)
	correlation := rollout.ID.String() + ":" + strconv.FormatInt(*rollout.HaltedAt, 10)
	if _, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant: pagination.TenantInfo{
			OrgID: rollout.OrganizationID,
			BuID:  rollout.BusinessUnitID,
		},
		Resource:    permission.ResourceAgentEvalSuite,
		Operation:   permission.OpUpdate,
		Limit:       haltNoticeLimit,
		Now:         s.now(),
		DedupeSince: *rollout.StartedAt,
		Notification: notification.Notification{
			EventType:     services.ExtractionRolloutHaltedEvent,
			Priority:      notification.PriorityHigh,
			Title:         title,
			Message:       message,
			Source:        haltNoticeSource,
			CorrelationID: &correlation,
			Data: map[string]any{
				"link":          haltNoticeLinkURL,
				"rolloutId":     rollout.ID.String(),
				"reason":        rollout.HaltReason.String(),
				"candidateRate": rollout.HaltCandidateRate,
				"baselineRate":  rollout.HaltBaselineRate,
			},
		},
	}); err != nil {
		s.l.Warn("could not tell anyone the extraction rollout stopped", zap.Error(err))
	}
}

func haltNotice(rollout *extractionrollout.ExtractionRollout) (title, message string) {
	candidate := percent(rollout.HaltCandidateRate)
	baseline := percent(rollout.HaltBaselineRate)
	if rollout.HaltReason == extractionrollout.HaltReasonRejections {
		return "Document extraction rollout stopped: too many unusable answers",
			fmt.Sprintf(
				"The candidate's answers were rejected on %s of its documents against %s for production, "+
					"so every document is back on production. Review the rollout in AI Control.",
				candidate,
				baseline,
			)
	}

	return "Document extraction rollout stopped: accuracy dropped",
		fmt.Sprintf(
			"The candidate read %s of confirmed fields correctly against %s for production, "+
				"so every document is back on production. Review the rollout in AI Control.",
			candidate, baseline,
		)
}

func percent(rate float64) string {
	return fmt.Sprintf("%.1f%%", rate*100)
}
