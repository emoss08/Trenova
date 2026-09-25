package aiauditservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	correlationSlack       = 1
	correlationPerEvent    = 20
	maxCorrelationsPerPage = 5000
)

// CorrelationMatch is how an event is matched to the audit log rows written
// for the same change: the record it touched, the principal the write was
// audited as, and the event's time window. A decision is audited against its
// proposal. Events that change nothing have no match.
func CorrelationMatch(event *aiaudit.AIAuditEvent) (repositories.AuditEntryMatch, bool) {
	if event == nil || event.WindowStart <= 0 {
		return repositories.AuditEntryMatch{}, false
	}

	resource := event.EntityID
	switch event.Kind {
	case aiaudit.KindProposalDecided:
		resource = event.ProposalID.String()
	case aiaudit.KindToolCall:
		if event.Outcome != aiaudit.OutcomeRan && event.Outcome != aiaudit.OutcomeFailed {
			return repositories.AuditEntryMatch{}, false
		}
	case aiaudit.KindProposalExecuted, aiaudit.KindProposalExecutionFailed:
	default:
		return repositories.AuditEntryMatch{}, false
	}
	if resource == "" {
		return repositories.AuditEntryMatch{}, false
	}

	var principal pulid.ID
	switch {
	case event.PrincipalType == aiaudit.PrincipalUser && event.PrincipalID != "":
		principal = pulid.ID(event.PrincipalID)
	case event.ActingUserID().IsNotNil():
		principal = event.ActingUserID()
	case event.PrincipalType == aiaudit.PrincipalAgent:
		principal = serviceports.AgentPrincipalID
	default:
		return repositories.AuditEntryMatch{}, false
	}

	return repositories.AuditEntryMatch{
		ResourceID:  resource,
		PrincipalID: principal,
		WindowStart: event.WindowStart,
		WindowEnd:   max(event.WindowEnd, event.WindowStart),
	}, true
}

func (m matchKey) holds(entry *audit.Entry) bool {
	return entry.ResourceID == m.match.ResourceID &&
		(entry.PrincipalID == m.match.PrincipalID || entry.UserID == m.match.PrincipalID) &&
		entry.Timestamp >= m.match.WindowStart-correlationSlack &&
		entry.Timestamp <= m.match.WindowEnd+correlationSlack
}

type matchKey struct {
	eventID pulid.ID
	match   repositories.AuditEntryMatch
}

// Correlate finds the audit log rows matched by time to each of a page of
// events, in one query.
func Correlate(
	ctx context.Context,
	ledger repositories.AIAuditRepository,
	tenantInfo pagination.TenantInfo,
	events []*aiaudit.AIAuditEvent,
) (map[pulid.ID][]*audit.Entry, error) {
	keys := make([]matchKey, 0, len(events))
	matches := make([]repositories.AuditEntryMatch, 0, len(events))
	for _, event := range events {
		match, ok := CorrelationMatch(event)
		if !ok {
			continue
		}
		keys = append(keys, matchKey{eventID: event.ID, match: match})
		matches = append(matches, match)
	}

	grouped := make(map[pulid.ID][]*audit.Entry, len(keys))
	if len(matches) == 0 {
		return grouped, nil
	}

	entries, err := ledger.ListCorrelatedAuditEntries(
		ctx,
		&repositories.ListCorrelatedAuditEntriesRequest{
			TenantInfo: tenantInfo,
			Matches:    matches,
			Limit:      min(len(matches)*correlationPerEvent, maxCorrelationsPerPage),
		},
	)
	if err != nil {
		return nil, err
	}

	for _, key := range keys {
		for _, entry := range entries {
			if key.holds(entry) {
				grouped[key.eventID] = append(grouped[key.eventID], entry)
			}
		}
	}

	return grouped, nil
}
