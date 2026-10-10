package aituneup

import "github.com/emoss08/trenova/shared/pulid"

type Plan struct {
	Insert []*TuneUp
	Update []*TuneUp
	Delete []pulid.ID
}

func (p *Plan) Empty() bool {
	return len(p.Insert) == 0 && len(p.Update) == 0 && len(p.Delete) == 0
}

func Reconcile(existing, candidates []*TuneUp, now int64) Plan {
	byFingerprint := make(map[string]*TuneUp, len(existing))
	for _, tuneUp := range existing {
		byFingerprint[tuneUp.Fingerprint] = tuneUp
	}

	plan := Plan{
		Insert: make([]*TuneUp, 0),
		Update: make([]*TuneUp, 0),
		Delete: make([]pulid.ID, 0),
	}
	found := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		found[candidate.Fingerprint] = struct{}{}
		stored, ok := byFingerprint[candidate.Fingerprint]
		if !ok {
			plan.Insert = append(plan.Insert, candidate)
			continue
		}
		if refreshed := refresh(stored, candidate, now); refreshed != nil {
			plan.Update = append(plan.Update, refreshed)
		}
	}

	for _, stored := range existing {
		if _, ok := found[stored.Fingerprint]; ok {
			continue
		}
		if expired(stored, now) {
			plan.Delete = append(plan.Delete, stored.ID)
		}
	}
	return plan
}

func refresh(stored, candidate *TuneUp, now int64) *TuneUp {
	next := *stored
	next.Evidence = candidate.Evidence
	next.AgentDefinitionID = candidate.AgentDefinitionID
	next.ProviderID = candidate.ProviderID
	next.OtherProviderID = candidate.OtherProviderID
	next.ToolName = candidate.ToolName
	next.Task = candidate.Task
	next.ComputedAt = candidate.ComputedAt

	switch stored.Status {
	case StatusApplied:
		if stored.DecidedAt != nil && now-*stored.DecidedAt < AppliedRetentionDays*secondsPerDay {
			return nil
		}
		reopen(&next)
	case StatusDismissed:
		if stored.DismissedUntil == nil || *stored.DismissedUntil <= now {
			reopen(&next)
		}
	case StatusOpen:
	}
	return &next
}

func reopen(tuneUp *TuneUp) {
	tuneUp.Status = StatusOpen
	tuneUp.DismissedUntil = nil
	tuneUp.DecidedAt = nil
	tuneUp.DecidedByID = pulid.Nil
}

func expired(stored *TuneUp, now int64) bool {
	switch stored.Status {
	case StatusDismissed:
		return stored.DismissedUntil == nil || *stored.DismissedUntil <= now
	case StatusApplied:
		return stored.DecidedAt == nil || now-*stored.DecidedAt >= AppliedRetentionDays*secondsPerDay
	default:
		return true
	}
}
