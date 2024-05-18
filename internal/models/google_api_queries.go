package models

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/googleapi"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/google/uuid"
)

func GetGoogleAPIKeyForOrganization(
	ctx context.Context, client *ent.Client, orgID, buID uuid.UUID,
) (string, error) {
	googleAPI, err := client.GoogleApi.Query().
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
