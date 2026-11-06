package homevisitservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Repo   telematicsports.TelematicsRepository
	Logger *zap.Logger
}

type Service struct {
	repo telematicsports.TelematicsRepository
	l    *zap.Logger
}

func New(p Params) telematicsports.HomeVisitRecorder {
	return &Service{repo: p.Repo, l: p.Logger.Named("service.home-visit")}
}

func (s *Service) RecordPositions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	positions []*telematics.VehiclePosition,
) {
	visits, err := s.visits(ctx, positions)
	if err == nil {
		err = s.repo.RecordHomeVisits(ctx, visits)
	}
	if err != nil {
		s.l.Warn("failed to record drivers seen at home; home time keeps its last value",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err))
	}
}

func (s *Service) visits(
	ctx context.Context,
	positions []*telematics.VehiclePosition,
) ([]telematicsports.HomeVisit, error) {
	if len(positions) == 0 {
		return nil, nil
	}

	tractorIDs := make([]pulid.ID, 0, len(positions))
	for _, position := range positions {
		tractorIDs = append(tractorIDs, position.TractorID)
	}
	homes, err := s.repo.ListTractorHomes(ctx, tractorIDs)
	if err != nil || len(homes) == 0 {
		return nil, err
	}

	return VisitsOf(positions, homes), nil
}

func VisitsOf(
	positions []*telematics.VehiclePosition,
	homes []telematicsports.TractorHome,
) []telematicsports.HomeVisit {
	byTractor := make(map[pulid.ID]telematicsports.TractorHome, len(homes))
	for _, home := range homes {
		byTractor[home.TractorID] = home
	}

	visits := make([]telematicsports.HomeVisit, 0, len(homes))
	for _, position := range positions {
		home, ok := byTractor[position.TractorID]
		if !ok || !worker.AtHome(
			home.HomeLatitude,
			home.HomeLongitude,
			position.Latitude,
			position.Longitude,
		) {
			continue
		}
		visits = append(visits, telematicsports.HomeVisit{
			WorkerID: home.WorkerID,
			At:       position.RecordedAt,
		})
	}

	return visits
}
