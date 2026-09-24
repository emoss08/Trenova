// Package agenttrustservice turns decisions into autonomy.
//
// Until this existed, the tier a tool ran at on an agent was whatever a person
// set when they built the agent, and it never moved. An organization that had
// approved the same reassignment five hundred times in a row was still asked
// the five hundred and first time, and an agent whose proposals were being
// rejected every day kept exactly the reach it started with. The ledger here
// records each outcome per agent and tool; a streak of clean approvals earns
// the next tier up, and a rejection or a failed execution takes an earned tier
// back. A tier a person chose is never touched.
package agenttrustservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EventToolPromoted = "agent.tool_promoted"
	EventToolDemoted  = "agent.tool_demoted"

	notificationSource = "agenttrustservice"
	agentsLink         = "/admin/agent-control?tab=agents"
)

// notifier is the one call this service makes on the notification service.
type notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

type actionLogger interface {
	LogAction(params *services.LogActionParams, opts ...services.LogOption) error
}

type Params struct {
	fx.In

	Logger *zap.Logger
	// DB holds a tier change and the ledger's record of it in one
	// transaction, so neither lands without the other.
	DB           ports.DBConnection
	Trust        repositories.AgentToolTrustRepository
	Definitions  repositories.AgentDefinitionRepository
	Controls     repositories.AgentControlRepository
	Runs         repositories.AgentRunRepository
	Tools        services.AgentToolRegistry
	AuditService services.AuditService
	// Notifications is optional so the ledger still counts where nothing can
	// be told about a change; the audit line is the durable record either way.
	Notifications *notificationservice.Service `optional:"true"`
}

type Service struct {
	l             *zap.Logger
	db            ports.DBConnection
	trust         repositories.AgentToolTrustRepository
	definitions   repositories.AgentDefinitionRepository
	controls      repositories.AgentControlRepository
	runs          repositories.AgentRunRepository
	tools         services.AgentToolRegistry
	audit         actionLogger
	notifications notifier
}

func New(p Params) services.AgentTrustService {
	svc := &Service{
		l:           p.Logger.Named("service.agenttrust"),
		db:          p.DB,
		trust:       p.Trust,
		definitions: p.Definitions,
		controls:    p.Controls,
		runs:        p.Runs,
		tools:       p.Tools,
		audit:       p.AuditService,
	}
	if p.Notifications != nil {
		svc.notifications = p.Notifications
	}

	return svc
}

// RecordDecision folds a person's decision into the ledger and moves the
// tool's tier when the ledger says to. The decision is already recorded and
// the proposal already decided by the time this runs; nothing here can undo
// either, and an error here is the caller's to log rather than to surface.
func (s *Service) RecordDecision(
	ctx context.Context,
	proposal *agent.AgentProposal,
	decision *agent.AgentDecision,
) error {
	if proposal == nil || decision == nil {
		return nil
	}

	outcome := agent.OutcomeOfDecision(decision.Decision, decision.Modifications)

	return s.record(ctx, proposal, outcome, decision.DecidedByUserID)
}

// RecordExecutionFailure counts a tool that was approved, or ran on its own,
// and did not go through. The world disagreed with the agent; that is a
// setback whichever tier the tool was at.
func (s *Service) RecordExecutionFailure(ctx context.Context, proposal *agent.AgentProposal) error {
	if proposal == nil {
		return nil
	}

	return s.record(ctx, proposal, agent.TrustOutcomeExecutionFailed, pulid.Nil)
}

func (s *Service) ListForDefinition(
	ctx context.Context,
	req repositories.ListToolTrustRequest,
) ([]*agent.ToolTrust, error) {
	return s.trust.ListByDefinition(ctx, req)
}

func (s *Service) record(
	ctx context.Context,
	proposal *agent.AgentProposal,
	outcome agent.TrustOutcome,
	decidedBy pulid.ID,
) error {
	tenantInfo := pagination.TenantInfo{
		OrgID: proposal.OrganizationID,
		BuID:  proposal.BusinessUnitID,
	}

	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         proposal.RunID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return fmt.Errorf("read run for trust ledger: %w", err)
	}
	// A run from before agents were definitions has no agent to credit.
	if run.AgentDefinitionID.IsNil() {
		return nil
	}

	now := timeutils.NowUnix()
	row, err := s.trust.Record(ctx, repositories.RecordToolTrustRequest{
		TenantInfo:        tenantInfo,
		AgentDefinitionID: run.AgentDefinitionID,
		ToolName:          proposal.ToolName,
		Outcome:           outcome,
		At:                now,
	})
	if err != nil {
		return err
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         run.AgentDefinitionID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return fmt.Errorf("read agent for trust ledger: %w", err)
	}

	change := tierChange{
		tenant:     tenantInfo,
		definition: definition,
		row:        row,
		current:    definition.EffectiveTier(proposal.ToolName, s.defaultTier(proposal.ToolName)),
		decidedBy:  decidedBy,
		at:         now,
	}

	switch {
	case outcome.Setback():
		// A setback takes back only what the ledger gave, and does so whether
		// or not the organization still has earned autonomy on: a tier that
		// was earned is only as good as the streak behind it, and the streak
		// is gone.
		if !row.HoldsEarnedTier(change.current) {
			return nil
		}

		return s.demote(ctx, change)
	case outcome.Clean():
		control, err := s.controls.GetOrCreate(ctx, tenantInfo)
		if err != nil {
			return fmt.Errorf("read agent control for trust ledger: %w", err)
		}
		if !control.EarnedAutonomy || !row.ReadyForPromotion(control.PromotionThreshold) {
			return nil
		}

		return s.promote(ctx, change, control)
	default:
		return nil
	}
}

