package development

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion"
	"github.com/emoss08/trenova/pkg/buncolgen/agentdefinitioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/aianalyticscol"
	"github.com/emoss08/trenova/pkg/buncolgen/aiprovidercol"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

const (
	SeedAIProviderBackupName = "Office vLLM"
	analyticsSeedMarker      = "seed-analytics-"

	analyticsSeedDays       = 30
	analyticsSeedQuietTail  = 20 * 60
	analyticsSeedHour       = int64(3600)
	analyticsSeedDay        = 24 * analyticsSeedHour
	analyticsSeedChunk      = 500
	analyticsSeedDecisionPc = 0.35
	analyticsSeedRatingPc   = 0.3
	analyticsSeedRandomSeed = 20261014
)

type AIAnalyticsSeed struct {
	seedhelpers.BaseSeed
}

func NewAIAnalyticsSeed() *AIAnalyticsSeed {
	seed := &AIAnalyticsSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AIAnalytics",
		"1.0.0",
		"Seeds a month of AI analytics across local, self-hosted and hosted providers",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(
		seedhelpers.SeedAIProvider,
		seedhelpers.SeedAgentDefinition,
		seedhelpers.SeedAgentActivity,
	)

	return seed
}

type analyticsRoute struct {
	task       aiprovider.Task
	runs       int
	completion float64
	agent      string
}

type analyticsErrorShare struct {
	kind  aianalytics.ProviderError
	share float64
}

type analyticsDeployment struct {
	provider     string
	model        string
	routes       []analyticsRoute
	medianMs     float64
	firstTokenMs float64
	toolCalls    float64
	toolFail     float64
	modelShare   float64
	errorRate    float64
	errorKinds   []analyticsErrorShare
	costPerRun   float64
	quietDays    int
	timeoutSecs  int
	approval     float64
	helpful      float64
}

var analyticsSeedTools = map[aiprovider.Task][]string{
	aiprovider.TaskAssistantChat: {
		"get_shipment", "search_drivers", "assign_driver", "get_hos_status", "update_load_status", "lookup_rate",
	},
	aiprovider.TaskBillingDiagnosis: {
		"release_billing_hold",
		"lookup_rate",
		"add_invoice_note",
	},
	aiprovider.TaskDocumentExtraction:     {"extract_document_fields"},
	aiprovider.TaskDocumentClassification: {"extract_document_fields"},
	aiprovider.TaskScopeClassification:    {"search_drivers"},
	aiprovider.TaskOperationalInsights:    {"run_report"},
	aiprovider.TaskDailyBriefing:          {"run_report"},
	aiprovider.TaskFormulaAssistant:       {"lookup_rate"},
}

var (
	analyticsModelCauses = []struct {
		cause aianalytics.Cause
		share float64
	}{
		{aianalytics.CauseWrongArguments, 0.72},
		{aianalytics.CauseRepeatedCall, 0.17},
		{aianalytics.CauseToolNotHeld, 0.11},
	}
	analyticsToolCauses = []struct {
		cause aianalytics.Cause
		share float64
	}{
		{aianalytics.CauseToolTimedOut, 0.42},
		{aianalytics.CauseBlockedByRule, 0.33},
		{aianalytics.CauseToolError, 0.25},
	}
)

