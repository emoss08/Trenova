package invoiceadjustmentservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In
}

type Validator struct{}

func NewValidator(_ ValidatorParams) *Validator {
	return &Validator{}
}

func (v *Validator) ValidateRequest(_ context.Context, req *services.InvoiceAdjustmentRequest) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	if req == nil {
		multiErr.Add("request", errortypes.ErrRequired, "Adjustment request is required")
		return multiErr
	}
	if req.InvoiceID.IsNil() {
		multiErr.Add("invoiceId", errortypes.ErrRequired, "Invoice ID is required")
	}
	if req.Kind == "" {
		multiErr.Add("kind", errortypes.ErrRequired, "Adjustment kind is required")
	}
	if strings.TrimSpace(req.IdempotencyKey) == "" {
		multiErr.Add("idempotencyKey", errortypes.ErrRequired, "Idempotency key is required")
	}
	if req.TenantInfo.OrgID.IsNil() {
		multiErr.Add("tenantInfo.orgId", errortypes.ErrRequired, "Organization ID is required")
	}
	if req.TenantInfo.BuID.IsNil() {
		multiErr.Add("tenantInfo.buId", errortypes.ErrRequired, "Business unit ID is required")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}
