package smsjobs

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/infrastructure/sms"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/smsjobs/smscontract"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"go.temporal.io/sdk/activity"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	SMSClient *sms.Client
	Logger    *zap.Logger
	Plans     platformplanports.PlanService `optional:"true"`
}

type Activities struct {
	smsClient *sms.Client
	logger    *zap.Logger
	plans     platformplanports.PlanService
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		smsClient: p.SMSClient,
		logger:    p.Logger.Named("sms-activities"),
		plans:     p.Plans,
	}
}

func (a *Activities) SendSMSActivity(
	ctx context.Context,
	payload *smscontract.SendSMSPayload,
) (*SendSMSResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Starting SMS send activity",
		"organizationId", payload.OrganizationID.String(),
		"businessUnitId", payload.BusinessUnitID.String(),
	)

	if err := planservice.RequireCapability(ctx, a.plans, pagination.TenantInfo{
		OrgID: payload.OrganizationID,
		BuID:  payload.BusinessUnitID,
	}, platformplan.CapabilitySMS); err != nil {
		logger.Info("SMS not sent: the organization's plan does not include SMS", "error", err)
		return &SendSMSResult{
			Success: false,
			Error:   err.Error(),
		}, temporaltype.ToPlanRefusal(err)
	}

	activity.RecordHeartbeat(ctx, "sending SMS")

	err := a.smsClient.Send(ctx, sms.SendRequest{
		To:   payload.PhoneNumber,
		Body: payload.Message,
	})
	if err != nil {
		logger.Error("Failed to send SMS", "error", err)
		return &SendSMSResult{
			Success: false,
			Error:   err.Error(),
		}, sendFailure(err)
	}

	logger.Info("SMS sent successfully")

	return &SendSMSResult{
		Success: true,
	}, nil
}

func sendFailure(err error) error {
	if errors.Is(err, sms.ErrNotConfigured) {
		return temporaltype.NewNonRetryableError("SMS is not configured", err).ToTemporalError()
	}

	var apiErr *sms.APIError
	if errors.As(err, &apiErr) {
		if !apiErr.Retryable() {
			return temporaltype.NewNonRetryableError("Twilio refused the SMS", err).
				ToTemporalError()
		}
		if apiErr.RetryAfter > 0 {
			return temporaltype.NewRetryableErrorWithDelay(
				"Failed to send SMS",
				err,
				apiErr.RetryAfter,
			).ToTemporalError()
		}
	}

	return temporaltype.NewRetryableError("Failed to send SMS", err).ToTemporalError()
}
