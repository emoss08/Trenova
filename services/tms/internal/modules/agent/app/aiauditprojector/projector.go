package aiauditprojector

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/observability/aimetrics"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditheartbeat"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditkeyring"
	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditredact"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunevent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrunstep"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversationturn"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantstreamports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// ProjectorTrail keeps the projector behind the newest rows, so a write
	// still committing when a pass reads is read by the next one.
	ProjectorTrail = 10 * time.Second
	// ProjectorOverlap is how far behind its watermark each pass reads
	// again, for a row whose timestamp was taken long before its transaction
	// committed. What was already projected is skipped by its source key.
	ProjectorOverlap = 15 * time.Minute
	// maxPagesPerSource bounds one pass's read of one source, so a backfill
	// advances a step each pass rather than holding a worker for hours.
	maxPagesPerSource = 10

	passResultOK     = "ok"
	passResultFailed = "failed"
)

// PassResult is what one projector pass did.
type PassResult struct {
	Inserted int                     `json:"inserted"`
	Tenants  int                     `json:"tenants"`
	Read     map[aiaudit.Source]int  `json:"read"`
	CaughtUp map[aiaudit.Source]bool `json:"caughtUp"`
}

// Projector writes the trail. It is the only writer: each pass reads every
// source past its watermark, derives the rows each source row stands for,
// and appends them per tenant under the tenant's chain lock.
type Projector struct {
	ledger   aiauditports.AIAuditRepository
	source   aiauditports.AIAuditSourceRepository
	keyring  *aiauditkeyring.Keyring
	deriver  *deriver
	metrics  *aimetrics.AIAudit
	batch    int
	now      func() time.Time
	trail    time.Duration
	overlap  time.Duration
	maxPages int
	l        *zap.Logger
}

type Params struct {
	Ledger    aiauditports.AIAuditRepository
	Source    aiauditports.AIAuditSourceRepository
	Keyring   *aiauditkeyring.Keyring
	Redactor  *aiauditredact.Redactor
	Metrics   *aimetrics.AIAudit
	BatchSize int
	Now       func() time.Time
	Logger    *zap.Logger
}

func New(p *Params) *Projector {
	now := p.Now
	if now == nil {
		now = time.Now
	}
	batch := p.BatchSize
	if batch <= 0 {
		batch = 1000
	}

	return &Projector{
		ledger:   p.Ledger,
		source:   p.Source,
		keyring:  p.Keyring,
		deriver:  &deriver{redactor: p.Redactor},
		metrics:  p.Metrics,
		batch:    batch,
		now:      now,
		trail:    ProjectorTrail,
		overlap:  ProjectorOverlap,
		maxPages: maxPagesPerSource,
		l:        p.Logger.Named("aiaudit.projector"),
	}
}

// cursor is a position in a source: the timestamp column it is scanned by
// and the id that breaks ties.
type cursor struct {
	ts int64
	id string
}

func (c cursor) after(other cursor) bool {
	return c.ts > other.ts || (c.ts == other.ts && c.id > other.id)
}

// tenantRows is one tenant's share of a pass: its source rows, deduplicated
// by id across the forward read and the overlap re-read.
type tenantRows struct {
	tenant    pagination.TenantInfo
	runs      map[pulid.ID]*agent.AgentRun
	turns     map[pulid.ID]*conversationturn.AssistantTurn
	usage     map[pulid.ID]*aiusage.AIUsageRecord
	steps     map[pulid.ID]*agentrunstep.AgentRunStep
	events    map[pulid.ID]*agentrunevent.AgentRunEvent
	proposals map[pulid.ID]*agent.AgentProposal
	decisions map[pulid.ID]*agentdecision.AgentDecision
}

func newTenantRows(tenant pagination.TenantInfo) *tenantRows {
	return &tenantRows{
		tenant:    tenant,
		runs:      map[pulid.ID]*agent.AgentRun{},
		turns:     map[pulid.ID]*conversationturn.AssistantTurn{},
		usage:     map[pulid.ID]*aiusage.AIUsageRecord{},
		steps:     map[pulid.ID]*agentrunstep.AgentRunStep{},
		events:    map[pulid.ID]*agentrunevent.AgentRunEvent{},
		proposals: map[pulid.ID]*agent.AgentProposal{},
		decisions: map[pulid.ID]*agentdecision.AgentDecision{},
	}
}

type pass struct {
	tenants map[pagination.TenantInfo]*tenantRows
}

func (p *pass) rows(orgID, buID pulid.ID) *tenantRows {
	key := pagination.TenantInfo{OrgID: orgID, BuID: buID}
	rows, ok := p.tenants[key]
	if !ok {
		rows = newTenantRows(key)
		p.tenants[key] = rows
	}

	return rows
}

