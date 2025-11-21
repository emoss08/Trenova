package workflowrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/utils/maputils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ConnectionParams struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type connectionRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewConnectionRepository(p ConnectionParams) repositories.WorkflowConnectionRepository {
	return &connectionRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.workflow-connection-repository"),
	}
}

func (r *connectionRepository) Clone(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.CloneConnectionRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Clone"),
		zap.Any("req", req),
	)

	newConnections := make([]*workflow.Connection, 0, len(req.SourceConnections))

	for _, conn := range req.SourceConnections {
		newConnections = append(newConnections, &workflow.Connection{
			OrganizationID:    req.OrgID,
			BusinessUnitID:    req.BuID,
			WorkflowVersionID: req.VersionID,
			SourceNodeID:      req.NodeIDMap[conn.SourceNodeID],
			TargetNodeID:      req.NodeIDMap[conn.TargetNodeID],
			Condition:         conn.Condition,
			IsDefaultBranch:   conn.IsDefaultBranch,
		})
	}

	if _, err := tx.NewInsert().Model(&newConnections).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to bulk insert cloned connections", zap.Error(err))
		return err
	}
	return nil
}

func (r *connectionRepository) GetByVersionID(
	ctx context.Context,
	req *repositories.GetConnectionsByVersionIDRequest,
) ([]*workflow.Connection, error) {
	log := r.l.With(
		zap.String("operation", "GetByVersionID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	connections := make([]*workflow.Connection, 0)
	err = db.NewSelect().Model(&connections).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfc.workflow_version_id = ?", req.VersionID).
				Where("wfc.organization_id = ?", req.OrgID).
				Where("wfc.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		log.Error("failed to scan workflow connections", zap.Error(err))
		return nil, err
	}

	return connections, nil
}

func (r *connectionRepository) Create(
	ctx context.Context,
	entity *workflow.Connection,
) (*workflow.Connection, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("versionID", entity.WorkflowVersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert workflow connection", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *connectionRepository) Update(
	ctx context.Context,
	entity *workflow.Connection,
) (*workflow.Connection, error) {
	log := r.l.With(
		zap.String("operation", "Update"),
		zap.String("entityID", entity.ID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	ov := entity.Version
	entity.Version++

	results, rErr := db.NewUpdate().
		Model(entity).
		WherePK().
		Where("wfc.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow connection", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Connection", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *connectionRepository) ImportConnections(
	ctx context.Context,
	req *repositories.ImportConnectionsRequest,
) error {
	log := r.l.With(
		zap.String("operation", "ImportConnections"),
		zap.Any("req", req),
	)

	for _, c := range req.Connections {
		connData, ok := c.(map[string]any)
		if !ok {
			continue
		}

		sourceIdx, err := maputils.GetInt(connData, "sourceNodeIndex")
		if err != nil {
			return errortypes.NewValidationError(
				"sourceNodeIndex",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		targetIdx, err := maputils.GetInt(connData, "targetNodeIndex")
		if err != nil {
			return errortypes.NewValidationError(
				"targetNodeIndex",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		condition, err := maputils.GetMap(connData, "condition")
		if err != nil {
			return errortypes.NewValidationError(
				"condition",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		isDefaultBranch, err := maputils.GetBool(connData, "isDefaultBranch")
		if err != nil {
			return errortypes.NewValidationError(
				"isDefaultBranch",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		newConn := &workflow.Connection{
			OrganizationID:    req.OrgID,
			BusinessUnitID:    req.BuID,
			WorkflowVersionID: req.VersionID,
			SourceNodeID:      req.NodeIDs[sourceIdx],
			TargetNodeID:      req.NodeIDs[targetIdx],
			Condition:         condition,
			IsDefaultBranch:   isDefaultBranch,
		}

		if _, err = r.Create(ctx, newConn); err != nil {
			log.Error("failed to create imported connection", zap.Error(err))
			return err
		}
	}

	return nil
}

func (r *connectionRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteConnectionRequest,
) error {
	log := r.l.With(
		zap.String("operation", "Delete"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	entity := &workflow.Connection{
		ID:             req.ConnectionID,
		OrganizationID: req.OrgID,
		BusinessUnitID: req.BuID,
	}

	results, dErr := db.NewDelete().Model(entity).WherePK().Exec(ctx)
	if dErr != nil {
		log.Error("failed to delete workflow connection", zap.Error(dErr))
		return dErr
	}

	return dberror.CheckRowsAffected(results, "Workflow Connection", req.ConnectionID.String())
}
