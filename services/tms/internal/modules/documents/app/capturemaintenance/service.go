package capturemaintenance

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/app/capturedevices"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturefiling"
	"github.com/emoss08/trenova/internal/modules/documents/app/captureintake"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturerequests"
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Batches captureports.CaptureBatchRepository
	Devices *capturedevices.Service
	Filing  *capturefiling.Service
	Intake  *captureintake.Service
	Logger  *zap.Logger
	// Notifications tells an owner their unfiled pages are about to go.
	// Without it the pages are still kept and deleted on time; nobody is told.
	Notifications Notifier `optional:"true"`
	Pages         captureports.CapturePageRepository
	Requests      *capturerequests.Service
	Support       *capturesupport.Service
	// Workflows is optional so an installation without a worker still
	// receives pages. A sealed batch then waits, and the reconcile sweep
	// starts its processing once a worker appears.
	Workflows kernelports.WorkflowStarter `optional:"true"`
}

type Service struct {
	batches       captureports.CaptureBatchRepository
	l             *zap.Logger
	notifications Notifier
	pages         captureports.CapturePageRepository
	workflows     kernelports.WorkflowStarter
	devices       *capturedevices.Service
	filing        *capturefiling.Service
	intake        *captureintake.Service
	requests      *capturerequests.Service
	support       *capturesupport.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		batches:       p.Batches,
		l:             p.Logger.Named("service.capture"),
		notifications: p.Notifications,
		pages:         p.Pages,
		workflows:     p.Workflows,
		devices:       p.Devices,
		filing:        p.Filing,
		intake:        p.Intake,
		requests:      p.Requests,
		support:       p.Support,
	}
}

// notifier is the one thing capture asks of notifications.
type Notifier interface {
	Create(
		ctx context.Context,
		entity *notification.Notification,
	) (*notification.Notification, error)
}

func AsNotifier(s *notificationservice.Service) Notifier {
	return s
}
