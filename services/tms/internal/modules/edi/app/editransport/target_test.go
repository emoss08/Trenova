package editransport

import (
	"maps"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/stretchr/testify/assert"
)

func TestTargetChangedComparesEveryAddressAndHostKey(t *testing.T) {
	t.Parallel()

	before := map[string]any{
		ConfigKeyEndpointURL:  "https://as2.partner.test/in",
		configKeyHost:         "sftp.partner.test",
		configKeyPort:         "22",
		configKeyKnownHostKey: "ssh-ed25519 AAAA",
		configKeyUsername:     "carrier",
	}

	for _, tc := range []struct {
		name    string
		key     string
		value   any
		changed bool
	}{
		{name: "endpoint", key: ConfigKeyEndpointURL, value: "https://evil.test/in", changed: true},
		{name: "host", key: configKeyHost, value: "sftp.evil.test", changed: true},
		{name: "port", key: configKeyPort, value: "2222", changed: true},
		{name: "known host key", key: configKeyKnownHostKey, value: "ssh-rsa B", changed: true},
		{name: "trailing slash", key: ConfigKeyEndpointURL, value: "https://as2.partner.test/in/"},
		{name: "numeric port", key: configKeyPort, value: float64(22)},
		{name: "username", key: configKeyUsername, value: "other"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			after := maps.Clone(before)
			after[tc.key] = tc.value

			assert.Equal(t, tc.changed, TargetChanged(before, after))
		})
	}
}

func TestSecretInUseFollowsTheMethodAndAuthMode(t *testing.T) {
	t.Parallel()

	password := map[string]any{configKeyAuthMode: AuthModePassword}
	privateKey := map[string]any{configKeyAuthMode: "privateKey"}

	assert.True(t, SecretInUse(edi.ConnectionMethodSFTP, password, secretKeyPassword))
	assert.False(t, SecretInUse(edi.ConnectionMethodSFTP, password, secretKeyPrivateKey))
	assert.True(t, SecretInUse(edi.ConnectionMethodVAN, privateKey, secretKeyPrivateKey))
	assert.False(t, SecretInUse(edi.ConnectionMethodVAN, privateKey, secretKeyPassword))
	assert.True(t, SecretInUse(edi.ConnectionMethodAS2, nil, SecretKeyAS2PrivateKey))
	assert.True(t, SecretInUse(edi.ConnectionMethodAS2, nil, SecretKeyBasicAuthPassword))
	assert.False(t, SecretInUse(edi.ConnectionMethodAS2, nil, secretKeyPassword))
	assert.False(t, SecretInUse(edi.ConnectionMethodInternal, nil, secretKeyPassword))
}

func TestAnAS2SigningKeyIsNotBoundToTheTarget(t *testing.T) {
	t.Parallel()

	assert.False(t, SecretBoundToTarget(edi.ConnectionMethodAS2, SecretKeyAS2PrivateKey))
	assert.True(t, SecretBoundToTarget(edi.ConnectionMethodAS2, SecretKeyBasicAuthPassword))
	assert.True(t, SecretBoundToTarget(edi.ConnectionMethodSFTP, secretKeyPrivateKey))
}
