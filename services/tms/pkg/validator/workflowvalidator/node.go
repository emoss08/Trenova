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

type NodeValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type NodeValidator struct {
	factory *framework.TenantedValidatorFactory[*workflow.Node]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewNodeValidator(p NodeValidatorParams) *NodeValidator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*workflow.Node](
		getDB,
	).
		WithModelName("WorkflowNode").
		WithCustomRules(func(entity *workflow.Node, vc *validator.ValidationContext) []framework.ValidationRule {
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

			// Validate version exists
			rules = append(rules,
				framework.NewBusinessRule("version_exists").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateVersionExists(ctx, entity, me, getDB)
						return nil
					}),

				// Validate node configuration based on type
				framework.NewBusinessRule("node_config_validation").
					WithStage(framework.ValidationStageCompliance).
					WithPriority(framework.ValidationPriorityMedium).
					WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
						validateNodeConfig(entity, me)
						return nil
					}),
			)

			return rules
		})

	return &NodeValidator{
		factory: factory,
		getDB:   getDB,
	}
}

func validateVersionExists(
	ctx context.Context,
	entity *workflow.Node,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	count, err := db.NewSelect().
		Model((*workflow.Version)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", entity.WorkflowVersionID).
				Where("wfv.organization_id = ?", entity.OrganizationID).
				Where("wfv.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check workflow version existence")
		return
	}

	if count == 0 {
		me.Add("workflowVersionId", errortypes.ErrInvalid, "Workflow version not found")
	}
}

func validateNodeConfig(entity *workflow.Node, me *errortypes.MultiError) {
	if len(entity.Config) == 0 {
		me.Add("config", errortypes.ErrInvalid, "Node configuration is required")
		return
	}

	switch entity.NodeType {
	case workflow.NodeTypeEntityUpdate:
		// Validate entity update node configuration
		if entityType, ok := entity.Config["entityType"].(string); !ok || entityType == "" {
			me.Add(
				"config.entityType",
				errortypes.ErrInvalid,
				"Entity type is required for entity update nodes",
			)
		}
		if fieldMappings, ok := entity.Config["fieldMappings"].(map[string]any); !ok || len(fieldMappings) == 0 {
			me.Add(
				"config.fieldMappings",
				errortypes.ErrInvalid,
				"Field mappings are required for entity update nodes",
			)
		}

	case workflow.NodeTypeCondition:
		// Validate condition node configuration
		if conditions, ok := entity.Config["conditions"].([]any); !ok || len(conditions) == 0 {
			me.Add(
				"config.conditions",
				errortypes.ErrInvalid,
				"Conditions are required for condition nodes",
			)
		}

	case workflow.NodeTypeTrigger:
		// Trigger nodes generally don't need specific config validation
		// but we can add validation here if needed
	}
}

func (v *NodeValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *workflow.Node,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
