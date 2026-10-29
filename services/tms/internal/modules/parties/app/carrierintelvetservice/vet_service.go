package carrierintelvetservice

import (
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierintelservice"
	"github.com/emoss08/trenova/internal/modules/parties/app/carrierservice"
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
	SnapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	SubjectRepo    carrierintelports.CarrierIntelSubjectRepository
	CarrierRepo    carrierports.CarrierRepository
	CarrierService *carrierservice.Service
	AuditService   auditports.AuditService
}

type Service struct {
	l              *zap.Logger
	intel          *carrierintelservice.Service
	controlRepo    carrierintelports.CarrierIntelControlRepository
	snapshotRepo   carrierintelports.CarrierIntelSnapshotRepository
	subjectRepo    carrierintelports.CarrierIntelSubjectRepository
	carrierRepo    carrierports.CarrierRepository
	carrierService carrierintelservice.CarrierWriter
	auditService   auditports.AuditService
	now            func() int64
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:              p.Logger.Named("service.carrier-intelligence"),
		intel:          p.Intel,
		controlRepo:    p.ControlRepo,
		snapshotRepo:   p.SnapshotRepo,
		subjectRepo:    p.SubjectRepo,
		carrierRepo:    p.CarrierRepo,
		carrierService: p.CarrierService,
		auditService:   p.AuditService,
		now:            timeutils.NowUnix,
	}
}
