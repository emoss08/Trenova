package agentqualityengine

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualitysettings"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentfingerprint"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentreplay"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aitask"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

var errSweepKeyUnclaimed = errors.New("no suite run holds the sweep key")

const staleRunningAfter = int64(24 * 60 * 60)

type SweepRequest struct {
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
	AgentDefinitionID pulid.ID              `json:"agentDefinitionId,omitzero"`
	Forced            bool                  `json:"forced"`
}

type PlannedAgent struct {
	AgentDefinitionID pulid.ID                      `json:"agentDefinitionId"`
	Name              string                        `json:"name"`
	Fingerprint       *agentfingerprint.Fingerprint `json:"fingerprint"`
	FingerprintHash   string                        `json:"fingerprintHash"`
	SuiteRevision     string                        `json:"suiteRevision"`
	ActiveCases       int                           `json:"activeCases"`
	Skip              bool                          `json:"skip"`
	SkipReason        string                        `json:"skipReason,omitempty"`
}

type SweepPlan struct {
	Enabled    bool                              `json:"enabled"`
	Reason     string                            `json:"reason,omitempty"`
	Timezone   string                            `json:"timezone"`
	PlannedAt  int64                             `json:"plannedAt"`
	DayStart   int64                             `json:"dayStart"`
	MonthStart int64                             `json:"monthStart"`
	Settings   agentqualitycontrol.SuiteSettings `json:"settings"`
	Agents     []PlannedAgent                    `json:"agents"`
}

