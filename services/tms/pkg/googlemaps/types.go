package googlemaps

import "github.com/emoss08/trenova/pkg/pulid"

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

type AutocompleteLocationResult struct {
	Details []*LocationDetails `json:"details"`
	Count   int                `json:"count"`
}

type AutoCompleteRequest struct {
	Input string `json:"input" form:"input"`
}