// RunOnce is one projector pass.
func (p *Projector) RunOnce(
	ctx context.Context,
	heartbeat aiauditheartbeat.Heartbeat,
) (*PassResult, error) {
	result, err := p.runOnce(ctx, heartbeat)
	if err != nil {
		p.metrics.RecordPass(passResultFailed)

		return nil, err
	}
	p.metrics.RecordPass(passResultOK)
	p.metrics.RecordProjected(result.Inserted)

	return result, nil
}

func (p *Projector) runOnce(
	ctx context.Context,
	heartbeat aiauditheartbeat.Heartbeat,
) (*PassResult, error) {
	now := p.now()
	until := now.Add(-p.trail).Unix()

	states, err := p.ledger.GetWatermarks(ctx)
	if err != nil {
		return nil, err
	}

	current := &pass{tenants: make(map[pagination.TenantInfo]*tenantRows)}
	result := &PassResult{
		Read:     make(map[aiaudit.Source]int, len(aiaudit.AllSources())),
		CaughtUp: make(map[aiaudit.Source]bool, len(aiaudit.AllSources())),
	}
	next := make(map[aiaudit.Source]cursor, len(aiaudit.AllSources()))

	for _, source := range aiaudit.AllSources() {
		from := cursor{}
		if state, ok := states[source]; ok {
			from = cursor{ts: state.WatermarkTS, id: state.WatermarkID}
		}

		reached, caughtUp, read, readErr := p.readSource(ctx, current, source, from, until)
		if readErr != nil {
			return nil, readErr
		}
		next[source] = reached
		result.Read[source] = read
		result.CaughtUp[source] = caughtUp
		aiauditheartbeat.Beat(heartbeat, source.String(), read)
	}

	inserted, err := p.project(ctx, current, now.Unix(), heartbeat)
	if err != nil {
		return nil, err
	}
	result.Inserted = inserted
	result.Tenants = len(current.tenants)

	for _, source := range aiaudit.AllSources() {
		reached := next[source]
		if err = p.ledger.SaveWatermark(ctx, &aiaudit.AIAuditProjectorState{
			Source:      source,
			WatermarkTS: reached.ts,
			WatermarkID: reached.id,
		}); err != nil {
			return nil, err
		}

		lag := float64(0)
		if !result.CaughtUp[source] && reached.ts > 0 {
			lag = now.Sub(time.Unix(reached.ts, 0)).Seconds()
		}
		p.metrics.RecordLag(source.String(), lag)
	}

	return result, nil
}

// readSource reads one source forward from its watermark, then re-reads the
// overlap behind it. It returns where the forward read reached and whether it
// reached the trailing bound.
func (p *Projector) readSource(
	ctx context.Context,
	current *pass,
	source aiaudit.Source,
	from cursor,
	until int64,
) (cursor, bool, int, error) {
	reached := from
	read := 0
	caughtUp := false

	for range p.maxPages {
		n, last, err := p.readPage(ctx, current, source, aiauditports.AIAuditSourcePage{
			AfterTS: reached.ts,
			AfterID: reached.id,
			UntilTS: until,
			Limit:   p.batch,
		})
		if err != nil {
			return from, false, read, err
		}
		read += n
		if n > 0 && last.after(reached) {
			reached = last
		}
		if n < p.batch {
			caughtUp = true
			break
		}
	}

	if from.ts <= 0 {
		return reached, caughtUp, read, nil
	}

	overlapFrom := cursor{ts: max(from.ts-int64(p.overlap.Seconds()), 0)}
	for range p.maxPages {
		n, last, err := p.readPage(ctx, current, source, aiauditports.AIAuditSourcePage{
			AfterTS: overlapFrom.ts,
			AfterID: overlapFrom.id,
			UntilTS: from.ts,
			Limit:   p.batch,
		})
		if err != nil {
			return from, false, read, err
		}
		if n < p.batch {
			break
		}
		overlapFrom = last
	}

	return reached, caughtUp, read, nil
}

