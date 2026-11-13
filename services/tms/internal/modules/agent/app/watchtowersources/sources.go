package watchtowersources

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/insightservice/detector"
	"github.com/emoss08/trenova/internal/modules/agent/app/watchtoweritem"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionqueueports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentexceptionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/insightports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edimessageports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

/*
What each source says is open right now.

The tower is filled by projections as records open, and corrected against
these snapshots: whatever a source no longer reports is resolved. That is
what makes a lost projection cost a night rather than a permanent ghost.

Two kinds of record never close on their own — a run that failed and an
hours violation, both of which are facts rather than states — so their
snapshot is a window. Past it, the reconcile resolves them: the tower is
about what needs a person now.
*/
const (
	// failedRunWindow and violationWindow are how long a fact stays on the
	// tower.
	failedRunWindow  = 72 * 60 * 60
	decisionPageSize = 100
	decisionPages    = 5
)

func tenantQuery(tenant pagination.TenantInfo, limit int) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: tenant,
		Pagination: pagination.Info{Limit: limit},
	}
}

func filterEq(field string, value any) domaintypes.FieldFilter {
	return domaintypes.FieldFilter{Field: field, Operator: dbtype.OpEqual, Value: value}
}

// InsightSource puts every active finding on the tower.
type InsightSource struct {
	repo      insightports.InsightRepository
	detectors *detector.Registry
}

func NewInsightSource(
	repo insightports.InsightRepository,
	detectors *detector.Registry,
) watchtowerports.WatchtowerSource {
	return &InsightSource{repo: repo, detectors: detectors}
}

func (s *InsightSource) Kind() watchtower.SourceKind { return watchtower.SourceInsight }

func (s *InsightSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	// Every detector's findings belong on the tower; who may see them is
	// decided when the feed is read, by the source kind's resource.
	keys := make(insightports.AllowedDetectorKeys, 0, len(s.detectors.All()))
	for _, d := range s.detectors.All() {
		keys = append(keys, d.Key())
	}

	insights, err := s.repo.ListActive(ctx, insightports.ListActiveInsightsRequest{
		TenantInfo:          tenant,
		AllowedDetectorKeys: keys,
		Limit:               watchtoweritem.SnapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(insights))
	for _, entity := range insights {
		items = append(items, DescribeInsight(entity))
	}

	return items, nil
}

// DecisionSource puts what is waiting on a person on the tower: proposals
// on their own and plans as one unit, the same queue the Desk decides.
type DecisionSource struct {
	queue       agentdecisionqueueports.AgentDecisionQueueService
	runs        agentruntimeports.AgentRunRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	kind        watchtower.SourceKind
}

func NewProposalSource(
	queue agentdecisionqueueports.AgentDecisionQueueService,
	runs agentruntimeports.AgentRunRepository,
	definitions agentdefinitionports.AgentDefinitionRepository,
) watchtowerports.WatchtowerSource {
	return &DecisionSource{
		queue:       queue,
		runs:        runs,
		definitions: definitions,
		kind:        watchtower.SourceAgentProposal,
	}
}

func NewPlanSource(
	queue agentdecisionqueueports.AgentDecisionQueueService,
	runs agentruntimeports.AgentRunRepository,
	definitions agentdefinitionports.AgentDefinitionRepository,
) watchtowerports.WatchtowerSource {
	return &DecisionSource{
		queue:       queue,
		runs:        runs,
		definitions: definitions,
		kind:        watchtower.SourceAgentPlan,
	}
}

func (s *DecisionSource) Kind() watchtower.SourceKind { return s.kind }

func (s *DecisionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	pending := make([]agentdecisionqueueports.PendingDecision, 0, decisionPageSize)
	after := ""
	for range decisionPages {
		page, err := s.queue.ListPending(ctx, &agentdecisionqueueports.ListPendingDecisionsRequest{
			TenantInfo: tenant,
			First:      decisionPageSize,
			After:      after,
		})
		if err != nil {
			return nil, err
		}
		pending = append(pending, page.Items...)
		if !page.HasNextPage || len(page.Items) == 0 {
			break
		}
		after = page.Items[len(page.Items)-1].Cursor
	}

	runIDs := make([]pulid.ID, 0, len(pending))
	for _, decision := range pending {
		switch {
		case decision.Proposal != nil:
			runIDs = append(runIDs, decision.Proposal.RunID)
		case decision.Plan != nil:
			runIDs = append(runIDs, decision.Plan.RunID)
		}
	}
	names := s.agentNames(ctx, tenant, runIDs)

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(pending))
	for _, decision := range pending {
		switch {
		case decision.Proposal != nil && s.kind == watchtower.SourceAgentProposal:
			items = append(
				items,
				DescribeProposal(decision.Proposal, names[decision.Proposal.RunID]),
			)
		case decision.Plan != nil && s.kind == watchtower.SourceAgentPlan:
			items = append(items, DescribePlan(decision.Plan, names[decision.Plan.RunID]))
		}
	}

	return items, nil
}