func analyticsDeployments() []analyticsDeployment {
	return []analyticsDeployment{
		{
			provider: SeedAIProviderLocalName,
			model:    "qwen2.5-14b-instruct",
			routes: []analyticsRoute{
				{aiprovider.TaskScopeClassification, 420, 0.93, ""},
				{aiprovider.TaskDocumentClassification, 320, 0.89, ""},
				{aiprovider.TaskOperationalInsights, 220, 0.74, ""},
				{aiprovider.TaskDailyBriefing, 140, 0.81, ""},
			},
			medianMs: 5700, firstTokenMs: 410, toolCalls: 1.4, toolFail: 0.09, modelShare: 0.78,
			errorRate: 0.012,
			errorKinds: []analyticsErrorShare{
				{aianalytics.ProviderErrorCouldNotConnect, 0.4},
				{aianalytics.ProviderErrorTimedOut, 0.6},
			},
			timeoutSecs: 60, approval: 0.52, helpful: 0.63,
		},
		{
			provider: SeedAIProviderServerName,
			model:    "meta-llama/Llama-3.3-70B-Instruct",
			routes: []analyticsRoute{
				{aiprovider.TaskAssistantChat, 620, 0.72, SeedAgentDispatchName},
				{aiprovider.TaskDocumentExtraction, 420, 0.70, ""},
				{aiprovider.TaskFormulaAssistant, 300, 0.81, ""},
			},
			medianMs: 12000, firstTokenMs: 1420, toolCalls: 4, toolFail: 0.06, modelShare: 0.35,
			errorRate: 0.06,
			errorKinds: []analyticsErrorShare{
				{aianalytics.ProviderErrorTimedOut, 0.62},
				{aianalytics.ProviderErrorCouldNotConnect, 0.32},
				{aianalytics.ProviderErrorServer, 0.06},
			},
			timeoutSecs: 60, approval: 0.6, helpful: 0.71,
		},
		{
			provider: SeedAIProviderBackupName,
			model:    "meta-llama/Llama-3.3-70B-Instruct",
			routes: []analyticsRoute{
				{aiprovider.TaskAssistantChat, 320, 0.88, SeedAgentDispatchName},
				{aiprovider.TaskDocumentExtraction, 220, 0.86, ""},
			},
			medianMs: 6000, firstTokenMs: 380, toolCalls: 4, toolFail: 0.045, modelShare: 0.35,
			errorRate:   0.005,
			errorKinds:  []analyticsErrorShare{{aianalytics.ProviderErrorTimedOut, 1}},
			timeoutSecs: 60, approval: 0.62, helpful: 0.74,
		},
		{
			provider: SeedAIProviderHostedName,
			model:    "claude-sonnet-5",
			routes: []analyticsRoute{
				{aiprovider.TaskBillingDiagnosis, 180, 0.95, SeedAgentBillingName},
				{aiprovider.TaskAssistantChat, 300, 0.96, SeedAgentDispatchName},
			},
			medianMs: 13000, firstTokenMs: 690, toolCalls: 5, toolFail: 0.018, modelShare: 0.14,
			errorRate:  0.004,
			errorKinds: []analyticsErrorShare{{aianalytics.ProviderErrorRateLimited, 1}},
			costPerRun: 0.052, quietDays: 6, timeoutSecs: 120, approval: 0.84, helpful: 0.93,
		},
	}
}

type analyticsSeedRefs struct {
	org       pulid.ID
	bu        pulid.ID
	admin     pulid.ID
	now       int64
	providers map[string]*aiprovider.Provider
	agents    map[string]pulid.ID
}

type analyticsDecision struct {
	unitIndex int
	decision  agent.DecisionType
	at        int64
}

type analyticsRating struct {
	unitIndex int
	rating    aifeedback.Rating
	at        int64
}

type analyticsSeedData struct {
	units     []*aianalytics.WorkUnit
	tools     []*aianalytics.UnitTool
	hours     []*aianalytics.ProviderHour
	decisions []analyticsDecision
	ratings   []analyticsRating
}

type hourKey struct {
	hour     int64
	provider string
	model    string
	task     aiprovider.Task
	agent    string
	kind     aianalytics.ProviderError
}

type hourTally struct {
	calls      int
	fellToNext int
	firstToken aianalytics.Histogram
}

func pickShare[T any](random *rand.Rand, items []T, share func(T) float64) T {
	roll := random.Float64()
	for _, item := range items {
		roll -= share(item)
		if roll <= 0 {
			return item
		}
	}

	return items[len(items)-1]
}

func completionOn(deployment *analyticsDeployment, route analyticsRoute, daysAgo int) float64 {
	if deployment.provider == SeedAIProviderServerName && daysAgo == 0 {
		return route.completion - 0.2
	}
	if deployment.provider == SeedAIProviderServerName && daysAgo == 13 {
		return route.completion - 0.12
	}

	return route.completion
}

func errorRateOn(deployment *analyticsDeployment, daysAgo int) float64 {
	if deployment.provider == SeedAIProviderServerName && daysAgo == 0 {
		return 0.45
	}

	return deployment.errorRate
}

