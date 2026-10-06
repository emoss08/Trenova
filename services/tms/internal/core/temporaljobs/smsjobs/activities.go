package smsjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/platformplan"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/planservice"
	"github.com/emoss08/trenova/internal/infrastructure/sms"
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
	Plans     services.PlanService `optional:"true"`
}

type Activities struct {
	smsClient *sms.Client
	logger    *zap.Logger
	plans     services.PlanService
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
	payload *SendSMSPayload,
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

	err := a.smsClient.Send(sms.SendRequest{
		To:   payload.PhoneNumber,
		Body: payload.Message,
	})
	if err != nil {
		logger.Error("Failed to send SMS", "error", err)
		return &SendSMSResult{
			Success: false,
			Error:   err.Error(),
		}, temporaltype.NewRetryableError("Failed to send SMS", err).ToTemporalError()
	}

	logger.Info("SMS sent successfully")

	return &SendSMSResult{
		Success: true,
	}, nil
}
