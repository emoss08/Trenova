package checkcallservice

import (
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/internal/infrastructure/config/tenderingconfig"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentmoveports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/trackingeventports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/emailports"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB                  kernelports.Transactor
	Logger              *zap.Logger
	AppConfig           *appconfig.Config
	TenderingConfig     *tenderingconfig.Config
	Repo                checkcallports.CheckCallRepository
	Links               checkcallports.TrackingLinkRepository
	MoveRepo            shipmentmoveports.ShipmentMoveRepository
	ShipmentRepo        shipmentports.ShipmentRepository
	CarrierAssignments  carrierassignmentports.CarrierAssignmentRepository
	CarrierRepo         carrierports.CarrierRepository
	OrgRepo             tenantports.OrganizationRepository
	DispatchControlRepo dispatchcontrolports.DispatchControlRepository
	TrackingEvents      trackingeventports.TrackingEventService
	Templates           documenttemplateports.DocumentTemplateResolver `optional:"true"`
	EmailService        emailports.EmailService                        `optional:"true"`
	AuditService        auditports.AuditService                        `optional:"true"`
	Observers           []checkcallports.CheckCallObserver             `                group:"check_call_observers"`
}

type Service struct {
	db                  kernelports.Transactor
	l                   *zap.Logger
	appConfig           *appconfig.Config
	tenderingConfig     *tenderingconfig.Config
	repo                checkcallports.CheckCallRepository
	links               checkcallports.TrackingLinkRepository
	moveRepo            shipmentmoveports.ShipmentMoveRepository
	shipmentRepo        shipmentports.ShipmentRepository
	carrierAssignments  carrierassignmentports.CarrierAssignmentRepository
	carrierRepo         carrierports.CarrierRepository
	orgRepo             tenantports.OrganizationRepository
	dispatchControlRepo dispatchcontrolports.DispatchControlRepository
	trackingEvents      trackingeventports.TrackingEventService
	templates           documenttemplateports.DocumentTemplateResolver
	emailService        emailports.EmailService
	auditService        auditports.AuditService
	observers           []checkcallports.CheckCallObserver
	now                 func() int64
}

func New(p Params) *Service { //nolint:gocritic // dependency injection
	return &Service{
		db:                  p.DB,
		l:                   p.Logger.Named("service.check-call"),
		appConfig:           p.AppConfig,
		tenderingConfig:     p.TenderingConfig,
		repo:                p.Repo,
		links:               p.Links,
		moveRepo:            p.MoveRepo,
		shipmentRepo:        p.ShipmentRepo,
		carrierAssignments:  p.CarrierAssignments,
		carrierRepo:         p.CarrierRepo,
		orgRepo:             p.OrgRepo,
		dispatchControlRepo: p.DispatchControlRepo,
		trackingEvents:      p.TrackingEvents,
		templates:           p.Templates,
		emailService:        p.EmailService,
		auditService:        p.AuditService,
		observers:           p.Observers,
		now:                 timeutils.NowUnix,
	}
}

var _ checkcallports.CheckCallService = (*Service)(nil)
