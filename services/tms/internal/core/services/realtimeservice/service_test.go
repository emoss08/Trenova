package realtimeservice

import (
	"testing"

	"github.com/ably/ably-go/ably"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestCreateTokenRequest(t *testing.T) {
	t.Parallel()

	client, err := ably.NewREST(ably.WithKey("app.test:super-secret"))
	require.NoError(t, err)

	svc := &Service{
		l:      zap.NewNop(),
		client: client,
	}

	req := &servicesport.CreateRealtimeTokenRequest{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	result, tokenErr := svc.CreateTokenRequest(req)
	require.NoError(t, tokenErr)
	require.NotNil(t, result)

	assert.Equal(t, req.UserID.String(), result.ClientID)
	assert.Equal(t, defaultTokenTTL, result.TTL)
	assert.Equal(
		t,
		tenantCapability(req.OrganizationID.String(), req.BusinessUnitID.String()),
		result.Capability,
	)
	assert.NotEmpty(t, result.KeyName)
	assert.NotEmpty(t, result.Nonce)
	assert.NotEmpty(t, result.MAC)
	assert.NotZero(t, result.Timestamp)
}

func TestCreateTokenRequest_ValidationErrors(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop()}

	_, err := svc.CreateTokenRequest(nil)
	require.Error(t, err)

	_, err = svc.CreateTokenRequest(&servicesport.CreateRealtimeTokenRequest{})
	require.Error(t, err)
}
