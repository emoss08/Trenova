package googlemapsservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/integrationservice"
	"github.com/google/uuid"
	"github.com/sourcegraph/conc/pool"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"googlemaps.github.io/maps"
)

type AutoCompleteServiceParams struct {
	fx.In

	Logger             *zap.Logger
	IntegrationService *integrationservice.Service
	UsStateRepo        repositories.UsStateRepository
}

type autoCompleteService struct {
	l                  *zap.Logger
	integrationService *integrationservice.Service
	usStateRepo        repositories.UsStateRepository
}

func NewAutoCompleteService(p AutoCompleteServiceParams) services.AutoCompleteService {
	log := p.Logger.Named("service.google.maps.autocomplete")
	return &autoCompleteService{
		l:                  log,
		integrationService: p.IntegrationService,
		usStateRepo:        p.UsStateRepo,
	}
}

func (s *autoCompleteService) GetPlaceDetails(
	ctx context.Context,
	req *services.AutoCompleteRequest,
) (*services.AutocompleteLocationResult, error) {
	log := s.l.With(
		zap.String("operation", "GetPlaceDetails"),
		zap.Any("request", req),
	)

	if req.Input == "" {
		return &services.AutocompleteLocationResult{
			Details: []*services.LocationDetails{},
			Count:   0,
		}, nil
	}

	client, err := s.resolveClient(ctx, req)
	if err != nil {
		log.Error("failed to resolve Google Maps client", zap.Error(err))
		return nil, err
	}

	var sessionToken maps.PlaceAutocompleteSessionToken
	if req.SessionToken != "" {
		parsed, parseErr := uuid.Parse(req.SessionToken)
		if parseErr != nil {
			log.Warn("invalid session token, ignoring", zap.Error(parseErr))
		} else {
			sessionToken = maps.PlaceAutocompleteSessionToken(parsed)
		}
	}

	paReq := &maps.PlaceAutocompleteRequest{
		Input: req.Input,
		//nolint:exhaustive // we only want to autocomplete US locations
		Components: map[maps.Component][]string{
			maps.ComponentCountry: {"us"},
		},
		SessionToken: sessionToken,
	}

	autocompleteResp, err := client.PlaceAutocomplete(ctx, paReq)
	if err != nil {
		log.Error("failed to get place details", zap.Error(err))
		return nil, err
	}

	if len(autocompleteResp.Predictions) == 0 {
		return &services.AutocompleteLocationResult{
			Details: []*services.LocationDetails{},
			Count:   0,
		}, nil
	}

	maxResults := min(5, len(autocompleteResp.Predictions))

	p := pool.New().WithMaxGoroutines(maxResults)
	results := make([]*services.LocationDetails, maxResults)

	for i := range maxResults {
		p.Go(func() {
			detail, detailErr := s.getDetailsByPlaceID(
				ctx,
				client,
				autocompleteResp.Predictions[i].PlaceID,
				sessionToken,
			)
			if detailErr != nil {
				log.Debug("skipping place details",
					zap.String("placeID", autocompleteResp.Predictions[i].PlaceID),
					zap.Error(detailErr))
				return
			}
			results[i] = detail
		})
	}

	p.Wait()

	details := make([]*services.LocationDetails, 0, maxResults)
	for _, r := range results {
		if r != nil {
			details = append(details, r)
		}
	}

	return &services.AutocompleteLocationResult{
		Details: details,
		Count:   len(details),
	}, nil
}

func (s *autoCompleteService) resolveClient(
	ctx context.Context,
	req *services.AutoCompleteRequest,
) (*maps.Client, error) {
	cfg, err := s.integrationService.GetRuntimeConfig(
		ctx, req.TenantInfo, integration.TypeGoogleMaps,
	)
	if err != nil {
		return nil, err
	}

	apiKey := cfg.Config["apiKey"]
	if apiKey == "" {
		return nil, ErrMissingAPIKey
	}

	client, err := maps.NewClient(maps.WithAPIKey(apiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Maps client: %w", err)
	}

	return client, nil
}

func (s *autoCompleteService) getDetailsByPlaceID(
	ctx context.Context,
	client *maps.Client,
	placeID string,
	sessionToken maps.PlaceAutocompleteSessionToken,
) (*services.LocationDetails, error) {
	log := s.l.With(
		zap.String("operation", "getDetailsByPlaceID"),
		zap.String("placeID", placeID),
	)

	if placeID == "" {
		return nil, ErrEmptyPlaceID
	}

	req := &maps.PlaceDetailsRequest{
		PlaceID: placeID,
		Fields: []maps.PlaceDetailsFieldMask{
			maps.PlaceDetailsFieldMaskName,
			maps.PlaceDetailsFieldMaskFormattedAddress,
			maps.PlaceDetailsFieldMaskAddressComponent,
			maps.PlaceDetailsFieldMaskGeometry,
			maps.PlaceDetailsFieldMaskPlaceID,
			maps.PlaceDetailsFieldMaskTypes,
		},
		SessionToken: sessionToken,
	}

	resp, err := client.PlaceDetails(ctx, req)
	if err != nil {
		log.Error("failed to get place details", zap.Error(err))
		return nil, err
	}

	details := s.parseLocationDetails(&resp)

	state, err := s.usStateRepo.GetByAbbreviation(ctx, details.State)
	if err != nil {
		log.Debug("state lookup failed",
			zap.String("state", details.State),
			zap.Error(err))
	} else if state != nil {
		details.StateID = state.ID
	}

	return details, nil
}

func (s *autoCompleteService) parseLocationDetails(
	resp *maps.PlaceDetailsResult,
) *services.LocationDetails {
	details := &services.LocationDetails{
		Name:      resp.Name,
		PlaceID:   resp.PlaceID,
		Longitude: resp.Geometry.Location.Lng,
		Latitude:  resp.Geometry.Location.Lat,
		Types:     resp.Types,
	}

	var streetNum, route string
	for _, component := range resp.AddressComponents {
		types := make(map[string]bool, len(component.Types))
		for _, t := range component.Types {
			types[t] = true
		}

		if types["street_number"] {
			streetNum = component.LongName
		}
		if types["route"] {
			route = component.LongName
		}
		if types["locality"] || types["sublocality"] || types["sublocality_level_1"] ||
			types["postal_town"] {
			if details.City == "" {
				details.City = component.LongName
			}
		}
		if types["administrative_area_level_1"] {
			details.State = component.ShortName
		}
		if types["postal_code"] {
			details.PostalCode = component.LongName
		}
	}

	switch {
	case streetNum != "" && route != "":
		details.AddressLine1 = fmt.Sprintf("%s %s", streetNum, route)
	case route != "":
		details.AddressLine1 = route
	case resp.FormattedAddress != "":
		details.AddressLine1 = resp.FormattedAddress
	}

	return details
}