// agentNames resolves run ids to the agent behind each, so the tower says
// who is asking rather than "an agent". A lookup that fails costs the name
// and not the item.
func (s *DecisionSource) agentNames(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runIDs []pulid.ID,
) map[pulid.ID]string {
	names := map[pulid.ID]string{}
	if len(runIDs) == 0 || s.runs == nil || s.definitions == nil {
		return names
	}

	runs, err := s.runs.ListByIDs(ctx, &agentruntimeports.ListAgentRunsByIDsRequest{
		IDs:        unique(runIDs),
		TenantInfo: tenant,
	})
	if err != nil {
		return names
	}

	definitionIDs := make([]pulid.ID, 0, len(runs))
	runDefinition := make(map[pulid.ID]pulid.ID, len(runs))
	for _, run := range runs {
		if run == nil || run.AgentDefinitionID.IsNil() {
			continue
		}
		runDefinition[run.ID] = run.AgentDefinitionID
		definitionIDs = append(definitionIDs, run.AgentDefinitionID)
	}
	if len(definitionIDs) == 0 {
		return names
	}

	definitions, err := s.definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{
			IDs:        unique(definitionIDs),
			TenantInfo: tenant,
		},
	)
	if err != nil {
		return names
	}
	byID := make(map[pulid.ID]string, len(definitions))
	for _, definition := range definitions {
		if definition != nil {
			byID[definition.ID] = definition.Name
		}
	}
	for runID, definitionID := range runDefinition {
		names[runID] = byID[definitionID]
	}

	return names
}