type tierChange struct {
	tenant     pagination.TenantInfo
	definition *agentdefinition.Definition
	row        *agent.ToolTrust
	current    agent.AutonomyTier
	decidedBy  pulid.ID
	at         int64
}

func (s *Service) promote(
	ctx context.Context,
	change tierChange,
	control *tenant.AgentControl,
) error {
	next, ok := change.current.Next()
	if !ok || !change.definition.WithinCeiling(next) ||
		next.Above(s.promotable(change.row.ToolName)) {
		// At the ceiling there is nowhere to go. The streak keeps counting so
		// the agent's page can show it, and a raised ceiling lets the next
		// clean approval promote.
		return nil
	}

	moved, err := s.moveTier(ctx, change, next, true)
	if err != nil || !moved {
		return err
	}

	s.l.Info("agent tool promoted on a clean streak",
		zap.String("agent", change.definition.ID.String()),
		zap.String("tool", change.row.ToolName),
		zap.String("from", string(change.current)),
		zap.String("to", string(next)),
		zap.Int("streak", change.row.Streak),
	)

	s.logTierChange(change, next, "Tool promoted after a streak of clean approvals")
	s.notify(ctx, change, next, tierNotice{
		eventType: EventToolPromoted,
		priority:  notification.PriorityMedium,
		title: fmt.Sprintf(
			"%s now runs %s at \"%s\"",
			change.definition.Name,
			stringutils.HumanizeSnakeCase(change.row.ToolName),
			tierLabel(next),
		),
		message: fmt.Sprintf(
			"%d approvals in a row without a change met the organization's threshold of %d, "+
				"so the tool moved up from \"%s\". A rejection or a failed run takes it back.",
			change.row.Streak, control.PromotionThreshold, tierLabel(change.current),
		),
	})

	return nil
}

func (s *Service) demote(ctx context.Context, change tierChange) error {
	previous, ok := change.current.Previous()
	if !ok {
		return nil
	}

	moved, err := s.moveTier(ctx, change, previous, false)
	if err != nil || !moved {
		return err
	}

	s.l.Warn("agent tool demoted after a setback",
		zap.String("agent", change.definition.ID.String()),
		zap.String("tool", change.row.ToolName),
		zap.String("from", string(change.current)),
		zap.String("to", string(previous)),
	)

	s.logTierChange(change, previous, "Earned tool tier taken back after a rejection or failed run")
	s.notify(ctx, change, previous, tierNotice{
		eventType: EventToolDemoted,
		priority:  notification.PriorityHigh,
		title: fmt.Sprintf(
			"%s lost \"%s\" on %s",
			change.definition.Name,
			tierLabel(change.current),
			stringutils.HumanizeSnakeCase(change.row.ToolName),
		),
		message: fmt.Sprintf(
			"The tier was earned from a streak of approvals, and a rejection or a failed run ended the streak. "+
				"The tool is back at \"%s\" and must earn its way up again.",
			tierLabel(previous),
		),
	})

	return nil
}

// moveTier sets the tool's tier on the agent and records on the ledger row
// that trust moved it, in one transaction. It reports false, with nothing
// changed, when the row has moved on since the change was decided from it.
//
// The ledger row is written first and conditioned on the version the decision
// read. Two approvals landing at once both used to read a streak past the
// threshold and both promote, one tier each; a rejection recorded between the
// read and the write was promoted over. Now only the decision that recorded
// the row's latest state can move the tier, and it holds the row's lock until
// the agent's tier is written, so the next decision reads the settled streak.
func (s *Service) moveTier(
	ctx context.Context,
	change tierChange,
	to agent.AutonomyTier,
	promoted bool,
) (bool, error) {
	mark := repositories.MarkToolTierChangeRequest{
		ID:         change.row.ID,
		TenantInfo: change.tenant,
		Version:    change.row.Version,
		Promoted:   promoted,
		At:         change.at,
	}
	label := "demotion"
	if promoted {
		mark.EarnedTier = to
		label = "promotion"
	}

	moved := true
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if _, err := s.trust.MarkTierChange(txCtx, mark); err != nil {
			if errortypes.IsVersionMismatchError(err) {
				moved = false
				return nil
			}

			return fmt.Errorf("mark tool %s: %w", label, err)
		}

		return s.setTier(txCtx, change, to)
	})
	if err != nil {
		return false, err
	}
	if !moved {
		s.l.Debug("a later decision moved the ledger on; leaving the tier to it",
			zap.String("agent", change.definition.ID.String()),
			zap.String("tool", change.row.ToolName),
			zap.String("change", label),
		)
	}

	return moved, nil
}

