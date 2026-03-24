package authservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/apikey"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubUsageRecorder struct {
	lastEvent services.APIKeyUsageEvent
	calls     int
}

func (s *stubUsageRecorder) RecordUsage(event services.APIKeyUsageEvent) {
	s.lastEvent = event
	s.calls++
}

func TestAuthenticateAPIKeyDisabled(t *testing.T) {
	t.Parallel()

	recorder := &stubUsageRecorder{}
	svc := &Service{
		akr:      mocks.NewMockAPIKeyRepository(t),
		usageBuf: recorder,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				APIToken: config.APITokenConfig{Enabled: false},
			},
		},
		l: zap.NewNop(),
	}

	result, err := svc.AuthenticateAPIKey(t.Context(), "trv_test.secret", "192.0.2.1", "test")

	require.Error(t, err)
	assert.Nil(t, result)
}

func TestAuthenticateAPIKeyRejectsRevokedKey(t *testing.T) {
	t.Parallel()

	generated, err := apikey.GenerateAPIKeySecret()
	require.NoError(t, err)

	key := &apikey.Key{
		ID:             pulid.MustNew("ak_"),
		KeyPrefix:      generated.Prefix,
		SecretHash:     generated.Hash,
		SecretSalt:     generated.Salt,
		Status:         apikey.StatusRevoked,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAPIKeyRepository(t)
	repo.EXPECT().GetByPrefix(mock.Anything, generated.Prefix).Return(key, nil)

	recorder := &stubUsageRecorder{}
	svc := &Service{
		akr:      repo,
		usageBuf: recorder,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				APIToken: config.APITokenConfig{Enabled: true},
			},
		},
		l: zap.NewNop(),
	}

	result, authErr := svc.AuthenticateAPIKey(t.Context(), generated.Token(), "192.0.2.1", "test")

	require.Error(t, authErr)
	assert.Nil(t, result)
}

func TestAuthenticateAPIKeyRejectsExpiredKey(t *testing.T) {
	t.Parallel()

	generated, err := apikey.GenerateAPIKeySecret()
	require.NoError(t, err)

	key := &apikey.Key{
		ID:             pulid.MustNew("ak_"),
		KeyPrefix:      generated.Prefix,
		SecretHash:     generated.Hash,
		SecretSalt:     generated.Salt,
		Status:         apikey.StatusActive,
		ExpiresAt:      timeutils.NowUnix() - 60,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAPIKeyRepository(t)
	repo.EXPECT().GetByPrefix(mock.Anything, generated.Prefix).Return(key, nil)

	recorder := &stubUsageRecorder{}
	svc := &Service{
		akr:      repo,
		usageBuf: recorder,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				APIToken: config.APITokenConfig{Enabled: true},
			},
		},
		l: zap.NewNop(),
	}

	result, authErr := svc.AuthenticateAPIKey(t.Context(), generated.Token(), "192.0.2.1", "test")

	require.Error(t, authErr)
	assert.Nil(t, result)
}

func TestAuthenticateAPIKeyRejectsInvalidSecret(t *testing.T) {
	t.Parallel()

	generated, err := apikey.GenerateAPIKeySecret()
	require.NoError(t, err)

	key := &apikey.Key{
		ID:             pulid.MustNew("ak_"),
		KeyPrefix:      generated.Prefix,
		SecretHash:     generated.Hash,
		SecretSalt:     generated.Salt,
		Status:         apikey.StatusActive,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAPIKeyRepository(t)
	repo.EXPECT().GetByPrefix(mock.Anything, generated.Prefix).Return(key, nil)

	recorder := &stubUsageRecorder{}
	svc := &Service{
		akr:      repo,
		usageBuf: recorder,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				APIToken: config.APITokenConfig{Enabled: true},
			},
		},
		l: zap.NewNop(),
	}

	result, authErr := svc.AuthenticateAPIKey(
		t.Context(),
		generated.Prefix+".wrong-secret",
		"192.0.2.1",
		"test",
	)

	require.Error(t, authErr)
	assert.Nil(t, result)
}

func TestAuthenticateAPIKeySuccessReturnsMachinePrincipal(t *testing.T) {
	t.Parallel()

	generated, err := apikey.GenerateAPIKeySecret()
	require.NoError(t, err)

	key := &apikey.Key{
		ID:             pulid.MustNew("ak_"),
		KeyPrefix:      generated.Prefix,
		SecretHash:     generated.Hash,
		SecretSalt:     generated.Salt,
		Status:         apikey.StatusActive,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}

	repo := mocks.NewMockAPIKeyRepository(t)
	repo.EXPECT().GetByPrefix(mock.Anything, generated.Prefix).Return(key, nil)
	recorder := &stubUsageRecorder{}

	svc := &Service{
		akr:      repo,
		usageBuf: recorder,
		cfg: &config.Config{
			Security: config.SecurityConfig{
				APIToken: config.APITokenConfig{Enabled: true},
			},
		},
		l: zap.NewNop(),
	}

	result, authErr := svc.AuthenticateAPIKey(
		t.Context(),
		generated.Token(),
		"192.0.2.1",
		"integration-test",
	)

	require.NoError(t, authErr)
	require.NotNil(t, result)
	assert.Equal(t, key.ID, result.PrincipalID)
	assert.Equal(t, key.ID, result.APIKeyID)
	assert.Equal(t, key.OrganizationID, result.OrganizationID)
	assert.Equal(t, key.BusinessUnitID, result.BusinessUnitID)
	assert.True(t, result.UserID.IsNil())
	assert.Equal(t, 1, recorder.calls)
	assert.Equal(t, key.ID, recorder.lastEvent.APIKeyID)
	assert.Equal(t, "192.0.2.1", recorder.lastEvent.IPAddress)
	assert.Equal(t, "integration-test", recorder.lastEvent.UserAgent)
	assert.False(t, recorder.lastEvent.OccurredAt.IsZero())
}
