package fixtures

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aianalytics/aianalyticsdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aifeedback/aifeedbackdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider/aiproviderdb"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/settingversion/settingversiondb"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
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
}

var AIAnalytics = Seeds.Add(seed.Seed{
	Name:        "ai_analytics",
	Kind:        seed.Fixture,
	Description: "Seeds a month of AI analytics across local, self-hosted and hosted providers",
	After:       []*seed.Seed{AIProvider, AgentDefinition, AgentActivity},
	Run:         (&AIAnalyticsSeed{}).Run,
})

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
		"shipment_get", "search_drivers", "assign_driver", "get_hos_status", "update_load_status", "lookup_rate",
	},
	aiprovider.TaskBillingDiagnosis: {
		"release_billing_hold",
		"lookup_rate",
		"add_invoice_note",
	},
	aiprovider.TaskDocumentExtraction:     {"extract_document_fields"},
	aiprovider.TaskDocumentClassification: {"extract_document_fields"},
	aiprovider.TaskScopeClassification:    {"search_drivers"},
	aiprovider.TaskOperationalInsights:    {"report_run"},
	aiprovider.TaskDailyBriefing:          {"report_run"},
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
	decision  agentdecision.DecisionType
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
						decision := agentdecision.DecisionRejected
						switch {
						case roll < deployment.approval:
							decision = agentdecision.DecisionAccepted
						case roll < deployment.approval+(1-deployment.approval)*0.55:
							decision = agentdecision.DecisionModified
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

func (s *AIAnalyticsSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	org, err := defaultOrganization(ctx, sr)
	if err != nil {
		return err
	}
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return fmt.Errorf("get admin user: %w", err)
	}

	refs := &analyticsSeedRefs{
		org:   org.ID,
		bu:    org.BusinessUnitID,
		admin: admin.ID,
		now:   timeutils.NowUnix(),
	}
	if err = s.ensureBackupProvider(ctx, db, sr, refs); err != nil {
		return err
	}
	if err = s.loadRefs(ctx, db, refs); err != nil {
		return err
	}

	return s.write(ctx, db, sr, refs, buildAnalyticsSeed(refs))
}

func (s *AIAnalyticsSeed) ensureBackupProvider(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	refs *analyticsSeedRefs,
) error {
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
		Priority:            30,
		Enabled:             true,
		Scripts:             true,
		ThinkingStyle:       aiprovider.ThinkingStyleAuto,
		Platform:            aiprovider.PlatformDirect,
		EmbeddingInputStyle: aiprovider.EmbeddingInputStyleNone,
		TimeoutSeconds:      60,
		MaxConcurrent:       8,
		OnCap:               aiprovider.CapActionNext,
	}
	if err := aiproviderdb.Providers.Create(ctx, db, provider); err != nil {
		return fmt.Errorf("insert backup provider: %w", err)
	}

	return nil
}

func (s *AIAnalyticsSeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	refs *analyticsSeedRefs,
) error {
	providers, err := aiproviderdb.Providers.AllTenants().Where(
		aiproviderdb.Providers.OrganizationID.Eq(refs.org),
		aiproviderdb.Providers.BusinessUnitID.Eq(refs.bu),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load providers: %w", err)
	}
	refs.providers = make(map[string]*aiprovider.Provider, len(providers))
	for _, provider := range providers {
		refs.providers[provider.Name] = provider
	}

	d := agentdefinitiondb.Definitions
	definitions, err := d.AllTenants().Where(
		d.OrganizationID.Eq(refs.org),
		d.BusinessUnitID.Eq(refs.bu),
		d.Name.In(SeedAgentDispatchName, SeedAgentBillingName),
	).All(ctx, db)
	if err != nil {
		return fmt.Errorf("load agents: %w", err)
	}
	refs.agents = make(map[string]pulid.ID, len(definitions))
	for _, definition := range definitions {
		refs.agents[definition.Name] = definition.ID
	}

	return nil
}

func insertChunks[T any](
	ctx context.Context,
	db *keel.DB,
	what string,
	h keel.Handle[*T],
	rows []*T,
) error {
	if _, err := h.Copy(ctx, db, rows); err != nil {
		return fmt.Errorf("insert %s: %w", what, err)
	}

	return nil
}