//nolint:funlen,gocognit
func buildAnalyticsSeed(refs *analyticsSeedRefs) *analyticsSeedData {
	random := rand.New(rand.NewPCG(analyticsSeedRandomSeed, analyticsSeedRandomSeed))
	data := new(analyticsSeedData)
	hours := make(map[hourKey]*hourTally, 4096)
	tally := func(key hourKey) *hourTally {
		item, ok := hours[key]
		if !ok {
			item = new(hourTally)
			hours[key] = item
		}
		return item
	}
	stamp := refs.now * 1000

	for _, deployment := range analyticsDeployments() {
		provider, ok := refs.providers[deployment.provider]
		if !ok {
			continue
		}
		modelKey := aianalytics.CanonicalModel(deployment.model)
		days := analyticsSeedDays - deployment.quietDays
		for _, route := range deployment.routes {
			agentID := refs.agents[route.agent]
			for index := range route.runs {
				daysAgo := deployment.quietDays + index*days/route.runs
				finishedAt := analyticsSeedEnd(refs.now) - int64(daysAgo)*analyticsSeedDay -
					analyticsSeedQuietTail -
					int64(random.Float64()*float64(analyticsSeedDay-analyticsSeedQuietTail))
				completed := math.Mod(float64(index)*0.6180339887+float64(route.runs)*0.137, 1) <
					completionOn(&deployment, route, daysAgo)
				outcome := aianalytics.OutcomeCompleted
				if !completed {
					roll := random.Float64()
					switch {
					case deployment.provider == SeedAIProviderServerName && roll < 0.15,
						deployment.provider != SeedAIProviderServerName && roll < 0.45:
						outcome = aianalytics.OutcomeHandedOff
					case deployment.provider == SeedAIProviderServerName && roll < 0.22,
						deployment.provider != SeedAIProviderServerName && roll < 0.6:
						outcome = aianalytics.OutcomeStopped
					default:
						outcome = aianalytics.OutcomeFailed
					}
				}
				duration := deployment.medianMs * math.Exp(0.5*random.NormFloat64())
				switch {
				case outcome == aianalytics.OutcomeFailed && deployment.provider == SeedAIProviderServerName:
					duration = float64(deployment.timeoutSecs)*1000 + random.Float64()*9000
				case outcome == aianalytics.OutcomeFailed:
					duration *= 1.8
				case outcome == aianalytics.OutcomeStopped:
					duration *= 4.5
				case outcome == aianalytics.OutcomeHandedOff:
					duration *= 0.8
				}
				firstToken := int64(deployment.firstTokenMs * math.Exp(0.35*random.NormFloat64()))

				unitID := pulid.MustNew("ar_").String()
				source := aianalytics.SourceCall
				if !agentID.IsNil() {
					source = aianalytics.SourceAgentRun
				}
				toolCount := int(
					math.Max(
						0,
						math.Round(
							deployment.toolCalls+random.NormFloat64()*deployment.toolCalls*0.4,
						),
					),
				)
				toolNames := analyticsSeedTools[route.task]
				calls := make(map[[2]string]int, toolCount)
				modelFailures, toolFailures := 0, 0
				for range toolCount {
					name := toolNames[random.IntN(len(toolNames))]
					cause := ""
					if random.Float64() < deployment.toolFail {
						if random.Float64() < deployment.modelShare {
							cause = string(
								pickShare(random, analyticsModelCauses, func(item struct {
									cause aianalytics.Cause
									share float64
								}) float64 {
									return item.share
								}).cause,
							)
							modelFailures++
						} else {
							cause = string(pickShare(random, analyticsToolCauses, func(item struct {
								cause aianalytics.Cause
								share float64
							}) float64 {
								return item.share
							}).cause)
							toolFailures++
						}
					}
					calls[[2]string{name, cause}]++
				}
				for key, count := range calls {
					data.tools = append(data.tools, &aianalytics.UnitTool{
						OrganizationID: refs.org,
						BusinessUnitID: refs.bu,
						UnitID:         unitID,
						ToolName:       key[0],
						Cause:          aianalytics.Cause(key[1]),
						FinishedAt:     finishedAt,
						Calls:          count,
					})
				}

				modelCalls := 1 + (toolCount+1)/2
				failedCalls := 0
				hour := finishedAt - finishedAt%analyticsSeedHour
				for range modelCalls {
					key := hourKey{
						hour:     hour,
						provider: provider.ID.String(),
						model:    deployment.model,
						task:     route.task,
						agent:    agentID.String(),
					}
					if random.Float64() < errorRateOn(&deployment, daysAgo) {
						failedCalls++
						key.kind = pickShare(
							random,
							deployment.errorKinds,
							func(item analyticsErrorShare) float64 { return item.share },
						).kind
						entry := tally(key)
						entry.calls++
						if outcome != aianalytics.OutcomeFailed && random.Float64() < 0.85 {
							entry.fellToNext++
						}
						continue
					}
					entry := tally(key)
					entry.calls++
					entry.firstToken.Observe(
						int64(deployment.firstTokenMs * math.Exp(0.35*random.NormFloat64())),
					)
				}
				if deployment.quietDays > 0 && daysAgo == deployment.quietDays && index%3 == 0 {
					entry := tally(hourKey{
						hour:     hour,
						provider: provider.ID.String(),
						model:    deployment.model,
						task:     route.task,
						agent:    agentID.String(),
						kind:     aianalytics.ProviderErrorAccessDenied,
					})
					entry.calls += 2
					entry.fellToNext += 2
				}

				unit := &aianalytics.WorkUnit{
					ID:                unitID,
					OrganizationID:    refs.org,
					BusinessUnitID:    refs.bu,
					Source:            source,
					AgentDefinitionID: agentID,
					ProviderID:        provider.ID,
					Model:             deployment.model,
					ModelKey:          modelKey,
					Task:              route.task,
					Outcome:           outcome,
					StartedAtMs:       finishedAt*1000 - int64(duration),
					FinishedAt:        finishedAt,
					DurationMs:        int64(duration),
					FirstTokenMs:      &firstToken,
					ModelCalls:        modelCalls,
					FailedCalls:       failedCalls,
					ToolCalls:         toolCount,
					ToolFailuresModel: modelFailures,
					ToolFailuresTool:  toolFailures,
					RefreshStamp:      stamp,
				}
				if deployment.costPerRun > 0 {
					cost := decimal.NewFromFloat(deployment.costPerRun * (0.6 + random.Float64()*0.8)).
						Round(6)
					unit.CostUSD = &cost
				}
				if source == aianalytics.SourceAgentRun {
					unit.DecisionRunID = pulid.ID(unitID)
					if random.Float64() < analyticsSeedDecisionPc {
						roll := random.Float64()
						decision := agent.DecisionRejected
						switch {
						case roll < deployment.approval:
							decision = agent.DecisionAccepted
						case roll < deployment.approval+(1-deployment.approval)*0.55:
							decision = agent.DecisionModified
						}
						data.decisions = append(data.decisions, analyticsDecision{
							unitIndex: len(data.units),
							decision:  decision,
							at:        finishedAt + 600,
						})
					}
					if route.task == aiprovider.TaskAssistantChat &&
						random.Float64() < analyticsSeedRatingPc {
						rating := aifeedback.RatingNegative
						if random.Float64() < deployment.helpful {
							rating = aifeedback.RatingPositive
						}
						data.ratings = append(data.ratings, analyticsRating{
							unitIndex: len(data.units),
							rating:    rating,
							at:        finishedAt + 300,
						})
					}
				}
				data.units = append(data.units, unit)
			}
		}
	}

	for key, item := range hours {
		data.hours = append(data.hours, &aianalytics.ProviderHour{
			OrganizationID:    refs.org,
			BusinessUnitID:    refs.bu,
			Hour:              key.hour,
			ProviderID:        key.provider,
			Model:             key.model,
			ModelKey:          aianalytics.CanonicalModel(key.model),
			Task:              key.task,
			AgentDefinitionID: key.agent,
			ErrorClass:        key.kind,
			Calls:             item.calls,
			FellToNext:        min(item.fellToNext, item.calls),
			FirstTokenHist:    item.firstToken.Buckets(),
		})
	}
	slices.SortFunc(data.hours, func(left, right *aianalytics.ProviderHour) int {
		switch {
		case left.Hour != right.Hour:
			return int(left.Hour - right.Hour)
		case left.ProviderID != right.ProviderID:
			if left.ProviderID < right.ProviderID {
				return -1
			}
			return 1
		default:
			return 0
		}
	})

	return data
}

