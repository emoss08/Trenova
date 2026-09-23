package watchtowersources

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
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
	snapshotLimit = 200
	// failedRunWindow and violationWindow are how long a fact stays on the
	// tower.
	failedRunWindow  = 72 * 60 * 60
	violationWindow  = 7 * 24 * 60 * 60
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
	repo      repositories.InsightRepository
	detectors *detector.Registry
}

func NewInsightSource(
	repo repositories.InsightRepository,
	detectors *detector.Registry,
) services.WatchtowerSource {
	return &InsightSource{repo: repo, detectors: detectors}
}

func (s *InsightSource) Kind() watchtower.SourceKind { return watchtower.SourceInsight }

func (s *InsightSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	// Every detector's findings belong on the tower; who may see them is
	// decided when the feed is read, by the source kind's resource.
	keys := make(repositories.AllowedDetectorKeys, 0, len(s.detectors.All()))
	for _, d := range s.detectors.All() {
		keys = append(keys, d.Key())
	}

	insights, err := s.repo.ListActive(ctx, repositories.ListActiveInsightsRequest{
		TenantInfo:          tenant,
		AllowedDetectorKeys: keys,
		Limit:               snapshotLimit,
	})
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(insights))
	for _, entity := range insights {
		items = append(items, DescribeInsight(entity))
	}

	return items, nil
}

// DecisionSource puts what is waiting on a person on the tower: proposals
// on their own and plans as one unit, the same queue the Desk decides.
type DecisionSource struct {
	queue       services.AgentDecisionQueueService
	runs        repositories.AgentRunRepository
	definitions repositories.AgentDefinitionRepository
	kind        watchtower.SourceKind
}

func NewProposalSource(
	queue services.AgentDecisionQueueService,
	runs repositories.AgentRunRepository,
	definitions repositories.AgentDefinitionRepository,
) services.WatchtowerSource {
	return &DecisionSource{
		queue:       queue,
		runs:        runs,
		definitions: definitions,
		kind:        watchtower.SourceAgentProposal,
	}
}

func NewPlanSource(
	queue services.AgentDecisionQueueService,
	runs repositories.AgentRunRepository,
	definitions repositories.AgentDefinitionRepository,
) services.WatchtowerSource {
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
) ([]services.WatchtowerItemInput, error) {
	pending := make([]services.PendingDecision, 0, decisionPageSize)
	after := ""
	for range decisionPages {
		page, err := s.queue.ListPending(ctx, services.ListPendingDecisionsRequest{
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

	items := make([]services.WatchtowerItemInput, 0, len(pending))
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

	runs, err := s.runs.ListByIDs(ctx, repositories.ListAgentRunsByIDsRequest{
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

	definitions, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        unique(definitionIDs),
		TenantInfo: tenant,
	})
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

// FailedRunSource puts recent agent failures on the tower, so a desk that
// relies on an agent learns when it stopped working.
type FailedRunSource struct {
	runs        repositories.AgentRunRepository
	definitions repositories.AgentDefinitionRepository
	now         func() int64
}

func NewFailedRunSource(
	runs repositories.AgentRunRepository,
	definitions repositories.AgentDefinitionRepository,
) services.WatchtowerSource {
	return &FailedRunSource{runs: runs, definitions: definitions, now: timeutils.NowUnix}
}

func (s *FailedRunSource) Kind() watchtower.SourceKind { return watchtower.SourceAgentRunFailed }

func (s *FailedRunSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, snapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("status", string(agent.RunStatusFailed)),
		{Field: "createdAt", Operator: dbtype.OpGreaterThan, Value: s.now() - failedRunWindow},
	}

	result, err := s.runs.List(ctx, &repositories.ListAgentRunRequest{Filter: filter})
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
	if len(definitionIDs) > 0 && s.definitions != nil {
		definitions, dErr := s.definitions.ListByIDs(
			ctx,
			repositories.ListAgentDefinitionsByIDsRequest{
				IDs:        unique(definitionIDs),
				TenantInfo: tenant,
			},
		)
		if dErr == nil {
			for _, definition := range definitions {
				if definition != nil {
					names[definition.ID] = definition.Name
				}
			}
		}
	}

	items := make([]services.WatchtowerItemInput, 0, len(result.Items))
	for _, run := range result.Items {
		if run == nil {
			continue
		}
		items = append(items, DescribeFailedRun(run, names[run.AgentDefinitionID]))
	}

	return items, nil
}

// ExceptionSource puts open agent exceptions on the tower.
type ExceptionSource struct {
	repo repositories.AgentExceptionRepository
}

func NewExceptionSource(repo repositories.AgentExceptionRepository) services.WatchtowerSource {
	return &ExceptionSource{repo: repo}
}

func (s *ExceptionSource) Kind() watchtower.SourceKind { return watchtower.SourceAgentException }

func (s *ExceptionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, snapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("resolutionState", string(agent.ResolutionStateOpen)),
	}

	result, err := s.repo.List(ctx, &repositories.ListAgentExceptionRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(result.Items))
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
	repo repositories.ServiceFailureRepository
}

func NewServiceFailureSource(repo repositories.ServiceFailureRepository) services.WatchtowerSource {
	return &ServiceFailureSource{repo: repo}
}

func (s *ServiceFailureSource) Kind() watchtower.SourceKind {
	return watchtower.SourceServiceFailure
}

func (s *ServiceFailureSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, snapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("status", string(servicefailure.StatusOpen)),
	}

	result, err := s.repo.List(ctx, &repositories.ListServiceFailuresRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if entity != nil {
			items = append(items, DescribeServiceFailure(entity))
		}
	}

	return items, nil
}

// CarrierIntelSource puts open carrier changes on the tower.
type CarrierIntelSource struct {
	repo repositories.CarrierIntelEventRepository
}

func NewCarrierIntelSource(
	repo repositories.CarrierIntelEventRepository,
) services.WatchtowerSource {
	return &CarrierIntelSource{repo: repo}
}

func (s *CarrierIntelSource) Kind() watchtower.SourceKind {
	return watchtower.SourceCarrierIntelEvent
}

func (s *CarrierIntelSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	result, err := s.repo.ListConnection(ctx, &repositories.ListCarrierIntelEventsRequest{
		Filter:   tenantQuery(tenant, snapshotLimit),
		Cursor:   pagination.CursorInfo{Limit: snapshotLimit},
		OpenOnly: true,
	})
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if !CarrierIntelEventOnTower(entity) {
			continue
		}
		items = append(items, DescribeCarrierIntelEvent(entity))
	}

	return items, nil
}

