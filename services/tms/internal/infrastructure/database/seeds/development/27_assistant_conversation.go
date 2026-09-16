package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

const (
	seedConversationModel  = "meta-llama/Llama-3.3-70B-Instruct"
	seedConversationMinute = int64(60)
)

type AssistantConversationSeed struct {
	seedhelpers.BaseSeed
}

// AssistantConversationSeed writes two conversations for the admin user: a
// dispatch thread that looked a shipment up through a tool, and a billing
// thread that ended in a proposal waiting for a decision. Together they exercise
// every message shape the thread view renders — a tool call, a tool result, a
// refusal, and a pending proposal card — without a model being reachable.
//
// Depends on:
//   - AgentDefinition: the agents the threads belong to
//   - Shipment: the record the dispatch thread looked up
func NewAssistantConversationSeed() *AssistantConversationSeed {
	seed := &AssistantConversationSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"AssistantConversation",
		"1.0.0",
		"Seeds assistant conversations with tool traffic and a pending proposal",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedAgentDefinition, seedhelpers.SeedShipment)
	return seed
}

type conversationSeedRefs struct {
	org      *tenant.Organization
	admin    *tenant.User
	dispatch *agentdefinition.Definition
	billing  *agentdefinition.Definition
	shipment *shipment.Shipment
	now      int64
}

func (s *AssistantConversationSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			refs, err := s.loadRefs(ctx, tx, sc)
			if err != nil {
				return err
			}

			cols := buncolgen.ThreadColumns
			count, err := tx.NewSelect().
				Model((*conversation.Thread)(nil)).
				Where(cols.OrganizationID.Eq(), refs.org.ID).
				Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
				Where(cols.UserID.Eq(), refs.admin.ID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing threads: %w", err)
			}
			if count > 0 {
				return nil
			}

			if err = s.seedDispatchThread(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("seed dispatch thread: %w", err)
			}
			if err = s.seedBillingThread(ctx, tx, sc, refs); err != nil {
				return fmt.Errorf("seed billing thread: %w", err)
			}

			return nil
		},
	)
}

