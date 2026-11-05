package edisetup

import (
	"maps"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sftpProfile() *edi.EDICommunicationProfile {
	return &edi.EDICommunicationProfile{
		Method: edi.ConnectionMethodSFTP,
		Config: map[string]any{
			"host":         "sftp.partner.test",
			"port":         "22",
			"authMode":     "password",
			"knownHostKey": "ssh-ed25519 AAAA",
		},
		EncryptedSecrets: map[string]string{"password": "sealed"},
	}
}

func as2Profile() *edi.EDICommunicationProfile {
	return &edi.EDICommunicationProfile{
		Method: edi.ConnectionMethodAS2,
		Config: map[string]any{"endpointUrl": "https://as2.partner.test/in"},
		EncryptedSecrets: map[string]string{
			"privateKey":           "sealed-key",
			"basicAuthPassword":    "sealed-basic",
			"decryptionPrivateKey": "sealed-decrypt",
		},
	}
}

func TestProfileRequiresThePasswordAgainWhenTheTargetChanges(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "host", key: "host", value: "sftp.attacker.test"},
		{name: "port", key: "port", value: "2222"},
		{name: "known host key", key: "knownHostKey", value: "ssh-ed25519 BBBB"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			existing := sftpProfile()
			config := maps.Clone(existing.Config)
			config[tc.key] = tc.value

			err := requireSecretsForNewTarget(&secretTargetCheck{
				method:   existing.Method,
				config:   config,
				existing: existing,
				secrets:  maps.Clone(existing.EncryptedSecrets),
			})

			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, "secrets.password", multiErr.Errors[0].Field)
		})
	}
}

func TestProfileRequiresAS2CredentialsAgainWhenTheEndpointChanges(t *testing.T) {
	t.Parallel()

	existing := as2Profile()

	err := requireSecretsForNewTarget(&secretTargetCheck{
		method:   existing.Method,
		config:   map[string]any{"endpointUrl": "https://evil.test/in"},
		existing: existing,
		secrets:  maps.Clone(existing.EncryptedSecrets),
	})

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}
	assert.ElementsMatch(t, []string{"secrets.privateKey", "secrets.basicAuthPassword"}, fields)
}

func TestProfileKeepsSecretsWhenTheTargetIsUnchanged(t *testing.T) {
	t.Parallel()

	existing := sftpProfile()
	config := maps.Clone(existing.Config)
	config["outboundDirectory"] = "/elsewhere"
	secrets := maps.Clone(existing.EncryptedSecrets)

	err := requireSecretsForNewTarget(&secretTargetCheck{
		method:   existing.Method,
		config:   config,
		existing: existing,
		secrets:  secrets,
	})

	require.NoError(t, err)
	assert.Equal(t, existing.EncryptedSecrets, secrets)
}

func TestProfileAcceptsANewTargetWithTheSecretReentered(t *testing.T) {
	t.Parallel()

	existing := sftpProfile()
	config := maps.Clone(existing.Config)
	config["host"] = "sftp.moved.test"

	err := requireSecretsForNewTarget(&secretTargetCheck{
		method:   existing.Method,
		config:   config,
		entered:  map[string]string{"password": "new-password"},
		existing: existing,
		secrets:  map[string]string{"password": "sealed-new"},
	})

	require.NoError(t, err)
}

func TestProfileDropsTheUnusedCredentialWhenTheTargetChanges(t *testing.T) {
	t.Parallel()

	existing := sftpProfile()
	existing.EncryptedSecrets["privateKey"] = "sealed-stale-key"
	config := maps.Clone(existing.Config)
	config["host"] = "sftp.moved.test"
	secrets := map[string]string{"password": "sealed-new", "privateKey": "sealed-stale-key"}

	err := requireSecretsForNewTarget(&secretTargetCheck{
		method:   existing.Method,
		config:   config,
		entered:  map[string]string{"password": "new-password"},
		existing: existing,
		secrets:  secrets,
	})

	require.NoError(t, err)
	assert.Equal(t, map[string]string{"password": "sealed-new"}, secrets)
}
