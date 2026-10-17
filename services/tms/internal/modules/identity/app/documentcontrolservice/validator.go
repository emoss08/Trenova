package documentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Config *config.Config
}

type Validator struct {
	conversion *config.DocumentConversionConfig
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{conversion: p.Config.GetDocumentConversionConfig()}
}

func (v *Validator) ValidateUpdate(
	_ context.Context,
	entity *tenant.DocumentControl,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if entity.UsesDocling() && !v.conversion.Enabled() {
		multiErr.Add(
			"extractionEngine",
			errortypes.ErrInvalid,
			"Docling needs the document converter, which is not configured on this server. "+
				"Set documentConversion.doclingUrl and documentConversion.apiKey, or keep the Legacy engine.",
		)
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

var _ validationframework.TenantedEntity = (*tenant.DocumentControl)(nil)
