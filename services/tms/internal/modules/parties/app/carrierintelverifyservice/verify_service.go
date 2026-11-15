package carrierintelverifyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/platformcatalogports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/carrierassignmentports"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
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
	Features       platformcatalogports.FeatureGate
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
	features       platformcatalogports.FeatureGate
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
		features:       p.Features,
		now:            timeutils.NowUnix,
	}
}

func (s *Service) requirePaidLookups(ctx context.Context) error {
	return platformcatalogports.RequireFeature(
		ctx,
		s.features,
		platformcatalog.FeatureCarrierBrokerage,
	)
}