//nolint:funlen
func (s *AIAnalyticsSeed) write(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
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

	if err := insertChunks(ctx, db, "analytics runs", agentdb.AgentRuns.Handle, runs); err != nil {
		return err
	}
	if err := backdate(ctx, db, agentdb.AgentRuns.Handle, runIDs, runAts); err != nil {
		return err
	}

	proposals := make([]*agent.AgentProposal, 0, len(data.decisions))
	decisions := make([]*agentdecision.AgentDecision, 0, len(data.decisions))
	proposalIDs := make([]string, 0, len(data.decisions))
	decisionIDs := make([]string, 0, len(data.decisions))
	decidedAts := make([]int64, 0, len(data.decisions))
	for _, item := range data.decisions {
		unit := data.units[item.unitIndex]
		proposalID := pulid.MustNew("ap_")
		status := agent.ProposalStatusExecuted
		if item.decision == agentdecision.DecisionRejected {
			status = agent.ProposalStatusRejected
		}
		proposals = append(proposals, &agent.AgentProposal{
			ID:             proposalID,
			OrganizationID: refs.org,
			BusinessUnitID: refs.bu,
			RunID:          pulid.ID(unit.ID),
			ToolName:       "shipment_add_comment",
			ToolParams:     map[string]any{"comment": "Seeded proposal"},
			Confidence:     decimal.NewFromFloat(0.8),
			Rationale:      "Seeded so analytics has decisions to read.",
			AutonomyTier:   agent.TierPropose,
			Status:         status,
			HeldBy:         []string{},
		})
		decision := &agentdecision.AgentDecision{
			ID:              pulid.MustNew("ad_"),
			OrganizationID:  refs.org,
			BusinessUnitID:  refs.bu,
			ProposalID:      &proposalID,
			DecidedByUserID: refs.admin,
			Decision:        item.decision,
			ReasonCode:      "Seeded",
		}
		if item.decision == agentdecision.DecisionModified {
			decision.Modifications = map[string]any{"comment": "Seeded proposal, edited"}
		}
		decisions = append(decisions, decision)
		proposalIDs = append(proposalIDs, proposalID.String())
		decisionIDs = append(decisionIDs, decision.ID.String())
		decidedAts = append(decidedAts, item.at)
	}
	if err := insertChunks(ctx, db, "analytics proposals", agentdb.AgentProposals.Handle, proposals); err != nil {
		return err
	}
	if err := insertChunks(ctx, db, "analytics decisions", agentdb.AgentDecisions.Handle, decisions); err != nil {
		return err
	}
	if err := backdate(ctx, db, agentdb.AgentProposals.Handle, proposalIDs, decidedAts); err != nil {
		return err
	}
	if err := backdate(ctx, db, agentdb.AgentDecisions.Handle, decisionIDs, decidedAts); err != nil {
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
	ratedAts := make([]int64, 0, len(feedback))
	for _, row := range feedback {
		ratedAts = append(ratedAts, row.CreatedAt)
	}
	if err := insertChunks(ctx, db, "analytics ratings", aifeedbackdb.Feedbacks.Handle, feedback); err != nil {
		return err
	}
	feedbackIDs := make([]string, 0, len(feedback))
	for _, row := range feedback {
		feedbackIDs = append(feedbackIDs, row.ID.String())
	}
	if err := backdate(ctx, db, aifeedbackdb.Feedbacks.Handle, feedbackIDs, ratedAts); err != nil {
		return err
	}

	if err := insertChunks(ctx, db, "ai work units", aianalyticsdb.WorkUnits.Handle, data.units); err != nil {
		return err
	}
	if err := insertChunks(ctx, db, "ai work unit tools", aianalyticsdb.UnitTools.Handle, data.tools); err != nil {
		return err
	}
	if err := insertChunks(ctx, db, "ai provider hours", aianalyticsdb.ProviderHours.Handle, data.hours); err != nil {
		return err
	}
	if err := s.writeProviderHistory(ctx, db, sr, refs); err != nil {
		return err
	}

	state := &aianalytics.RefreshState{
		OrganizationID:   refs.org,
		BusinessUnitID:   refs.bu,
		RefreshedThrough: refs.now,
		RefreshedAt:      refs.now,
	}
	if err := aianalyticsdb.RefreshStates.Create(ctx, db, state); err != nil {
		return fmt.Errorf("insert ai analytics refresh state: %w", err)
	}

	return nil
}

func (s *AIAnalyticsSeed) writeProviderHistory(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
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
	if err := settingversiondb.SettingVersions.CreateMany(ctx, db, versions); err != nil {
		return fmt.Errorf("insert provider history: %w", err)
	}
	quietDays := int64(analyticsDeployments()[3].quietDays)
	if err := backdate(
		ctx,
		db,
		settingversiondb.SettingVersions.Handle,
		[]string{versions[0].ID.String(), versions[1].ID.String()},
		[]int64{
			refs.now - 45*analyticsSeedDay,
			refs.now - quietDays*analyticsSeedDay + 3*analyticsSeedHour,
		},
	); err != nil {
		return fmt.Errorf("date provider history: %w", err)
	}

	return nil
}

func analyticsSeedEnd(now int64) int64 {
	return now - now%analyticsSeedHour - analyticsSeedHour
}