func (s *Engine) PlanSweep(ctx context.Context, req *SweepRequest) (*SweepPlan, error) {
	control, err := s.settings.Control(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()
	timezone := s.settings.Timezone(ctx, control, req.TenantInfo)
	dayStart, monthStart, err := agentqualitysettings.Windows(now, timezone)
	if err != nil {
		return nil, err
	}

	plan := &SweepPlan{
		Enabled:    control.Enabled || req.Forced,
		Timezone:   timezone,
		PlannedAt:  now,
		DayStart:   dayStart,
		MonthStart: monthStart,
		Settings:   control.Settings(),
		Agents:     []PlannedAgent{},
	}
	if !plan.Enabled {
		plan.Reason = "The nightly quality sweep is turned off."

		return plan, nil
	}

	definitions, err := s.agentsToPlan(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(definitions) == 0 {
		return plan, nil
	}

	provider, err := s.chatProvider(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	for _, definition := range definitions {
		planned, planErr := s.planAgent(ctx, &planAgentParams{
			tenant:     req.TenantInfo,
			definition: definition,
			provider:   provider,
			control:    control,
			now:        now,
			forced:     req.Forced,
		})
		if planErr != nil {
			return nil, planErr
		}
		plan.Agents = append(plan.Agents, *planned)
	}

	return plan, nil
}

func (s *Engine) agentsToPlan(
	ctx context.Context,
	req *SweepRequest,
) ([]*agentdefinition.Definition, error) {
	ids := []pulid.ID{req.AgentDefinitionID}
	if req.AgentDefinitionID.IsNil() {
		withCases, err := s.cases.ListAgentsWithActiveCases(
			ctx,
			agentqualityports.ListAgentsWithActiveEvalCasesRequest{TenantInfo: req.TenantInfo},
		)
		if err != nil {
			return nil, err
		}
		ids = withCases
	}
	if len(ids) == 0 {
		return nil, nil
	}

	definitions, err := s.definitions.ListByIDs(
		ctx,
		agentdefinitionports.ListAgentDefinitionsByIDsRequest{
			IDs:        ids,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("read the agents to plan: %w", err)
	}

	planned := make([]*agentdefinition.Definition, 0, len(definitions))
	for _, definition := range definitions {
		if definition == nil {
			continue
		}
		if !definition.Enabled && !req.Forced {
			continue
		}
		planned = append(planned, definition)
	}
	slices.SortFunc(planned, func(a, b *agentdefinition.Definition) int {
		return compareIDs(a.ID, b.ID)
	})

	return planned, nil
}

func compareIDs(a, b pulid.ID) int {
	switch {
	case a.String() < b.String():
		return -1
	case a.String() > b.String():
		return 1
	default:
		return 0
	}
}

type resolvedProvider struct {
	candidates []*aiprovider.Provider
}

func (r resolvedProvider) forAgent(preferred pulid.ID) (providerID pulid.ID, model string) {
	if len(r.candidates) == 0 {
		return preferred, ""
	}
	for _, candidate := range r.candidates {
		if candidate.ID == preferred {
			return candidate.ID, candidate.Model
		}
	}

	return r.candidates[0].ID, r.candidates[0].Model
}

func (s *Engine) chatProvider(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (resolvedProvider, error) {
	if s.providers == nil {
		return resolvedProvider{}, nil
	}

	candidates, err := s.providers.ListForTask(ctx, aiproviderports.ListAIProvidersForTaskRequest{
		Task:       aitask.TaskAssistantChat,
		TenantInfo: tenant,
	})
	if err != nil {
		return resolvedProvider{}, fmt.Errorf("read the providers that serve chat: %w", err)
	}

	usable := make([]*aiprovider.Provider, 0, len(candidates))
	for _, candidate := range candidates {
		if ok, _ := candidate.CanServeTask(aitask.TaskAssistantChat); ok {
			usable = append(usable, candidate)
		}
	}

	return resolvedProvider{candidates: usable}, nil
}

type planAgentParams struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	provider   resolvedProvider
	control    *agentqualitycontrol.Control
	now        int64
	forced     bool
}

func (s *Engine) fingerprint(
	definition *agentdefinition.Definition,
	provider resolvedProvider,
) *agentfingerprint.Fingerprint {
	providerID, model := provider.forAgent(definition.PreferredProviderID)
	if s.runtime == nil {
		return agentquality.FingerprintOf(definition, "").Served(model, providerID)
	}

	return s.runtime.Fingerprint(definition, providerID, model)
}

func (s *Engine) planAgent(ctx context.Context, p *planAgentParams) (*PlannedAgent, error) {
	fingerprint := s.fingerprint(p.definition, p.provider)
	cases, err := s.cases.ListSamplingCases(ctx, agentqualityports.ListEvalCaseSamplingRequest{
		TenantInfo:        p.tenant,
		AgentDefinitionID: p.definition.ID,
	})
	if err != nil {
		return nil, err
	}

	planned := &PlannedAgent{
		AgentDefinitionID: p.definition.ID,
		Name:              p.definition.Name,
		Fingerprint:       fingerprint,
		FingerprintHash:   fingerprint.Hash(),
		SuiteRevision:     agentqualitysuite.SuiteRevision(cases),
		ActiveCases:       len(cases),
	}
	if len(cases) == 0 {
		planned.Skip = true
		planned.SkipReason = "The agent has no active evaluation cases."

		return planned, nil
	}

	last, err := s.suiteRuns.Last(ctx, agentqualityports.LastAgentSuiteRunRequest{
		AgentDefinitionID: p.definition.ID,
		TenantInfo:        p.tenant,
		Statuses: []agentqualitysuite.SuiteRunStatus{
			agentqualitysuite.SuiteRunStatusCompleted,
		},
	})
	if err != nil {
		return nil, err
	}

	var lastRun *agentqualitysuite.LastRun
	if last != nil && last.FinishedAt != nil {
		lastRun = &agentqualitysuite.LastRun{
			FingerprintHash: last.FingerprintHash,
			SuiteRevision:   last.SuiteRevision,
			FinishedAt:      *last.FinishedAt,
		}
	}

	decision := agentqualitysuite.DecideSkip(agentqualitysuite.SkipInput{
		FingerprintHash: planned.FingerprintHash,
		SuiteRevision:   planned.SuiteRevision,
		Last:            lastRun,
		Now:             p.now,
		ForceRerunDays:  p.control.ForceRerunDays,
		Forced:          p.forced,
	})
	planned.Skip = decision.Skip
	planned.SkipReason = decision.Reason

	return planned, nil
}

type OpenSuiteRequest struct {
	TenantInfo        pagination.TenantInfo             `json:"tenantInfo"`
	Agent             PlannedAgent                      `json:"agent"`
	SweepKey          string                            `json:"sweepKey,omitempty"`
	Trigger           agentqualitysuite.SuiteRunTrigger `json:"trigger"`
	RequestedByUserID pulid.ID                          `json:"requestedByUserId,omitzero"`
	Settings          agentqualitycontrol.SuiteSettings `json:"settings"`
}

type OpenedSuite struct {
	SuiteRunID pulid.ID                         `json:"suiteRunId"`
	Status     agentqualitysuite.SuiteRunStatus `json:"status"`
	CasesTotal int                              `json:"casesTotal"`
	SampleSeed int64                            `json:"sampleSeed"`
	WorkflowID string                           `json:"workflowId"`
}

func openedFrom(run *agentqualitysuite.SuiteRun) *OpenedSuite {
	return &OpenedSuite{
		SuiteRunID: run.ID,
		Status:     run.Status,
		CasesTotal: run.CasesTotal,
		SampleSeed: run.SampleSeed,
		WorkflowID: run.WorkflowID,
	}
}

func (s *Engine) OpenSuite(ctx context.Context, req *OpenSuiteRequest) (*OpenedSuite, error) {
	existing, err := s.bySweepKey(ctx, req.TenantInfo, req.SweepKey)
	if err == nil {
		return openedFrom(existing), nil
	}
	if !errors.Is(err, errSweepKeyUnclaimed) {
		return nil, err
	}

	cases, err := s.cases.ListSamplingCases(ctx, agentqualityports.ListEvalCaseSamplingRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.Agent.AgentDefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, errortypes.NewBusinessError(
			"This agent has no active evaluation cases to run",
		)
	}

	baseline, err := s.suiteRuns.Last(ctx, agentqualityports.LastAgentSuiteRunRequest{
		AgentDefinitionID: req.Agent.AgentDefinitionID,
		TenantInfo:        req.TenantInfo,
		Statuses:          ScoredStatuses(),
	})
	if err != nil {
		return nil, err
	}

	id := agentqualitysuite.NewSuiteRunID()
	seed := agentqualitysuite.SeedFromID(id)
	sample := agentqualitysuite.SampleCases(cases, seed, req.Settings.MaxCasesPerAgent)
	run := s.newSuiteRun(newSuiteRunParams{
		id:       id,
		seed:     seed,
		req:      req,
		revision: agentqualitysuite.SuiteRevision(cases),
		baseline: baseline,
		total:    len(sample),
	})

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	evaluations := suiteEvaluations(run, sample, req.Agent.Fingerprint)
	created, err := s.createSuite(ctx, run, evaluations)
	if errors.Is(err, agentqualityports.ErrSuiteRunAlreadyRunning) {
		if cleared, clearErr := s.clearStaleRunning(ctx, req); clearErr != nil {
			return nil, clearErr
		} else if cleared {
			created, err = s.createSuite(ctx, run, evaluations)
		}
	}
	if errors.Is(err, agentqualityports.ErrSuiteRunSweepKeyTaken) {
		return s.openedAfterSweepKeyTaken(ctx, req.TenantInfo, req.SweepKey, err)
	}
	if errors.Is(err, agentqualityports.ErrSuiteRunAlreadyRunning) {
		return nil, errortypes.NewBusinessError(
			"This agent's suite is already running; wait for it to finish",
		).WithInternal(err)
	}
	if err != nil {
		return nil, err
	}

	return openedFrom(created), nil
}

func (s *Engine) bySweepKey(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sweepKey string,
) (*agentqualitysuite.SuiteRun, error) {
	if sweepKey == "" {
		return nil, errSweepKeyUnclaimed
	}

	existing, err := s.suiteRuns.GetBySweepKey(
		ctx,
		agentqualityports.GetAgentSuiteRunBySweepKeyRequest{
			SweepKey:   sweepKey,
			TenantInfo: tenant,
		},
	)
	if err == nil {
		return existing, nil
	}
	if errortypes.IsNotFoundError(err) {
		return nil, errSweepKeyUnclaimed
	}

	return nil, err
}

func (s *Engine) openedAfterSweepKeyTaken(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sweepKey string,
	createErr error,
) (*OpenedSuite, error) {
	taken, err := s.bySweepKey(ctx, tenant, sweepKey)
	if errors.Is(err, errSweepKeyUnclaimed) {
		return nil, createErr
	}
	if err != nil {
		return nil, errors.Join(createErr, err)
	}

	return openedFrom(taken), nil
}

type newSuiteRunParams struct {
	id       pulid.ID
	seed     int64
	req      *OpenSuiteRequest
	revision string
	baseline *agentqualitysuite.SuiteRun
	total    int
}

func (s *Engine) newSuiteRun(p newSuiteRunParams) *agentqualitysuite.SuiteRun {
	run := &agentqualitysuite.SuiteRun{
		ID:                p.id,
		OrganizationID:    p.req.TenantInfo.OrgID,
		BusinessUnitID:    p.req.TenantInfo.BuID,
		AgentDefinitionID: p.req.Agent.AgentDefinitionID,
		Trigger:           p.req.Trigger,
		SweepKey:          p.req.SweepKey,
		Fingerprint:       p.req.Agent.Fingerprint,
		FingerprintHash:   p.req.Agent.FingerprintHash,
		SuiteRevision:     p.revision,
		SampleSeed:        p.seed,
		Status:            agentqualitysuite.SuiteRunStatusRunning,
		CasesTotal:        p.total,
		StartedAt:         s.now(),
		WorkflowID:        agentqualitycontrol.SuiteRunWorkflowID(p.id),
	}
	if p.req.RequestedByUserID.IsNotNil() {
		userID := p.req.RequestedByUserID
		run.RequestedByUserID = &userID
	}
	if p.baseline != nil {
		baselineID := p.baseline.ID
		run.BaselineRunID = &baselineID
		run.FingerprintChanges = p.req.Agent.Fingerprint.Changes(p.baseline.Fingerprint)
	}

	return run
}

func suiteEvaluations(
	run *agentqualitysuite.SuiteRun,
	sample []agentqualitysuite.SamplingCase,
	fingerprint *agentfingerprint.Fingerprint,
) []*agentevaluation.Evaluation {
	evaluations := make([]*agentevaluation.Evaluation, 0, len(sample))
	for idx, evalCase := range sample {
		caseID := evalCase.ID
		suiteRunID := run.ID
		ordinal := idx + 1
		evaluationID := pulid.MustNew(agentevaluation.EvaluationIDPrefix)
		evaluations = append(evaluations, &agentevaluation.Evaluation{
			ID:                evaluationID,
			OrganizationID:    run.OrganizationID,
			BusinessUnitID:    run.BusinessUnitID,
			AgentDefinitionID: run.AgentDefinitionID,
			EvalCaseID:        &caseID,
			Status:            agentevaluation.EvaluationStatusPending,
			Trigger:           evalCase.Trigger,
			SubjectType:       evalCase.SubjectType,
			SubjectID:         evalCase.SubjectID,
			DefinitionVersion: fingerprintVersion(fingerprint),
			Fingerprint:       fingerprint,
			WorkflowID:        agentqualitycontrol.EvaluationWorkflowID(evaluationID),
			RequestedByUserID: run.RequestedByUserID,
			SuiteRunID:        &suiteRunID,
			SuiteOrdinal:      &ordinal,
			Actions:           []agentreplay.ReplayAction{},
		})
	}

	return evaluations
}

func fingerprintVersion(fingerprint *agentfingerprint.Fingerprint) int64 {
	if fingerprint == nil {
		return 0
	}

	return fingerprint.DefinitionVersion
}

func (s *Engine) createSuite(
	ctx context.Context,
	run *agentqualitysuite.SuiteRun,
	evaluations []*agentevaluation.Evaluation,
) (*agentqualitysuite.SuiteRun, error) {
	var created *agentqualitysuite.SuiteRun
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		saved, err := s.suiteRuns.Create(txCtx, run)
		if err != nil {
			return err
		}
		if err = s.evaluations.CreateMany(txCtx, evaluations); err != nil {
			return err
		}
		created = saved

		return nil
	})
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Engine) clearStaleRunning(ctx context.Context, req *OpenSuiteRequest) (bool, error) {
	running, err := s.suiteRuns.Last(ctx, agentqualityports.LastAgentSuiteRunRequest{
		AgentDefinitionID: req.Agent.AgentDefinitionID,
		TenantInfo:        req.TenantInfo,
		Statuses: []agentqualitysuite.SuiteRunStatus{
			agentqualitysuite.SuiteRunStatusRunning,
		},
	})
	if err != nil || running == nil {
		return false, err
	}
	if s.now()-running.StartedAt < staleRunningAfter {
		return false, nil
	}

	if err = s.fail(
		ctx,
		running,
		"The run never finished and was closed when the next one began.",
	); err != nil {
		return false, err
	}

	return true, nil
}

type RecordSkippedRequest struct {
	TenantInfo pagination.TenantInfo             `json:"tenantInfo"`
	Agent      PlannedAgent                      `json:"agent"`
	SweepKey   string                            `json:"sweepKey"`
	Trigger    agentqualitysuite.SuiteRunTrigger `json:"trigger"`
}

func (s *Engine) RecordSkipped(
	ctx context.Context,
	req *RecordSkippedRequest,
) (*OpenedSuite, error) {
	existing, err := s.bySweepKey(ctx, req.TenantInfo, req.SweepKey)
	if err == nil {
		return openedFrom(existing), nil
	}
	if !errors.Is(err, errSweepKeyUnclaimed) {
		return nil, err
	}

	now := s.now()
	id := agentqualitysuite.NewSuiteRunID()
	run := &agentqualitysuite.SuiteRun{
		ID:                id,
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentDefinitionID: req.Agent.AgentDefinitionID,
		Trigger:           req.Trigger,
		SweepKey:          req.SweepKey,
		Fingerprint:       req.Agent.Fingerprint,
		FingerprintHash:   req.Agent.FingerprintHash,
		SuiteRevision:     req.Agent.SuiteRevision,
		SampleSeed:        agentqualitysuite.SeedFromID(id),
		Status:            agentqualitysuite.SuiteRunStatusSkipped,
		StartedAt:         now,
		FinishedAt:        &now,
		Comments: stringutils.Ellipsize(
			req.Agent.SkipReason,
			agentqualitysuite.MaxSuiteComments,
		),
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.suiteRuns.Create(ctx, run)
	if errors.Is(err, agentqualityports.ErrSuiteRunSweepKeyTaken) {
		return s.openedAfterSweepKeyTaken(ctx, req.TenantInfo, req.SweepKey, err)
	}
	if err != nil {
		return nil, err
	}

	return openedFrom(created), nil
}
