package policyrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
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

func NewRepository(p Params) ports.PolicyRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.policy-repository"),
	}
}

func (r *repository) GetByID(
	ctx context.Context,
	entityID pulid.ID,
) (*permission.Policy, error) {
	log := r.l.With(
		zap.String("operation", "GetByID"),
		zap.String("policyID", entityID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entity := new(permission.Policy)

	if err = db.NewSelect().
		Model(entity).
		Where("p.id = ?", entityID).
		Scan(ctx); err != nil {
		return nil, dberror.HandleNotFoundError(err, "Policy")
	}

	return entity, nil
}

func (r *repository) GetByBusinessUnit(
	ctx context.Context,
	businessUnitID pulid.ID,
) ([]*permission.Policy, error) {
	log := r.l.With(
		zap.String("operation", "GetByBusinessUnit"),
		zap.String("buID", businessUnitID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*permission.Policy, 0)

	if err = db.NewSelect().
		Model(&entities).
		Where("pol.scope->>'businessUnitId' = ?", businessUnitID.String()).
		Order("pol.priority DESC", "pol.created_at DESC").
		Scan(ctx); err != nil {
		log.Error("failed to scan policies", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetByOrganization(
	ctx context.Context,
	businessUnitID, organizationID pulid.ID,
) ([]*permission.Policy, error) {
	log := r.l.With(

		zap.String("operation", "GetByOrganization"),
		zap.String("buID", businessUnitID.String()),
		zap.String("orgID", organizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*permission.Policy, 0)

	if err = db.NewSelect().
		Model(&entities).
		Where("pol.scope->>'businessUnitId' = ?", businessUnitID.String()).
		Where(
			"(pol.scope->'organizationIds' = '[]'::jsonb OR pol.scope->'organizationIds' ? ?)",
			organizationID.String(),
		).
		Order("pol.priority DESC", "pol.created_at DESC").
		Scan(ctx); err != nil {
		log.Error("failed to scan policies", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) GetUserPolicies(
	ctx context.Context,
	userID, organizationID pulid.ID,
) ([]*permission.Policy, error) {
	log := r.l.With(
		zap.String("operation", "GetUserPolicies"),
		zap.String("userID", userID.String()),
		zap.String("orgID", organizationID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	policyIDs := make([]string, 0)

	if err = db.NewSelect().
		Column("policy_id").
		TableExpr("user_effective_policies").
		Where("user_id = ?", userID).
		Where("organization_id = ?", organizationID).
		Scan(ctx, &policyIDs); err != nil {
		log.Error("failed to get user policy IDs", zap.Error(err))
		return nil, err
	}

	log.Debug("found user policies", zap.Int("count", len(policyIDs)))

	if len(policyIDs) == 0 {
		return []*permission.Policy{}, nil
	}

	entities := make([]*permission.Policy, 0)

	if err = db.NewSelect().
		Model(&entities).
		Where("pol.id IN (?)", bun.In(policyIDs)).
		Order("pol.priority DESC", "pol.effect ASC"). // Deny policies first
		Scan(ctx); err != nil {
		log.Error("failed to scan policies", zap.Error(err))
		return nil, err
	}

	return entities, nil
}

func (r *repository) Create(
	ctx context.Context,
	entity *permission.Policy,
) error {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("policyID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	if _, err = db.NewInsert().
		Model(entity).
		Returning("*").
		Exec(ctx); err != nil {
		log.Error("failed to insert policy", zap.Error(err))
		return err
	}

	return nil
}

func (r *repository) Update(
	ctx context.Context,
	entity *permission.Policy,
) error {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("policyID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewUpdate().
		Model(entity).
		WherePK().
		Exec(ctx)
	if err != nil {
		log.Error("failed to update policy", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Policy", entity.ID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) Delete(
	ctx context.Context,
	entityID pulid.ID,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.String("policyID", entityID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	result, err := db.NewDelete().
		Model((*permission.Policy)(nil)).
		Where("id = ?", entityID).
		Exec(ctx)
	if err != nil {
		log.Error("failed to delete policy", zap.Error(err))
		return err
	}

	roErr := dberror.CheckRowsAffected(result, "Policy", entityID.String())
	if roErr != nil {
		return roErr
	}

	return nil
}

func (r *repository) GetResourcePolicies(
	ctx context.Context,
	businessUnitID pulid.ID,
	resourceType string,
) ([]*permission.Policy, error) {
	log := r.l.With(
		zap.String("operation", "GetResourcePolicies"),
		zap.String("buID", businessUnitID.String()),
		zap.String("resourceType", resourceType),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	entities := make([]*permission.Policy, 0)

	if err = db.NewSelect().
		Model(&entities).
		Where("pol.scope->>'businessUnitId' = ?", businessUnitID.String()).
		Where("pol.resources @> ?", bun.Safe("'[{\"resourceType\":\""+resourceType+"\"}]'")).
		Order("pol.priority DESC", "pol.created_at DESC").
		Scan(ctx); err != nil {
		log.Error("failed to scan policies", zap.Error(err))
		return nil, err
	}

	return entities, nil
}
