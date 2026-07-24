package agentjobs

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const billingExceptionSystemPrompt = "You are a billing exception analyst for a transportation " +
	"management system. You inspect a blocked billing queue item, determine why it is blocked, and " +
	"propose resolutions for a human biller to approve. You never execute changes yourself. Prefer " +
	"a proposal that a registered tool can carry out; when you cannot resolve the blocker or your " +
	"confidence is low, raise an exception instead."

type ActivitiesParams struct {
	fx.In

	BillingQueue serviceports.BillingQueueService
	Shipment     serviceports.ShipmentService
	Completion   serviceports.CompletionService
	ToolRegistry serviceports.AgentToolRegistry
	ExceptionSvc serviceports.AgentExceptionService
	RunRepo      repositories.AgentRunRepository
	ProposalRepo repositories.AgentProposalRepository
	Logger       *zap.Logger
}

type Activities struct {
	billingQueue serviceports.BillingQueueService
	shipment     serviceports.ShipmentService
	completion   serviceports.CompletionService
	toolRegistry serviceports.AgentToolRegistry
	exceptionSvc serviceports.AgentExceptionService
	runRepo      repositories.AgentRunRepository
	proposalRepo repositories.AgentProposalRepository
	logger       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		billingQueue: p.BillingQueue,
		shipment:     p.Shipment,
		completion:   p.Completion,
		toolRegistry: p.ToolRegistry,
		exceptionSvc: p.ExceptionSvc,
		runRepo:      p.RunRepo,
		proposalRepo: p.ProposalRepo,
		logger:       p.Logger.Named("billing-agent-activities"),
	}
}

func (a *Activities) GatherContextActivity(
	ctx context.Context,
	payload *AgentRunPayload,
) (*GatherContextResult, error) {
	tenant := payload.tenantInfo()

	item, err := a.billingQueue.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		TenantInfo:            tenant,
		ItemID:                payload.SubjectID,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, fmt.Errorf("gather context: load billing queue item: %w", err)
	}

	sections := make([]serviceports.ContextSection, 0, 3)
	sections = append(sections, serviceports.ContextSection{
		Title:   "Billing Queue Item",
		Trusted: true,
		Content: marshalTrusted(item),
	})

	if item.ShipmentID.IsNotNil() {
		readiness, rErr := a.shipment.GetBillingReadiness(ctx, item.ShipmentID, tenant)
		if rErr != nil {
			a.logger.Warn("failed to load billing readiness", zap.Error(rErr))
		} else {
			sections = append(sections, serviceports.ContextSection{
				Title:   "Billing Readiness",
				Trusted: true,
				Content: marshalTrusted(map[string]any{
					"validationFailures":  readiness.ValidationFailures,
					"missingRequirements": readiness.MissingRequirements,
					"warnings":            readiness.Warnings,
					"serviceFailures":     readiness.ServiceFailureContext,
				}),
			})
		}
	}

	notes := strings.TrimSpace(
		strings.Join([]string{item.ReviewNotes, item.ExceptionNotes, item.CancelReason}, "\n"),
	)
	if notes != "" {
		sections = append(sections, serviceports.ContextSection{
			Title:   "Notes and Comments",
			Trusted: false,
			Content: notes,
		})
	}

	deliminated := serviceports.DelimitedContext{Sections: sections}
	hash := hashContext(deliminated)

	if err = a.updateRun(ctx, tenant, payload.RunID, func(run *agent.AgentRun) {
		run.Status = agent.RunStatusDiagnosing
		run.InputContextHash = hash
		run.StartedAt = run.CreatedAt
	}); err != nil {
		return nil, err
	}

	return &GatherContextResult{
		Context:          deliminated,
		InputContextHash: hash,
		SubjectID:        payload.SubjectID,
	}, nil
}

func (a *Activities) DiagnoseActivity(
	ctx context.Context,
	input *DiagnoseActivityInput,
) (*DiagnoseActivityResult, error) {
	req := &serviceports.DiagnoseRequest{
		TenantInfo:    input.TenantInfo,
		PromptVersion: input.PromptVersion,
		SystemPrompt:  billingExceptionSystemPrompt,
		Context:       input.Context,
		ToolSchemas:   a.toolRegistry.Descriptors(),
	}

	result, err := a.completion.Diagnose(ctx, req)
	if err == nil {
		return toDiagnoseResult(result), nil
	}

	if !errors.Is(err, serviceports.ErrModelSchemaValidation) {
		return nil, err
	}

	req.SystemPrompt += "\n\nYour previous response did not satisfy the required schema: " +
		err.Error() + "\nReturn output that strictly matches the schema."

	result, err = a.completion.Diagnose(ctx, req)
	if err != nil {
		return nil, temporal.NewNonRetryableApplicationError(
			"model output failed schema validation after retry",
			"SchemaValidation",
			err,
		)
	}

	return toDiagnoseResult(result), nil
}

func (a *Activities) PersistDiagnosisActivity(
	ctx context.Context,
	input *PersistDiagnosisInput,
) (*PersistDiagnosisResult, error) {
	result := &PersistDiagnosisResult{}

	for _, proposed := range input.Proposals {
		persisted, err := a.persistProposal(ctx, input, proposed)
		if err != nil {
			return nil, err
		}
		if persisted {
			result.ProposalsPersisted++
		}
	}

	for _, raised := range input.Exceptions {
		if err := a.flagException(ctx, input, raised); err != nil {
			return nil, err
		}
		result.ExceptionsPersisted++
	}

	if err := a.updateRun(ctx, input.TenantInfo, input.RunID, func(run *agent.AgentRun) {
		run.ModelIdentifier = input.ModelIdentifier
	}); err != nil {
		return nil, err
	}

	return result, nil
}

