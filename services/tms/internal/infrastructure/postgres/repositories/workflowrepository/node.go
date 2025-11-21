package workflowrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/maputils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type NodeParams struct {
	fx.In

	DB             *postgres.Connection
	ConnectionRepo repositories.WorkflowConnectionRepository
	Logger         *zap.Logger
}

type nodeRepository struct {
	db       *postgres.Connection
	connRepo repositories.WorkflowConnectionRepository
	l        *zap.Logger
}

func NewNodeRepository(p NodeParams) repositories.WorkflowNodeRepository {
	return &nodeRepository{
		db:       p.DB,
		connRepo: p.ConnectionRepo,
		l:        p.Logger.Named("postgres.workflow-node-repository"),
	}
}

func (r *nodeRepository) GetByVersionID(
	ctx context.Context,
	req *repositories.GetNodesByVersionIDRequest,
) ([]*workflow.Node, error) {
	log := r.l.With(
		zap.String("operation", "GetNodesByVersionID"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return nil, err
	}

	nodes := make([]*workflow.Node, 0)
	err = db.NewSelect().Model(&nodes).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfn.workflow_version_id = ?", req.VersionID).
				Where("wfn.organization_id = ?", req.OrgID).
				Where("wfn.business_unit_id = ?", req.BuID)
		}).Scan(ctx)
	if err != nil {
		log.Error("failed to scan workflow nodes", zap.Error(err))
		return nil, err
	}

	return nodes, nil
}

func (r *nodeRepository) Create(
	ctx context.Context,
	entity *workflow.Node,
) (*workflow.Node, error) {
	log := r.l.With(
		zap.String("operation", "Create"),
		zap.String("versionID", entity.WorkflowVersionID.String()),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		return nil, err
	}

	if _, err = db.NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to insert node", zap.Error(err))
		return nil, err
	}

	return entity, nil
}

func (r *nodeRepository) Clone(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.CloneNodesRequest,
) (map[pulid.ID]pulid.ID, error) {
	log := r.l.With(
		zap.String("operation", "CloneNodes"),
		zap.String("versionID", req.VersionID.String()),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	if len(req.SourceNodes) == 0 {
		return make(map[pulid.ID]pulid.ID), nil
	}

	newNodes := make([]*workflow.Node, len(req.SourceNodes))
	oldIDs := make([]pulid.ID, len(req.SourceNodes))

	for i, node := range req.SourceNodes {
		oldIDs[i] = node.ID
		newNodes[i] = &workflow.Node{
			OrganizationID:    req.OrgID,
			BusinessUnitID:    req.BuID,
			WorkflowVersionID: req.VersionID,
			Name:              node.Name,
			NodeType:          node.NodeType,
			Config:            node.Config,
			PositionX:         node.PositionX,
			PositionY:         node.PositionY,
		}
	}

	if _, err := tx.NewInsert().Model(&newNodes).Returning("*").Exec(ctx); err != nil {
		log.Error("failed to bulk insert cloned nodes", zap.Error(err))
		return nil, err
	}

	nodeIDMap := make(map[pulid.ID]pulid.ID, len(newNodes))
	for i, newNode := range newNodes {
		nodeIDMap[oldIDs[i]] = newNode.ID
	}

	return nodeIDMap, nil
}

func (r *nodeRepository) CloneVersionNodes(
	ctx context.Context,
	tx bun.IDB,
	req *repositories.CloneVersionNodesRequest,
) error {
	nodeIDMap, err := r.Clone(ctx, tx, &repositories.CloneNodesRequest{
		SourceNodes: req.SourceVersion.Nodes,
		VersionID:   req.NewVersionID,
		OrgID:       req.OrgID,
		BuID:        req.BuID,
	})
	if err != nil {
		return err
	}

	return r.connRepo.Clone(
		ctx,
		tx,
		&repositories.CloneConnectionRequest{
			SourceConnections: req.SourceVersion.Connections,
			VersionID:         req.NewVersionID,
			OrgID:             req.OrgID,
			BuID:              req.BuID,
			NodeIDMap:         nodeIDMap,
		},
	)
}

func (r *nodeRepository) Update(
	ctx context.Context,
	entity *workflow.Node,
) (*workflow.Node, error) {
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
		Where("wfn.version = ?", ov).
		Returning("*").
		Exec(ctx)
	if rErr != nil {
		log.Error("failed to update workflow node", zap.Error(rErr))
		return nil, rErr
	}

	roErr := dberror.CheckRowsAffected(results, "Workflow Node", entity.ID.String())
	if roErr != nil {
		return nil, roErr
	}

	return entity, nil
}

func (r *nodeRepository) ImportNodes(
	ctx context.Context,
	req *repositories.ImportNodesRequest,
) ([]pulid.ID, error) {
	log := r.l.With(
		zap.String("operation", "ImportNodes"),
		zap.String("versionID", req.VersionID.String()),
		zap.String("orgID", req.OrgID.String()),
		zap.String("buID", req.BuID.String()),
	)

	nodeIDs := make([]pulid.ID, len(req.Nodes))

	for i, n := range req.Nodes {
		nodeData, ok := n.(map[string]any)
		if !ok {
			return nil, errortypes.NewValidationError(
				"node",
				errortypes.ErrInvalid,
				"invalid node data",
			)
		}

		name, err := maputils.GetString(nodeData, "name")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.name",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		nodeType, err := maputils.GetString(nodeData, "nodeType")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.nodeType",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		config, err := maputils.GetMap(nodeData, "config")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.config",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		position, err := maputils.GetMap(nodeData, "position")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.position",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		posX, err := maputils.GetInt(position, "x")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.position.x",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		posY, err := maputils.GetInt(position, "y")
		if err != nil {
			return nil, errortypes.NewValidationError(
				"node.position.y",
				errortypes.ErrInvalid,
				err.Error(),
			)
		}

		newNode := &workflow.Node{
			OrganizationID:    req.OrgID,
			BusinessUnitID:    req.BuID,
			WorkflowVersionID: req.VersionID,
			Name:              name,
			NodeType:          workflow.NodeType(nodeType),
			Config:            config,
			PositionX:         posX,
			PositionY:         posY,
		}

		if _, err = r.Create(ctx, newNode); err != nil {
			log.Error("failed to create imported node", zap.Error(err))
			return nil, err
		}

		nodeIDs[i] = newNode.ID
	}

	return nodeIDs, nil
}

func (r *nodeRepository) Delete(
	ctx context.Context,
	req *repositories.DeleteNodeRequest,
) error {
	log := r.l.With(
		zap.String("operation", "DeleteNode"),
		zap.Any("req", req),
	)

	db, err := r.db.DB(ctx)
	if err != nil {
		log.Error("failed to get database connection", zap.Error(err))
		return err
	}

	results, dErr := db.NewDelete().
		Model((*workflow.Node)(nil)).
		WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
			return dq.Where("wfn.id = ?", req.NodeID).
				Where("wfn.organization_id = ?", req.OrgID).
				Where("wfn.business_unit_id = ?", req.BuID)
		}).
		Exec(ctx)
	if dErr != nil {
		log.Error("failed to delete workflow node", zap.Error(dErr))
		return dErr
	}

	return dberror.CheckRowsAffected(results, "Workflow Node", req.NodeID.String())
}
