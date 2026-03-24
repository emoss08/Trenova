package documenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func createSystemDocumentTypeProtectionRule() validationframework.TenantedRule[*documenttype.DocumentType] {
	return validationframework.NewTenantedRule[*documenttype.DocumentType](
		"system_document_type_protection",
	).
		OnUpdate().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *documenttype.DocumentType,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.IsSystem {
				multiErr.Add(
					"isSystem",
					errortypes.ErrInvalid,
					"System document types cannot be modified",
				)
			}
			return nil
		})
}