func (p *Projector) readPage(
	ctx context.Context,
	current *pass,
	source aiaudit.Source,
	page aiauditports.AIAuditSourcePage,
) (int, cursor, error) {
	switch source {
	case aiaudit.SourceAgentRuns:
		rows, err := p.source.ListRuns(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).runs[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.UpdatedAt, id: last.ID.String()}, nil
	case aiaudit.SourceAssistantTurns:
		rows, err := p.source.ListTurns(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).turns[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.UpdatedAt, id: last.ID.String()}, nil
	case aiaudit.SourceUsage:
		rows, err := p.source.ListUsage(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).usage[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.CreatedAt, id: last.ID.String()}, nil
	case aiaudit.SourceRunSteps:
		rows, err := p.source.ListSteps(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).steps[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.UpdatedAt, id: last.ID.String()}, nil
	case aiaudit.SourceRunEvents:
		rows, err := p.source.ListEvents(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).events[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.CreatedAt, id: last.ID.String()}, nil
	case aiaudit.SourceProposals:
		rows, err := p.source.ListProposals(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).proposals[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.UpdatedAt, id: last.ID.String()}, nil
	case aiaudit.SourceDecisions:
		rows, err := p.source.ListDecisions(ctx, page)
		if err != nil || len(rows) == 0 {
			return 0, cursor{}, err
		}
		for _, row := range rows {
			current.rows(row.OrganizationID, row.BusinessUnitID).decisions[row.ID] = row
		}
		last := rows[len(rows)-1]
		return len(rows), cursor{ts: last.CreatedAt, id: last.ID.String()}, nil
	default:
		return 0, cursor{}, fmt.Errorf("unknown AI audit source %q", source)
	}
}

// project derives and appends each tenant's rows. One tenant's failure fails
// the pass, so no watermark moves past rows that were not written; the other
// tenants' rows are skipped by their source keys when the pass is re-read.
func (p *Projector) project(
	ctx context.Context,
	current *pass,
	now int64,
	heartbeat aiauditheartbeat.Heartbeat,
) (int, error) {
	tenants := make([]*tenantRows, 0, len(current.tenants))
	for _, rows := range current.tenants {
		tenants = append(tenants, rows)
	}
	slices.SortFunc(tenants, func(a, b *tenantRows) int {
		return cmp.Or(
			cmp.Compare(a.tenant.OrgID, b.tenant.OrgID),
			cmp.Compare(a.tenant.BuID, b.tenant.BuID),
		)
	})

	sign := p.keyring.Signer()
	inserted := 0
	var failures []error
	for _, rows := range tenants {
		events, err := p.deriveTenant(dbscope.WithTenant(ctx, rows.tenant.DBTenant()), rows)
		if err != nil {
			failures = append(failures, fmt.Errorf("derive AI audit rows for %s: %w",
				rows.tenant.OrgID, err))

			continue
		}
		if len(events) == 0 {
			continue
		}

		appended, err := p.ledger.Append(ctx, &aiauditports.AppendAIAuditEventsRequest{
			TenantInfo: rows.tenant,
			Events:     events,
			Sign:       sign,
			KeyID:      p.keyring.ActiveKeyID(),
			Now:        now,
		})
		if err != nil {
			failures = append(failures, fmt.Errorf("append AI audit rows for %s: %w",
				rows.tenant.OrgID, err))

			continue
		}
		inserted += appended.Inserted
		aiauditheartbeat.Beat(heartbeat, rows.tenant.OrgID.String(), appended.Inserted)
	}

	if len(failures) > 0 {
		return inserted, errors.Join(failures...)
	}

	return inserted, nil
}

// deriveTenant turns one tenant's rows into trail rows, skipping source rows
// whose every row is already on the trail before reading anything more.
func (p *Projector) deriveTenant(
	ctx context.Context,
	rows *tenantRows,
) ([]*aiaudit.AIAuditEvent, error) {
	existing, err := p.ledger.ExistingSourceKeys(ctx, rows.tenant, candidateKeys(rows))
	if err != nil {
		return nil, err
	}
	pending := pendingRows(rows, existing)

	need := newNeeds()
	collectNeeds(need, pending)
	found, err := load(ctx, p.source, rows.tenant, need)
	if err != nil {
		return nil, err
	}

	events, err := p.derive(found, pending)
	if err != nil {
		return nil, err
	}

	unknown, err := p.unknownSteps(ctx, rows.tenant, found, pending, existing)
	if err != nil {
		return nil, err
	}
	events = append(events, unknown...)
	if len(events) == 0 {
		return nil, nil
	}

	if err = loadNames(ctx, p.source, rows.tenant, found, events); err != nil {
		return nil, err
	}
	for _, event := range events {
		finish(event, found)
	}

	slices.SortStableFunc(events, func(a, b *aiaudit.AIAuditEvent) int {
		return cmp.Or(
			cmp.Compare(a.OccurredAt, b.OccurredAt),
			cmp.Compare(a.SourceKey, b.SourceKey),
		)
	})

	return events, nil
}