func (s *AIAnalyticsSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			admin, err := sc.GetUserByUsername(ctx, "admin")
			if err != nil {
				return fmt.Errorf("get admin user: %w", err)
			}

			stateCols := aianalyticscol.RefreshStateColumns
			exists, err := tx.NewSelect().
				Model((*aianalytics.RefreshState)(nil)).
				Where(stateCols.OrganizationID.Eq(), org.ID).
				Where(stateCols.BusinessUnitID.Eq(), org.BusinessUnitID).
				Exists(ctx)
			if err != nil {
				return fmt.Errorf("check ai analytics refresh state: %w", err)
			}
			if exists {
				return nil
			}

			refs := &analyticsSeedRefs{
				org:   org.ID,
				bu:    org.BusinessUnitID,
				admin: admin.ID,
				now:   timeutils.NowUnix(),
			}
			if err = s.ensureBackupProvider(ctx, tx, sc, refs); err != nil {
				return err
			}
			if err = s.loadRefs(ctx, tx, refs); err != nil {
				return err
			}

			return s.write(ctx, tx, sc, refs, buildAnalyticsSeed(refs))
		},
	)
}

func (s *AIAnalyticsSeed) ensureBackupProvider(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *analyticsSeedRefs,
) error {
	cols := aiprovidercol.ProviderColumns
	exists, err := tx.NewSelect().
		Model((*aiprovider.Provider)(nil)).
		Where(cols.OrganizationID.Eq(), refs.org).
		Where(cols.BusinessUnitID.Eq(), refs.bu).
		Where(cols.Name.Eq(), SeedAIProviderBackupName).
		Exists(ctx)
	if err != nil {
		return fmt.Errorf("check backup provider: %w", err)
	}
	if exists {
		return nil
	}

	provider := &aiprovider.Provider{
		OrganizationID: refs.org,
		BusinessUnitID: refs.bu,
		Name:           SeedAIProviderBackupName,
		Description: "A second OpenAI-compatible server on the office network serving the same " +
			"Llama model, kept as a fallback for the workstation.",
		Kind:                 aiprovider.KindOpenAIChat,
		BaseURL:              "http://localhost:8001/v1",
		Model:                "meta-llama/Llama-3.3-70B-Instruct",
		AllowPrivateNetwork:  true,
		StructuredOutputMode: aiprovider.StructuredOutputJSONMode,
		ReasoningEffort:      aiprovider.ReasoningOff,
		MaxTokens:            16384,
		Tasks: []aiprovider.Task{
			aiprovider.TaskAssistantChat,
			aiprovider.TaskDocumentExtraction,
			aiprovider.TaskGeneral,
		},
		Priority: 30,
		Enabled:  true,
	}
	if _, err = tx.NewInsert().Model(provider).Exec(ctx); err != nil {
		return fmt.Errorf("insert backup provider: %w", err)
	}

	return sc.TrackCreated(ctx, "ai_providers", provider.ID, s.Name())
}

