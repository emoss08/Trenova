package agentqualityservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const staleRunningAfter = int64(24 * 60 * 60)

type SweepRequest struct {
	TenantInfo        pagination.TenantInfo `json:"tenantInfo"`
	AgentDefinitionID pulid.ID              `json:"agentDefinitionId,omitzero"`
	Forced            bool                  `json:"forced"`
}

type PlannedAgent struct {
	AgentDefinitionID pulid.ID           `json:"agentDefinitionId"`
	Name              string             `json:"name"`
	Fingerprint       *agent.Fingerprint `json:"fingerprint"`
	FingerprintHash   string             `json:"fingerprintHash"`
	SuiteRevision     string             `json:"suiteRevision"`
	ActiveCases       int                `json:"activeCases"`
	Skip              bool               `json:"skip"`
	SkipReason        string             `json:"skipReason,omitempty"`
}

type SweepPlan struct {
	Enabled    bool                       `json:"enabled"`
	Reason     string                     `json:"reason,omitempty"`
	Timezone   string                     `json:"timezone"`
	PlannedAt  int64                      `json:"plannedAt"`
	DayStart   int64                      `json:"dayStart"`
	MonthStart int64                      `json:"monthStart"`
	Settings   agentquality.SuiteSettings `json:"settings"`
	Agents     []PlannedAgent             `json:"agents"`
}

