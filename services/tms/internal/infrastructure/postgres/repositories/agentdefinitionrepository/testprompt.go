package agentdefinitionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type TestPromptParams struct {
	fx.In

	DB *postgres.Connection
}

type testPromptRepository struct {
	db *postgres.Connection
}

func NewTestPromptRepository(p TestPromptParams) repositories.AgentTestPromptRepository {
	return &testPromptRepository{db: p.DB}
}

func (r *testPromptRepository) List(
	ctx context.Context,
	req *repositories.ListAgentTestPromptsRequest,
) ([]*agentdefinition.TestPrompt, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentdefinition.TestPrompt, error) {
		cols := buncolgen.TestPromptColumns
		prompts := make([]*agentdefinition.TestPrompt, 0, agentdefinition.MaxTestPrompts)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&prompts).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TestPromptScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
			}).
			Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(agentdefinition.MaxTestPrompts).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list agent test prompts: %w", err)
		}
		return prompts, nil
	})
}

func (r *testPromptRepository) Count(
	ctx context.Context,
	req *repositories.ListAgentTestPromptsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := buncolgen.TestPromptColumns
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agentdefinition.TestPrompt)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.TestPromptScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
			}).
			Count(ctx)
		if err != nil {
			return 0, fmt.Errorf("count agent test prompts: %w", err)
		}
		return count, nil
	})
}

func (r *testPromptRepository) Create(ctx context.Context, prompt *agentdefinition.TestPrompt) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(prompt).Exec(ctx); err != nil {
			return fmt.Errorf("keep agent test prompt: %w", err)
		}
		return nil
	})
}

func (r *testPromptRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteAgentTestPromptRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := buncolgen.TestPromptColumns
		result, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*agentdefinition.TestPrompt)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return buncolgen.TestPromptScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID).
					Where(cols.ID.Eq(), req.ID)
			}).
			Exec(ctx)
		if err != nil {
			return fmt.Errorf("delete agent test prompt: %w", err)
		}
		return dberror.CheckRowsAffected(result, "Agent test prompt", req.ID.String())
	})
}
