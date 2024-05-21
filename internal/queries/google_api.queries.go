package queries

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/ent/googleapi"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

// Parameters:
//   - ctx context.Context: Context which may contain deadlines, cancellation signals, and other request-scoped values.
//   - orgID uuid.UUID: The identifier of the organization.
//   - buID uuid.UUID: The identifier of the business unit.
//
// Returns:
//   - string: The Google API key.
//   - error: An error object that indicates why the retrieval failed, nil if no error occurred.
func (r *QueryService) GetGoogleAPIKeyForOrganization(ctx context.Context, orgID, buID uuid.UUID) (string, error) {
	googleAPI, err := r.Client.GoogleApi.Query().
		Where(
			googleapi.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			)).
		Only(ctx)
	if err != nil {
		return "", err
	}

	if googleAPI.APIKey == "" {
		return "", errors.New("google API key not found")
	}

	return googleAPI.APIKey, nil
}