func (s *Service) PlanSweep(ctx context.Context, req *SweepRequest) (*SweepPlan, error) {
	control, err := s.control(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	now := s.now()
	timezone := s.timezoneOf(ctx, control, req.TenantInfo)
	dayStart, monthStart, err := windows(now, timezone)
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
		planned, planErr := s.planAgent(ctx, planAgentParams{
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

func (s *Service) agentsToPlan(
	ctx context.Context,
	req *SweepRequest,
) ([]*agentdefinition.Definition, error) {
	ids := []pulid.ID{req.AgentDefinitionID}
	if req.AgentDefinitionID.IsNil() {
		withCases, err := s.cases.ListAgentsWithActiveCases(
			ctx,
			repositories.ListAgentsWithActiveEvalCasesRequest{TenantInfo: req.TenantInfo},
		)
		if err != nil {
			return nil, err
		}
		ids = withCases
	}
	if len(ids) == 0 {
		return nil, nil
	}

	definitions, err := s.definitions.ListByIDs(ctx, repositories.ListAgentDefinitionsByIDsRequest{
		IDs:        ids,
		TenantInfo: req.TenantInfo,
	})
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

func (r resolvedProvider) forAgent(preferred pulid.ID) (pulid.ID, string) {
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

func (s *Service) chatProvider(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (resolvedProvider, error) {
	if s.providers == nil {
		return resolvedProvider{}, nil
	}

	candidates, err := s.providers.ListForTask(ctx, repositories.ListAIProvidersForTaskRequest{
		Task:       aiprovider.TaskAssistantChat,
		TenantInfo: tenant,
	})
	if err != nil {
		return resolvedProvider{}, fmt.Errorf("read the providers that serve chat: %w", err)
	}

	usable := make([]*aiprovider.Provider, 0, len(candidates))
	for _, candidate := range candidates {
		if ok, _ := candidate.CanServeTask(aiprovider.TaskAssistantChat); ok {
			usable = append(usable, candidate)
		}
	}

	return resolvedProvider{candidates: usable}, nil
}

type planAgentParams struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	provider   resolvedProvider
	control    *agentquality.Control
	now        int64
	forced     bool
}

func (s *Service) fingerprint(
	definition *agentdefinition.Definition,
	provider resolvedProvider,
) *agent.Fingerprint {
	providerID, model := provider.forAgent(definition.PreferredProviderID)
	if s.runtime == nil {
		return agentquality.FingerprintOf(definition, "").Served(model, providerID)
	}

	return s.runtime.Fingerprint(definition, providerID, model)
}

func (s *Service) planAgent(ctx context.Context, p planAgentParams) (*PlannedAgent, error) {
	fingerprint := s.fingerprint(p.definition, p.provider)
	cases, err := s.cases.ListSamplingCases(ctx, repositories.ListEvalCaseSamplingRequest{
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
		SuiteRevision:     agentquality.SuiteRevision(cases),
		ActiveCases:       len(cases),
	}
	if len(cases) == 0 {
		planned.Skip = true
		planned.SkipReason = "The agent has no active evaluation cases."

		return planned, nil
	}

	last, err := s.suiteRuns.Last(ctx, repositories.LastAgentSuiteRunRequest{
		AgentDefinitionID: p.definition.ID,
		TenantInfo:        p.tenant,
		Statuses:          []agentquality.SuiteRunStatus{agentquality.SuiteRunStatusCompleted},
	})
	if err != nil {
		return nil, err
	}

	var lastRun *agentquality.LastRun
	if last != nil && last.FinishedAt != nil {
		lastRun = &agentquality.LastRun{
			FingerprintHash: last.FingerprintHash,
			SuiteRevision:   last.SuiteRevision,
			FinishedAt:      *last.FinishedAt,
		}
	}

	decision := agentquality.DecideSkip(agentquality.SkipInput{
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
	TenantInfo        pagination.TenantInfo        `json:"tenantInfo"`
	Agent             PlannedAgent                 `json:"agent"`
	SweepKey          string                       `json:"sweepKey,omitempty"`
	Trigger           agentquality.SuiteRunTrigger `json:"trigger"`
	RequestedByUserID pulid.ID                     `json:"requestedByUserId,omitzero"`
	Settings          agentquality.SuiteSettings   `json:"settings"`
}

type OpenedSuite struct {
	SuiteRunID pulid.ID                    `json:"suiteRunId"`
	Status     agentquality.SuiteRunStatus `json:"status"`
	CasesTotal int                         `json:"casesTotal"`
	SampleSeed int64                       `json:"sampleSeed"`
	WorkflowID string                      `json:"workflowId"`
}

func openedFrom(run *agentquality.SuiteRun) *OpenedSuite {
	return &OpenedSuite{
		SuiteRunID: run.ID,
		Status:     run.Status,
		CasesTotal: run.CasesTotal,
		SampleSeed: run.SampleSeed,
		WorkflowID: run.WorkflowID,
	}
}

func (s *Service) OpenSuite(ctx context.Context, req *OpenSuiteRequest) (*OpenedSuite, error) {
	existing, err := s.bySweepKey(ctx, req.TenantInfo, req.SweepKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return openedFrom(existing), nil
	}

	cases, err := s.cases.ListSamplingCases(ctx, repositories.ListEvalCaseSamplingRequest{
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

	baseline, err := s.suiteRuns.Last(ctx, repositories.LastAgentSuiteRunRequest{
		AgentDefinitionID: req.Agent.AgentDefinitionID,
		TenantInfo:        req.TenantInfo,
		Statuses:          scoredStatuses(),
	})
	if err != nil {
		return nil, err
	}

	id := agentquality.NewSuiteRunID()
	seed := agentquality.SeedFromID(id)
	sample := agentquality.SampleCases(cases, seed, req.Settings.MaxCasesPerAgent)
	run := s.newSuiteRun(newSuiteRunParams{
		id:       id,
		seed:     seed,
		req:      req,
		revision: agentquality.SuiteRevision(cases),
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
	if errors.Is(err, repositories.ErrSuiteRunAlreadyRunning) {
		if cleared, clearErr := s.clearStaleRunning(ctx, req); clearErr != nil {
			return nil, clearErr
		} else if cleared {
			created, err = s.createSuite(ctx, run, evaluations)
		}
	}
	if errors.Is(err, repositories.ErrSuiteRunSweepKeyTaken) {
		existing, getErr := s.bySweepKey(ctx, req.TenantInfo, req.SweepKey)
		if getErr != nil || existing == nil {
			return nil, errors.Join(err, getErr)
		}

		return openedFrom(existing), nil
	}
	if errors.Is(err, repositories.ErrSuiteRunAlreadyRunning) {
		return nil, errortypes.NewBusinessError(
			"This agent's suite is already running; wait for it to finish",
		).WithInternal(err)
	}
	if err != nil {
		return nil, err
	}

	return openedFrom(created), nil
}

func (s *Service) bySweepKey(
	ctx context.Context,
	tenant pagination.TenantInfo,
	sweepKey string,
) (*agentquality.SuiteRun, error) {
	if sweepKey == "" {
		return nil, nil
	}

	existing, err := s.suiteRuns.GetBySweepKey(ctx, repositories.GetAgentSuiteRunBySweepKeyRequest{
		SweepKey:   sweepKey,
		TenantInfo: tenant,
	})
	if err == nil {
		return existing, nil
	}
	if errortypes.IsNotFoundError(err) {
		return nil, nil
	}

	return nil, err
}

type newSuiteRunParams struct {
	id       pulid.ID
	seed     int64
	req      *OpenSuiteRequest
	revision string
	baseline *agentquality.SuiteRun
	total    int
}

func (s *Service) newSuiteRun(p newSuiteRunParams) *agentquality.SuiteRun {
	run := &agentquality.SuiteRun{
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
		Status:            agentquality.SuiteRunStatusRunning,
		CasesTotal:        p.total,
		StartedAt:         s.now(),
		WorkflowID:        agentquality.SuiteRunWorkflowID(p.id),
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
	run *agentquality.SuiteRun,
	sample []agentquality.SamplingCase,
	fingerprint *agent.Fingerprint,
) []*agent.Evaluation {
	evaluations := make([]*agent.Evaluation, 0, len(sample))
	for idx, evalCase := range sample {
		caseID := evalCase.ID
		suiteRunID := run.ID
		ordinal := idx + 1
		evaluationID := pulid.MustNew(agent.EvaluationIDPrefix)
		evaluations = append(evaluations, &agent.Evaluation{
			ID:                evaluationID,
			OrganizationID:    run.OrganizationID,
			BusinessUnitID:    run.BusinessUnitID,
			AgentDefinitionID: run.AgentDefinitionID,
			EvalCaseID:        &caseID,
			Status:            agent.EvaluationStatusPending,
			Trigger:           evalCase.Trigger,
			SubjectType:       evalCase.SubjectType,
			SubjectID:         evalCase.SubjectID,
			DefinitionVersion: fingerprintVersion(fingerprint),
			Fingerprint:       fingerprint,
			WorkflowID:        agentquality.EvaluationWorkflowID(evaluationID),
			RequestedByUserID: run.RequestedByUserID,
			SuiteRunID:        &suiteRunID,
			SuiteOrdinal:      &ordinal,
			Actions:           []agent.ReplayAction{},
		})
	}

	return evaluations
}

func fingerprintVersion(fingerprint *agent.Fingerprint) int64 {
	if fingerprint == nil {
		return 0
	}

	return fingerprint.DefinitionVersion
}

func (s *Service) createSuite(
	ctx context.Context,
	run *agentquality.SuiteRun,
	evaluations []*agent.Evaluation,
) (*agentquality.SuiteRun, error) {
	var created *agentquality.SuiteRun
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
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

func (s *Service) clearStaleRunning(ctx context.Context, req *OpenSuiteRequest) (bool, error) {
	running, err := s.suiteRuns.Last(ctx, repositories.LastAgentSuiteRunRequest{
		AgentDefinitionID: req.Agent.AgentDefinitionID,
		TenantInfo:        req.TenantInfo,
		Statuses:          []agentquality.SuiteRunStatus{agentquality.SuiteRunStatusRunning},
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
	TenantInfo pagination.TenantInfo        `json:"tenantInfo"`
	Agent      PlannedAgent                 `json:"agent"`
	SweepKey   string                       `json:"sweepKey"`
	Trigger    agentquality.SuiteRunTrigger `json:"trigger"`
}

func (s *Service) RecordSkipped(
	ctx context.Context,
	req *RecordSkippedRequest,
) (*OpenedSuite, error) {
	existing, err := s.bySweepKey(ctx, req.TenantInfo, req.SweepKey)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return openedFrom(existing), nil
	}

	now := s.now()
	id := agentquality.NewSuiteRunID()
	run := &agentquality.SuiteRun{
		ID:                id,
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentDefinitionID: req.Agent.AgentDefinitionID,
		Trigger:           req.Trigger,
		SweepKey:          req.SweepKey,
		Fingerprint:       req.Agent.Fingerprint,
		FingerprintHash:   req.Agent.FingerprintHash,
		SuiteRevision:     req.Agent.SuiteRevision,
		SampleSeed:        agentquality.SeedFromID(id),
		Status:            agentquality.SuiteRunStatusSkipped,
		StartedAt:         now,
		FinishedAt:        &now,
		Comments: stringutils.Ellipsize(
			req.Agent.SkipReason,
			agentquality.MaxSuiteComments,
		),
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.suiteRuns.Create(ctx, run)
	if errors.Is(err, repositories.ErrSuiteRunSweepKeyTaken) {
		existing, getErr := s.bySweepKey(ctx, req.TenantInfo, req.SweepKey)
		if getErr != nil || existing == nil {
			return nil, errors.Join(err, getErr)
		}

		return openedFrom(existing), nil
	}
	if err != nil {
		return nil, err
	}

	return openedFrom(created), nil
}

func (s *Service) RunSuite(
	ctx context.Context,
	req *services.RunAgentSuiteRequest,
	actor *services.RequestActor,
) (*agentquality.SuiteRun, error) {
	if s.starter == nil {
		return nil, errortypes.NewBusinessError("The workflow engine is not available")
	}

	plan, err := s.PlanSweep(ctx, &SweepRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Forced:            true,
	})
	if err != nil {
		return nil, err
	}
	if len(plan.Agents) == 0 {
		return nil, errortypes.NewNotFoundError("Agent not found within your organization")
	}

	planned := plan.Agents[0]
	if planned.ActiveCases == 0 {
		return nil, errortypes.NewBusinessError(
			"This agent has no active evaluation cases to run; activate a case first",
		)
	}

	opened, err := s.OpenSuite(ctx, &OpenSuiteRequest{
		TenantInfo:        req.TenantInfo,
		Agent:             planned,
		Trigger:           agentquality.SuiteRunTriggerManual,
		RequestedByUserID: actor.UserIDOrNil(),
		Settings:          plan.Settings,
	})
	if err != nil {
		return nil, err
	}

	run, err := s.suiteRuns.GetByID(ctx, repositories.GetAgentSuiteRunRequest{
		ID:         opened.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.starter.StartSuiteRun(ctx, &services.AgentSuiteRunStart{
		TenantInfo:        req.TenantInfo,
		UserID:            actor.UserIDOrNil(),
		SuiteRunID:        run.ID,
		AgentDefinitionID: run.AgentDefinitionID,
		SampleSeed:        run.SampleSeed,
		Settings:          plan.Settings,
		DayStart:          plan.DayStart,
		MonthStart:        plan.MonthStart,
	}); err != nil {
		if failErr := s.fail(
			ctx,
			run,
			"The suite could not be started: "+err.Error(),
		); failErr != nil {
			s.l.Error("could not close a suite run that failed to start", zap.Error(failErr))
		}

		return nil, err
	}

	s.logRun(run, actor)

	return run, nil
}

func (s *Service) logRun(run *agentquality.SuiteRun, actor *services.RequestActor) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(run),
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
	}, auditservice.WithComment("Agent suite run started by hand")); err != nil {
		s.l.Error("failed to log agent suite run audit", zap.Error(err))
	}
}
