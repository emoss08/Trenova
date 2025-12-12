package workflowvalidator

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/workflow"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/framework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ConnectionValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type ConnectionValidator struct {
	factory *framework.TenantedValidatorFactory[*workflow.Connection]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewConnectionValidator(p ConnectionValidatorParams) *ConnectionValidator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*workflow.Connection](
		getDB,
	).
		WithModelName("WorkflowConnection").
		WithCustomRules(func(entity *workflow.Connection, vc *validator.ValidationContext) []framework.ValidationRule {
			var rules []framework.ValidationRule

			if vc.IsCreate {
				rules = append(rules, framework.NewBusinessRule("id_validation").
					WithValidation(func(_ context.Context, multiErr *errortypes.MultiError) error {
						if entity.ID.IsNotNil() {
							multiErr.Add("id", errortypes.ErrInvalid, "ID cannot be set on create")
						}
						return nil
					}),
				)
			}

			// Validate nodes exist and belong to the same version
			rules = append(rules,
				framework.NewBusinessRule("nodes_exist_and_match_version").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateNodesExist(ctx, entity, me, getDB)
						return nil
					}),

				// Validate no circular dependencies
				framework.NewBusinessRule("no_circular_dependencies").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityMedium).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateNoCircularDependencies(ctx, entity, me, getDB)
						return nil
					}),
			)

			return rules
		})

	return &ConnectionValidator{
		factory: factory,
		getDB:   getDB,
	}
}

func validateNodesExist(
	ctx context.Context,
	entity *workflow.Connection,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Check source node exists and belongs to the same version
	var sourceNode workflow.Node
	err = db.NewSelect().
		Model(&sourceNode).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfn.id = ?", entity.SourceNodeID).
				Where("wfn.organization_id = ?", entity.OrganizationID).
				Where("wfn.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("sourceNodeId", errortypes.ErrInvalid, "Source node not found")
		return
	}

	// Check target node exists and belongs to the same version
	var targetNode workflow.Node
	err = db.NewSelect().
		Model(&targetNode).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfn.id = ?", entity.TargetNodeID).
				Where("wfn.organization_id = ?", entity.OrganizationID).
				Where("wfn.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("targetNodeId", errortypes.ErrInvalid, "Target node not found")
		return
	}

	// Ensure both nodes belong to the same version
	if sourceNode.WorkflowVersionID != entity.WorkflowVersionID {
		me.Add(
			"sourceNodeId",
			errortypes.ErrInvalid,
			"Source node does not belong to the specified workflow version",
		)
	}

	if targetNode.WorkflowVersionID != entity.WorkflowVersionID {
		me.Add(
			"targetNodeId",
			errortypes.ErrInvalid,
			"Target node does not belong to the specified workflow version",
		)
	}

	if sourceNode.WorkflowVersionID != targetNode.WorkflowVersionID {
		me.Add(
			"workflowVersionId",
			errortypes.ErrInvalid,
			"Source and target nodes must belong to the same workflow version",
		)
	}
}

func validateNoCircularDependencies(
	ctx context.Context,
	entity *workflow.Connection,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	// Check if creating this connection would create a circular dependency
	// We do this by checking if there's already a path from target to source
	// If we're adding source -> target, and target -> source already exists (directly or indirectly),
	// we have a cycle

	// For now, we'll just check for direct cycles (source -> target -> source)
	// A more comprehensive check would use graph traversal to detect any cycle
	count, err := db.NewSelect().
		Model((*workflow.Connection)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfc.workflow_version_id = ?", entity.WorkflowVersionID).
				Where("wfc.source_node_id = ?", entity.TargetNodeID).
				Where("wfc.target_node_id = ?", entity.SourceNodeID).
				Where("wfc.organization_id = ?", entity.OrganizationID).
				Where("wfc.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check for circular dependencies")
		return
	}

	if count > 0 {
		me.Add(
			"targetNodeId",
			errortypes.ErrInvalid,
			"Creating this connection would create a circular dependency",
		)
	}
}

func (v *ConnectionValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *workflow.Connection,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
