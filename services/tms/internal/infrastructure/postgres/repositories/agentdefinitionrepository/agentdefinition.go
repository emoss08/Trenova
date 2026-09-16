package agentdefinitionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AgentDefinitionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentdefinition-repository"),
	}
}

func (r *repository) List(
	ctx context.Context,
	req *repositories.ListAgentDefinitionRequest,
) (*pagination.ListResult[*agentdefinition.Definition], error) {
	cols := buncolgen.DefinitionColumns

	entities := make([]*agentdefinition.Definition, 0, req.Filter.Pagination.SafeLimit())
	total, err := r.db.DB().
		NewSelect().
		Model(&entities).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = querybuilder.ApplyFilters(
				sq,
				buncolgen.DefinitionTable.Alias,
				req.Filter,
				(*agentdefinition.Definition)(nil),
			)
			sq = sq.Apply(buncolgen.DefinitionApplyTenant(req.Filter.TenantInfo))
			if req.EnabledOnly {
				sq = sq.Where(cols.Enabled.IsTrue())
			}

			return sq.
				Limit(req.Filter.Pagination.SafeLimit()).
				Offset(req.Filter.Pagination.SafeOffset()).
				Order(cols.Name.OrderAsc())
		}).ScanAndCount(ctx)
	if err != nil {
		r.l.Error("failed to list agent definitions", zap.Error(err))
		return nil, err
	}

	return &pagination.ListResult[*agentdefinition.Definition]{Items: entities, Total: total}, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentDefinitionByIDRequest,
) (*agentdefinition.Definition, error) {
	entity := new(agentdefinition.Definition)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.DefinitionScopeTenant(sq, req.TenantInfo).
				Where(buncolgen.DefinitionColumns.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentDefinition")
	}

	return entity, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (*agentdefinition.Definition, error) {
	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateName(entity.Name)
		}
		r.l.Error("failed to create agent definition", zap.Error(err))

		return nil, err
	}

	return entity, nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *agentdefinition.Definition,
) (*agentdefinition.Definition, error) {
	cols := buncolgen.DefinitionColumns
	ov := entity.Version
	entity.Version++

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.DefinitionScopeTenantUpdate(uq, pagination.TenantInfo{
				OrgID: entity.OrganizationID,
				BuID:  entity.BusinessUnitID,
			}).Where(cols.ID.Eq(), entity.ID).
				Where(cols.Version.Eq(), ov)
		}).
		Set(cols.Name.Set(), entity.Name).
		Set(cols.Description.Set(), entity.Description).
		Set(cols.Kind.Set(), entity.Kind).
		Set(cols.Focus.Set(), entity.Focus).
		Set(cols.ToolNames.Set(), entity.ToolNames).
		Set(cols.AutonomyCeiling.Set(), entity.AutonomyCeiling).
		Set(cols.Enabled.Set(), entity.Enabled).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Set(cols.Version.Set(), entity.Version).
		Exec(ctx)
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return nil, duplicateName(entity.Name)
		}

		return nil, fmt.Errorf("update agent definition: %w", err)
	}

	if err = dberror.CheckRowsAffected(res, "AgentDefinition", entity.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

func (r *repository) Delete(
	ctx context.Context,
	req repositories.DeleteAgentDefinitionRequest,
) error {
	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*agentdefinition.Definition)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return buncolgen.DefinitionScopeTenantDelete(dq, req.TenantInfo).
				Where(buncolgen.DefinitionColumns.ID.Eq(), req.ID)
		}).
		Exec(ctx)
	if err != nil {
		return fmt.Errorf("delete agent definition: %w", err)
	}

	return dberror.CheckRowsAffected(res, "AgentDefinition", req.ID.String())
}

func duplicateName(name string) error {
	multiErr := errortypes.NewMultiError()
	multiErr.Add(
		"name",
		errortypes.ErrDuplicate,
		fmt.Sprintf("An agent named %q already exists", name),
	)

	return multiErr
}
