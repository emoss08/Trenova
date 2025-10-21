package googlemaps

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"googlemaps.github.io/maps"
)

type AutoCompleteParams struct {
	fx.In

	Client *maps.Client
	Repo   repositories.UsStateRepository
	Logger *zap.Logger
}

type AutoCompleteService struct {
	client *maps.Client
	repo   repositories.UsStateRepository
	l      *zap.Logger
}

func NewAutoCompleteService(p AutoCompleteParams) *AutoCompleteService {
	return &AutoCompleteService{
		client: p.Client,
		repo:   p.Repo,
		l:      p.Logger.Named("service.google.maps.autocomplete"),
	}
}

func (s *AutoCompleteService) GetPlaceDetails(
	ctx context.Context,
	req *AutoCompleteRequest,
) (*AutocompleteLocationResult, error) {
	if req.Input == "" {
		return &AutocompleteLocationResult{Details: []*LocationDetails{}, Count: 0}, nil
	}

	paReq := &maps.PlaceAutocompleteRequest{
		Input: req.Input,
		//nolint:exhaustive // we only want to autocomplete US locations
		Components: map[maps.Component][]string{
			maps.ComponentCountry: {"us"},
		},
	}

	autocompleteResp, err := s.placesAutocomplete(ctx, paReq)
	if err != nil {
		return nil, fmt.Errorf("places autocomplete failed: %w", err)
	}

	if len(autocompleteResp.Predictions) == 0 {
		return &AutocompleteLocationResult{Details: []*LocationDetails{}, Count: 0}, nil
	}

	maxResults := min(5, len(autocompleteResp.Predictions))

	details := make([]*LocationDetails, 0, maxResults)
	for i := range maxResults {
		detail, detailErr := s.getPlaceDetails(ctx, autocompleteResp.Predictions[i].PlaceID)
		if detailErr != nil {
			s.l.Debug("skipping place details",
				zap.String("placeID", autocompleteResp.Predictions[i].PlaceID),
				zap.Error(detailErr))
			continue
		}
		details = append(details, detail)
	}

	return &AutocompleteLocationResult{
		Details: details,
		Count:   len(details),
	}, nil
}

func (s *AutoCompleteService) placesAutocomplete(
	ctx context.Context,
	req *maps.PlaceAutocompleteRequest,
) (*maps.AutocompleteResponse, error) {
	resp, err := s.client.PlaceAutocomplete(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("google places autocomplete API error: %w", err)
	}

	return &resp, nil
}

func (s *AutoCompleteService) getPlaceDetails(
	ctx context.Context,
	placeID string,
) (*LocationDetails, error) {
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
	}

	resp, err := s.client.PlaceDetails(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("google place details API error for placeID %s: %w", placeID, err)
	}

	details := s.parseLocationDetails(&resp)

	state, err := s.repo.GetByAbbreviation(ctx, details.State)
	if err != nil {
		s.l.Debug("state lookup failed",
			zap.String("state", details.State),
			zap.Error(err))
	} else {
		details.StateID = state.ID
	}

	return details, nil
}

func (s *AutoCompleteService) parseLocationDetails(resp *maps.PlaceDetailsResult) *LocationDetails {
	details := &LocationDetails{
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
