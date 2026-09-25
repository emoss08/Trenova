package workercredentialservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/timeutils"
)

var errRenewalUndeliverable = errors.New(
	"driver notifications are not configured, so the renewal ask would reach nobody",
)

// RenewalPreview is a renewal ask as the driver would receive it and the
// papers it names.
type RenewalPreview struct {
	Credentials  []*worker.WorkerCredential
	Notification *serviceports.DriverNotificationPreview
}

// PreviewRenewal checks the ask as RequestRenewal does and renders the
// message it would send, sending nothing.
func (s *Service) PreviewRenewal(ctx context.Context, req RenewalRequest) (*RenewalPreview, error) {
	named, err := s.planRenewal(ctx, req)
	if err != nil {
		return nil, err
	}
	if s.driverNotify == nil {
		return nil, errRenewalUndeliverable
	}

	notice, _ := renewalNotification(req, named, timeutils.NowUnix())
	rendered, err := s.driverNotify.Preview(ctx, notice)
	if err != nil {
		return nil, err
	}

	return &RenewalPreview{Credentials: named, Notification: rendered}, nil
}
