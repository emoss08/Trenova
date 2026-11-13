package agentreflectionservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	secondsPerDay   = int64(24 * 60 * 60)
	maxLearningDays = 365
)

type learningKey struct {
	source  agentmemory.MemorySource
	trigger agentreflection.ReflectionSignalKind
}

func (s *Service) LearningStats(
	ctx context.Context,
	req *agentreflectionports.LearningStatsRequest,
) (*agentreflectionports.LearningStats, error) {
	if req.Days < 1 || req.Days > maxLearningDays {
		return nil, errortypes.NewValidationError(
			"days",
			errortypes.ErrInvalid,
			"Choose between 1 and {0} days",
			maxLearningDays,
		)
	}
	since := s.now() - int64(req.Days)*secondsPerDay

	triggers, err := s.repo.ListLearningTriggers(
		ctx,
		&agentreflectionports.ListLearningTriggersRequest{
			TenantInfo: req.TenantInfo,
			Since:      since,
		},
	)
	if err != nil {
		return nil, err
	}
	outcomes, err := s.memories.Outcomes(ctx, req.TenantInfo, since)
	if err != nil {
		return nil, err
	}

	return &agentreflectionports.LearningStats{
		Days:  req.Days,
		Since: since,
		Rows:  LearningRows(triggers, outcomes),
	}, nil
}

// LearningRows joins what look backs cost with what became of the memories
// every source recorded. A look back's rows are one per trigger; a trigger
// that kept nothing still has a row, since what it cost is the point.
func LearningRows(
	triggers []agentreflectionports.LearningTrigger,
	outcomes []agentmemoryports.MemoryOutcome,
) []*agentreflectionports.LearningRow {
	rows := make(map[learningKey]*agentreflectionports.LearningRow, len(triggers)+len(outcomes))
	row := func(key learningKey) *agentreflectionports.LearningRow {
		if found, ok := rows[key]; ok {
			return found
		}
		created := &agentreflectionports.LearningRow{Source: key.source}
		if key.trigger != "" {
			created.Trigger = &key.trigger
		}
		rows[key] = created

		return created
	}

	for idx := range triggers {
		trigger := &triggers[idx]
		target := row(learningKey{
			source:  agentmemory.MemorySourceReflection,
			trigger: knownTrigger(trigger.Trigger),
		})
		target.LookBacks += int(trigger.LookBacks)
		target.TaughtNothing += int(trigger.TaughtNothing)
		target.ToldAgain += int(trigger.ToldAgain)
		target.InputTokens += int(trigger.InputTokens)
		target.OutputTokens += int(trigger.OutputTokens)
	}

	for idx := range outcomes {
		outcome := &outcomes[idx]
		key := learningKey{source: outcome.Source}
		if outcome.Source == agentmemory.MemorySourceReflection {
			key.trigger = knownTrigger(outcome.Trigger)
		}
		target := row(key)
		count := int(outcome.Count)
		switch outcome.Status {
		case agentmemory.MemoryStatusActive, agentmemory.MemoryStatusPaused:
			target.Kept += count
		case agentmemory.MemoryStatusSuggested:
			target.Waiting += count
		case agentmemory.MemoryStatusDismissed:
			target.Dismissed += count - int(outcome.Lapsed)
			target.Lapsed += int(outcome.Lapsed)
		case agentmemory.MemoryStatusRetired:
			target.Retired += count
		}
		target.Used += int(outcome.Used)
	}

	out := make([]*agentreflectionports.LearningRow, 0, len(rows))
	for _, found := range rows {
		out = append(out, found)
	}
	slices.SortFunc(out, func(a, b *agentreflectionports.LearningRow) int {
		if a.Source != b.Source {
			return sourceOrder(a.Source) - sourceOrder(b.Source)
		}

		return triggerOrder(triggerOf(a)) - triggerOrder(triggerOf(b))
	})

	return out
}

func triggerOf(row *agentreflectionports.LearningRow) agentreflection.ReflectionSignalKind {
	if row.Trigger == nil {
		return ""
	}

	return *row.Trigger
}

func knownTrigger(raw string) agentreflection.ReflectionSignalKind {
	kind := agentreflection.ReflectionSignalKind(raw)
	if kind.IsValid() {
		return kind
	}

	return ""
}

func sourceOrder(source agentmemory.MemorySource) int {
	order := []agentmemory.MemorySource{
		agentmemory.MemorySourceReflection,
		agentmemory.MemorySourceAgent,
		agentmemory.MemorySourceUser,
		agentmemory.MemorySourceDecision,
		agentmemory.MemorySourceFeedback,
	}
	if idx := slices.Index(order, source); idx >= 0 {
		return idx
	}

	return len(order)
}

func triggerOrder(trigger agentreflection.ReflectionSignalKind) int {
	order := []agentreflection.ReflectionSignalKind{
		agentreflection.ReflectionSignalPersonCorrected,
		agentreflection.ReflectionSignalStandingRequest,
		agentreflection.ReflectionSignalProposalModified,
		agentreflection.ReflectionSignalProposalRejected,
		agentreflection.ReflectionSignalNegativeFeedback,
		agentreflection.ReflectionSignalPersonTold,
	}
	if idx := slices.Index(order, trigger); idx >= 0 {
		return idx
	}

	return len(order)
}
