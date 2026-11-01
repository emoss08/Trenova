package weatheralertresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/operations/ports/weatheralertports"
	"github.com/emoss08/trenova/internal/testutil/mocks/weatheralertportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestWeatherAlerts_ReadsTheCallersTenant(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	service := weatheralertportsmock.NewMockWeatherAlertService(t)
	collection := &weatheralertports.WeatherAlertFeatureCollection{Type: "FeatureCollection"}
	service.EXPECT().
		GetActiveAlerts(mock.Anything, pagination.TenantInfo{
			OrgID:  authCtx.OrganizationID,
			BuID:   authCtx.BusinessUnitID,
			UserID: authCtx.UserID,
		}).
		Return(collection, nil)
	deps := &Deps{Core: &resolvercore.Core{}, WeatherAlertService: service}

	got, err := (&QueryResolver{deps}).WeatherAlerts(resolvertest.AuthedContext(t, authCtx))
	require.NoError(t, err)
	assert.Same(t, collection, got)
}

func TestWeatherAlert_ReadsTheNamedAlert(t *testing.T) {
	t.Parallel()

	authCtx := resolvertest.UserAuthContext()
	alertID := pulid.MustNew("wa_")
	service := weatheralertportsmock.NewMockWeatherAlertService(t)
	detail := &weatheralertports.WeatherAlertDetail{}
	service.EXPECT().
		GetAlertDetail(mock.Anything, &weatheralertports.GetWeatherAlertDetailRequest{
			ID: alertID,
			TenantInfo: pagination.TenantInfo{
				OrgID:  authCtx.OrganizationID,
				BuID:   authCtx.BusinessUnitID,
				UserID: authCtx.UserID,
			},
		}).
		Return(detail, nil)
	deps := &Deps{Core: &resolvercore.Core{}, WeatherAlertService: service}

	got, err := (&QueryResolver{deps}).WeatherAlert(
		resolvertest.AuthedContext(t, authCtx),
		alertID.String(),
	)
	require.NoError(t, err)
	assert.Same(t, detail, got)
}

func TestWeatherAlert_NamesAMalformedID(t *testing.T) {
	t.Parallel()

	deps := &Deps{Core: &resolvercore.Core{}}

	_, err := (&QueryResolver{deps}).WeatherAlert(
		resolvertest.AuthedContext(t, resolvertest.UserAuthContext()),
		"nope",
	)

	var fieldErr *errortypes.Error
	require.ErrorAs(t, err, &fieldErr)
	assert.Equal(t, "id", fieldErr.Field)
}

func TestWeatherAlertFeature_ReturnsTheGeometry(t *testing.T) {
	t.Parallel()

	geometry := weatheralertports.WeatherAlertGeometry{"type": "Polygon"}

	got, err := (&WeatherAlertFeatureResolver{&Deps{}}).Geometry(
		t.Context(),
		&weatheralertports.WeatherAlertFeature{Geometry: geometry},
	)
	require.NoError(t, err)
	assert.Equal(t, "Polygon", got["type"])
}