func unique(ids []pulid.ID) []pulid.ID {
	seen := make(map[pulid.ID]struct{}, len(ids))
	out := make([]pulid.ID, 0, len(ids))
	for _, id := range ids {
		if id.IsNil() {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}

	return out
}

// FailedRunSource puts an agent that has stopped working on the tower, so a
// desk that relies on it learns of it. A run that failed is only that news
// while it is the last word: an agent that has since finished a run is
// working again, and a timeout it got over days ago is nothing for a person
// to do. So a failure is on the tower until the agent's next run that ends
// well, or until it ages out of the window.
//
// An agent in shadow, by its own switch or the organization's pause, is not
// working for anyone yet, so its failures are not news for the tower.
type FailedRunSource struct {
	runs        agentruntimeports.AgentRunRepository
	definitions agentdefinitionports.AgentDefinitionRepository
	controls    tenantports.AgentControlRepository
	now         func() int64
}

func NewFailedRunSource(
	runs agentruntimeports.AgentRunRepository,
	definitions agentdefinitionports.AgentDefinitionRepository,
	controls tenantports.AgentControlRepository,
) watchtowerports.WatchtowerSource {
	return &FailedRunSource{
		runs:        runs,
		definitions: definitions,
		controls:    controls,
		now:         timeutils.NowUnix,
	}
}

func (s *FailedRunSource) Kind() watchtower.SourceKind { return watchtower.SourceAgentRunFailed }

func (s *FailedRunSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	if s.controls != nil {
		control, cErr := s.controls.GetOrCreate(ctx, tenant)
		if cErr != nil {
			return nil, cErr
		}
		if control != nil && control.ShadowMode {
			return []watchtowerports.WatchtowerItemInput{}, nil
		}
	}

	filter := tenantQuery(tenant, watchtoweritem.SnapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("status", string(agent.RunStatusFailed)),
		{Field: "createdAt", Operator: dbtype.OpGreaterThan, Value: s.now() - failedRunWindow},
	}

	result, err := s.runs.List(ctx, &agentruntimeports.ListAgentRunRequest{
		Filter: filter,
		Reader: agentruntimeports.PersonRunRead(pulid.Nil),
	})
	if err != nil {
		return nil, err
	}

	definitionIDs := make([]pulid.ID, 0, len(result.Items))
	for _, run := range result.Items {
		if run != nil && run.AgentDefinitionID.IsNotNil() {
			definitionIDs = append(definitionIDs, run.AgentDefinitionID)
		}
	}
	names := map[pulid.ID]string{}
	shadowed := map[pulid.ID]struct{}{}
	if len(definitionIDs) > 0 && s.definitions != nil {
		definitions, dErr := s.definitions.ListByIDs(
			ctx,
			agentdefinitionports.ListAgentDefinitionsByIDsRequest{
				IDs:        unique(definitionIDs),
				TenantInfo: tenant,
			},
		)
		if dErr == nil {
			for _, definition := range definitions {
				if definition != nil {
					names[definition.ID] = definition.Name
					if definition.ShadowMode {
						shadowed[definition.ID] = struct{}{}
					}
				}
			}
		}
	}

	latest, err := s.stillFailing(ctx, tenant, latestFailurePerAgent(result.Items))
	if err != nil {
		return nil, err
	}
	items := make([]watchtowerports.WatchtowerItemInput, 0, len(latest))
	for _, run := range latest {
		// An agent in shadow is watched in AI control; its failures are not
		// put in front of the people working the queue.
		if _, shadow := shadowed[run.AgentDefinitionID]; shadow {
			continue
		}
		items = append(items, DescribeFailedRun(run, names[run.AgentDefinitionID]))
	}

	return items, nil
}

// stillFailing keeps the failures no later run of the same agent has got
// past. One read covers every failing agent: its runs that ended well since
// the oldest of the failures.
func (s *FailedRunSource) stillFailing(
	ctx context.Context,
	tenant pagination.TenantInfo,
	failures []*agent.AgentRun,
) ([]*agent.AgentRun, error) {
	if len(failures) == 0 {
		return failures, nil
	}

	agents := make([]string, 0, len(failures))
	oldest := failures[0].CreatedAt
	for _, run := range failures {
		agents = append(agents, run.AgentDefinitionID.String())
		oldest = min(oldest, run.CreatedAt)
	}

	filter := tenantQuery(tenant, watchtoweritem.SnapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		{Field: "status", Operator: dbtype.OpIn, Value: []string{
			string(agent.RunStatusCompleted),
			string(agent.RunStatusShadowCompleted),
			string(agent.RunStatusAwaitingDecision),
		}},
		{Field: "agentDefinitionId", Operator: dbtype.OpIn, Value: agents},
		{Field: "createdAt", Operator: dbtype.OpGreaterThan, Value: oldest},
	}
	recovered, err := s.runs.List(ctx, &agentruntimeports.ListAgentRunRequest{
		Filter: filter,
		Reader: agentruntimeports.PersonRunRead(pulid.Nil),
	})
	if err != nil {
		return nil, err
	}

	return failuresNotGotPast(failures, recovered.Items), nil
}

// failuresNotGotPast drops each failure its agent has finished a run after.
func failuresNotGotPast(failures, finished []*agent.AgentRun) []*agent.AgentRun {
	lastGood := make(map[pulid.ID]int64, len(failures))
	for _, run := range finished {
		if run != nil && run.CreatedAt > lastGood[run.AgentDefinitionID] {
			lastGood[run.AgentDefinitionID] = run.CreatedAt
		}
	}

	out := make([]*agent.AgentRun, 0, len(failures))
	for _, run := range failures {
		if lastGood[run.AgentDefinitionID] <= run.CreatedAt {
			out = append(out, run)
		}
	}

	return out
}

// latestFailurePerAgent keeps one failed run for each agent, its newest. An
// agent that fails five runs in a row has one problem, and five items saying
// so bury the rest of the feed; the earlier failures clear as the sweep finds
// them gone from here, and the run list in AI control still holds them all.
func latestFailurePerAgent(runs []*agent.AgentRun) []*agent.AgentRun {
	newest := make(map[pulid.ID]*agent.AgentRun, len(runs))
	order := make([]pulid.ID, 0, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		known, seen := newest[run.AgentDefinitionID]
		if !seen {
			order = append(order, run.AgentDefinitionID)
		}
		if !seen || run.CreatedAt > known.CreatedAt {
			newest[run.AgentDefinitionID] = run
		}
	}

	out := make([]*agent.AgentRun, 0, len(order))
	for _, id := range order {
		out = append(out, newest[id])
	}

	return out
}

// ExceptionSource puts open agent exceptions on the tower.
type ExceptionSource struct {
	repo agentexceptionports.AgentExceptionRepository
}

func NewExceptionSource(
	repo agentexceptionports.AgentExceptionRepository,
) watchtowerports.WatchtowerSource {
	return &ExceptionSource{repo: repo}
}

func (s *ExceptionSource) Kind() watchtower.SourceKind { return watchtower.SourceAgentException }

func (s *ExceptionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, watchtoweritem.SnapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("resolutionState", string(agentexception.ResolutionStateOpen)),
	}

	result, err := s.repo.List(ctx, &agentexceptionports.ListAgentExceptionRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if entity != nil {
			items = append(items, DescribeException(entity))
		}
	}

	return items, nil
}

// ServiceFailureSource puts the late and missed stops nobody has resolved
// on the tower.
type ServiceFailureSource struct {
	repo servicefailureports.ServiceFailureRepository
}

func NewServiceFailureSource(
	repo servicefailureports.ServiceFailureRepository,
) watchtowerports.WatchtowerSource {
	return &ServiceFailureSource{repo: repo}
}

func (s *ServiceFailureSource) Kind() watchtower.SourceKind {
	return watchtower.SourceServiceFailure
}

