package customfieldservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customfield"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const MaxCustomFieldsPerResourceType = 50

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.CustomFieldDefinitionRepository
	ValueRepo    repositories.CustomFieldValueRepository
	Validator    *Validator
	AuditService services.AuditService
}

type Service struct {
	l            *zap.Logger
	repo         repositories.CustomFieldDefinitionRepository
	valueRepo    repositories.CustomFieldValueRepository
	validator    *Validator
	auditService services.AuditService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.customfield"),
		repo:         p.Repo,
		valueRepo:    p.ValueRepo,
		validator:    p.Validator,
		auditService: p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListCustomFieldDefinitionsRequest,
) (*pagination.ListResult[*customfield.CustomFieldDefinition], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req repositories.GetCustomFieldDefinitionByIDRequest,
) (*customfield.CustomFieldDefinition, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) GetActiveByResourceType(
	ctx context.Context,
	req repositories.GetActiveByResourceTypeRequest,
) ([]*customfield.CustomFieldDefinition, error) {
	return s.repo.GetActiveByResourceType(ctx, req)
}

func (s *Service) GetSupportedResourceTypes() []string {
	return customfield.GetSupportedResourceTypes()
}

func (s *Service) Create(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
	userID pulid.ID,
) (*customfield.CustomFieldDefinition, error) {
	count, err := s.repo.CountByResourceType(ctx, repositories.CountByResourceTypeRequest{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		ResourceType: entity.ResourceType,
	})
	if err != nil {
		s.l.Error("failed to count custom fields", zap.Error(err))
		return nil, err
	}

	if count >= MaxCustomFieldsPerResourceType {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("resourceType", errortypes.ErrInvalid,
			"Maximum number of custom fields reached for this resource type")
		return nil, multiErr
	}

	if multiErr := s.validator.ValidateCreate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create custom field definition", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCustomFieldDefinition,
		ResourceID:     createdEntity.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(createdEntity),
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
	},
		auditservice.WithComment("Custom field definition created"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *customfield.CustomFieldDefinition,
	userID pulid.ID,
) (*customfield.CustomFieldDefinition, error) {
	if multiErr := s.validator.ValidateUpdate(ctx, entity); multiErr != nil {
		return nil, multiErr
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: entity.GetOrganizationID(),
		BuID:  entity.GetBusinessUnitID(),
	}

	original, err := s.repo.GetByID(ctx, repositories.GetCustomFieldDefinitionByIDRequest{
		ID:         entity.GetID(),
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Error("failed to get original custom field definition", zap.Error(err))
		return nil, err
	}

	usageStats, err := s.GetUsageStats(ctx, entity.GetID(), tenantInfo)
	if err != nil {
		s.l.Error("failed to get usage stats for update check", zap.Error(err))
		return nil, err
	}

	breakingChanges := s.detectBreakingChanges(original, entity, usageStats)
	if breakingChanges.HasBlockingChanges {
		multiErr := errortypes.NewMultiError()
		for _, change := range breakingChanges.Changes {
			if change.ChangeType == customfield.BreakingChangeTypeBlocked {
				multiErr.Add(change.Field, errortypes.ErrBreakingChange, change.Message)
			}
		}
		return nil, multiErr
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		s.l.Error("failed to update custom field definition", zap.Error(err))
		return nil, err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCustomFieldDefinition,
		ResourceID:     updatedEntity.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(updatedEntity),
		PreviousState:  jsonutils.MustToJSON(original),
		OrganizationID: updatedEntity.OrganizationID,
		BusinessUnitID: updatedEntity.BusinessUnitID,
	},
		auditservice.WithComment("Custom field definition updated"),
		auditservice.WithDiff(original, updatedEntity),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) Delete(
	ctx context.Context,
	req repositories.GetCustomFieldDefinitionByIDRequest,
	userID pulid.ID,
) error {
	definition, err := s.repo.GetByID(ctx, req)
	if err != nil {
		s.l.Error("failed to get custom field definition for delete", zap.Error(err))
		return err
	}

	usageStats, err := s.GetUsageStats(ctx, req.ID, req.TenantInfo)
	if err != nil {
		s.l.Error("failed to get usage stats for delete check", zap.Error(err))
		return err
	}

	if usageStats.TotalValueCount > 0 {
		return errortypes.NewConflictError(
			fmt.Sprintf(
				"Cannot delete custom field definition: %d values exist across %d resources. Deactivate the field instead to preserve data.",
				usageStats.TotalValueCount,
				usageStats.ResourceCount,
			),
		).WithUsageStats(usageStats)
	}

	if err = s.repo.Delete(ctx, req); err != nil {
		s.l.Error("failed to delete custom field definition", zap.Error(err))
		return err
	}

	if err = s.auditService.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceCustomFieldDefinition,
		ResourceID:     definition.GetID().String(),
		Operation:      permission.OpDelete,
		UserID:         userID,
		PreviousState:  jsonutils.MustToJSON(definition),
		OrganizationID: definition.OrganizationID,
		BusinessUnitID: definition.BusinessUnitID,
	},
		auditservice.WithComment("Custom field definition deleted"),
	); err != nil {
		s.l.Error("failed to log audit action", zap.Error(err))
	}

	return nil
}

