package agentreflectionservice

import (
	"cmp"
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpagecontext"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreflection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentreflectionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/textcase"
)

const (
	// MinCorrectedRecords is how many different records must have had the
	// same field changed before it reads as a habit. One record changed
	// three times is someone making up their mind; three records changed the
	// same way is how the organization does it.
	MinCorrectedRecords = 3
	// correctionWindowDays is how far back hand edits are read for a habit,
	// and how long a habit already looked at is left alone.
	correctionWindowDays = 30
	// correctionWriteDays is how long before the window an agent's write may
	// have happened and still be the work an edit in the window corrects.
	correctionWriteDays = 7
	// correctionFreshDays is how recent a habit's newest edit must be for a
	// look back now: an old habit that stopped is not worth a model call.
	correctionFreshDays      = 2
	maxCorrectionLookBacks   = 5
	maxCorrectionInstances   = 20
	correctionPatternKeySep  = "|"
	correctionDetailSep      = " · "
	maxCorrectionDetailChars = 120
)

// CorrectionPattern is one field of one kind of record that one agent set
// and people changed by hand, with the latest change on each record.
type CorrectionPattern struct {
	AgentDefinitionID pulid.ID
	Resource          string
	Field             string
	Work              []agentreflectionports.CorrectedWork
	Newest            int64
}

func (p *CorrectionPattern) Key() string {
	return strings.Join(
		[]string{p.AgentDefinitionID.String(), p.Resource, p.Field},
		correctionPatternKeySep,
	)
}

// What names the pattern in plain words: "shipment · delivery window start".
func (p *CorrectionPattern) What() string {
	return strings.ToLower(textcase.HumanizeCamelCaseSentence(p.Resource)) + correctionDetailSep +
		strings.ToLower(textcase.HumanizeCamelCaseSentence(p.Field))
}

// CorrectionPatterns groups hand edits of agents' work by agent, kind of
// record and field, and keeps the groups that are a habit: the field changed
// on MinCorrectedRecords different records, the newest of them lately. The
// most widespread comes first.
func CorrectionPatterns(
	work []agentreflectionports.CorrectedWork,
	now int64,
) []*CorrectionPattern {
	type recordKey struct {
		pattern string
		record  string
	}
	patterns := make(map[string]*CorrectionPattern, len(work))
	latest := make(map[recordKey]int, len(work))

	for idx := range work {
		item := &work[idx]
		for _, field := range item.Fields {
			pattern := &CorrectionPattern{
				AgentDefinitionID: item.AgentDefinitionID,
				Resource:          item.Resource,
				Field:             field.Field,
			}
			key := pattern.Key()
			if found, ok := patterns[key]; ok {
				pattern = found
			} else {
				patterns[key] = pattern
			}

			one := *item
			one.Fields = []agentreflectionports.CorrectedField{field}
			record := recordKey{pattern: key, record: item.ResourceID}
			if at, seen := latest[record]; seen {
				if pattern.Work[at].ChangedAt < item.ChangedAt {
					pattern.Work[at] = one
				}
			} else {
				latest[record] = len(pattern.Work)
				pattern.Work = append(pattern.Work, one)
			}
			pattern.Newest = max(pattern.Newest, item.ChangedAt)
		}
	}

	fresh := now - correctionFreshDays*secondsPerDay
	habits := make([]*CorrectionPattern, 0, len(patterns))
	for _, pattern := range patterns {
		if len(pattern.Work) < MinCorrectedRecords || pattern.Newest < fresh {
			continue
		}
		slices.SortFunc(pattern.Work, func(a, b agentreflectionports.CorrectedWork) int {
			return cmp.Compare(b.ChangedAt, a.ChangedAt)
		})
		if len(pattern.Work) > maxCorrectionInstances {
			pattern.Work = pattern.Work[:maxCorrectionInstances]
		}
		habits = append(habits, pattern)
	}
	slices.SortFunc(habits, func(a, b *CorrectionPattern) int {
		if byRecords := cmp.Compare(len(b.Work), len(a.Work)); byRecords != 0 {
			return byRecords
		}

		return strings.Compare(a.Key(), b.Key())
	})

	return habits
}

