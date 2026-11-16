package supportlogservice

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Repo supportlogports.Repository
}

type Validator struct {
	validator tenantedvalidation.EntityValidator[*supportlog.LogRequest]
}

func NewValidator(p ValidatorParams) *Validator {
	return &Validator{
		validator: tenantedvalidation.NewBuilder[*supportlog.LogRequest]().
			WithModelName("Log request").
			WithCustomRule(instanceBounds(p.Repo)).
			Build(),
	}
}

func (v *Validator) ValidateCreate(ctx context.Context, entity *supportlog.LogRequest) error {
	return v.validator.ValidateCreate(ctx, entity)
}

func instanceBounds(
	repo supportlogports.Repository,
) validationframework.Rule[*supportlog.LogRequest] {
	return validationframework.NewRule[*supportlog.LogRequest]("instance_log_request_bounds").
		OnCreate().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			ctx context.Context,
			entity *supportlog.LogRequest,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			load, err := repo.InstanceLoad(ctx, entity.CreatedAt-int64(supportlog.DailyWindow/time.Second))
			if err != nil {
				return err
			}
			switch {
			case load.InFlight:
				multiErr.Add(
					"period",
					errortypes.ErrInvalid,
					"Logs are already being sent to Trenova support. Wait until that request has finished",
				)
			case load.Remaining() == 0:
				multiErr.Add(
					"period",
					errortypes.ErrInvalid,
					"This instance has sent logs to Trenova support {0} times in the last 24 hours. Try again later",
					supportlog.DailyLimit,
				)
			}
			return nil
		})
}