func (s *AIAnalyticsSeed) loadRefs(ctx context.Context, tx bun.Tx, refs *analyticsSeedRefs) error {
	providerCols := aiprovidercol.ProviderColumns
	providers := make([]*aiprovider.Provider, 0, 4)
	if err := tx.NewSelect().
		Model(&providers).
		Where(providerCols.OrganizationID.Eq(), refs.org).
		Where(providerCols.BusinessUnitID.Eq(), refs.bu).
		Scan(ctx); err != nil {
		return fmt.Errorf("load providers: %w", err)
	}
	refs.providers = make(map[string]*aiprovider.Provider, len(providers))
	for _, provider := range providers {
		refs.providers[provider.Name] = provider
	}

	defCols := agentdefinitioncol.DefinitionColumns
	definitions := make([]*agentdefinition.Definition, 0, 2)
	if err := tx.NewSelect().
		Model(&definitions).
		Where(defCols.OrganizationID.Eq(), refs.org).
		Where(defCols.BusinessUnitID.Eq(), refs.bu).
		Where(defCols.Name.In(), bun.In([]string{SeedAgentDispatchName, SeedAgentBillingName})).
		Scan(ctx); err != nil {
		return fmt.Errorf("load agents: %w", err)
	}
	refs.agents = make(map[string]pulid.ID, len(definitions))
	for _, definition := range definitions {
		refs.agents[definition.Name] = definition.ID
	}

	return nil
}