func (a *Activities) persistProposal(
	ctx context.Context,
	input *PersistDiagnosisInput,
	proposed serviceports.ProposedAction,
) (bool, error) {
	tool, ok := a.toolRegistry.Get(proposed.ToolName)
	if !ok {
		return false, a.flagException(ctx, input, serviceports.RaisedException{
			Category:       string(agent.CategoryUnableToDiagnose),
			Severity:       string(agent.SeverityMedium),
			AttemptSummary: "Model proposed an unknown tool: " + proposed.ToolName,
			Evidence:       proposed.Evidence,
		})
	}

	proposal := &agent.AgentProposal{
		OrganizationID: input.TenantInfo.OrgID,
		BusinessUnitID: input.TenantInfo.BuID,
		RunID:          input.RunID,
		ToolName:       proposed.ToolName,
		ToolParams:     proposed.ToolParams,
		Confidence:     decimal.NewFromFloat(proposed.Confidence),
		Rationale:      proposed.Rationale,
		Evidence:       proposed.Evidence,
		AutonomyTier:   tool.DefaultAutonomyTier(),
		Status:         agent.ProposalStatusPending,
	}

	me := errortypes.NewMultiError()
	proposal.Validate(me)
	if me.HasErrors() {
		return false, a.flagException(ctx, input, serviceports.RaisedException{
			Category:       string(agent.CategoryUnableToDiagnose),
			Severity:       string(agent.SeverityMedium),
			AttemptSummary: "Model produced an invalid proposal for tool " + proposed.ToolName,
			Evidence:       proposed.Evidence,
		})
	}

	if _, err := a.proposalRepo.Create(ctx, proposal); err != nil {
		return false, fmt.Errorf("persist proposal: %w", err)
	}

	return true, nil
}

func (a *Activities) flagException(
	ctx context.Context,
	input *PersistDiagnosisInput,
	raised serviceports.RaisedException,
) error {
	category := agent.ExceptionCategory(raised.Category)
	if !category.IsValid() {
		category = agent.CategoryOther
	}

	severity := agent.Severity(raised.Severity)
	if !severity.IsValid() {
		severity = agent.SeverityMedium
	}

	evidence := raised.Evidence
	if len(evidence) == 0 {
		evidence = []agent.EvidenceRef{{
			Type: "agent_run",
			ID:   input.RunID.String(),
			Note: "raised by billing exception agent",
		}}
	}

	_, err := a.exceptionSvc.Flag(ctx, &serviceports.FlagAgentExceptionRequest{
		RunID:          input.RunID,
		Category:       category,
		Severity:       severity,
		SubjectType:    input.SubjectType,
		SubjectID:      input.SubjectID,
		AttemptSummary: raised.AttemptSummary,
		Evidence:       evidence,
		BlastRadius:    raised.BlastRadius,
		TenantInfo:     input.TenantInfo,
	}, agentActor(input.TenantInfo))
	if err != nil {
		return fmt.Errorf("flag exception: %w", err)
	}

	return nil
}

func (a *Activities) CompleteRunActivity(ctx context.Context, input *CompleteRunInput) error {
	return a.updateRun(ctx, input.TenantInfo, input.RunID, func(run *agent.AgentRun) {
		run.Status = input.Status
		completedAt := run.UpdatedAt
		run.CompletedAt = &completedAt
	})
}

func (a *Activities) ExpireProposalsActivity(ctx context.Context, input *ExpireProposalsInput) error {
	if _, err := a.proposalRepo.ExpirePendingByRun(
		ctx,
		repositories.ExpireAgentProposalsByRunRequest{
			RunID:      input.RunID,
			TenantInfo: input.TenantInfo,
		},
	); err != nil {
		return fmt.Errorf("expire proposals: %w", err)
	}

	return a.updateRun(ctx, input.TenantInfo, input.RunID, func(run *agent.AgentRun) {
		run.Status = agent.RunStatusCompleted
		completedAt := run.UpdatedAt
		run.CompletedAt = &completedAt
	})
}

func (a *Activities) updateRun(
	ctx context.Context,
	tenant pagination.TenantInfo,
	runID pulid.ID,
	mutate func(run *agent.AgentRun),
) error {
	run, err := a.runRepo.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         runID,
		TenantInfo: &tenant,
	})
	if err != nil {
		return fmt.Errorf("load agent run: %w", err)
	}

	mutate(run)

	if _, err = a.runRepo.Update(ctx, run); err != nil {
		return fmt.Errorf("update agent run: %w", err)
	}

	return nil
}

func agentActor(tenant pagination.TenantInfo) *serviceports.RequestActor {
	return &serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeAgent,
		PrincipalID:    serviceports.AgentPrincipalID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}
}

func toDiagnoseResult(result *serviceports.DiagnoseResult) *DiagnoseActivityResult {
	return &DiagnoseActivityResult{
		Proposals:       result.Proposals,
		Exceptions:      result.Exceptions,
		ModelIdentifier: result.ModelIdentifier,
	}
}

func marshalTrusted(value any) string {
	encoded, err := sonic.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}

	return string(encoded)
}

func hashContext(deliminated serviceports.DelimitedContext) string {
	encoded, err := sonic.Marshal(deliminated)
	if err != nil {
		encoded = []byte(fmt.Sprintf("%v", deliminated))
	}

	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
