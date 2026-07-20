package realtimeservice

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"

	realtime "github.com/Foony-Limited/realtime-go"
	servicesport "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testAPIKey = "app.test:super-secret"

func TestCreateToken(t *testing.T) {
	t.Parallel()

	client, err := realtime.NewRest(realtime.RestOptions{Key: testAPIKey})
	require.NoError(t, err)

	svc := &Service{
		l:      zap.NewNop(),
		apiKey: testAPIKey,
		client: client,
	}

	req := &servicesport.CreateRealtimeTokenRequest{
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	result, tokenErr := svc.CreateToken(req)
	require.NoError(t, tokenErr)
	require.NotNil(t, result)

	assert.Equal(t, req.UserID.String(), result.ClientID)
	assert.NotEmpty(t, result.Token)
	assert.Greater(t, result.ExpiresAt, time.Now().UnixMilli())

	segments := strings.Split(result.Token, ".")
	require.Len(t, segments, 3)

	payload, decErr := base64.RawURLEncoding.DecodeString(segments[1])
	require.NoError(t, decErr)

	claims := string(payload)
	assert.Contains(t, claims, req.UserID.String())
	assert.Contains(
		t,
		claims,
		"tenant:"+req.OrganizationID.String()+":"+req.BusinessUnitID.String()+":*",
	)
}

func TestCreateToken_ValidationErrors(t *testing.T) {
	t.Parallel()

	svc := &Service{l: zap.NewNop()}

	_, err := svc.CreateToken(nil)
	require.Error(t, err)

	_, err = svc.CreateToken(&servicesport.CreateRealtimeTokenRequest{})
	require.Error(t, err)
}
