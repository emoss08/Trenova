package distancecalculator

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/distanceoverride"
	"github.com/emoss08/trenova/internal/core/domain/location"
	shipmentsvc "github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/distancehelpers"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrInsufficientStops = errors.New(
		"move must have at least 2 stops for distance calculation",
	)
	ErrNoDistanceOverrideFound = errors.New("no distance override found")
)

type Params struct {
	fx.In

	Logger                     *zap.Logger
	DistanceOverrideRepository repositories.DistanceOverrideRepository
}

type service struct {
	l   *zap.Logger
	dor repositories.DistanceOverrideRepository
}

func NewService(p Params) services.DistanceCalculatorService {
	return &service{
		l:   p.Logger.Named("service.distance-calculator"),
		dor: p.DistanceOverrideRepository,
	}
}

func (s *service) CalculateDistance(
	ctx context.Context,
	tx bun.IDB,
	req *services.DistanceCalculationRequest,
) (*services.DistanceCalculationResult, error) {
	log := s.l.With(
		zap.String("operation", "CalculateDistance"),
		zap.String("moveID", req.MoveID.String()),
	)

	stopSeqs, err := s.fetchStopSequences(ctx, tx, req.MoveID)
	if err != nil {
		log.Error("failed to fetch stop sequences", zap.Error(err))
		return nil, fmt.Errorf("fetch stop sequences: %w", err)
	}

	if len(stopSeqs) < 2 {
		log.Debug("insufficient stops for distance calculation", zap.Int("count", len(stopSeqs)))
		return nil, ErrInsufficientStops
	}

	distance, source, err := s.determineDistance(ctx, tx, req, stopSeqs, log)
	if err != nil {
		log.Error("failed to determine distance", zap.Error(err))
		return nil, fmt.Errorf("determine distance: %w", err)
	}

	log.Debug("distance calculation completed",
		zap.Float64("distance", distance),
		zap.String("source", string(source)),
	)

	return &services.DistanceCalculationResult{
		Distance: distance,
		Source:   source,
	}, nil
}

type stopSequence struct {
	Sequence   int      `bun:"sequence"`
	LocationID pulid.ID `bun:"location_id"`
}

func (s *service) fetchStopSequences(
	ctx context.Context,
	tx bun.IDB,
	moveID pulid.ID,
) ([]stopSequence, error) {
	stopSeqs := make([]stopSequence, 0, 8)

	err := tx.NewSelect().
		Model((*shipmentsvc.Stop)(nil)).
		Column("sequence", "location_id").
		Where("shipment_move_id = ?", moveID).
		Order("sequence ASC").
		Scan(ctx, &stopSeqs)
	if err != nil {
		return nil, fmt.Errorf("query stop sequences: %w", err)
	}

	return stopSeqs, nil
}

func (s *service) determineDistance(
	ctx context.Context,
	tx bun.IDB,
	req *services.DistanceCalculationRequest,
	stopSeqs []stopSequence,
	log *zap.Logger,
) (float64, services.DistanceSource, error) {
	originLocationID := stopSeqs[0].LocationID
	destinationLocationID := stopSeqs[len(stopSeqs)-1].LocationID

	distanceOverride, err := s.getDistanceOverride(
		ctx,
		originLocationID,
		destinationLocationID,
		req.OrganizationID,
		req.BusinessUnitID,
		log,
	)
	if err != nil && !errors.Is(err, ErrNoDistanceOverrideFound) {
		return 0, "", fmt.Errorf("check distance override: %w", err)
	}

	if distanceOverride != nil {
		log.Debug("using configured distance override",
			zap.String("originLocationID", originLocationID.String()),
			zap.String("destinationLocationID", destinationLocationID.String()),
			zap.Float64("distance", distanceOverride.Distance),
		)
		return distanceOverride.Distance, services.DistanceSourceOverride, nil
	}

	calculatedDistance, err := s.calculateHaversineDistance(ctx, tx, stopSeqs, log)
	if err != nil {
		return 0, "", fmt.Errorf("calculate haversine distance: %w", err)
	}

	return calculatedDistance, services.DistanceSourceCalculated, nil
}

func (s *service) getDistanceOverride(
	ctx context.Context,
	originLocationID, destinationLocationID, orgID, buID pulid.ID,
	log *zap.Logger,
) (*distanceoverride.Override, error) {
	override, err := s.dor.GetByLocationIDs(ctx, &repositories.GetByLocationIDsRequest{
		OriginLocationID:      originLocationID,
		DestinationLocationID: destinationLocationID,
		OrgID:                 orgID,
		BuID:                  buID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			log.Debug("no distance override found, will calculate using haversine",
				zap.String("originLocationID", originLocationID.String()),
				zap.String("destinationLocationID", destinationLocationID.String()),
			)
			return nil, ErrNoDistanceOverrideFound
		}
		return nil, fmt.Errorf("query distance override: %w", err)
	}

	return override, nil
}

func (s *service) calculateHaversineDistance(
	ctx context.Context,
	tx bun.IDB,
	stopSeqs []stopSequence,
	log *zap.Logger,
) (float64, error) {
	locationIDs := make([]pulid.ID, len(stopSeqs))
	for i, seq := range stopSeqs {
		locationIDs[i] = seq.LocationID
	}

	locations := make([]*location.Location, 0, len(locationIDs))
	err := tx.NewSelect().
		Model(&locations).
		Column("id", "longitude", "latitude").
		Where("id IN (?)", bun.In(locationIDs)).
		Scan(ctx)
	if err != nil {
		return 0, fmt.Errorf("query location coordinates: %w", err)
	}

	locationMap := make(map[pulid.ID]*location.Location, len(locations))
	for _, loc := range locations {
		locationMap[loc.ID] = loc
	}

	totalDistance := 0.0
	for i := 0; i < len(stopSeqs)-1; i++ {
		currentLoc := locationMap[stopSeqs[i].LocationID]
		nextLoc := locationMap[stopSeqs[i+1].LocationID]

		if currentLoc == nil || nextLoc == nil {
			log.Warn("location not found in map, skipping segment",
				zap.Int("segment", i),
				zap.String("currentLocationID", stopSeqs[i].LocationID.String()),
				zap.String("nextLocationID", stopSeqs[i+1].LocationID.String()),
			)
			continue
		}

		if currentLoc.Longitude == nil || currentLoc.Latitude == nil ||
			nextLoc.Longitude == nil || nextLoc.Latitude == nil {
			log.Warn("location missing coordinates, skipping segment",
				zap.Int("segment", i),
				zap.String("currentLocationID", currentLoc.ID.String()),
				zap.String("nextLocationID", nextLoc.ID.String()),
			)
			continue
		}

		segmentDistance := distancehelpers.CalculateHaversineDistance(
			*currentLoc.Latitude, *currentLoc.Longitude,
			*nextLoc.Latitude, *nextLoc.Longitude,
		)
		totalDistance += segmentDistance

		log.Debug("calculated segment distance",
			zap.Int("segment", i),
			zap.Float64("segmentDistance", segmentDistance),
			zap.Float64("totalDistance", totalDistance),
		)
	}

	return totalDistance, nil
}
