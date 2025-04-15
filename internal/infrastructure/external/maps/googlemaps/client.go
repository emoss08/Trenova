package googlemaps

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"googlemaps.github.io/maps"
)

var ErrAPIKeyBlank = eris.New("API Key is blank")

// LocationDetails contains the details of a location returned from the Google Maps API
type LocationDetails struct {
	Name         string   `json:"name"`
	AddressLine1 string   `json:"addressLine1"`
	AddressLine2 string   `json:"addressLine2"`
	City         string   `json:"city"`
	State        string   `json:"state"`
	PostalCode   string   `json:"postalCode"`
	Longitude    float64  `json:"longitude"`
	Latitude     float64  `json:"latitude"`
	PlaceID      string   `json:"placeId"`
	Types        []string `json:"types"`
}

// AutocompleteLocationResult combines autocomplete prediction with full location details
type AutocompleteLocationResult struct {
	// Predictions from the autocomplete API
	Predictions []maps.AutocompletePrediction `json:"predictions"`
	// Location details for each prediction (same order as predictions)
	Details []*LocationDetails `json:"details"`
}

type Client interface {
	PlaceAutocomplete(ctx context.Context, orgID pulid.ID, req *maps.PlaceAutocompleteRequest) (*maps.AutocompleteResponse, error)
	GetPlaceDetails(ctx context.Context, orgID pulid.ID, placeID string) (*LocationDetails, error)
	AutocompleteWithDetails(ctx context.Context, orgID pulid.ID, req *maps.PlaceAutocompleteRequest) (*AutocompleteLocationResult, error)
}

// ClientParams contains the dependencies for the Google Maps client
type ClientParams struct {
	fx.In

	GCrepo repositories.GoogleMapsConfigRepository
	Logger *logger.Logger
}

// client is the Google Maps client implementation
type client struct {
	l      *zerolog.Logger
	gcRepo repositories.GoogleMapsConfigRepository
}

// NewClient creates a new Google Maps client
//
// Parameters:
//   - p[ClientParams]: The client parameters
//
// Returns:
//   - c[Client]: The Google Maps client
func NewClient(p ClientParams) Client {
	log := p.Logger.With().
		Str("client", "googlemaps").
		Logger()

	return &client{
		l:      &log,
		gcRepo: p.GCrepo,
	}
}

// getClient gets a client for the Google Maps API
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - orgID[pulid.ID]: The organization ID
//
// Returns:
//   - gc[*maps.Client]: The Google Maps client
//   - err[error]: The error
func (c *client) getClient(ctx context.Context, orgID pulid.ID) (*maps.Client, error) {
	// * Get the API key from the database
	apiKey, err := c.gcRepo.GetAPIKeyByOrgID(ctx, orgID)
	if err != nil {
		c.l.Error().
			Err(err).
			Str("orgID", orgID.String()).
			Msg("failed to get API key from database")
		return nil, err
	}

	// * Check if the API Key is an empty string
	if apiKey == "" {
		return nil, ErrAPIKeyBlank
	}

	// * Initialize the Google Maps client
	gc, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, err
	}

	return gc, nil
}

// PlaceAutocomplete performs a place autocomplete request to the Google Maps API
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - orgID[pulid.ID]: The organization ID
//   - req[*maps.PlaceAutocompleteRequest]: The place autocomplete request
//
// Returns:
//   - resp[*maps.AutocompleteResponse]: The place autocomplete response
//   - err[error]: The error
func (c *client) PlaceAutocomplete(ctx context.Context, orgID pulid.ID, req *maps.PlaceAutocompleteRequest) (*maps.AutocompleteResponse, error) {
	log := c.l.With().
		Str("operation", "PlaceAutocomplete").
		Str("orgID", orgID.String()).
		Logger()

	// * Get the client
	gc, err := c.getClient(ctx, orgID)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get client")
		return nil, err
	}

	// * Make the request
	resp, err := gc.PlaceAutocomplete(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to make request")
		return nil, err
	}

	return &resp, nil
}

