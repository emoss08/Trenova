package googlemaps

import (
	"context"

	"github.com/emoss08/trenova/pkg/types/pulid"
)

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
	StateID      pulid.ID `json:"stateId"`
	Types        []string `json:"types"`
}

// AutocompleteLocationResult combines autocomplete prediction with full location details
type AutocompleteLocationResult struct {
	// Location details for each prediction (same order as predictions)
	Details []*LocationDetails `json:"details"`
	// Count of predictions
	Count int `json:"count"`
}

type AutoCompleteRequest struct {
	OrgID pulid.ID `json:"orgId" query:"orgId"`
	BuID  pulid.ID `json:"buId"  query:"buId"`
	Input string   `json:"input" query:"input"`
}

type Client interface {
	CheckAPIKey(ctx context.Context, orgID pulid.ID, buID pulid.ID) (bool, error)
	AutocompleteWithDetails(
		ctx context.Context,
		req *AutoCompleteRequest,
	) (*AutocompleteLocationResult, error)
}
