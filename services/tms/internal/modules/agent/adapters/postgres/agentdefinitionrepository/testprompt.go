package agentdefinitionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition/agentdefinitiondb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttestprompt"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/pkg/dberror"
	"go.uber.org/fx"
)

type TestPromptParams struct {
	fx.In

	DB *postgres.Connection
}

type testPromptRepository struct {
	db *postgres.Connection
}

func NewTestPromptRepository(p TestPromptParams) agentdefinitionports.AgentTestPromptRepository {
	return &testPromptRepository{db: p.DB}
}

func (r *testPromptRepository) List(
	ctx context.Context,
	req *agentdefinitionports.ListAgentTestPromptsRequest,
) ([]*agenttestprompt.TestPrompt, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agenttestprompt.TestPrompt, error) {
		p := agentdefinitiondb.TestPrompts
		prompts, err := p.Where(p.AgentDefinitionID.Eq(req.AgentDefinitionID)).
			Order(p.CreatedAt.Asc(), p.ID.Asc()).
			Limit(agenttestprompt.MaxTestPrompts).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent test prompts: %w", err)
		}
		return prompts, nil
	})
}

func (r *testPromptRepository) Count(
	ctx context.Context,
	req *agentdefinitionports.ListAgentTestPromptsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		p := agentdefinitiondb.TestPrompts
		count, err := p.Where(p.AgentDefinitionID.Eq(req.AgentDefinitionID)).
			Count(ctx, r.db.Keel(ctx))
		if err != nil {
			return 0, fmt.Errorf("count agent test prompts: %w", err)
		}
		return int(count), nil
	})
}

func (r *testPromptRepository) Create(
	ctx context.Context,
	prompt *agenttestprompt.TestPrompt,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if err := agentdefinitiondb.TestPrompts.Create(ctx, r.db.Keel(ctx), prompt); err != nil {
			return fmt.Errorf("keep agent test prompt: %w", err)
		}
		return nil
	})
}

func (r *testPromptRepository) Delete(
	ctx context.Context,
	req *agentdefinitionports.DeleteAgentTestPromptRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		p := agentdefinitiondb.TestPrompts
		n, err := p.Where(p.AgentDefinitionID.Eq(req.AgentDefinitionID), p.ID.Eq(req.ID)).
			Delete(ctx, r.db.Keel(ctx))
		if err != nil {
			return fmt.Errorf("delete agent test prompt: %w", err)
		}
		return dberror.CheckAffected(n, "Agent test prompt", req.ID.String())
	})
}