func (s *Service) PlanCorrections(
	ctx context.Context,
	tenant pagination.TenantInfo,
	now int64,
) ([]*agentreflectionports.ReflectOnCorrectionsRequest, error) {
	control, err := s.controls.GetOrCreate(ctx, tenant)
	if err != nil {
		return nil, err
	}
	if control.LearningOff || control.ShadowMode {
		return nil, nil
	}

	since := now - correctionWindowDays*secondsPerDay
	work, err := s.corrections.ListCorrectedWork(
		ctx,
		&agentreflectionports.ListCorrectedWorkRequest{
			TenantInfo:   tenant,
			WroteSince:   since - correctionWriteDays*secondsPerDay,
			ChangedSince: since,
			Until:        now,
		},
	)
	if err != nil {
		return nil, err
	}
	habits := CorrectionPatterns(work, now)
	if len(habits) == 0 {
		return nil, nil
	}

	keys := make([]string, 0, len(habits))
	for _, habit := range habits {
		keys = append(keys, habit.Key())
	}
	looked, err := s.repo.ListCorrectionLookBacks(
		ctx,
		&agentreflectionports.ListCorrectionLookBacksRequest{
			TenantInfo:  tenant,
			PatternKeys: keys,
			Since:       since,
		},
	)
	if err != nil {
		return nil, err
	}

	requests := make(
		[]*agentreflectionports.ReflectOnCorrectionsRequest,
		0,
		min(len(habits), maxCorrectionLookBacks),
	)
	for _, habit := range habits {
		if len(requests) == maxCorrectionLookBacks {
			break
		}
		if slices.Contains(looked, habit.Key()) {
			continue
		}
		requests = append(requests, &agentreflectionports.ReflectOnCorrectionsRequest{
			TenantInfo:        tenant,
			AgentDefinitionID: habit.AgentDefinitionID,
			PatternKey:        habit.Key(),
			Resource:          habit.Resource,
			Field:             habit.Field,
			Day:               int(now / secondsPerDay),
			Work:              habit.Work,
		})
	}

	return requests, nil
}

func (s *Service) PrepareCorrections(
	ctx context.Context,
	req *agentreflectionports.ReflectOnCorrectionsRequest,
) (*agentreflectionports.ReflectionPlan, error) {
	tenant := req.TenantInfo
	if len(req.Work) == 0 || req.AgentDefinitionID.IsNil() {
		return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, nil
	}

	reflection, err := s.claim(ctx, &agentreflection.Reflection{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		AgentDefinitionID: req.AgentDefinitionID,
		SubjectType:       agentreflection.ReflectionSubjectCorrection,
		PatternKey:        req.PatternKey,
		FromSequence:      req.Day,
		ThroughSequence:   req.Day,
	})
	if err != nil || reflection == nil {
		return &agentreflectionports.ReflectionPlan{TenantInfo: tenant}, err
	}

	plan := &agentreflectionports.ReflectionPlan{
		TenantInfo:        tenant,
		ReflectionID:      reflection.ID,
		Subject:           agentreflection.ReflectionSubjectCorrection,
		AgentDefinitionID: req.AgentDefinitionID,
	}
	pattern := &CorrectionPattern{Resource: req.Resource, Field: req.Field}

	return s.claimed(ctx, reflection, func() (*agentreflectionports.ReflectionPlan, error) {
		definition, _, skip, agentErr := s.usableAgent(ctx, tenant, req.AgentDefinitionID)
		if agentErr != nil {
			return nil, agentErr
		}
		if skip != "" {
			return s.skip(ctx, reflection, plan, skip)
		}

		return s.ready(ctx, &readyParams{
			reflection: reflection,
			plan:       plan,
			definition: definition,
			window: &Window{
				Corrections:   req.Work,
				CorrectedWhat: pattern.What(),
			},
			trigger:  "people changing by hand what the agent had set",
			provider: definition.PreferredProviderID,
			records:  correctedRecords(req.Work),
		})
	})
}

func correctedRecords(work []agentreflectionports.CorrectedWork) []agentpagecontext.EntityRef {
	records := make([]agentpagecontext.EntityRef, 0, min(len(work), maxWindowRecords))
	for idx := range work {
		if len(records) == maxWindowRecords {
			break
		}
		records = append(records, agentpagecontext.EntityRef{
			Type: work[idx].Resource,
			ID:   work[idx].ResourceID,
		})
	}

	return records
}
