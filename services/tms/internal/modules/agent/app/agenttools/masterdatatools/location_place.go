package masterdatatools

import (
	"context"
	"errors"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	postalCodePrefix   = 5
	placeLookupMissing = "this organization has no address lookup connected"
	placeLookupFailed  = "the address lookup did not answer"
	placeNotFound      = "no place at that address was found"
	unplacedNote       = " It is saved without map coordinates, because %s, so it has no geofence " +
		"until it is placed on the map in Locations."
)

type placeFinder interface {
	GetPlaceDetails(
		ctx context.Context,
		req *kernelports.AutoCompleteRequest,
	) (*kernelports.AutocompleteLocationResult, error)
}

func (t *CreateLocationTool) place(
	ctx context.Context,
	draft *locationDraft,
	tenant pagination.TenantInfo,
) (details *kernelports.LocationDetails, unplacedReason string) {
	if t.places == nil {
		return nil, placeLookupMissing
	}

	found, err := t.places.GetPlaceDetails(ctx, &kernelports.AutoCompleteRequest{
		TenantInfo: tenant,
		Input:      draft.searchText(),
	})
	switch {
	case errors.Is(err, kernelports.ErrPlaceLookupUnavailable):
		return nil, placeLookupMissing
	case err != nil:
		return nil, placeLookupFailed
	case found == nil:
		return nil, placeNotFound
	}
	for _, detail := range found.Details {
		if detail != nil && draft.isAt(detail) {
			return detail, ""
		}
	}

	return nil, placeNotFound
}

func (d *locationDraft) searchText() string {
	return strings.Join([]string{d.addressLine1, d.city, d.state + " " + d.postalCode}, ", ")
}

func (d *locationDraft) isAt(detail *kernelports.LocationDetails) bool {
	if !strings.EqualFold(strings.TrimSpace(detail.State), d.state) {
		return false
	}

	return samePostalCode(detail.PostalCode, d.postalCode) ||
		strings.EqualFold(strings.TrimSpace(detail.City), d.city)
}

func samePostalCode(a, b string) bool {
	a, b = strings.TrimSpace(a), strings.TrimSpace(b)
	if len(a) < postalCodePrefix || len(b) < postalCodePrefix {
		return false
	}

	return a[:postalCodePrefix] == b[:postalCodePrefix]
}