func insertChunks[T any](ctx context.Context, tx bun.Tx, what string, rows []T) error {
	for start := 0; start < len(rows); start += analyticsSeedChunk {
		chunk := rows[start:min(start+analyticsSeedChunk, len(rows))]
		if _, err := tx.NewInsert().Model(&chunk).Exec(ctx); err != nil {
			return fmt.Errorf("insert %s: %w", what, err)
		}
	}

	return nil
}

func backdate(
	ctx context.Context,
	tx bun.Tx,
	refs *analyticsSeedRefs,
	table string,
	ids []string,
	ats []int64,
) error {
	_, err := tx.NewRaw(`UPDATE ? AS t SET created_at = v.at, updated_at = v.at
		FROM (SELECT unnest(?::varchar[]) AS id, unnest(?::bigint[]) AS at) AS v
		WHERE t.id = v.id AND t.organization_id = ? AND t.business_unit_id = ?`,
		bun.Ident(table), pgdialect.Array(ids), pgdialect.Array(ats), refs.org, refs.bu,
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("backdate %s: %w", table, err)
	}

	return nil
}

//nolint:funlen
func (s *AIAnalyticsSeed) write(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *analyticsSeedRefs,
	data *analyticsSeedData,
) error {
	runs := make([]*agent.AgentRun, 0, len(data.decisions)+len(data.ratings))
	runIDs := make([]string, 0, cap(runs))
	runAts := make([]int64, 0, cap(runs))
	seen := make(map[int]struct{}, cap(runs))
	addRun := func(index int) {
		if _, ok := seen[index]; ok {
			return
		}
		seen[index] = struct{}{}
		unit := data.units[index]
		completed := unit.FinishedAt
		status := agent.RunStatusCompleted
		if unit.Outcome == aianalytics.OutcomeFailed {
			status = agent.RunStatusFailed
		}
		agentType := agent.TypeAssistantChat
		if unit.Task == aiprovider.TaskBillingDiagnosis {
			agentType = agent.TypeBillingException
		}
		runs = append(runs, &agent.AgentRun{
			ID:                pulid.ID(unit.ID),
			OrganizationID:    refs.org,
			BusinessUnitID:    refs.bu,
			AgentDefinitionID: unit.AgentDefinitionID,
			AgentType:         agentType,
			Trigger:           agent.RunTriggerManual,
			SubjectType:       agent.SubjectOrganization,
			SubjectID:         refs.org,
			Status:            status,
			Summary:           "Analytics seed run",
			ModelIdentifier:   unit.Model,
			PromptVersion:     "seed/v1",
			InputContextHash:  analyticsSeedMarker + unit.ID,
			StartedAt:         unit.StartedAtMs / 1000,
			CompletedAt:       &completed,
		})
		runIDs = append(runIDs, unit.ID)
		runAts = append(runAts, unit.FinishedAt)
	}
	for _, decision := range data.decisions {
		addRun(decision.unitIndex)
	}
	for _, rating := range data.ratings {
		addRun(rating.unitIndex)
	}

	if err := insertChunks(ctx, tx, "analytics runs", runs); err != nil {
		return err
	}
	if err := backdate(ctx, tx, refs, "agent_runs", runIDs, runAts); err != nil {
		return err
	}

	proposals := make([]*agent.AgentProposal, 0, len(data.decisions))
	decisions := make([]*agent.AgentDecision, 0, len(data.decisions))
	proposalIDs := make([]string, 0, len(data.decisions))
	decisionIDs := make([]string, 0, len(data.decisions))
	decidedAts := make([]int64, 0, len(data.decisions))
	for _, item := range data.decisions {
		unit := data.units[item.unitIndex]
		proposalID := pulid.MustNew("ap_")
		status := agent.ProposalStatusExecuted
		if item.decision == agent.DecisionRejected {
			status = agent.ProposalStatusRejected
		}
		proposals = append(proposals, &agent.AgentProposal{
			ID:             proposalID,
			OrganizationID: refs.org,
			BusinessUnitID: refs.bu,
			RunID:          pulid.ID(unit.ID),
			ToolName:       "add_shipment_comment",
			ToolParams:     map[string]any{"comment": "Seeded proposal"},
			Confidence:     decimal.NewFromFloat(0.8),
			Rationale:      "Seeded so analytics has decisions to read.",
			AutonomyTier:   agent.TierPropose,
			Status:         status,
		})
		decision := &agent.AgentDecision{
			ID:              pulid.MustNew("ad_"),
			OrganizationID:  refs.org,
			BusinessUnitID:  refs.bu,
			ProposalID:      &proposalID,
			DecidedByUserID: refs.admin,
			Decision:        item.decision,
			ReasonCode:      "Seeded",
		}
		if item.decision == agent.DecisionModified {
			decision.Modifications = map[string]any{"comment": "Seeded proposal, edited"}
		}
		decisions = append(decisions, decision)
		proposalIDs = append(proposalIDs, proposalID.String())
		decisionIDs = append(decisionIDs, decision.ID.String())
		decidedAts = append(decidedAts, item.at)
	}
	if err := insertChunks(ctx, tx, "analytics proposals", proposals); err != nil {
		return err
	}
	if err := insertChunks(ctx, tx, "analytics decisions", decisions); err != nil {
		return err
	}
	if err := backdate(ctx, tx, refs, "agent_proposals", proposalIDs, decidedAts); err != nil {
		return err
	}
	if err := backdate(ctx, tx, refs, "agent_decisions", decisionIDs, decidedAts); err != nil {
		return err
	}

	feedback := make([]*aifeedback.Feedback, 0, len(data.ratings))
	for _, item := range data.ratings {
		unit := data.units[item.unitIndex]
		runID := pulid.ID(unit.ID)
		feedback = append(feedback, &aifeedback.Feedback{
			OrganizationID:    refs.org,
			BusinessUnitID:    refs.bu,
			UserID:            refs.admin,
			TargetType:        aifeedback.TargetAssistantMessage,
			TargetID:          runID,
			RunID:             &runID,
			Task:              string(unit.Task),
			Model:             unit.Model,
			ProviderID:        &unit.ProviderID,
			FingerprintSource: aifeedback.FingerprintNone,
			Rating:            item.rating,
			Reasons:           []aifeedback.Reason{},
			PatternKey:        aifeedback.PatternKey(nil, "", "AgentRun"),
			CreatedAt:         item.at,
		})
	}
	if err := insertChunks(ctx, tx, "analytics ratings", feedback); err != nil {
		return err
	}

	if err := insertChunks(ctx, tx, "ai work units", data.units); err != nil {
		return err
	}
	if err := insertChunks(ctx, tx, "ai work unit tools", data.tools); err != nil {
		return err
	}
	if err := insertChunks(ctx, tx, "ai provider hours", data.hours); err != nil {
		return err
	}
	if err := s.writeProviderHistory(ctx, tx, sc, refs); err != nil {
		return err
	}

	state := &aianalytics.RefreshState{
		OrganizationID:   refs.org,
		BusinessUnitID:   refs.bu,
		RefreshedThrough: refs.now,
		RefreshedAt:      refs.now,
	}
	if _, err := tx.NewInsert().Model(state).Exec(ctx); err != nil {
		return fmt.Errorf("insert ai analytics refresh state: %w", err)
	}

	return nil
}

