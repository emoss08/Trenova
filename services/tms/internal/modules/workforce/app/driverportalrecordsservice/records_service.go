package driverportalrecordsservice

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/driverportalservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/performancereviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/schedulingservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workersafetyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workertrainingservice"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Portal          *driverportalservice.Service
	DocumentService *documentservice.Service
	PTOLedger       *ptoledgerservice.Service
	Credentials     *workercredentialservice.Service
	Training        *workertrainingservice.Service
	Safety          *workersafetyservice.Service
	Reviews         *performancereviewservice.Service
	Leave           LeaveReader
	Scheduling      *schedulingservice.Service
	SelfService     *selfserviceservice.Service
}

type Service struct {
	portal          *driverportalservice.Service
	documentService *documentservice.Service
	ptoLedger       *ptoledgerservice.Service
	credentials     *workercredentialservice.Service
	training        *workertrainingservice.Service
	safety          *workersafetyservice.Service
	reviews         *performancereviewservice.Service
	leave           LeaveReader
	scheduling      *schedulingservice.Service
	selfService     *selfserviceservice.Service
}

//nolint:gocritic // fx param structs are passed by value
func New(p Params) *Service {
	return &Service{
		portal:          p.Portal,
		documentService: p.DocumentService,
		ptoLedger:       p.PTOLedger,
		credentials:     p.Credentials,
		training:        p.Training,
		safety:          p.Safety,
		reviews:         p.Reviews,
		leave:           p.Leave,
		scheduling:      p.Scheduling,
		selfService:     p.SelfService,
	}
}
