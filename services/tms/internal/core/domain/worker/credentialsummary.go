package worker

import (
	"sort"

	"github.com/emoss08/trenova/shared/pulid"
)

// CredentialSummaryItem is one slot in a worker's credential picture: every
// required type appears (possibly Missing), and every optional type the worker
// actually holds.
type CredentialSummaryItem struct {
	CredentialType  *WorkerCredentialType
	Credential      *WorkerCredential
	Health          CredentialHealth
	DaysUntilExpiry *int64
	Required        bool
}

type WorkerCredentialSummary struct {
	WorkerID         pulid.ID
	ComplianceStatus ComplianceStatus
	RequiredCount    int
	ValidCount       int
	ExpiringCount    int
	ExpiredCount     int
	MissingCount     int
	Items            []*CredentialSummaryItem
}

// BuildCredentialSummary is the pure roll-up. Only active credentials count;
// a required slot with no active credential is Missing. A worker is compliant
// when no required slot blocks, and an expiring-soon slot warns but does not
// block. Workers with no required types are Compliant by construction.
func BuildCredentialSummary(
	wrk *Worker,
	types []*WorkerCredentialType,
	credentials []*WorkerCredential,
	now int64,
) *WorkerCredentialSummary {
	summary := &WorkerCredentialSummary{
		ComplianceStatus: ComplianceStatusCompliant,
		Items:            make([]*CredentialSummaryItem, 0, len(types)),
	}
	if wrk != nil {
		summary.WorkerID = wrk.ID
	}

	activeByType := make(map[pulid.ID]*WorkerCredential, len(credentials))
	for _, cred := range credentials {
		if cred == nil || !cred.IsActive() {
			continue
		}
		activeByType[cred.CredentialTypeID] = cred
	}

	seen := make(map[pulid.ID]struct{}, len(types))
	for _, credentialType := range types {
		if credentialType == nil {
			continue
		}
		seen[credentialType.ID] = struct{}{}
		required := credentialType.AppliesTo(wrk)
		cred, held := activeByType[credentialType.ID]
		if !required && !held {
			continue
		}

		item := &CredentialSummaryItem{
			CredentialType: credentialType,
			Credential:     cred,
			Required:       required,
			Health:         CredentialHealthMissing,
		}
		if held {
			cred.CredentialType = credentialType
			item.Health = EvaluateCredentialHealth(
				cred.ExpiresAt,
				credentialType.RenewalWindowDays,
				now,
			)
			item.DaysUntilExpiry = cred.DaysUntilExpiry(now)
		}
		summary.add(item)
	}

	for _, cred := range credentials {
		if cred == nil || !cred.IsActive() {
			continue
		}
		if _, ok := seen[cred.CredentialTypeID]; ok {
			continue
		}
		if cred.CredentialType == nil {
			continue
		}
		summary.add(&CredentialSummaryItem{
			CredentialType:  cred.CredentialType,
			Credential:      cred,
			Health:          cred.Health(now),
			DaysUntilExpiry: cred.DaysUntilExpiry(now),
		})
	}

	sort.SliceStable(summary.Items, func(i, j int) bool {
		a, b := summary.Items[i], summary.Items[j]
		if a.Required != b.Required {
			return a.Required
		}
		if a.CredentialType.SortOrder != b.CredentialType.SortOrder {
			return a.CredentialType.SortOrder < b.CredentialType.SortOrder
		}
		return a.CredentialType.Name < b.CredentialType.Name
	})

	return summary
}

func (s *WorkerCredentialSummary) add(item *CredentialSummaryItem) {
	s.Items = append(s.Items, item)
	if item.Required {
		s.RequiredCount++
	}
	switch item.Health {
	case CredentialHealthValid:
		s.ValidCount++
	case CredentialHealthExpiringSoon:
		s.ExpiringCount++
	case CredentialHealthExpired:
		s.ExpiredCount++
	case CredentialHealthMissing:
		s.MissingCount++
	}
	if item.Required && item.Health.Blocks() {
		s.ComplianceStatus = ComplianceStatusNonCompliant
	}
}

// Attention returns the slots that need action, worst first.
func (s *WorkerCredentialSummary) Attention() []*CredentialSummaryItem {
	out := make([]*CredentialSummaryItem, 0, len(s.Items))
	for _, item := range s.Items {
		if item.Health != CredentialHealthValid {
			out = append(out, item)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return healthRank(out[i].Health) < healthRank(out[j].Health)
	})
	return out
}

func healthRank(h CredentialHealth) int {
	switch h {
	case CredentialHealthMissing:
		return 0
	case CredentialHealthExpired:
		return 1
	case CredentialHealthExpiringSoon:
		return 2
	case CredentialHealthValid:
		return 3
	default:
		return 4
	}
}
