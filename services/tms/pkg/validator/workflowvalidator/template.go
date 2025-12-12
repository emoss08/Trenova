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

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type TemplateValidator struct {
	factory *framework.TenantedValidatorFactory[*workflow.Template]
}

func NewTemplateValidator(p ValidatorParams) *TemplateValidator {
	factory := framework.NewTenantedValidatorFactory[*workflow.Template](
		func(ctx context.Context) (*bun.DB, error) {
			return p.DB.DB(ctx)
		},
	).
		WithModelName("WorkflowTemplate").
		WithUniqueFields(func(entity *workflow.Template) []framework.UniqueField {
			return []framework.UniqueField{
				{
					Name:     "name",
					GetValue: func() string { return entity.Name },
					Message:  "Workflow template with name ':value' already exists in the organization.",
				},
			}
		}).
		WithCustomRules(func(entity *workflow.Template, vc *validator.ValidationContext) []framework.ValidationRule {
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

			return rules
		})

	return &TemplateValidator{
		factory: factory,
	}
}

func (v *TemplateValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *workflow.Template,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