func (s *ServiceFailureSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, watchtoweritem.SnapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("status", string(servicefailure.StatusOpen)),
	}

	result, err := s.repo.List(ctx, &servicefailureports.ListServiceFailuresRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if entity != nil {
			items = append(items, DescribeServiceFailure(entity))
		}
	}

	return items, nil
}

// CarrierIntelSource puts open carrier changes on the tower.
type CarrierIntelSource struct {
	repo carrierintelports.CarrierIntelEventRepository
}

func NewCarrierIntelSource(
	repo carrierintelports.CarrierIntelEventRepository,
) watchtowerports.WatchtowerSource {
	return &CarrierIntelSource{repo: repo}
}

func (s *CarrierIntelSource) Kind() watchtower.SourceKind {
	return watchtower.SourceCarrierIntelEvent
}

func (s *CarrierIntelSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	result, err := s.repo.ListConnection(ctx, &carrierintelports.ListCarrierIntelEventsRequest{
		Filter:   tenantQuery(tenant, watchtoweritem.SnapshotLimit),
		Cursor:   pagination.CursorInfo{Limit: watchtoweritem.SnapshotLimit},
		OpenOnly: true,
	})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if !CarrierIntelEventOnTower(entity) {
			continue
		}
		items = append(items, DescribeCarrierIntelEvent(entity))
	}

	return items, nil
}

// EDIQuarantineSource puts held-back inbound files on the tower.
type EDIQuarantineSource struct {
	repo edimessageports.EDIInboundFileRepository
}

func NewEDIQuarantineSource(
	repo edimessageports.EDIInboundFileRepository,
) watchtowerports.WatchtowerSource {
	return &EDIQuarantineSource{repo: repo}
}

func (s *EDIQuarantineSource) Kind() watchtower.SourceKind {
	return watchtower.SourceEDIInboundQuarantined
}

func (s *EDIQuarantineSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	files, err := s.repo.ListRecentQuarantined(
		ctx,
		edimessageports.ListRecentQuarantinedEDIInboundFilesRequest{
			TenantInfo: tenant,
			Limit:      watchtoweritem.SnapshotLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(files))
	for _, file := range files {
		if file != nil {
			items = append(items, DescribeQuarantinedFile(file))
		}
	}

	return items, nil
}

// BillingExceptionSource puts items that cannot be invoiced on the tower.
type BillingExceptionSource struct {
	repo billingqueueports.BillingQueueRepository
}

func NewBillingExceptionSource(
	repo billingqueueports.BillingQueueRepository,
) watchtowerports.WatchtowerSource {
	return &BillingExceptionSource{repo: repo}
}

func (s *BillingExceptionSource) Kind() watchtower.SourceKind {
	return watchtower.SourceBillingException
}

func (s *BillingExceptionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, watchtoweritem.SnapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("status", string(billingqueue.StatusException)),
	}

	result, err := s.repo.List(ctx, &billingqueueports.ListBillingQueueItemsRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if entity != nil {
			items = append(items, DescribeBillingException(entity))
		}
	}

	return items, nil
}

// DetentionSource puts running detention clocks on the tower.
type DetentionSource struct {
	repo detentionports.DetentionOccurrenceRepository
}

func NewDetentionSource(
	repo detentionports.DetentionOccurrenceRepository,
) watchtowerports.WatchtowerSource {
	return &DetentionSource{repo: repo}
}

func (s *DetentionSource) Kind() watchtower.SourceKind {
	return watchtower.SourceDetentionOccurrence
}

func (s *DetentionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	occurrences, err := s.repo.ListOpen(ctx, tenant)
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(occurrences))
	for _, entity := range occurrences {
		if entity != nil {
			items = append(items, DescribeDetentionOccurrence(entity))
		}
	}

	return items, nil
}

// InboundMessageSource puts mail that is waiting on a person on the tower.
type InboundMessageSource struct {
	repo inboundmessageports.InboundMessageRepository
}

func NewInboundMessageSource(
	repo inboundmessageports.InboundMessageRepository,
) watchtowerports.WatchtowerSource {
	return &InboundMessageSource{repo: repo}
}

func (s *InboundMessageSource) Kind() watchtower.SourceKind {
	return watchtower.SourceInboundMessage
}

func (s *InboundMessageSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]watchtowerports.WatchtowerItemInput, error) {
	messages, err := s.repo.ListRecentForReview(
		ctx,
		inboundmessageports.ListRecentInboundMessagesForReviewRequest{
			TenantInfo: tenant,
			Limit:      watchtoweritem.SnapshotLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]watchtowerports.WatchtowerItemInput, 0, len(messages))
	for _, message := range messages {
		if message != nil {
			items = append(items, DescribeInboundMessage(message))
		}
	}

	return items, nil
}