func (s *Service) GetUsageStats(
	ctx context.Context,
	definitionID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*customfield.DefinitionUsageStats, error) {
	valueReq := &repositories.GetValuesByDefinitionRequest{
		TenantInfo:   tenantInfo,
		DefinitionID: definitionID,
	}

	totalCount, err := s.valueRepo.CountByDefinition(ctx, valueReq)
	if err != nil {
		return nil, err
	}

	resourceCount, err := s.valueRepo.CountResourcesByDefinition(ctx, valueReq)
	if err != nil {
		return nil, err
	}

	stats := &customfield.DefinitionUsageStats{
		DefinitionID:    definitionID,
		TotalValueCount: totalCount,
		ResourceCount:   resourceCount,
	}

	if totalCount > 0 {
		s.enrichWithOptionUsage(ctx, stats, definitionID, tenantInfo)
	}

	return stats, nil
}

func (s *Service) enrichWithOptionUsage(
	ctx context.Context,
	stats *customfield.DefinitionUsageStats,
	definitionID pulid.ID,
	tenantInfo pagination.TenantInfo,
) {
	definition, err := s.repo.GetByID(ctx, repositories.GetCustomFieldDefinitionByIDRequest{
		ID:         definitionID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("failed to get definition for option usage enrichment", zap.Error(err))
		return
	}

	if !definition.FieldType.RequiresOptions() || len(definition.Options) == 0 {
		return
	}

	optionUsage, err := s.valueRepo.GetOptionUsageCounts(
		ctx,
		&repositories.GetOptionUsageRequest{
			TenantInfo:   tenantInfo,
			DefinitionID: definitionID,
		},
	)
	if err != nil {
		s.l.Warn("failed to get option usage counts", zap.Error(err))
		return
	}

	optionMap := make(map[string]customfield.SelectOption)
	for _, opt := range definition.Options {
		optionMap[opt.Value] = opt
	}

	stats.OptionUsage = make([]customfield.OptionUsageStats, 0, len(optionUsage))
	for value, count := range optionUsage {
		label := value
		if opt, ok := optionMap[value]; ok {
			label = opt.Label
		}
		stats.OptionUsage = append(stats.OptionUsage, customfield.OptionUsageStats{
			Value:      value,
			Label:      label,
			UsageCount: count,
		})
	}
}

func (s *Service) detectBreakingChanges(
	original *customfield.CustomFieldDefinition,
	updated *customfield.CustomFieldDefinition,
	usageStats *customfield.DefinitionUsageStats,
) *customfield.BreakingChangeResult {
	result := &customfield.BreakingChangeResult{
		HasBlockingChanges: false,
		Changes:            make([]customfield.BreakingChange, 0),
	}

	if usageStats.TotalValueCount == 0 {
		return result
	}

	if original.FieldType != updated.FieldType {
		result.HasBlockingChanges = true
		result.Changes = append(result.Changes, customfield.BreakingChange{
			Field:      "fieldType",
			ChangeType: customfield.BreakingChangeTypeBlocked,
			Code:       "FIELD_TYPE_CHANGE",
			Message: fmt.Sprintf(
				"Cannot change field type from '%s' to '%s' because %d values exist",
				original.FieldType, updated.FieldType, usageStats.TotalValueCount,
			),
		})
	}

	if original.FieldType.RequiresOptions() && updated.FieldType.RequiresOptions() {
		updatedValues := make(map[string]bool)
		for _, opt := range updated.Options {
			updatedValues[opt.Value] = true
		}

		for _, optUsage := range usageStats.OptionUsage {
			if optUsage.UsageCount > 0 && !updatedValues[optUsage.Value] {
				result.HasBlockingChanges = true
				result.Changes = append(result.Changes, customfield.BreakingChange{
					Field:      "options",
					ChangeType: customfield.BreakingChangeTypeBlocked,
					Code:       "OPTION_IN_USE",
					Message: fmt.Sprintf(
						"Cannot remove option '%s' because it is used by %d resources",
						optUsage.Label, optUsage.UsageCount,
					),
					Details: map[string]any{
						"optionValue": optUsage.Value,
						"usageCount":  optUsage.UsageCount,
					},
				})
			}
		}
	}

	if !original.IsRequired && updated.IsRequired {
		result.Changes = append(result.Changes, customfield.BreakingChange{
			Field:      "isRequired",
			ChangeType: customfield.BreakingChangeTypeWarning,
			Code:       "REQUIRED_FLAG_ADDED",
			Message: fmt.Sprintf(
				"Making field required may affect %d existing resources without values",
				usageStats.ResourceCount,
			),
		})
	}

	return result
}