func (s *AIAnalyticsSeed) writeProviderHistory(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *analyticsSeedRefs,
) error {
	hosted, ok := refs.providers[SeedAIProviderHostedName]
	if !ok {
		return nil
	}
	snapshot := func(enabled bool) map[string]any {
		return map[string]any{"name": hosted.Name, "model": hosted.Model, "enabled": enabled}
	}
	versions := []*settingversion.SettingVersion{
		{
			OrganizationID: refs.org,
			BusinessUnitID: refs.bu,
			Kind:           settingversion.KindAIProvider,
			SubjectID:      hosted.ID,
			Version:        1,
			Snapshot:       snapshot(true),
			AuthorID:       &refs.admin,
		},
		{
			OrganizationID: refs.org,
			BusinessUnitID: refs.bu,
			Kind:           settingversion.KindAIProvider,
			SubjectID:      hosted.ID,
			Version:        2,
			Snapshot:       snapshot(false),
			AuthorID:       &refs.admin,
		},
	}
	if _, err := tx.NewInsert().Model(&versions).Exec(ctx); err != nil {
		return fmt.Errorf("insert provider history: %w", err)
	}
	for _, version := range versions {
		if err := sc.TrackCreated(ctx, "ai_setting_versions", version.ID, s.Name()); err != nil {
			return err
		}
	}
	quietDays := int64(analyticsDeployments()[3].quietDays)
	_, err := tx.NewRaw(`UPDATE ai_setting_versions AS v SET created_at = x.at
		FROM (SELECT unnest(?::varchar[]) AS id, unnest(?::bigint[]) AS at) AS x
		WHERE v.id = x.id AND v.organization_id = ? AND v.business_unit_id = ?`,
		pgdialect.Array([]string{versions[0].ID.String(), versions[1].ID.String()}),
		pgdialect.Array([]int64{
			refs.now - 45*analyticsSeedDay,
			refs.now - quietDays*analyticsSeedDay + 3*analyticsSeedHour,
		}),
		refs.org, refs.bu,
	).Exec(ctx)
	if err != nil {
		return fmt.Errorf("date provider history: %w", err)
	}

	return nil
}

