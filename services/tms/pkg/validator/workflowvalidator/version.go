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

type VersionValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type VersionValidator struct {
	factory *framework.TenantedValidatorFactory[*workflow.Version]
	getDB   func(context.Context) (*bun.DB, error)
}

func NewVersionValidator(p VersionValidatorParams) *VersionValidator {
	getDB := func(ctx context.Context) (*bun.DB, error) {
		return p.DB.DB(ctx)
	}

	factory := framework.NewTenantedValidatorFactory[*workflow.Version](
		getDB,
	).
		WithModelName("WorkflowVersion").
		WithCustomRules(func(entity *workflow.Version, vc *validator.ValidationContext) []framework.ValidationRule {
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

			// Validate template exists
			rules = append(rules,
				framework.NewBusinessRule("template_exists").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateTemplateExists(ctx, entity, me, getDB)
						return nil
					}),
			)

			// Validate version number uniqueness
			rules = append(rules,
				framework.NewBusinessRule("version_number_unique").
					WithStage(framework.ValidationStageDataIntegrity).
					WithPriority(framework.ValidationPriorityHigh).
					WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
						validateVersionNumberUnique(ctx, entity, me, getDB, vc.IsCreate)
						return nil
					}),
			)

			// Validate only one published version per template
			if entity.VersionStatus == workflow.VersionStatusPublished {
				rules = append(rules,
					framework.NewBusinessRule("single_published_version").
						WithStage(framework.ValidationStageDataIntegrity).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							validateSinglePublishedVersion(ctx, entity, me, getDB, vc.IsCreate)
							return nil
						}),
				)
			}

			// Validate version can be edited (only Draft versions)
			if vc.IsUpdate {
				rules = append(rules,
					framework.NewBusinessRule("version_editable").
						WithStage(framework.ValidationStageBusinessRules).
						WithPriority(framework.ValidationPriorityHigh).
						WithValidation(func(ctx context.Context, me *errortypes.MultiError) error {
							validateVersionEditable(ctx, entity, me, getDB)
							return nil
						}),
				)
			}

			// Validate trigger-specific configurations
			rules = append(rules,
				framework.NewBusinessRule("trigger_configuration").
					WithStage(framework.ValidationStageCompliance).
					WithPriority(framework.ValidationPriorityMedium).
					WithValidation(func(_ context.Context, me *errortypes.MultiError) error {
						validateTriggerConfig(entity, me)
						return nil
					}),
			)

			return rules
		})

	return &VersionValidator{
		factory: factory,
		getDB:   getDB,
	}
}

func validateTemplateExists(
	ctx context.Context,
	entity *workflow.Version,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	count, err := db.NewSelect().
		Model((*workflow.Template)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wft.id = ?", entity.WorkflowTemplateID).
				Where("wft.organization_id = ?", entity.OrganizationID).
				Where("wft.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check template existence")
		return
	}

	if count == 0 {
		me.Add("workflowTemplateId", errortypes.ErrInvalid, "Workflow template not found")
	}
}

func validateVersionNumberUnique(
	ctx context.Context,
	entity *workflow.Version,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
	isCreate bool,
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	query := db.NewSelect().
		Model((*workflow.Version)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.workflow_template_id = ?", entity.WorkflowTemplateID).
				Where("wfv.version_number = ?", entity.VersionNumber).
				Where("wfv.organization_id = ?", entity.OrganizationID).
				Where("wfv.business_unit_id = ?", entity.BusinessUnitID)
		})

	// On update, exclude the current version
	if !isCreate {
		query = query.Where("wfv.id != ?", entity.ID)
	}

	count, err := query.Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check version number uniqueness")
		return
	}

	if count > 0 {
		me.Add("versionNumber", errortypes.ErrInvalid, "Version number already exists for this template")
	}
}

func validateSinglePublishedVersion(
	ctx context.Context,
	entity *workflow.Version,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
	isCreate bool,
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	query := db.NewSelect().
		Model((*workflow.Version)(nil)).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.workflow_template_id = ?", entity.WorkflowTemplateID).
				Where("wfv.version_status = ?", workflow.VersionStatusPublished).
				Where("wfv.organization_id = ?", entity.OrganizationID).
				Where("wfv.business_unit_id = ?", entity.BusinessUnitID)
		})

	// On update, exclude the current version
	if !isCreate {
		query = query.Where("wfv.id != ?", entity.ID)
	}

	count, err := query.Count(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check published version uniqueness")
		return
	}

	if count > 0 {
		me.Add("versionStatus", errortypes.ErrInvalid,
			"Another version is already published for this template. Please archive it first.")
	}
}

func validateVersionEditable(
	ctx context.Context,
	entity *workflow.Version,
	me *errortypes.MultiError,
	getDB func(context.Context) (*bun.DB, error),
) {
	db, err := getDB(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Database connection error")
		return
	}

	var currentVersion workflow.Version
	err = db.NewSelect().
		Model(&currentVersion).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where("wfv.id = ?", entity.ID).
				Where("wfv.organization_id = ?", entity.OrganizationID).
				Where("wfv.business_unit_id = ?", entity.BusinessUnitID)
		}).
		Scan(ctx)
	if err != nil {
		me.Add("__all__", errortypes.ErrSystemError, "Failed to check current version status")
		return
	}

	// Only Draft versions can be edited
	if !currentVersion.VersionStatus.IsDraft() {
		me.Add("versionStatus", errortypes.ErrInvalid,
			"Only Draft versions can be edited. Published or Archived versions are immutable.")
	}
}

func validateTriggerConfig(entity *workflow.Version, me *errortypes.MultiError) {
	switch entity.TriggerType {
	case workflow.TriggerTypeScheduled:
		if len(entity.ScheduleConfig) == 0 {
			me.Add(
				"scheduleConfig",
				errortypes.ErrInvalid,
				"Schedule configuration is required when trigger type is Scheduled",
			)
		}
		// Validate cron expression if present
		if cronExpr, ok := entity.ScheduleConfig["cronExpression"].(string); ok && cronExpr == "" {
			me.Add(
				"scheduleConfig.cronExpression",
				errortypes.ErrInvalid,
				"Cron expression is required for scheduled workflows",
			)
		}

	case workflow.TriggerTypeEvent:
		if len(entity.TriggerConfig) == 0 {
			me.Add(
				"triggerConfig",
				errortypes.ErrInvalid,
				"Trigger configuration is required when trigger type is Event",
			)
		}
		// Validate event type if present
		if eventType, ok := entity.TriggerConfig["eventType"].(string); ok && eventType == "" {
			me.Add(
				"triggerConfig.eventType",
				errortypes.ErrInvalid,
				"Event type is required for event-triggered workflows",
			)
		}
	}
}

func (v *VersionValidator) Validate(
	ctx context.Context,
	valCtx *validator.ValidationContext,
	entity *workflow.Version,
) *errortypes.MultiError {
	return v.factory.Validate(ctx, entity, valCtx)
}