func (s *AssistantConversationSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
) (*conversationSeedRefs, error) {
	org, err := sc.GetDefaultOrganization(ctx)
	if err != nil {
		return nil, err
	}
	admin, err := sc.GetUserByUsername(ctx, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	refs := &conversationSeedRefs{org: org, admin: admin, now: timeutils.NowUnix()}

	agents := make([]*agentdefinition.Definition, 0, 2)
	agentCols := buncolgen.DefinitionColumns
	if err = tx.NewSelect().
		Model(&agents).
		Where(agentCols.OrganizationID.Eq(), org.ID).
		Where(agentCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(agentCols.Name.In(), bun.In([]string{SeedAgentDispatchName, SeedAgentBillingName})).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load seeded agents: %w", err)
	}
	for _, definition := range agents {
		switch definition.Name {
		case SeedAgentDispatchName:
			refs.dispatch = definition
		case SeedAgentBillingName:
			refs.billing = definition
		}
	}
	if refs.dispatch == nil || refs.billing == nil {
		return nil, fmt.Errorf("seeded agents not found: %w", seedhelpers.ErrEntityNotFound)
	}

	shipmentCols := buncolgen.ShipmentColumns
	refs.shipment = new(shipment.Shipment)
	if err = tx.NewSelect().
		Model(refs.shipment).
		Where(shipmentCols.OrganizationID.Eq(), org.ID).
		Where(shipmentCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(shipmentCols.CreatedAt.OrderAsc()).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load a seeded shipment: %w", err)
	}

	return refs, nil
}

func (s *AssistantConversationSeed) seedDispatchThread(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *conversationSeedRefs,
) error {
	startedAt := refs.now - 3*24*60*seedConversationMinute
	thread, err := s.insertThread(ctx, tx, sc, refs, refs.dispatch, threadSeed{
		title:     fmt.Sprintf("Where is %s?", refs.shipment.ProNumber),
		startedAt: startedAt,
		turns:     4,
	})
	if err != nil {
		return err
	}

	callID := "call_" + pulid.MustNew("").String()
	messages := []*conversation.Message{
		{
			ThreadID:      thread.ID,
			Sequence:      0,
			Role:          conversation.RoleUser,
			Content:       fmt.Sprintf("Where is %s right now and who is on it?", refs.shipment.ProNumber),
			ScopeStage:    "Deterministic",
			ScopeCategory: "TransportationOperations",
			CreatedAt:     startedAt,
		},
		{
			ThreadID: thread.ID,
			Sequence: 1,
			Role:     conversation.RoleAssistant,
			ToolCalls: []conversation.ToolCallRecord{{
				ID:        callID,
				Name:      "get_shipment",
				Arguments: map[string]any{"proNumber": refs.shipment.ProNumber},
			}},
			Model:        seedConversationModel,
			InputTokens:  412,
			OutputTokens: 38,
			CreatedAt:    startedAt + 2,
		},
		{
			ThreadID:   thread.ID,
			Sequence:   2,
			Role:       conversation.RoleTool,
			ToolCallID: callID,
			ToolName:   "get_shipment",
			Content: fmt.Sprintf(
				`{"proNumber":%q,"status":%q,"id":%q}`,
				refs.shipment.ProNumber,
				refs.shipment.Status,
				refs.shipment.ID,
			),
			CreatedAt: startedAt + 3,
		},
		{
			ThreadID: thread.ID,
			Sequence: 3,
			Role:     conversation.RoleAssistant,
			Content: fmt.Sprintf(
				"%s is currently %s. I checked the shipment record directly; the assigned "+
					"driver and equipment are on the move tab if you want to open it.",
				refs.shipment.ProNumber,
				refs.shipment.Status,
			),
			ScopeStage:   "Output",
			Model:        seedConversationModel,
			InputTokens:  688,
			OutputTokens: 71,
			CreatedAt:    startedAt + 6,
		},
	}

	return s.insertMessages(ctx, tx, sc, refs, messages)
}

func (s *AssistantConversationSeed) seedBillingThread(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *conversationSeedRefs,
) error {
	startedAt := refs.now - 90*seedConversationMinute
	thread, err := s.insertThread(ctx, tx, sc, refs, refs.billing, threadSeed{
		title:     "Blocked invoices this week",
		startedAt: startedAt,
		turns:     4,
	})
	if err != nil {
		return err
	}

	messages := []*conversation.Message{
		{
			ThreadID:      thread.ID,
			Sequence:      0,
			Role:          conversation.RoleUser,
			Content:       "Write me a Python script that exports every invoice to a spreadsheet.",
			ScopeStage:    "Deterministic",
			ScopeCategory: "CodeGeneration",
			ScopeReason:   "CodeGeneration",
			Refused:       true,
			CreatedAt:     startedAt,
		},
		{
			ThreadID: thread.ID,
			Sequence: 1,
			Role:     conversation.RoleAssistant,
			Content: "I can't write code, but I can help with billing. Ask me about a blocked " +
				"invoice or a shipment and I'll look it up.",
			ScopeStage:    "Deterministic",
			ScopeCategory: "CodeGeneration",
			ScopeReason:   "CodeGeneration",
			Refused:       true,
			CreatedAt:     startedAt + 1,
		},
		{
			ThreadID: thread.ID,
			Sequence: 2,
			Role:     conversation.RoleUser,
			Content: fmt.Sprintf(
				"Fine. %s has been sitting in the billing queue for a week. What's holding it?",
				refs.shipment.ProNumber,
			),
			ScopeStage:    "Deterministic",
			ScopeCategory: "TransportationOperations",
			CreatedAt:     startedAt + 40*seedConversationMinute,
		},
		{
			ThreadID: thread.ID,
			Sequence: 3,
			Role:     conversation.RoleAssistant,
			Content: fmt.Sprintf(
				"%s is blocked because the signed bill of lading has not been attached, and "+
					"the rate on the shipment does not match the customer's rate agreement. I "+
					"can't fix the rate — that needs a person — but I can flag the item for "+
					"manual review with what I found so it does not sit any longer.",
				refs.shipment.ProNumber,
			),
			ScopeStage:   "Output",
			Model:        seedConversationModel,
			InputTokens:  1204,
			OutputTokens: 143,
			CreatedAt:    startedAt + 40*seedConversationMinute + 5,
		},
	}
	if err = s.insertMessages(ctx, tx, sc, refs, messages); err != nil {
		return err
	}

	return s.insertProposal(ctx, tx, sc, refs, thread, messages[3])
}

type threadSeed struct {
	title     string
	startedAt int64
	turns     int
}

func (s *AssistantConversationSeed) insertThread(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *conversationSeedRefs,
	definition *agentdefinition.Definition,
	seed threadSeed,
) (*conversation.Thread, error) {
	thread := &conversation.Thread{
		OrganizationID:    refs.org.ID,
		BusinessUnitID:    refs.org.BusinessUnitID,
		UserID:            refs.admin.ID,
		AgentDefinitionID: definition.ID,
		Title:             seed.title,
		Status:            conversation.ThreadStatusActive,
		LastMessageAt:     seed.startedAt + int64(seed.turns)*40*seedConversationMinute,
	}
	if _, err := tx.NewInsert().Model(thread).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert thread: %w", err)
	}
	if err := sc.TrackCreated(ctx, "assistant_threads", thread.ID, s.Name()); err != nil {
		return nil, err
	}

	return thread, nil
}

