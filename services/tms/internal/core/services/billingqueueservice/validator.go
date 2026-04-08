package billingqueueservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB *postgres.Connection
}

type Validator struct {
	validator *validationframework.TenantedValidator[*billingqueue.BillingQueueItem]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: validationframework.NewTenantedValidatorBuilder[*billingqueue.BillingQueueItem]().
			WithModelName("Billing Queue Item").
			WithUniquenessChecker(validationframework.NewBunUniquenessCheckerLazy(func() bun.IDB { return p.DB.DB() })).
			WithCustomRule(createStatusConstraintsRule()).
			Build(),
	}
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *billingqueue.BillingQueueItem,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

func createStatusConstraintsRule() validationframework.TenantedRule[*billingqueue.BillingQueueItem] {
	return validationframework.NewTenantedRule[*billingqueue.BillingQueueItem](
		"status_constraints",
	).
		OnBoth().
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(ctx context.Context, entity *billingqueue.BillingQueueItem, valCtx *validationframework.TenantedValidationContext, multiErr *errortypes.MultiError) error {
			switch entity.Status { //nolint:exhaustive // only a few statuses are supported
			case billingqueue.StatusInReview:
				if entity.AssignedBillerID == nil || entity.AssignedBillerID.IsNil() {
					multiErr.Add(
						"assignedBillerId",
						errortypes.ErrRequired,
						"Assigned biller is required when status is InReview",
					)
				}
			case billingqueue.StatusSentBackToOps, billingqueue.StatusException:
				if entity.ExceptionReasonCode == nil {
					multiErr.Add(
						"exceptionReasonCode",
						errortypes.ErrRequired,
						"Exception reason code is required",
					)
				} else if !entity.ExceptionReasonCode.IsValid() {
					multiErr.Add(
						"exceptionReasonCode",
						errortypes.ErrInvalid,
						"Invalid exception reason code",
					)
				}

				notesRequired := entity.Status == billingqueue.StatusException ||
					(entity.ExceptionReasonCode != nil && *entity.ExceptionReasonCode == billingqueue.ExceptionOther)

				if notesRequired && entity.ExceptionNotes == "" {
					multiErr.Add(
						"exceptionNotes",
						errortypes.ErrRequired,
						"Exception notes are required",
					)
				}
			case billingqueue.StatusCanceled:
				if entity.CanceledByID == nil || entity.CanceledByID.IsNil() {
					multiErr.Add(
						"canceledById",
						errortypes.ErrRequired,
						"Canceled by is required when status is Canceled",
					)
				}
				if entity.CancelReason == "" {
					multiErr.Add(
						"cancelReason",
						errortypes.ErrRequired,
						"Cancel reason is required when status is Canceled",
					)
				}
			}

			return nil
		})
}
