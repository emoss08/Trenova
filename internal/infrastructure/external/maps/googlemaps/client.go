// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package googlemaps

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"googlemaps.github.io/maps"
)

var ErrAPIKeyBlank = eris.New("API Key is blank")

// ClientParams contains the dependencies for the Google Maps client
type ClientParams struct {
	fx.In

	IntegrationRepo repositories.IntegrationRepository
	UsStateRepo     repositories.UsStateRepository
	Logger          *logger.Logger
}

// client is the Google Maps client implementation
type client struct {
	l               *zerolog.Logger
	integrationRepo repositories.IntegrationRepository
	usStateRepo     repositories.UsStateRepository
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
		l:               &log,
		integrationRepo: p.IntegrationRepo,
		usStateRepo:     p.UsStateRepo,
	}
}

// getClient gets a client for the Google Maps API
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - orgID[pulid.ID]: The organization ID
//   - buID[pulid.ID]: The business unit ID
//
// Returns:
//   - gc[*maps.Client]: The Google Maps client
//   - err[error]: The error
func (c *client) getClient(ctx context.Context, orgID, buID pulid.ID) (*maps.Client, error) {
	// * Get the API key from the database
	intg, err := c.integrationRepo.GetByType(ctx, repositories.GetIntegrationByTypeRequest{
		Type:  integration.GoogleMapsIntegrationType,
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		c.l.Error().
			Err(err).
			Str("orgID", orgID.String()).
			Msg("failed to get API key from database")
		return nil, err
	}

	apiKey, ok := intg.Configuration["apiKey"]
	if !ok {
		return nil, ErrAPIKeyBlank
	}

	// * convert the api key to a string
	apiKeyStr, ok := apiKey.(string)
	if !ok {
		return nil, ErrAPIKeyBlank
	}

	// * Initialize the Google Maps client
	gc, err := maps.NewClient(maps.WithAPIKey(apiKeyStr))
	if err != nil {
		return nil, err
	}

	return gc, nil
}

// CheckAPIKey checks if the API key is valid
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - orgID[pulid.ID]: The organization ID
//
// Returns:
//   - valid[bool]: Whether the API key is valid
//   - err[error]: The error
func (c *client) CheckAPIKey(ctx context.Context, orgID, buID pulid.ID) (bool, error) {
	_, err := c.getClient(ctx, orgID, buID)
	if err != nil {
		return false, err
	}

	return true, nil
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
func (c *client) placeAutocomplete(
	ctx context.Context,
	orgID, buID pulid.ID,
	req *maps.PlaceAutocompleteRequest,
) (*maps.AutocompleteResponse, error) {
	log := c.l.With().
		Str("operation", "PlaceAutocomplete").
		Str("orgID", orgID.String()).
		Logger()

	// * Get the client
	gc, err := c.getClient(ctx, orgID, buID)
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
func (c *client) getPlaceDetails(
	ctx context.Context,
	orgID, buID pulid.ID,
	placeID string,
) (*LocationDetails, error) {
	log := c.l.With().
		Str("operation", "GetPlaceDetails").
		Str("orgID", orgID.String()).
		Str("placeID", placeID).
		Logger()

	// * Get the client
	gc, err := c.getClient(ctx, orgID, buID)
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

	// * Get the state ID by abbreviation
	state, err := c.usStateRepo.GetByAbbreviation(ctx, details.State)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get state by abbreviation")
		return nil, err
	}

	details.StateID = state.ID

	return details, nil
}

// AutocompleteWithDetails performs place autocomplete and then fetches details for each prediction
// This eliminates the need to make separate API calls from the client
//
// Parameters:
//   - ctx[context.Context]: The context of the request
//   - req[*AutoCompleteRequest]: The place autocomplete request
//
// Returns:
//   - result[*AutocompleteLocationResult]: Combined autocomplete predictions and location details
//   - err[error]: The error
func (c *client) AutocompleteWithDetails(
	ctx context.Context,
	req *AutoCompleteRequest,
) (*AutocompleteLocationResult, error) {
	log := c.l.With().
		Str("operation", "AutocompleteWithDetails").
		Str("orgID", req.OrgID.String()).
		Logger()

	// * Create a request
	paReq := maps.PlaceAutocompleteRequest{
		Input: req.Input,
		// * Add country component to restrict results to the United States
		//nolint:exhaustive // We only want to restrict to the United States
		Components: map[maps.Component][]string{
			maps.ComponentCountry: {"us"},
		},
	}

	// * Get autocomplete predictions first
	autocompleteResp, err := c.placeAutocomplete(ctx, req.OrgID, req.BuID, &paReq)
	if err != nil {
		log.Error().
			Err(err).
			Msg("failed to get autocomplete predictions")
		return nil, err
	}

	result := &AutocompleteLocationResult{
		Details: make([]*LocationDetails, 0, len(autocompleteResp.Predictions)),
	}

	// * Fetch details for each prediction
	for i := range autocompleteResp.Predictions {
		// * Get a pointer to the current prediction to avoid copying the struct
		currentPrediction := &autocompleteResp.Predictions[i]
		details, dErr := c.getPlaceDetails(ctx, req.OrgID, req.BuID, currentPrediction.PlaceID)
		if dErr != nil {
			log.Error().
				Err(dErr).
				Str("placeID", currentPrediction.PlaceID).
				Msg("failed to get place details, continuing with next prediction")
			// * Add nil for this prediction to maintain order
			result.Details = append(result.Details, nil)
			continue
		}
		result.Details = append(result.Details, details)
	}

	// * Append the count of predictions
	result.Count = len(result.Details)

	return result, nil
}