func (s *AssistantConversationSeed) insertMessages(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *conversationSeedRefs,
	messages []*conversation.Message,
) error {
	for _, message := range messages {
		message.OrganizationID = refs.org.ID
		message.BusinessUnitID = refs.org.BusinessUnitID
		if _, err := tx.NewInsert().Model(message).Exec(ctx); err != nil {
			return fmt.Errorf("insert message %d: %w", message.Sequence, err)
		}
		if err := sc.TrackCreated(ctx, "assistant_messages", message.ID, s.Name()); err != nil {
			return err
		}
	}

	return nil
}

// insertProposal records the write the billing assistant asked for, in the
// same shape the chat loop persists one: a run anchored to the thread, then a
// pending proposal citing the thread and the message that proposed it.
func (s *AssistantConversationSeed) insertProposal(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	refs *conversationSeedRefs,
	thread *conversation.Thread,
	source *conversation.Message,
) error {
	run := &agent.AgentRun{
		OrganizationID:   refs.org.ID,
		BusinessUnitID:   refs.org.BusinessUnitID,
		AgentType:        agent.TypeAssistantChat,
		SubjectType:      agent.SubjectAssistantThread,
		SubjectID:        thread.ID,
		Status:           agent.RunStatusAwaitingDecision,
		ModelIdentifier:  seedConversationModel,
		PromptVersion:    "assistant-chat/v1",
		InputContextHash: "seed-" + thread.ID.String(),
		StartedAt:        source.CreatedAt,
	}
	if _, err := tx.NewInsert().Model(run).Exec(ctx); err != nil {
		return fmt.Errorf("insert agent run: %w", err)
	}
	if err := sc.TrackCreated(ctx, "agent_runs", run.ID, s.Name()); err != nil {
		return err
	}

	proposal := &agent.AgentProposal{
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		RunID:          run.ID,
		ToolName:       "flag_for_manual_review",
		ToolParams: map[string]any{
			"runId":     run.ID.String(),
			"subjectId": refs.shipment.ID.String(),
			"category":  string(agent.CategoryMissingBOL),
			"severity":  string(agent.SeverityMedium),
			"attemptSummary": "Signed bill of lading is missing and the shipment rate " +
				"disagrees with the customer's rate agreement.",
			"evidence": []map[string]any{{
				"type": "Shipment",
				"id":   refs.shipment.ID.String(),
				"note": "Rate on shipment does not match the agreement on file",
			}},
		},
		Confidence: decimal.NewFromFloat(0.82),
		Rationale: "The item cannot be billed until a person resolves the rate, and " +
			"flagging it records what was found so the reviewer does not start over.",
		Evidence: []agent.EvidenceRef{
			{Type: "AssistantThread", ID: thread.ID.String()},
			{Type: "AssistantMessage", ID: source.ID.String()},
			{Type: "Shipment", ID: refs.shipment.ID.String(), Note: "Blocked in billing queue"},
		},
		AutonomyTier:    agent.TierPropose,
		Status:          agent.ProposalStatusPending,
		SourceMessageID: source.ID,
	}
	if _, err := tx.NewInsert().Model(proposal).Exec(ctx); err != nil {
		return fmt.Errorf("insert proposal: %w", err)
	}

	return sc.TrackCreated(ctx, "agent_proposals", proposal.ID, s.Name())
}

func (s *AssistantConversationSeed) Down(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			return seedhelpers.DeleteTrackedEntities(ctx, tx, s.Name(), sc)
		},
	)
}

func (s *AssistantConversationSeed) CanRollback() bool {
	return true
}
