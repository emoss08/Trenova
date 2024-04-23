// Package google provides functionalities to interact with Google Maps services.
// It allows clients to geocode locations and calculate distance matrices using
// Google Maps' APIs, supporting multi-tenancy by allowing separate API keys for each tenant.
package google

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	"googlemaps.github.io/maps"
)

// Client holds a logger instance and can create Google Maps API clients tailored to specific organizations.
type Client struct {
	Logger *zerolog.Logger // Logger is used for logging messages in the methods of Client.
}

// NewClient initializes a new instance of Client using a given Server's logger.
// It serves as a constructor for Client struct.
//
// Parameters:
//
//	s *api.Server: A pointer to an instance of api.Server which contains configuration and state needed by Client.
//
// Returns:
//
//	*Client: A pointer to the newly created Client instance.
func NewClient(logger *zerolog.Logger) *Client {
	return &Client{
		Logger: logger,
	}
}

// GetClientForOrganization creates a new Google Maps API client configured with a specific API key.
// This method supports multi-tenancy by allowing different API keys for each tenant.
//
// Parameters:
//
//	apiKey string: The API key for the tenant, used to authenticate with Google Maps API.
//
// Returns:
//
//	*maps.Client: A pointer to the maps.Client configured with the apiKey.
//	error: An error object that indicates why the client creation failed, nil if no error occurred.
func (gc *Client) GetClientForOrganization(apiKey string) (*maps.Client, error) {
	c, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		gc.Logger.Error().Err(err).Msg("Error creating google maps client")
		return nil, err
	}
	return c, nil
}

// ForwardGeocode attempts to forward geocode a location described by address components and an API key.
// It logs and returns errors encountered during the geocoding process.
//
// Parameters:
//
//	ctx context.Context: The context to control cancellation and deadlines.
//	addressLine1, city, state, zipCode string: Components of the address to be geocoded.
//	apiKey string: API key for the tenant, used for authorization with Google Maps API.
//
// Returns:
//
//	[]maps.GeocodingResult: A slice of GeocodingResult containing geocoded information.
//	error: An error object that indicates why the geocoding failed, nil if no error occurred.
func (gc *Client) ForwardGeocode(
	ctx context.Context, addressLine1, city, state, zipCode, apiKey string,
) ([]maps.GeocodingResult, error) {
	c, err := gc.GetClientForOrganization(apiKey)
	if err != nil {
		gc.Logger.Error().Err(err).Msg("Error geocoding location")
		return nil, err
	}
	req := &maps.GeocodingRequest{
		Address: fmt.Sprintf("%s, %s, %s, %s", addressLine1, city, state, zipCode),
	}
	geocodeResponse, err := c.Geocode(ctx, req)
	if err != nil {
		gc.Logger.Error().Err(err).Msg("Error geocoding location")
		return nil, err
	}
	return geocodeResponse, nil
}

// GetDistanceMatrix computes the distance matrix for given origins and destinations, using a specified unit system.
// It utilizes a specific API key to allow multi-tenant access.
//
// Parameters:
//
//	ctx context.Context: The context to control cancellation and deadlines.
//	origins, destinations []string: Lists of starting points and endpoints for the distance calculations.
//	units maps.Units: The unit system to be used for the distances.
//	apiKey string: API key for the tenant, used for authorization with Google Maps API.
//
// Returns:
//
//	*maps.DistanceMatrixResponse: A pointer to DistanceMatrixResponse containing the calculated distances.
//	error: An error object that indicates why the distance matrix computation failed, nil if no error occurred.
func (gc *Client) GetDistanceMatrix(
	ctx context.Context, origins, destinations []string, units maps.Units, apiKey string,
) (*maps.DistanceMatrixResponse, error) {
	c, err := gc.GetClientForOrganization(apiKey)
	if err != nil {
		gc.Logger.Error().Err(err).Msg("Error getting distance matrix")
		return nil, err
	}

	// TODO(Wolfred): Add the ability to specify additional departure and arrival times. This will allow us to
	// account for traffic conditions when calculating the distance matrix.
	req := &maps.DistanceMatrixRequest{
		Origins:      origins,
		Destinations: destinations,
		Units:        units,
		Mode:         maps.TravelModeDriving,
		Language:     "en",
	}
	distanceMatrixResponse, err := c.DistanceMatrix(ctx, req)
	if err != nil {
		gc.Logger.Error().Err(err).Msg("Error getting distance matrix")
		return nil, err
	}
	return distanceMatrixResponse, nil
}