func collectNeeds(need *needs, rows *tenantRows) {
	for _, run := range rows.runs {
		need.owner(string(agent.RunOwnerAgentRun), run.ID)
	}
	for _, turn := range rows.turns {
		need.owner(string(agent.RunOwnerAssistantTurn), turn.ID)
	}
	for _, record := range rows.usage {
		switch {
		case record.OwnerID.IsNotNil():
			need.owner(string(record.OwnerKind), record.OwnerID)
		case record.RunID.IsNotNil():
			need.owner(string(agent.RunOwnerAgentRun), record.RunID)
		default:
			need.usageWindow(record.ThreadID, record.CreatedAt)
		}
	}
	for _, step := range rows.steps {
		need.owner(step.OwnerKind, step.OwnerID)
	}
	for _, event := range rows.events {
		need.owner(event.OwnerKind, event.OwnerID)
		if event.Kind == assistantstreamports.AssistantEventToolFinished && event.CallID != "" {
			need.calls = append(need.calls, aiauditports.OwnerCall{
				OwnerID: event.OwnerID,
				CallID:  event.CallID,
			})
		}
	}
	for _, proposal := range rows.proposals {
		need.owner(string(agent.RunOwnerAgentRun), proposal.RunID)
	}
	for _, decision := range rows.decisions {
		if decision.ProposalID != nil {
			need.proposals[*decision.ProposalID] = struct{}{}
		}
	}
}

// derive turns every pending row into its trail rows. Decisions name their
// proposal's run, which is read after the proposals themselves.
func (p *Projector) derive(found *lookups, rows *tenantRows) ([]*aiaudit.AIAuditEvent, error) {
	events := make([]*aiaudit.AIAuditEvent, 0,
		2*len(rows.runs)+2*len(rows.turns)+len(rows.usage)+len(rows.steps)+
			len(rows.events)+2*len(rows.proposals)+len(rows.decisions))

	for _, run := range rows.runs {
		events = append(events, p.deriver.runEvents(found, run)...)
	}
	for _, turn := range rows.turns {
		events = append(events, p.deriver.turnEvents(found, turn)...)
	}
	for _, record := range rows.usage {
		events = append(events, p.deriver.usageEvent(found, record))
	}
	for _, step := range rows.steps {
		if step.Status == string(agentruntimeports.RunStepStarted) {
			owner := found.ownerOf(step.OwnerKind, step.OwnerID)
			if !owner.terminal {
				continue
			}
			event, err := p.deriver.unknownStepEvent(found, step)
			if err != nil {
				return nil, err
			}
			events = append(events, event)

			continue
		}
		event, err := p.deriver.stepEvent(found, step)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	for _, row := range rows.events {
		if event := p.deriver.runEventEvent(found, row); event != nil {
			events = append(events, event)
		}
	}
	for _, proposal := range rows.proposals {
		derived, err := p.deriver.proposalEvents(found, proposal)
		if err != nil {
			return nil, err
		}
		events = append(events, derived...)
	}

	for _, decision := range rows.decisions {
		event, err := p.deriver.decisionEvent(found, decision)
		if err != nil {
			return nil, err
		}
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// unknownSteps finds the calls left unsettled by the runs and turns whose end
// this pass records: they can never settle now, and whether they ran cannot
// be known.
func (p *Projector) unknownSteps(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	found *lookups,
	rows *tenantRows,
	existing map[string]struct{},
) ([]*aiaudit.AIAuditEvent, error) {
	owners := make([]pulid.ID, 0, len(rows.runs)+len(rows.turns))
	for _, run := range rows.runs {
		if run.CompletedAt != nil {
			owners = append(owners, run.ID)
		}
	}
	for _, turn := range rows.turns {
		if turn.Status.Terminal() {
			owners = append(owners, turn.ID)
		}
	}
	if len(owners) == 0 {
		return nil, nil
	}

	steps, err := p.source.StartedToolSteps(ctx, tenantInfo, owners)
	if err != nil {
		return nil, err
	}

	events := make([]*aiaudit.AIAuditEvent, 0, len(steps))
	for _, step := range steps {
		if _, pendingInPass := rows.steps[step.ID]; pendingInPass {
			continue
		}
		key := sourceKey("step", step.OwnerID.String(), step.StepKey)
		if _, done := existing[key]; done {
			continue
		}
		event, deriveErr := p.deriver.unknownStepEvent(found, step)
		if deriveErr != nil {
			return nil, deriveErr
		}
		events = append(events, event)
	}

	return events, nil
}

// SourcePruneHorizon is the newest time a source's own retention sweep may
// delete before. Rows after it may not be on the trail yet: the watermark
// less the overlap every pass re-reads. A source never read has no horizon.
func (p *Projector) SourcePruneHorizon(ctx context.Context, source aiaudit.Source) (int64, error) {
	states, err := p.ledger.GetWatermarks(ctx)
	if err != nil {
		return 0, err
	}

	state, ok := states[source]
	if !ok || state.WatermarkTS <= 0 {
		return 0, nil
	}

	return max(state.WatermarkTS-int64(p.overlap.Seconds()), 0), nil
}
