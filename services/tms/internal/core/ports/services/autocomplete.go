package services

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AutoCompleteService interface {
	GetPlaceDetails(
		ctx context.Context,
		req *AutoCompleteRequest,
	) (*AutocompleteLocationResult, error)
}

type LocationDetails struct {
	Name         string   `json:"name"`
	AddressLine1 string   `json:"addressLine1"`
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
	TenantInfo   pagination.TenantInfo `json:"-"`
	Input        string                `json:"input"        form:"input"`
	SessionToken string                `json:"sessionToken" form:"sessionToken"`
}
