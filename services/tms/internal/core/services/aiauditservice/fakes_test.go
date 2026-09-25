package aiauditservice

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/aiaudit"
	"github.com/emoss08/trenova/internal/core/domain/aiusage"
	"github.com/emoss08/trenova/internal/core/domain/audit"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// fakeLedger keeps the trail in memory with the repository's semantics:
// existing source keys skipped, seqs assigned in order, every batch sealed.
type fakeLedger struct {
	repositories.AIAuditRepository

	mu         sync.Mutex
	events     map[pagination.TenantInfo][]*aiaudit.AIAuditEvent
	heads      map[pagination.TenantInfo]*aiaudit.AIAuditChainHead
	seals      map[pagination.TenantInfo][]*aiaudit.AIAuditSeal
	watermarks map[aiaudit.Source]*aiaudit.AIAuditProjectorState
	entries    []*audit.Entry
}

func newFakeLedger() *fakeLedger {
	return &fakeLedger{
		events:     map[pagination.TenantInfo][]*aiaudit.AIAuditEvent{},
		heads:      map[pagination.TenantInfo]*aiaudit.AIAuditChainHead{},
		seals:      map[pagination.TenantInfo][]*aiaudit.AIAuditSeal{},
		watermarks: map[aiaudit.Source]*aiaudit.AIAuditProjectorState{},
	}
}

func (f *fakeLedger) Append(
	_ context.Context,
	req *repositories.AppendAIAuditEventsRequest,
) (*repositories.AppendAIAuditEventsResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	head := f.heads[req.TenantInfo]
	if head == nil {
		head = &aiaudit.AIAuditChainHead{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
		}
		f.heads[req.TenantInfo] = head
	}

	held := map[string]struct{}{}
	for _, event := range f.events[req.TenantInfo] {
		held[event.SourceKey] = struct{}{}
	}

	prev := head.LastHash
	if head.LastSeq == 0 {
		prev = aiaudit.GenesisHash
	}
	seq := head.LastSeq
	from := seq + 1
	inserted := 0
	for _, event := range req.Events {
		if _, dup := held[event.SourceKey]; dup {
			continue
		}
		held[event.SourceKey] = struct{}{}
		seq++
		event.Seq = seq
		event.OrganizationID = req.TenantInfo.OrgID
		event.BusinessUnitID = req.TenantInfo.BuID
		if event.ID.IsNil() {
			event.ID = pulid.MustNew(aiaudit.EventIDPrefix)
		}
		if event.RecordedAt == 0 {
			event.RecordedAt = req.Now
		}
		if err := req.Sign(event, prev); err != nil {
			return nil, err
		}
		prev = event.Hash
		f.events[req.TenantInfo] = append(f.events[req.TenantInfo], event)
		inserted++
	}

	result := &repositories.AppendAIAuditEventsResult{Inserted: inserted}
	if inserted == 0 {
		return result, nil
	}

	head.LastSeq = seq
	head.LastHash = prev
	f.seals[req.TenantInfo] = append(f.seals[req.TenantInfo], &aiaudit.AIAuditSeal{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		FromSeq:        from,
		ToSeq:          seq,
		HeadHash:       prev,
		HashKeyID:      req.KeyID,
		RowCount:       inserted,
		SealedAt:       req.Now,
	})
	result.FromSeq, result.ToSeq, result.HeadHash = from, seq, prev

	return result, nil
}

func (f *fakeLedger) ExistingSourceKeys(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	keys []string,
) (map[string]struct{}, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	wanted := map[string]struct{}{}
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	found := map[string]struct{}{}
	for _, event := range f.events[tenantInfo] {
		if _, ok := wanted[event.SourceKey]; ok {
			found[event.SourceKey] = struct{}{}
		}
	}

	return found, nil
}

func (f *fakeLedger) all(tenantInfo pagination.TenantInfo) []*aiaudit.AIAuditEvent {
	f.mu.Lock()
	defer f.mu.Unlock()

	return slices.Clone(f.events[tenantInfo])
}

func (f *fakeLedger) GetChainHead(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*aiaudit.AIAuditChainHead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if head, ok := f.heads[tenantInfo]; ok {
		copied := *head

		return &copied, nil
	}

	return &aiaudit.AIAuditChainHead{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}, nil
}

func (f *fakeLedger) ListChainHeads(context.Context) ([]*aiaudit.AIAuditChainHead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	heads := make([]*aiaudit.AIAuditChainHead, 0, len(f.heads))
	for _, head := range f.heads {
		heads = append(heads, head)
	}

	return heads, nil
}

