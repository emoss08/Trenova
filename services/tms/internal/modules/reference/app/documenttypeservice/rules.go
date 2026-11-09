package documenttypeservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
)

func createSystemDocumentTypeProtectionRule() validationframework.Rule[*documenttype.DocumentType] {
	return validationframework.NewRule[*documenttype.DocumentType](
		"system_document_type_protection",
	).
		OnUpdate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			_ context.Context,
			entity *documenttype.DocumentType,
			_ *validationframework.Context,
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
