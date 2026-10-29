package carrierintelreviewservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
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

	Logger       *zap.Logger
	Intel        *carrierintelservice.Service
	SnapshotRepo carrierintelports.CarrierIntelSnapshotRepository
	RawRepo      carrierintelports.CarrierIntelRawPayloadRepository
	EventRepo    carrierintelports.CarrierIntelEventRepository
	OverrideRepo carrierintelports.CarrierIntelOverrideRepository
	CarrierRepo  carrierports.CarrierRepository
	AuditService auditports.AuditService
	Watchtower   watchtowerports.WatchtowerProjector `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	intel        *carrierintelservice.Service
	snapshotRepo carrierintelports.CarrierIntelSnapshotRepository
	rawRepo      carrierintelports.CarrierIntelRawPayloadRepository
	eventRepo    carrierintelports.CarrierIntelEventRepository
	overrideRepo carrierintelports.CarrierIntelOverrideRepository
	carrierRepo  carrierports.CarrierRepository
	auditService auditports.AuditService
	watchtower   watchtowerports.WatchtowerProjector
	now          func() int64
}

var _ carrierintelports.CarrierIntelEventReader = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.carrier-intelligence"),
		intel:        p.Intel,
		snapshotRepo: p.SnapshotRepo,
		rawRepo:      p.RawRepo,
		eventRepo:    p.EventRepo,
		overrideRepo: p.OverrideRepo,
		carrierRepo:  p.CarrierRepo,
		auditService: p.AuditService,
		watchtower:   p.Watchtower,
		now:          timeutils.NowUnix,
	}
}