func (f *fakeLedger) FirstSeq(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, bool, error) {
	events := f.all(tenantInfo)
	if len(events) == 0 {
		return 0, false, nil
	}

	return events[0].Seq, true, nil
}

func (f *fakeLedger) ListChainRange(
	_ context.Context,
	req repositories.ListAIAuditChainRangeRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	out := make([]*aiaudit.AIAuditEvent, 0, req.Limit)
	for _, event := range f.all(req.TenantInfo) {
		if event.Seq > req.AfterSeq && len(out) < req.Limit {
			out = append(out, event)
		}
	}

	return out, nil
}

func (f *fakeLedger) GetSealEndingAt(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	toSeq int64,
) (*aiaudit.AIAuditSeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, seal := range f.seals[tenantInfo] {
		if seal.ToSeq == toSeq {
			return seal, nil
		}
	}

	return nil, nil //nolint:nilnil // no seal ends there
}

func (f *fakeLedger) ListSeals(
	_ context.Context,
	req repositories.ListAIAuditSealsRequest,
) ([]*aiaudit.AIAuditSeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]*aiaudit.AIAuditSeal, 0, req.Limit)
	for _, seal := range f.seals[req.TenantInfo] {
		if seal.ToSeq >= req.FromSeq && len(out) < req.Limit {
			out = append(out, seal)
		}
	}

	return out, nil
}

func (f *fakeLedger) LastSealBefore(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	sealedBefore int64,
) (*aiaudit.AIAuditSeal, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	var last *aiaudit.AIAuditSeal
	for _, seal := range f.seals[tenantInfo] {
		if seal.SealedAt < sealedBefore {
			last = seal
		}
	}

	return last, nil
}

func (f *fakeLedger) Prune(
	_ context.Context,
	req repositories.PruneAIAuditEventsRequest,
) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	kept := f.events[req.TenantInfo][:0]
	deleted := 0
	for _, event := range f.events[req.TenantInfo] {
		if event.Seq <= req.ThroughSeq {
			deleted++

			continue
		}
		kept = append(kept, event)
	}
	f.events[req.TenantInfo] = kept

	return deleted, nil
}

func (f *fakeLedger) RecordVerification(
	_ context.Context,
	req *repositories.RecordAIAuditVerificationRequest,
) (*aiaudit.AIAuditChainHead, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	head := f.heads[req.TenantInfo]
	if head == nil {
		return nil, errors.New("no head")
	}
	head.LastVerificationStatus = req.Status
	head.LastVerificationFailedSeq = req.FailedSeq
	head.LastVerificationDetail = req.Detail
	at := req.At
	head.LastVerifiedAt = &at
	if req.Status == aiaudit.VerificationVerified {
		head.LastVerifiedSeq = req.VerifiedSeq
	}

	return head, nil
}

func (f *fakeLedger) GetWatermarks(
	context.Context,
) (map[aiaudit.Source]*aiaudit.AIAuditProjectorState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make(map[aiaudit.Source]*aiaudit.AIAuditProjectorState, len(f.watermarks))
	for source, state := range f.watermarks {
		copied := *state
		out[source] = &copied
	}

	return out, nil
}

func (f *fakeLedger) SaveWatermark(_ context.Context, state *aiaudit.AIAuditProjectorState) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	copied := *state
	f.watermarks[state.Source] = &copied

	return nil
}

func (f *fakeLedger) Summarize(
	_ context.Context,
	scope *repositories.AIAuditEventScope,
) (*repositories.AIAuditEventSummary, error) {
	summary := &repositories.AIAuditEventSummary{}
	for _, event := range f.inScope(scope) {
		if summary.Count == 0 || event.Seq < summary.FirstSeq {
			summary.FirstSeq = event.Seq
		}
		summary.LastSeq = max(summary.LastSeq, event.Seq)
		summary.Count++
	}

	return summary, nil
}

func (f *fakeLedger) inScope(scope *repositories.AIAuditEventScope) []*aiaudit.AIAuditEvent {
	out := make([]*aiaudit.AIAuditEvent, 0)
	for _, event := range f.all(scope.TenantInfo) {
		if event.OccurredAt < scope.From || event.OccurredAt > scope.To {
			continue
		}
		if scope.SnapshotSeq > 0 && event.Seq > scope.SnapshotSeq {
			continue
		}
		if scope.Filter != nil {
			if !matchesFilters(event, scope.Filter) {
				continue
			}
		}
		out = append(out, event)
	}
	slices.SortFunc(out, func(a, b *aiaudit.AIAuditEvent) int {
		return cmp.Or(cmp.Compare(a.OccurredAt, b.OccurredAt), cmp.Compare(a.ID, b.ID))
	})

	return out
}

