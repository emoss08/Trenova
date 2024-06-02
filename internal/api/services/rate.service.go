package services

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/rate"
	"github.com/emoss08/trenova/internal/queries"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// RateService provides  methods for managing rates.
type RateService struct {
	Client       *ent.Client               // Client is the database client used for querying and mutating customer records.
	Logger       *zerolog.Logger           // Logger is used for logging messages.
	QueryService *queries.RateQueryService // QueryService provides methods for querying the database.
}

// NewRateService creates a new RateService.
// s is the server instance containing necessary dependencies.
//
// Parameters:
//   - s *api.Server: A pointer to an instance of api.Server which contains configuration and state needed by
//     RateService.
//
// Returns:
//   - *RateService: A pointer to the newly created RateService instance.
func NewRateService(s *api.Server) *RateService {
	return &RateService{
		Client: s.Client,
		Logger: s.Logger,
		QueryService: &queries.RateQueryService{
			Client: s.Client,
			Logger: s.Logger,
		},
	}
}

// GetRates retrieves a list of rates for a given organization and business unit.
// It returns a slice of Rate entities, the total number of Rate records, and an error object.
//
// Parameters:
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - limit int: The maximum number of records to return.
//   - offset int: The number of records to skip before starting to return records.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//   - statuses string: A comma-separated string of statuses to filter the rates by.
//
// Returns:
//   - []*ent.Rate: A slice of Rate entities.
//   - int: The total number of Rate records.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (r *RateService) GetRates(ctx context.Context, limit, offset int, orgID, buID uuid.UUID, statuses string) ([]*ent.Rate, int, error) {
	ps := ParseStatuses(statuses)
	params := queries.GetRatesParams{
		Limit:    limit,
		Offset:   offset,
		OrgID:    orgID,
		BuID:     buID,
		Statuses: ps,
	}

	return r.QueryService.GetRates(ctx, params)
}

func ParseStatuses(statusStr string) []rate.Status {
	if statusStr == "" {
		// Include all of the statuses
		return []rate.Status{rate.StatusA, rate.StatusI}
	}

	statuses := []rate.Status{}
	statusStrs := strings.Split(statusStr, ",")
	for _, s := range statusStrs {
		status := rate.Status(s)
		statuses = append(statuses, status)
	}

	return statuses
}

// CreateRate creates a new Rate. It returns a pointer to the newly created Rate entity and an error object.
//
// Parameters:
//
//   - ctx: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - entity *ent.Rate: The Rate request containing the details of the Rate to be created.
//
// Returns:
//   - *ent.Rate: A pointer to the newly created Rate entity.
//
// Possible errors:
//   - Error creating Rate entity
func (r *RateService) CreateRate(ctx context.Context, entity *ent.Rate) (*ent.Rate, error) {
	createdEntity := new(ent.Rate)

	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error

		_, err = r.QueryService.CreateRateEntity(ctx, tx, entity)
		if err != nil {
			r.Logger.Err(err).Msg("Error creating rate entity")
			return err
		}

		return nil
	})
	if err != nil {
		r.Logger.Err(err).Msg("Error creating rate entity")
		return nil, err
	}

	return createdEntity, nil
}

func (r *RateService) GetsRatesNearExpiration(ctx context.Context, orgID, buID uuid.UUID) ([]*ent.Rate, int, error) {
	return r.QueryService.GetsRatesNearExpiration(ctx, orgID, buID)
}
