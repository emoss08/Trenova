package aiaudittest

import (
	"cmp"
	"context"
	"errors"
	"slices"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/config/aiauditconfig"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditcorrelate"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditkeyring"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunevent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/dataretention"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// FakeLedger keeps the trail in memory with the repository's semantics:
// existing source keys skipped, seqs assigned in order, every batch sealed.
type FakeLedger struct {
	aiauditports.AIAuditRepository

	Mu         sync.Mutex
	Events     map[pagination.TenantInfo][]*aiaudit.AIAuditEvent
	Heads      map[pagination.TenantInfo]*aiaudit.AIAuditChainHead
	Seals      map[pagination.TenantInfo][]*aiaudit.AIAuditSeal
	Watermarks map[aiaudit.Source]*aiaudit.AIAuditProjectorState
	Entries    []*audit.Entry
}

func NewFakeLedger() *FakeLedger {
	return &FakeLedger{
		Events:     map[pagination.TenantInfo][]*aiaudit.AIAuditEvent{},
		Heads:      map[pagination.TenantInfo]*aiaudit.AIAuditChainHead{},
		Seals:      map[pagination.TenantInfo][]*aiaudit.AIAuditSeal{},
		Watermarks: map[aiaudit.Source]*aiaudit.AIAuditProjectorState{},
	}
}

func (f *FakeLedger) Append(
	_ context.Context,
	req *aiauditports.AppendAIAuditEventsRequest,
) (*aiauditports.AppendAIAuditEventsResult, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	head := f.Heads[req.TenantInfo]
	if head == nil {
		head = &aiaudit.AIAuditChainHead{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
		}
		f.Heads[req.TenantInfo] = head
	}

	held := map[string]struct{}{}
	for _, event := range f.Events[req.TenantInfo] {
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
		f.Events[req.TenantInfo] = append(f.Events[req.TenantInfo], event)
		inserted++
	}

	result := &aiauditports.AppendAIAuditEventsResult{Inserted: inserted}
	if inserted == 0 {
		return result, nil
	}

	head.LastSeq = seq
	head.LastHash = prev
	f.Seals[req.TenantInfo] = append(f.Seals[req.TenantInfo], &aiaudit.AIAuditSeal{
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

func (f *FakeLedger) ExistingSourceKeys(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	keys []string,
) (map[string]struct{}, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	wanted := map[string]struct{}{}
	for _, key := range keys {
		wanted[key] = struct{}{}
	}
	found := map[string]struct{}{}
	for _, event := range f.Events[tenantInfo] {
		if _, ok := wanted[event.SourceKey]; ok {
			found[event.SourceKey] = struct{}{}
		}
	}

	return found, nil
}

func (f *FakeLedger) All(tenantInfo pagination.TenantInfo) []*aiaudit.AIAuditEvent {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	return slices.Clone(f.Events[tenantInfo])
}

func (f *FakeLedger) GetChainHead(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (*aiaudit.AIAuditChainHead, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	if head, ok := f.Heads[tenantInfo]; ok {
		copied := *head

		return &copied, nil
	}

	return &aiaudit.AIAuditChainHead{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
	}, nil
}

func (f *FakeLedger) ListChainHeads(context.Context) ([]*aiaudit.AIAuditChainHead, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	heads := make([]*aiaudit.AIAuditChainHead, 0, len(f.Heads))
	for _, head := range f.Heads {
		heads = append(heads, head)
	}

	return heads, nil
}

func (f *FakeLedger) FirstSeq(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
) (int64, bool, error) {
	events := f.All(tenantInfo)
	if len(events) == 0 {
		return 0, false, nil
	}

	return events[0].Seq, true, nil
}

func (f *FakeLedger) ListChainRange(
	_ context.Context,
	req aiauditports.ListAIAuditChainRangeRequest,
) ([]*aiaudit.AIAuditEvent, error) {
	out := make([]*aiaudit.AIAuditEvent, 0, req.Limit)
	for _, event := range f.All(req.TenantInfo) {
		if event.Seq > req.AfterSeq && len(out) < req.Limit {
			out = append(out, event)
		}
	}

	return out, nil
}

func (f *FakeLedger) GetSealEndingAt(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	toSeq int64,
) (*aiaudit.AIAuditSeal, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	for _, seal := range f.Seals[tenantInfo] {
		if seal.ToSeq == toSeq {
			return seal, nil
		}
	}

	return nil, nil //nolint:nilnil // no seal ends there
}

func (f *FakeLedger) ListSeals(
	_ context.Context,
	req aiauditports.ListAIAuditSealsRequest,
) ([]*aiaudit.AIAuditSeal, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	out := make([]*aiaudit.AIAuditSeal, 0, req.Limit)
	for _, seal := range f.Seals[req.TenantInfo] {
		if seal.ToSeq >= req.FromSeq && len(out) < req.Limit {
			out = append(out, seal)
		}
	}

	return out, nil
}

func (f *FakeLedger) LastSealBefore(
	_ context.Context,
	tenantInfo pagination.TenantInfo,
	sealedBefore int64,
) (*aiaudit.AIAuditSeal, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	var last *aiaudit.AIAuditSeal
	for _, seal := range f.Seals[tenantInfo] {
		if seal.SealedAt < sealedBefore {
			last = seal
		}
	}

	return last, nil
}

func (f *FakeLedger) Prune(
	_ context.Context,
	req aiauditports.PruneAIAuditEventsRequest,
) (int, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	kept := f.Events[req.TenantInfo][:0]
	deleted := 0
	for _, event := range f.Events[req.TenantInfo] {
		if event.Seq <= req.ThroughSeq {
			deleted++

			continue
		}
		kept = append(kept, event)
	}
	f.Events[req.TenantInfo] = kept

	return deleted, nil
}

func (f *FakeLedger) RecordVerification(
	_ context.Context,
	req *aiauditports.RecordAIAuditVerificationRequest,
) (*aiaudit.AIAuditChainHead, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	head := f.Heads[req.TenantInfo]
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

func (f *FakeLedger) GetWatermarks(
	context.Context,
) (map[aiaudit.Source]*aiaudit.AIAuditProjectorState, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	out := make(map[aiaudit.Source]*aiaudit.AIAuditProjectorState, len(f.Watermarks))
	for source, state := range f.Watermarks {
		copied := *state
		out[source] = &copied
	}

	return out, nil
}

func (f *FakeLedger) SaveWatermark(_ context.Context, state *aiaudit.AIAuditProjectorState) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()

	copied := *state
	f.Watermarks[state.Source] = &copied

	return nil
}

func (f *FakeLedger) Summarize(
	_ context.Context,
	scope *aiauditports.AIAuditEventScope,
) (*aiauditports.AIAuditEventSummary, error) {
	summary := &aiauditports.AIAuditEventSummary{}
	for _, event := range f.inScope(scope) {
		if summary.Count == 0 || event.Seq < summary.FirstSeq {
			summary.FirstSeq = event.Seq
		}
		summary.LastSeq = max(summary.LastSeq, event.Seq)
		summary.Count++
	}

	return summary, nil
}

func (f *FakeLedger) inScope(scope *aiauditports.AIAuditEventScope) []*aiaudit.AIAuditEvent {
	out := make([]*aiaudit.AIAuditEvent, 0)
	for _, event := range f.All(scope.TenantInfo) {
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

func (f *FakeLedger) ListPage(
	_ context.Context,
	req *aiauditports.ListAIAuditEventPageRequest,
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

func (f *FakeLedger) ListCorrelatedAuditEntries(
	_ context.Context,
	req *aiauditports.ListCorrelatedAuditEntriesRequest,
) ([]*audit.Entry, error) {
	out := make([]*audit.Entry, 0)
	for _, entry := range f.Entries {
		for _, match := range req.Matches {
			if aiauditcorrelate.EntryMatches(entry, match) {
				out = append(out, entry)

				break
			}
		}
	}

	return out, nil
}

type EmptyRetention struct {
	tenantports.DataRetentionRepository
}

func (EmptyRetention) List(
	context.Context,
) (*pagination.ListResult[*dataretention.DataRetention], error) {
	return &pagination.ListResult[*dataretention.DataRetention]{}, nil
}

// FakeSource holds source rows in memory and pages them in (ts, id) order
// across tenants, as the repository does.
type FakeSource struct {
	aiauditports.AIAuditSourceRepository

	Runs        []*agent.AgentRun
	Turns       []*conversationturn.AssistantTurn
	Usage       []*aiusage.AIUsageRecord
	Steps       []*agentrunstep.AgentRunStep
	Events      []*agentrunevent.AgentRunEvent
	Proposals   []*agent.AgentProposal
	Decisions   []*agentdecision.AgentDecision
	Threads     []*conversation.Thread
	Evaluations []*agentevaluation.Evaluation
	Users       map[pulid.ID]string
	Agents      map[pulid.ID]string
}

func page[T any](
	rows []T,
	ts func(T) int64,
	id func(T) string,
	req aiauditports.AIAuditSourcePage,
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

func (f *FakeSource) ListRuns(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*agent.AgentRun, error) {
	return page(f.Runs, func(r *agent.AgentRun) int64 { return r.UpdatedAt },
		func(r *agent.AgentRun) string { return r.ID.String() }, req, nil), nil
}

func (f *FakeSource) ListTurns(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*conversationturn.AssistantTurn, error) {
	return page(f.Turns, func(r *conversationturn.AssistantTurn) int64 { return r.UpdatedAt },
		func(r *conversationturn.AssistantTurn) string { return r.ID.String() }, req, nil), nil
}

func (f *FakeSource) ListUsage(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*aiusage.AIUsageRecord, error) {
	return page(f.Usage, func(r *aiusage.AIUsageRecord) int64 { return r.CreatedAt },
		func(r *aiusage.AIUsageRecord) string { return r.ID.String() }, req, nil), nil
}

func (f *FakeSource) ListSteps(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*agentrunstep.AgentRunStep, error) {
	return page(
		f.Steps,
		func(r *agentrunstep.AgentRunStep) int64 { return r.UpdatedAt },
		func(r *agentrunstep.AgentRunStep) string { return r.ID.String() },
		req,
		func(r *agentrunstep.AgentRunStep) bool { return r.Kind == string(agentruntimeports.RunStepTool) },
	), nil
}

func (f *FakeSource) ListEvents(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*agentrunevent.AgentRunEvent, error) {
	return page(f.Events, func(r *agentrunevent.AgentRunEvent) int64 { return r.CreatedAt },
		func(r *agentrunevent.AgentRunEvent) string { return r.ID.String() }, req, nil), nil
}

func (f *FakeSource) ListProposals(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*agent.AgentProposal, error) {
	return page(f.Proposals, func(r *agent.AgentProposal) int64 { return r.UpdatedAt },
		func(r *agent.AgentProposal) string { return r.ID.String() }, req, nil), nil
}

func (f *FakeSource) ListDecisions(
	_ context.Context,
	req aiauditports.AIAuditSourcePage,
) ([]*agentdecision.AgentDecision, error) {
	return page(f.Decisions, func(r *agentdecision.AgentDecision) int64 { return r.CreatedAt },
		func(r *agentdecision.AgentDecision) string { return r.ID.String() }, req, nil), nil
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

func (f *FakeSource) RunsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentRun, error) {
	return byIDs(f.Runs, func(r *agent.AgentRun) pulid.ID { return r.ID }, ids), nil
}

func (f *FakeSource) TurnsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversationturn.AssistantTurn, error) {
	return byIDs(
		f.Turns,
		func(r *conversationturn.AssistantTurn) pulid.ID { return r.ID },
		ids,
	), nil
}

func (f *FakeSource) TurnsByRunIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversationturn.AssistantTurn, error) {
	return byIDs(
		f.Turns,
		func(r *conversationturn.AssistantTurn) pulid.ID { return r.RunID },
		ids,
	), nil
}

func (f *FakeSource) TurnsInWindow(
	_ context.Context,
	req *aiauditports.AIAuditTurnWindow,
) ([]*conversationturn.AssistantTurn, error) {
	return byIDs(f.Turns, func(r *conversationturn.AssistantTurn) pulid.ID { return r.ThreadID },
		req.ThreadIDs), nil
}

func (f *FakeSource) ThreadsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*conversation.Thread, error) {
	return byIDs(f.Threads, func(r *conversation.Thread) pulid.ID { return r.ID }, ids), nil
}

func (f *FakeSource) EvaluationsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agentevaluation.Evaluation, error) {
	return byIDs(
		f.Evaluations,
		func(r *agentevaluation.Evaluation) pulid.ID { return r.ID },
		ids,
	), nil
}

func (f *FakeSource) ProposalsByIDs(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) ([]*agent.AgentProposal, error) {
	return byIDs(f.Proposals, func(r *agent.AgentProposal) pulid.ID { return r.ID }, ids), nil
}

func (f *FakeSource) StartedToolSteps(
	_ context.Context,
	_ pagination.TenantInfo,
	owners []pulid.ID,
) ([]*agentrunstep.AgentRunStep, error) {
	out := make([]*agentrunstep.AgentRunStep, 0)
	for _, step := range f.Steps {
		if slices.Contains(owners, step.OwnerID) &&
			step.Status == string(agentruntimeports.RunStepStarted) &&
			step.Kind == string(agentruntimeports.RunStepTool) {
			out = append(out, step)
		}
	}

	return out, nil
}

func (f *FakeSource) ToolStepCalls(
	_ context.Context,
	_ pagination.TenantInfo,
	calls []aiauditports.OwnerCall,
) (map[aiauditports.OwnerCall]struct{}, error) {
	found := map[aiauditports.OwnerCall]struct{}{}
	for _, step := range f.Steps {
		call := aiauditports.OwnerCall{OwnerID: step.OwnerID, CallID: step.CallID}
		if slices.Contains(calls, call) {
			found[call] = struct{}{}
		}
	}

	return found, nil
}

func (f *FakeSource) UserNames(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	out := map[pulid.ID]string{}
	for _, id := range ids {
		if name, ok := f.Users[id]; ok {
			out[id] = name
		}
	}

	return out, nil
}

func (f *FakeSource) AgentNames(
	_ context.Context,
	_ pagination.TenantInfo,
	ids []pulid.ID,
) (map[pulid.ID]string, error) {
	out := map[pulid.ID]string{}
	for _, id := range ids {
		if name, ok := f.Agents[id]; ok {
			out[id] = name
		}
	}

	return out, nil
}

const FixedNow = int64(1_800_000_000)

func TestKeyring(signed bool) *aiauditkeyring.Keyring {
	if !signed {
		return aiauditkeyring.New(&aiauditconfig.ChainConfig{}, zap.NewNop())
	}

	return aiauditkeyring.New(&aiauditconfig.ChainConfig{
		Keys:        []aiauditconfig.ChainKey{{ID: "k1", Secret: strings.Repeat("s", 32)}},
		ActiveKeyID: "k1",
	}, zap.NewNop())
}

type FakePolicies map[string]agenttoolports.ToolPolicy

func (f FakePolicies) Get(name string) (agenttoolports.ToolPolicy, bool) {
	policy, ok := f[name]

	return policy, ok
}

func TestPolicies() FakePolicies {
	return FakePolicies{
		"update_worker": {
			Name:     "update_worker",
			Kind:     agenttooltrust.ToolKindAction,
			Resource: permission.ResourceWorker,
		},
		"pin_my_home": {
			Name:  "pin_my_home",
			Kind:  agenttooltrust.ToolKindAction,
			Scope: agenttooltrust.ToolScopeSelf,
		},
		"shipment_get": {
			Name:     "shipment_get",
			Kind:     agenttooltrust.ToolKindQuery,
			Resource: permission.ResourceShipment,
		},
	}
}

type FakeNotifier struct {
	Mu       sync.Mutex
	Permits  []notificationservice.NotifyPermittedRequest
	Personal []*notification.Notification
}

func (f *FakeNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Permits = append(f.Permits, req)

	return 1, nil
}

func (f *FakeNotifier) Create(
	_ context.Context,
	entity *notification.Notification,
) (*notification.Notification, error) {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Personal = append(f.Personal, entity)

	return entity, nil
}

type FakeAudit struct {
	auditports.AuditService

	Mu      sync.Mutex
	Entries []*auditports.LogActionParams
}

func (f *FakeAudit) LogAction(
	params *auditports.LogActionParams,
	_ ...auditports.LogOption,
) error {
	f.Mu.Lock()
	defer f.Mu.Unlock()
	f.Entries = append(f.Entries, params)

	return nil
}