func matchesFilters(event *aiaudit.AIAuditEvent, filter *aiaudit.ExportFilter) bool {
	for _, field := range filter.FieldFilters {
		if field.Field == "kind" && string(event.Kind) != field.Value {
			return false
		}
	}

	return true
}

func (f *fakeLedger) ListPage(
	_ context.Context,
	req *repositories.ListAIAuditEventPageRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	out := make([]*aiaudit.AIAuditEvent, 0, req.Limit)
	for _, event := range f.inScope(&req.Scope) {
		if req.HasAfterCursor && (event.OccurredAt < req.AfterOccurred ||
			(event.OccurredAt == req.AfterOccurred && event.ID <= req.AfterID)) {
			continue
		}
		if len(out) == req.Limit {
			break
		}
		out = append(out, event)
	}

	return out, nil
}

func (f *fakeLedger) ListCorrelatedAuditEntries(
	_ context.Context,
	req *repositories.ListCorrelatedAuditEntriesRequest,
) ([]*audit.Entry, error) {
	out := make([]*audit.Entry, 0)
	for _, entry := range f.entries {
		for _, match := range req.Matches {
			key := matchKey{match: match}
			if key.holds(entry) {
				out = append(out, entry)

				break
			}
		}
	}

	return out, nil
}

type emptyRetention struct {
	repositories.DataRetentionRepository
}

func (emptyRetention) List(context.Context) (*pagination.ListResult[*tenant.DataRetention], error) {
	return &pagination.ListResult[*tenant.DataRetention]{}, nil
}

func pruneThrough(s *scenario, seq int64) repositories.PruneAIAuditEventsRequest {
	return repositories.PruneAIAuditEventsRequest{TenantInfo: s.tenant, ThroughSeq: seq}
}

// fakeSource holds source rows in memory and pages them in (ts, id) order
// across tenants, as the repository does.
type fakeSource struct {
	repositories.AIAuditSourceRepository

	runs        []*agent.AgentRun
	turns       []*conversation.AssistantTurn
	usage       []*aiusage.AIUsageRecord
	steps       []*agent.AgentRunStep
	events      []*agent.AgentRunEvent
	proposals   []*agent.AgentProposal
	decisions   []*agent.AgentDecision
	threads     []*conversation.Thread
	evaluations []*agent.Evaluation
	users       map[pulid.ID]string
	agents      map[pulid.ID]string
}

func page[T any](
	rows []T,
	ts func(T) int64,
	id func(T) string,
	req repositories.AIAuditSourcePage,
	keep func(T) bool,
) []T {
	sorted := slices.Clone(rows)
	slices.SortFunc(sorted, func(a, b T) int {
		return cmp.Or(cmp.Compare(ts(a), ts(b)), cmp.Compare(id(a), id(b)))
	})

	out := make([]T, 0, req.Limit)
	for _, row := range sorted {
		if keep != nil && !keep(row) {
			continue
		}
		if ts(row) < req.AfterTS || (ts(row) == req.AfterTS && id(row) <= req.AfterID) {
			continue
		}
		if ts(row) > req.UntilTS || len(out) == req.Limit {
			break
		}
		out = append(out, row)
	}

	return out
}

func (f *fakeSource) ListRuns(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*agent.AgentRun, error) {
	return page(f.runs, func(r *agent.AgentRun) int64 { return r.UpdatedAt },
		func(r *agent.AgentRun) string { return r.ID.String() }, req, nil), nil
}

func (f *fakeSource) ListTurns(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*conversation.AssistantTurn, error) {
	return page(f.turns, func(r *conversation.AssistantTurn) int64 { return r.UpdatedAt },
		func(r *conversation.AssistantTurn) string { return r.ID.String() }, req, nil), nil
}

func (f *fakeSource) ListUsage(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*aiusage.AIUsageRecord, error) {
	return page(f.usage, func(r *aiusage.AIUsageRecord) int64 { return r.CreatedAt },
		func(r *aiusage.AIUsageRecord) string { return r.ID.String() }, req, nil), nil
}

