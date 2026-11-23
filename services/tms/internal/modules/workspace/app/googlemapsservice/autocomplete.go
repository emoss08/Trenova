package googlemapsservice

import (
	"context"
	"fmt"
	"net/http"
	"time"

	places "cloud.google.com/go/maps/places/apiv1"
	"cloud.google.com/go/maps/places/apiv1/placespb"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/integrationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/google/uuid"
	"github.com/googleapis/gax-go/v2/apierror"
	"github.com/googleapis/gax-go/v2/callctx"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/api/option"
)

const (
	placesRequestTimeout = 10 * time.Second
	maxPlaceResults      = 5
	placeFieldMask       = "id,displayName,formattedAddress,addressComponents,location,types"
	headerAPIKey         = "x-goog-api-key"
	headerFieldMask      = "x-goog-fieldmask"
)

type AutoCompleteServiceParams struct {
	fx.In

	Logger             *zap.Logger
	IntegrationService *integrationservice.Service
	UsStateRepo        usstateports.UsStateRepository
}

type autoCompleteService struct {
	l                  *zap.Logger
	places             *places.Client
	integrationService *integrationservice.Service
	usStateRepo        usstateports.UsStateRepository
}

func NewAutoCompleteService(p AutoCompleteServiceParams) (kernelports.AutoCompleteService, error) {
	client, err := places.NewRESTClient(
		context.Background(),
		option.WithHTTPClient(&http.Client{Timeout: placesRequestTimeout}),
	)
	if err != nil {
		return nil, fmt.Errorf("create Google Places client: %w", err)
	}

	return &autoCompleteService{
		l:                  p.Logger.Named("service.google.maps.autocomplete"),
		places:             client,
		integrationService: p.IntegrationService,
		usStateRepo:        p.UsStateRepo,
	}, nil
}

func (s *autoCompleteService) GetPlaceDetails(
	ctx context.Context,
	req *kernelports.AutoCompleteRequest,
) (*kernelports.AutocompleteLocationResult, error) {
	log := s.l.With(zap.String("operation", "GetPlaceDetails"))

	if req.Input == "" {
		return emptyResult(), nil
	}

	apiKey, err := s.resolveAPIKey(ctx, req)
	if err != nil {
		return nil, err
	}
	ctx = callctx.SetHeaders(ctx, headerAPIKey, apiKey)

	sessionToken := ""
	if req.SessionToken != "" {
		if _, parseErr := uuid.Parse(req.SessionToken); parseErr != nil {
			log.Warn("invalid session token, ignoring", zap.Error(parseErr))
		} else {
			sessionToken = req.SessionToken
		}
	}

	resp, err := s.places.AutocompletePlaces(ctx, &placespb.AutocompletePlacesRequest{
		Input:               req.Input,
		IncludedRegionCodes: []string{"us"},
		SessionToken:        sessionToken,
	})
	if err != nil {
		log.Error("failed to autocomplete places", zap.Error(err))
		return nil, mapPlacesError(err)
	}

	placeIDs := make([]string, 0, maxPlaceResults)
	for _, suggestion := range resp.GetSuggestions() {
		if id := suggestion.GetPlacePrediction().GetPlaceId(); id != "" {
			placeIDs = append(placeIDs, id)
		}
		if len(placeIDs) == maxPlaceResults {
			break
		}
	}
	if len(placeIDs) == 0 {
		return emptyResult(), nil
	}

	results := make([]*kernelports.LocationDetails, len(placeIDs))
	p := pool.New().WithMaxGoroutines(len(placeIDs))
	for i, placeID := range placeIDs {
		p.Go(func() {
			detail, detailErr := s.getDetailsByPlaceID(ctx, placeID, sessionToken)
			if detailErr != nil {
				log.Debug("skipping place details",
					zap.String("placeID", placeID),
					zap.Error(detailErr))
				return
			}
			results[i] = detail
		})
	}
	p.Wait()

	details := make([]*kernelports.LocationDetails, 0, len(results))
	for _, r := range results {
		if r != nil {
			details = append(details, r)
		}
	}

	return &kernelports.AutocompleteLocationResult{
		Details: details,
		Count:   len(details),
	}, nil
}