var analyticsSeedTeardown = []string{
	`DELETE FROM ai_feedback AS f
	WHERE f.organization_id = ?org AND f.business_unit_id = ?bu
		AND f.run_id IN (SELECT r.id FROM agent_runs AS r
			WHERE r.organization_id = ?org AND r.business_unit_id = ?bu AND r.input_context_hash LIKE ?marker)`,
	`DELETE FROM agent_decisions AS d
	WHERE d.organization_id = ?org AND d.business_unit_id = ?bu
		AND d.proposal_id IN (SELECT p.id FROM agent_proposals AS p
			JOIN agent_runs AS r ON r.id = p.run_id
				AND r.organization_id = p.organization_id AND r.business_unit_id = p.business_unit_id
			WHERE p.organization_id = ?org AND p.business_unit_id = ?bu AND r.input_context_hash LIKE ?marker)`,
	`DELETE FROM agent_proposals AS p
	WHERE p.organization_id = ?org AND p.business_unit_id = ?bu
		AND p.run_id IN (SELECT r.id FROM agent_runs AS r
			WHERE r.organization_id = ?org AND r.business_unit_id = ?bu AND r.input_context_hash LIKE ?marker)`,
	`DELETE FROM agent_runs
	WHERE organization_id = ?org AND business_unit_id = ?bu AND input_context_hash LIKE ?marker`,
	`DELETE FROM ai_work_unit_tools WHERE organization_id = ?org AND business_unit_id = ?bu`,
	`DELETE FROM ai_work_units WHERE organization_id = ?org AND business_unit_id = ?bu`,
	`DELETE FROM ai_provider_hourly_stats WHERE organization_id = ?org AND business_unit_id = ?bu`,
	`DELETE FROM ai_analytics_refresh_state WHERE organization_id = ?org AND business_unit_id = ?bu`,
}

type analyticsTeardownArgs struct {
	Org    string `bun:"org"`
	Bu     string `bun:"bu"`
	Marker string `bun:"marker"`
}

func (s *AIAnalyticsSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			args := &analyticsTeardownArgs{
				Org:    org.ID.String(),
				Bu:     org.BusinessUnitID.String(),
				Marker: analyticsSeedMarker + "%",
			}
			for _, statement := range analyticsSeedTeardown {
				if _, err = tx.NewRaw(statement, args).Exec(ctx); err != nil {
					return fmt.Errorf("remove ai analytics seed: %w", err)
				}
			}

			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *AIAnalyticsSeed) CanRollback() bool {
	return true
}

func analyticsSeedEnd(now int64) int64 {
	return now - now%analyticsSeedHour - analyticsSeedHour
}