func (f *fakeSource) ListSteps(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*agent.AgentRunStep, error) {
	return page(f.steps, func(r *agent.AgentRunStep) int64 { return r.UpdatedAt },
		func(r *agent.AgentRunStep) string { return r.ID.String() }, req,
		func(r *agent.AgentRunStep) bool { return r.Kind == string(serviceports.RunStepTool) }), nil
}

func (f *fakeSource) ListEvents(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*agent.AgentRunEvent, error) {
	return page(f.events, func(r *agent.AgentRunEvent) int64 { return r.CreatedAt },
		func(r *agent.AgentRunEvent) string { return r.ID.String() }, req, nil), nil
}

func (f *fakeSource) ListProposals(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*agent.AgentProposal, error) {
	return page(f.proposals, func(r *agent.AgentProposal) int64 { return r.UpdatedAt },
		func(r *agent.AgentProposal) string { return r.ID.String() }, req, nil), nil
}

func (f *fakeSource) ListDecisions(
	_ context.Context,
	req repositories.AIAuditSourcePage,
) ([]*agent.AgentDecision, error) {
	return page(f.decisions, func(r *agent.AgentDecision) int64 { return r.CreatedAt },
		func(r *agent.AgentDecision) string { return r.ID.String() }, req, nil), nil
}

func byIDs[T any](rows []T, id func(T) pulid.ID, ids []pulid.ID) []T {
	out := make([]T, 0, len(ids))
	for _, row := range rows {
		if slices.Contains(ids, id(row)) {
			out = append(out, row)
		}
	}

	return out
}

func (f *fakeSource) RunsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentRun, error) {
	return byIDs(f.runs, func(r *agent.AgentRun) pulid.ID { return r.ID }, ids), nil
}

func (f *fakeSource) TurnsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.AssistantTurn, error) {
	return byIDs(f.turns, func(r *conversation.AssistantTurn) pulid.ID { return r.ID }, ids), nil
}

func (f *fakeSource) TurnsByRunIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.AssistantTurn, error) {
	return byIDs(f.turns, func(r *conversation.AssistantTurn) pulid.ID { return r.RunID }, ids), nil
}

func (f *fakeSource) TurnsInWindow(
	_ context.Context,
	req repositories.AIAuditTurnWindow,
) ([]*conversation.AssistantTurn, error) {
	return byIDs(f.turns, func(r *conversation.AssistantTurn) pulid.ID { return r.ThreadID },
		req.ThreadIDs), nil
}

func (f *fakeSource) ThreadsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.Thread, error) {
	return byIDs(f.threads, func(r *conversation.Thread) pulid.ID { return r.ID }, ids), nil
}

func (f *fakeSource) EvaluationsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.Evaluation, error) {
	return byIDs(f.evaluations, func(r *agent.Evaluation) pulid.ID { return r.ID }, ids), nil
}

func (f *fakeSource) ProposalsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentProposal, error) {
	return byIDs(f.proposals, func(r *agent.AgentProposal) pulid.ID { return r.ID }, ids), nil
}

func (f *fakeSource) StartedToolSteps(
	_ context.Context,
	_ pagination.TenantInfo,
	owners []pulid.ID,
) ([]*agent.AgentRunStep, error) {
	out := make([]*agent.AgentRunStep, 0)
	for _, step := range f.steps {
		if slices.Contains(owners, step.OwnerID) &&
			step.Status == string(serviceports.RunStepStarted) &&
			step.Kind == string(serviceports.RunStepTool) {
			out = append(out, step)
		}
	}

	return out, nil
}

func (f *fakeSource) ToolStepCalls(
	_ context.Context,
	_ pagination.TenantInfo,
	calls []repositories.OwnerCall,
) (map[repositories.OwnerCall]struct{}, error) {
	found := map[repositories.OwnerCall]struct{}{}
	for _, step := range f.steps {
		call := repositories.OwnerCall{OwnerID: step.OwnerID, CallID: step.CallID}
		if slices.Contains(calls, call) {
			found[call] = struct{}{}
		}
	}

	return found, nil
}

func (f *fakeSource) UserNames(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	out := map[pulid.ID]string{}
	for _, id := range ids {
		if name, ok := f.users[id]; ok {
			out[id] = name
		}
	}

	return out, nil
}

func (f *fakeSource) AgentNames(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	out := map[pulid.ID]string{}
	for _, id := range ids {
		if name, ok := f.agents[id]; ok {
			out[id] = name
		}
	}

	return out, nil
}