func emptyResult() *kernelports.AutocompleteLocationResult {
	return &kernelports.AutocompleteLocationResult{
		Details: []*kernelports.LocationDetails{},
		Count:   0,
	}
}

func (s *autoCompleteService) resolveAPIKey(
	ctx context.Context,
	req *kernelports.AutoCompleteRequest,
) (string, error) {
	cfg, err := s.integrationService.GetRuntimeConfig(
		ctx, req.TenantInfo, integration.TypeGoogleMaps,
	)
	if err != nil {
		return "", fmt.Errorf("%w: %w", kernelports.ErrPlaceLookupUnavailable, err)
	}

	apiKey := cfg.Config["apiKey"]
	if apiKey == "" {
		return "", fmt.Errorf("%w: %w", kernelports.ErrPlaceLookupUnavailable, ErrMissingAPIKey)
	}

	return apiKey, nil
}

func mapPlacesError(err error) error {
	apiErr, ok := apierror.FromError(err)
	if !ok {
		return err
	}

	switch {
	case apiErr.HTTPCode() == http.StatusUnauthorized,
		apiErr.HTTPCode() == http.StatusForbidden,
		apiErr.Reason() == "API_KEY_INVALID":
		return errortypes.NewBusinessError(
			"Google Maps rejected the API key. Check that the key is valid and that the Places API (New) is enabled for its Google Cloud project.",
		).WithInternal(fmt.Errorf("%w: %w", kernelports.ErrPlaceLookupUnavailable, err))
	default:
		return err
	}
}

func (s *autoCompleteService) getDetailsByPlaceID(
	ctx context.Context,
	placeID string,
	sessionToken string,
) (*kernelports.LocationDetails, error) {
	if placeID == "" {
		return nil, ErrEmptyPlaceID
	}

	place, err := s.places.GetPlace(
		callctx.SetHeaders(ctx, headerFieldMask, placeFieldMask),
		&placespb.GetPlaceRequest{
			Name:         "places/" + placeID,
			SessionToken: sessionToken,
		},
	)
	if err != nil {
		return nil, mapPlacesError(err)
	}

	details := parseLocationDetails(place)

	state, err := s.usStateRepo.GetByAbbreviation(ctx, details.State)
	if err != nil {
		s.l.Debug("state lookup failed",
			zap.String("state", details.State),
			zap.Error(err))
	} else if state != nil {
		details.StateID = state.ID
	}

	return details, nil
}

func parseLocationDetails(place *placespb.Place) *kernelports.LocationDetails {
	details := &kernelports.LocationDetails{
		Name:      place.GetDisplayName().GetText(),
		PlaceID:   place.GetId(),
		Longitude: place.GetLocation().GetLongitude(),
		Latitude:  place.GetLocation().GetLatitude(),
		Types:     place.GetTypes(),
	}

	var streetNum, route string
	for _, component := range place.GetAddressComponents() {
		types := make(map[string]bool, len(component.GetTypes()))
		for _, t := range component.GetTypes() {
			types[t] = true
		}

		if types["street_number"] {
			streetNum = component.GetLongText()
		}
		if types["route"] {
			route = component.GetLongText()
		}
		if types["locality"] || types["sublocality"] || types["sublocality_level_1"] ||
			types["postal_town"] {
			if details.City == "" {
				details.City = component.GetLongText()
			}
		}
		if types["administrative_area_level_1"] {
			details.State = component.GetShortText()
		}
		if types["postal_code"] {
			details.PostalCode = component.GetLongText()
		}
	}

	switch {
	case streetNum != "" && route != "":
		details.AddressLine1 = streetNum + " " + route
	case route != "":
		details.AddressLine1 = route
	case place.GetFormattedAddress() != "":
		details.AddressLine1 = place.GetFormattedAddress()
	}

	return details
}
