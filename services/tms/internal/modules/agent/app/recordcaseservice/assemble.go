package recordcaseservice

import (
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordcase"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/shared/stringutils"
)

const approvalReason = "AI control lets this agent only propose this change; a person approves it."

func assemble(
	built *composed,
	reading *recordcase.Reading,
	status recordcase.ReadingStatus,
) *recordcase.Case {
	out := &recordcase.Case{
		Record: recordcase.Record{
			Kind:     built.record.Kind,
			ID:       built.record.ID,
			Label:    built.record.Label,
			Readable: built.record.Readable,
		},
		Ownership: built.standing.Ownership(),
		Away:      built.away,
		HandOff:   built.handOff,
		Head:      headOf(built),
		Options:   built.options,
		Chosen:    built.chosen,
		Drafting:  built.drafting,
		DraftNote: built.draftNote,
		Decision:  built.decision,
		Exhibits:  built.exhibits,
		Starters:  built.subject.starters,
		Open:      len(built.items),
		Severity:  severityOf(built.items),
		UpdatedAt: built.now,
	}
	if len(out.Starters) == 0 {
		out.Starters = []string{
			"What happens if nothing changes?",
			"What are my options here?",
			"Who else should know about this?",
		}
	}
	out.Finding = findingOf(built, reading, status)
	out.Note = noteOf(built)

	return out
}

func headOf(built *composed) recordcase.Head {
	head := built.subject.head
	if head.Title == "" {
		head.Title = stringutils.FirstNonEmptyTrimmed(built.record.Subtitle, built.record.Label)
	}
	if head.Status == "" {
		head.Status = built.record.Status
		head.StatusTone = severityTone(severityOf(built.items))
	}
	if head.StatusTone == "" {
		head.StatusTone = recordcase.ToneNeutral
	}
	for _, item := range built.items {
		if item.DueAt == nil {
			continue
		}
		if head.Deadline == nil || *item.DueAt < head.Deadline.At {
			head.Deadline = &recordcase.Deadline{At: *item.DueAt, Label: item.DueLabel}
		}
	}

	return head
}

func severityOf(items []*watchtower.Item) watchtower.Severity {
	severity := watchtower.SeverityInfo
	for _, item := range items {
		if item.Severity.Rank() > severity.Rank() {
			severity = item.Severity
		}
	}

	return severity
}

func severityTone(severity watchtower.Severity) recordcase.Tone {
	switch severity {
	case watchtower.SeverityCritical:
		return recordcase.ToneBad
	case watchtower.SeverityWarning:
		return recordcase.ToneWarning
	case watchtower.SeverityInfo:
		return recordcase.ToneInfo
	default:
		return recordcase.ToneNeutral
	}
}

func findingOf(
	built *composed,
	reading *recordcase.Reading,
	status recordcase.ReadingStatus,
) *recordcase.Finding {
	if len(built.items) == 0 && built.decision == nil {
		return nil
	}
	finding := &recordcase.Finding{
		Sources:       len(built.exhibits),
		ReadingStatus: status,
	}
	if built.decision != nil {
		finding.AgentDefinitionID = built.decision.AgentDefinitionID
		finding.FoundAt = built.decision.ProposedAt
	}
	for _, item := range built.items {
		if finding.AgentDefinitionID.IsNil() {
			finding.AgentDefinitionID = item.AgentDefinitionID
		}
		if finding.FoundAt == 0 || item.OccurredAt < finding.FoundAt {
			finding.FoundAt = item.OccurredAt
		}
	}

	for _, option := range built.options {
		if finding.AgentDefinitionID.IsNil() {
			finding.AgentDefinitionID = option.AgentDefinitionID
		}
	}

	primary := primaryCondition(built)
	if primary != nil {
		finding.Raised = "Watchtower raised this at " + clockIn(primary.OccurredAt, built) +
			" from " + strings.ToLower(primary.SourceKind.Label()) + "."
		finding.Computed = stringutils.FirstNonEmptyTrimmed(primary.Summary, primary.Title)
	}

	matters, unchanged := "", built.subject.unchanged
	if primary != nil {
		matters = strings.Join(primary.Impact, " · ")
	}
	if unchanged == "" && built.subject.head.Deadline == nil {
		if head := headOf(built); head.Deadline != nil && head.Deadline.Label != "" &&
			head.Deadline.At > built.now {
			unchanged = head.Deadline.Label + " at " + clockIn(head.Deadline.At, built) + "."
		}
	}
	if reading != nil {
		finding.Segments = reading.Segments
		matters = stringutils.FirstNonEmptyTrimmed(reading.WhyItMatters, matters)
		unchanged = stringutils.FirstNonEmptyTrimmed(reading.IfNothingChanges, unchanged)
	}

	finding.Trust = make([]recordcase.TrustCell, 0, 4)
	if matters != "" {
		finding.Trust = append(finding.Trust, recordcase.TrustCell{
			Key:  recordcase.TrustMatters,
			Text: matters,
		})
	}
	if unchanged != "" {
		finding.Trust = append(finding.Trust, recordcase.TrustCell{
			Key:  recordcase.TrustUnchanged,
			Text: unchanged,
		})
	}
	if built.decision != nil && built.decision.AgentDefinitionID.IsNotNil() {
		proposes := built.decision.Title
		if built.chosen != nil && built.chosen.Description != "" {
			proposes = built.chosen.Description
		}
		finding.Trust = append(finding.Trust, recordcase.TrustCell{
			Key:  recordcase.TrustProposes,
			Text: proposes,
		})
		if built.decision.Phase == recordcase.PhaseAwaiting {
			finding.Trust = append(finding.Trust, recordcase.TrustCell{
				Key:  recordcase.TrustApproval,
				Text: approvalReason,
			})
		}
	}

	return finding
}

func primaryCondition(built *composed) *watchtower.Item {
	found := conditions(built)
	if len(found) == 0 {
		return nil
	}

	return found[0]
}

func clockIn(ts int64, built *composed) string {
	read := subjectRead{zone: built.zone, now: built.now}

	return read.clock(ts)
}

func noteOf(built *composed) string {
	primary := primaryCondition(built)
	if primary == nil {
		if built.decision != nil && built.decision.Phase == recordcase.PhaseDone {
			return "Nothing else is open about this record."
		}

		return ""
	}

	return "Watchtower: " + strings.ToLower(string(primary.Severity)) + ", raised " +
		clockIn(primary.OccurredAt, built) + " by " + strings.ToLower(primary.SourceKind.Label()) +
		" monitoring. It clears on its own when the record is back on track."
}

func fingerprintOf(built *composed) string {
	signals := make([]recordcase.Signal, 0, len(built.items))
	for _, item := range built.items {
		signals = append(signals, recordcase.Signal{
			Kind:     string(item.SourceKind),
			SourceID: item.SourceID,
			Severity: string(item.Severity),
			Title:    item.Title,
			DueAt:    item.DueAt,
		})
	}

	return recordcase.ReadingFingerprint(headOf(built).Status, signals)
}
