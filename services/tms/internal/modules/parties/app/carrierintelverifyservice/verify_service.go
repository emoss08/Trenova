package carrierintelverifyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	Intel          *carrierintelservice.Service
	ControlRepo    carrierintelports.CarrierIntelControlRepository
	EventRepo      carrierintelports.CarrierIntelEventRepository
	VerifyRepo     carrierintelports.CarrierEquipmentVerificationRepository
	CarrierRepo    carrierports.CarrierRepository
	AssignmentRepo carrierassignmentports.CarrierAssignmentRepository
	AuditService   auditports.AuditService
	Plans          platformplanports.PlanService `optional:"true"`
}

type Service struct {
	l              *zap.Logger
	intel          *carrierintelservice.Service
	controlRepo    carrierintelports.CarrierIntelControlRepository
	eventRepo      carrierintelports.CarrierIntelEventRepository
	verifyRepo     carrierintelports.CarrierEquipmentVerificationRepository
	carrierRepo    carrierports.CarrierRepository
	assignmentRepo carrierassignmentports.CarrierAssignmentRepository
	auditService   auditports.AuditService
	plans          platformplanports.PlanService
	now            func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.carrier-intelligence"),
		intel:          p.Intel,
		controlRepo:    p.ControlRepo,
		eventRepo:      p.EventRepo,
		verifyRepo:     p.VerifyRepo,
		carrierRepo:    p.CarrierRepo,
		assignmentRepo: p.AssignmentRepo,
		auditService:   p.AuditService,
		plans:          p.Plans,
		now:            timeutils.NowUnix,
	}
}

func (s *Service) requirePaidLookups(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) error {
	return planservice.RequireCapability(
		ctx,
		s.plans,
		tenantInfo,
		platformplan.CapabilityCarrierIntelligencePaid,
	)
}
