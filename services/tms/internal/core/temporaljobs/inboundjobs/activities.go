package inboundjobs

import (
	"context"

	"github.com/emoss08/trenova/internal/core/services/inboundmessageservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Inbound *inboundmessageservice.Service
	Logger  *zap.Logger
}

type Activities struct {
	inbound *inboundmessageservice.Service
	l       *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{inbound: p.Inbound, l: p.Logger.Named("inbound-activities")}
}

// SettleInboundMessageActivity reads a staged message and decides what it is.
//
// The name carries its domain because Temporal's activity registry is keyed by
// method name across a whole task queue, and this one shares system-queue with
// several other packages.
func (a *Activities) SettleInboundMessageActivity(
	ctx context.Context,
	payload *ProcessInboundMessagePayload,
) (*ProcessInboundMessageResult, error) {
	result, err := a.inbound.ProcessMessage(
		ctx,
		payload.MessageID,
		pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
	)
	if err != nil {
		a.l.Error("failed to settle an inbound message",
			zap.String("messageId", payload.MessageID.String()), zap.Error(err))

		return nil, err
	}

	return &ProcessInboundMessageResult{
		MessageID:      result.MessageID,
		Status:         result.Status,
		Classification: result.Classification,
		Handled:        result.Handled,
		Matched:        result.Matched,
	}, nil
}

// FailInboundMessageActivity records that the pipeline gave up.
//
// A message left mid-pipeline is worse than one that failed visibly: it looks
// like it is still being worked on, so nobody goes to look at it.
func (a *Activities) FailInboundMessageActivity(
	ctx context.Context,
	payload *FailInboundMessagePayload,
) error {
	return a.inbound.MarkFailed(
		ctx,
		payload.MessageID,
		pagination.TenantInfo{
			OrgID: payload.OrganizationID,
			BuID:  payload.BusinessUnitID,
		},
		payload.Code,
		payload.Reason,
	)
}