func (s *Service) setTier(ctx context.Context, change tierChange, tier agent.AutonomyTier) error {
	if err := s.definitions.SetToolTier(ctx, repositories.SetAgentDefinitionToolTierRequest{
		ID:         change.definition.ID,
		TenantInfo: change.tenant,
		ToolName:   change.row.ToolName,
		Tier:       tier,
	}); err != nil {
		return fmt.Errorf("set tool tier: %w", err)
	}

	return nil
}

func (s *Service) defaultTier(toolName string) agent.AutonomyTier {
	policy, ok := s.policyOf(toolName)
	if !ok {
		return agent.TierPropose
	}

	return policy.DefaultTier
}

// promotable is the most a tool may ever be promoted to: the most its policy
// lets it run at, held below approval for a tool whose work leaves the
// organization.
func (s *Service) promotable(toolName string) agent.AutonomyTier {
	policy, ok := s.policyOf(toolName)
	if !ok {
		return agent.TierAutoExecute
	}

	return agenttoolpolicy.Promotable(policy)
}

func (s *Service) policyOf(toolName string) (services.ToolPolicy, bool) {
	if s.tools == nil {
		return services.ToolPolicy{}, false
	}
	tool, ok := s.tools.Get(toolName)
	if !ok {
		return services.ToolPolicy{}, false
	}

	return tool.Policy(), true
}

func (s *Service) logTierChange(change tierChange, to agent.AutonomyTier, comment string) {
	if s.audit == nil {
		return
	}

	actor := services.SystemAuditActor()
	if change.decidedBy.IsNotNil() {
		actor = services.AuditActor{
			PrincipalType: services.PrincipalTypeUser,
			PrincipalID:   change.decidedBy,
			UserID:        change.decidedBy,
		}
	}

	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:      permission.ResourceAgentDefinition,
		ResourceID:    change.definition.ID.String(),
		Operation:     permission.OpUpdate,
		UserID:        actor.UserID,
		PrincipalType: actor.PrincipalType,
		PrincipalID:   actor.PrincipalID,
		CurrentState: jsonutils.MustToJSON(map[string]any{
			"tool":              change.row.ToolName,
			"fromTier":          change.current,
			"toTier":            to,
			"streak":            change.row.Streak,
			"approvals":         change.row.Approvals,
			"rejections":        change.row.Rejections,
			"executionFailures": change.row.ExecutionFailures,
		}),
		OrganizationID: change.tenant.OrgID,
		BusinessUnitID: change.tenant.BuID,
		Critical:       true,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log tool tier change", zap.Error(err))
	}
}

type tierNotice struct {
	eventType string
	priority  notification.Priority
	title     string
	message   string
}

func (s *Service) notify(
	ctx context.Context,
	change tierChange,
	to agent.AutonomyTier,
	notice tierNotice,
) {
	if s.notifications == nil {
		return
	}

	buID := change.tenant.BuID
	if _, err := s.notifications.Create(ctx, &notification.Notification{
		OrganizationID: change.tenant.OrgID,
		BusinessUnitID: &buID,
		EventType:      notice.eventType,
		Priority:       notice.priority,
		Channel:        notification.ChannelGlobal,
		Title:          notice.title,
		Message:        notice.message,
		Source:         notificationSource,
		Data: map[string]any{
			"link":      agentsLink,
			"agentId":   change.definition.ID.String(),
			"agentName": change.definition.Name,
			"toolName":  change.row.ToolName,
			"fromTier":  string(change.current),
			"toTier":    string(to),
		},
		RelatedEntities: map[string]any{
			"agentDefinitionId": change.definition.ID.String(),
		},
	}); err != nil {
		s.l.Warn("failed to create tool tier notification",
			zap.String("agent", change.definition.ID.String()),
			zap.Error(err),
		)
	}
}

func tierLabel(tier agent.AutonomyTier) string {
	switch tier {
	case agent.TierActWithApproval:
		return "Ask first"
	case agent.TierAutoExecute:
		return "Automatic"
	default:
		return "Propose"
	}
}