// GetPlaceDetails retrieves detailed information about a place from the Google Maps API
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - orgID[pulid.ID]: The organization ID
//   - placeID[string]: The place ID to retrieve details for
//
// Returns:
//   - details[*LocationDetails]: The location details
//   - err[error]: The error
func (c *client) GetPlaceDetails(ctx context.Context, orgID pulid.ID, placeID string) (*LocationDetails, error) {
	log := c.l.With().
		Str("operation", "GetPlaceDetails").
		Str("orgID", orgID.String()).
		Str("placeID", placeID).
		Logger()

	// * Get the client
	gc, err := c.getClient(ctx, orgID)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get client")
		return nil, err
	}

	// * Define the fields to request
	fields := []maps.PlaceDetailsFieldMask{
		maps.PlaceDetailsFieldMaskName,
		maps.PlaceDetailsFieldMaskFormattedAddress,
		maps.PlaceDetailsFieldMaskAddressComponent,
		maps.PlaceDetailsFieldMaskGeometry,
		maps.PlaceDetailsFieldMaskPlaceID,
		maps.PlaceDetailsFieldMaskTypes,
	}

	// * Create the request
	req := &maps.PlaceDetailsRequest{
		PlaceID: placeID,
		Fields:  fields,
	}

	// * Make the request
	resp, err := gc.PlaceDetails(ctx, req)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to make place details request")
		return nil, err
	}

	// * Create a new LocationDetails object
	details := &LocationDetails{
		Name:      resp.Name,
		PlaceID:   resp.PlaceID,
		Longitude: resp.Geometry.Location.Lng,
		Latitude:  resp.Geometry.Location.Lat,
		Types:     resp.Types,
	}

	// * Process address components
	var streetNumber, route string
	for _, component := range resp.AddressComponents {
		for _, t := range component.Types {
			switch t {
			case "street_number":
				streetNumber = component.LongName
			case "route":
				route = component.LongName
			case "locality", "sublocality", "sublocality_level_1", "postal_town":
				details.City = component.LongName
			case "administrative_area_level_1":
				details.State = component.ShortName
			case "postal_code":
				details.PostalCode = component.LongName
			}
		}
	}

	// * Combine street number and route for address line 1
	if streetNumber != "" && route != "" {
		details.AddressLine1 = streetNumber + " " + route
	} else if route != "" {
		details.AddressLine1 = route
	}

	// * If we don't have all the address components but have a formatted address,
	// * we'll use it as a fallback
	if details.AddressLine1 == "" && resp.FormattedAddress != "" {
		details.AddressLine1 = resp.FormattedAddress
	}

	return details, nil
}

// AutocompleteWithDetails performs place autocomplete and then fetches details for each prediction
// This eliminates the need to make separate API calls from the client
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - orgID[pulid.ID]: The organization ID
//   - req[*maps.PlaceAutocompleteRequest]: The place autocomplete request
//
// Returns:
//   - result[*AutocompleteLocationResult]: Combined autocomplete predictions and location details
//   - err[error]: The error
func (c *client) AutocompleteWithDetails(ctx context.Context, orgID pulid.ID, req *maps.PlaceAutocompleteRequest) (*AutocompleteLocationResult, error) {
	log := c.l.With().
		Str("operation", "AutocompleteWithDetails").
		Str("orgID", orgID.String()).
		Logger()

	// * Get autocomplete predictions first
	autocompleteResp, err := c.PlaceAutocomplete(ctx, orgID, req)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get autocomplete predictions")
		return nil, err
	}

	result := &AutocompleteLocationResult{
		Predictions: autocompleteResp.Predictions,
		Details:     make([]*LocationDetails, 0, len(autocompleteResp.Predictions)),
	}

	// * Fetch details for each prediction
	for _, prediction := range autocompleteResp.Predictions {
		details, dErr := c.GetPlaceDetails(ctx, orgID, prediction.PlaceID)
		if dErr != nil {
			log.Error().
				Err(dErr).
				Str("placeID", prediction.PlaceID).
				Msg("failed to get place details, continuing with next prediction")
			// * Add nil for this prediction to maintain order
			result.Details = append(result.Details, nil)
			continue
		}
		result.Details = append(result.Details, details)
	}

	return result, nil
}
