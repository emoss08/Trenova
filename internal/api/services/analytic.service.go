package services

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/shipment"
	"github.com/rs/zerolog"
)

// AnalyticService provides methods for interacting with the analytic service.
//
// Fields:
//   - Client: A *ent.Client object for database operations related to users.
//   - Logger: A *zerolog.Logger object used for logging messages in the service.
//   - config: A *config.Server object containing the server configuration.
//   - Server: A *api.Server object representing the server instance.
type AnalyticService struct {
	Client *ent.Client
	Logger *zerolog.Logger
	Server *api.Server
}

func NewAnalyticService(s *api.Server) *AnalyticService {
	return &AnalyticService{
		Client: s.Client,
		Logger: s.Logger,
		Server: s,
	}
}

// GetDailyShipmentCounts returns a slice of daily counts of new shipments created between the given start and end dates.
//
// Parameters:
//   - ctx: A context.Context object used for the database query.
//   - startDate: A time.Time object representing the start date for the query.
//   - endDate: A time.Time object representing the end date for the query.
//
// Returns:
//   - []map[string]any: A slice of maps with day and count of new shipments for each day.
//   - error: An error if the query fails.
func (r *AnalyticService) GetDailyShipmentCounts(ctx context.Context, startDate, endDate time.Time) ([]map[string]any, error) {
	// Define a struct to match the expected query output
	type Result struct {
		CreatedAt time.Time `json:"created_at"` // Ensure this matches the column name in the query
		Count     int       `json:"count"`
	}

	var shipments []Result
	if err := r.Client.Debug().Shipment.
		Query().
		Where(
			shipment.StatusEQ("New"),
			shipment.CreatedAtGTE(startDate),
			shipment.CreatedAtLTE(endDate),
		).
		GroupBy(shipment.FieldCreatedAt).
		Aggregate(ent.Count()).
		Scan(ctx, &shipments); err != nil {
		r.Logger.Error().Err(err).Msg("Error getting daily shipment counts")
		return nil, err
	}

	// Process results into the desired format
	var results []map[string]any
	for _, s := range shipments {
		results = append(results, map[string]any{
			"day":   s.CreatedAt.Format("2006-01-02"), // Format date as YYYY-MM-DD
			"value": s.Count,
		})
	}

	return results, nil
}