// WeatherSource puts severe and extreme alerts on the tower.
type WeatherSource struct {
	repo repositories.WeatherAlertRepository
}

func NewWeatherSource(repo repositories.WeatherAlertRepository) services.WatchtowerSource {
	return &WeatherSource{repo: repo}
}

func (s *WeatherSource) Kind() watchtower.SourceKind { return watchtower.SourceWeatherAlert }

func (s *WeatherSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	alerts, err := s.repo.GetActiveAlerts(ctx, tenant)
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(alerts))
	for _, alert := range alerts {
		if alert == nil {
			continue
		}
		if _, worthy := WeatherSeverity(alert); !worthy {
			continue
		}
		items = append(items, DescribeWeatherAlert(alert))
	}

	return items, nil
}

// EDIQuarantineSource puts held-back inbound files on the tower.
type EDIQuarantineSource struct {
	repo repositories.EDIInboundFileRepository
}

func NewEDIQuarantineSource(repo repositories.EDIInboundFileRepository) services.WatchtowerSource {
	return &EDIQuarantineSource{repo: repo}
}

func (s *EDIQuarantineSource) Kind() watchtower.SourceKind {
	return watchtower.SourceEDIInboundQuarantined
}

func (s *EDIQuarantineSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	files, err := s.repo.ListRecentQuarantined(
		ctx,
		repositories.ListRecentQuarantinedEDIInboundFilesRequest{
			TenantInfo: tenant,
			Limit:      snapshotLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(files))
	for _, file := range files {
		if file != nil {
			items = append(items, DescribeQuarantinedFile(file))
		}
	}

	return items, nil
}

// BillingExceptionSource puts items that cannot be invoiced on the tower.
type BillingExceptionSource struct {
	repo repositories.BillingQueueRepository
}

func NewBillingExceptionSource(repo repositories.BillingQueueRepository) services.WatchtowerSource {
	return &BillingExceptionSource{repo: repo}
}

func (s *BillingExceptionSource) Kind() watchtower.SourceKind {
	return watchtower.SourceBillingException
}

func (s *BillingExceptionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	filter := tenantQuery(tenant, snapshotLimit)
	filter.FieldFilters = []domaintypes.FieldFilter{
		filterEq("status", string(billingqueue.StatusException)),
	}

	result, err := s.repo.List(ctx, &repositories.ListBillingQueueItemsRequest{Filter: filter})
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(result.Items))
	for _, entity := range result.Items {
		if entity != nil {
			items = append(items, DescribeBillingException(entity))
		}
	}

	return items, nil
}

// DetentionSource puts running detention clocks on the tower.
type DetentionSource struct {
	repo repositories.DetentionOccurrenceRepository
}

func NewDetentionSource(repo repositories.DetentionOccurrenceRepository) services.WatchtowerSource {
	return &DetentionSource{repo: repo}
}

func (s *DetentionSource) Kind() watchtower.SourceKind {
	return watchtower.SourceDetentionOccurrence
}

func (s *DetentionSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	occurrences, err := s.repo.ListOpen(ctx, tenant)
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(occurrences))
	for _, entity := range occurrences {
		if entity != nil {
			items = append(items, DescribeDetentionOccurrence(entity))
		}
	}

	return items, nil
}

// InboundMessageSource puts mail that is waiting on a person on the tower.
type InboundMessageSource struct {
	repo repositories.InboundMessageRepository
}

func NewInboundMessageSource(
	repo repositories.InboundMessageRepository,
) services.WatchtowerSource {
	return &InboundMessageSource{repo: repo}
}

func (s *InboundMessageSource) Kind() watchtower.SourceKind {
	return watchtower.SourceInboundMessage
}

func (s *InboundMessageSource) Snapshot(
	ctx context.Context,
	tenant pagination.TenantInfo,
) ([]services.WatchtowerItemInput, error) {
	messages, err := s.repo.ListRecentForReview(
		ctx,
		repositories.ListRecentInboundMessagesForReviewRequest{
			TenantInfo: tenant,
			Limit:      snapshotLimit,
		},
	)
	if err != nil {
		return nil, err
	}

	items := make([]services.WatchtowerItemInput, 0, len(messages))
	for _, message := range messages {
		if message != nil {
			items = append(items, DescribeInboundMessage(message))
		}
	}

	return items, nil
}
