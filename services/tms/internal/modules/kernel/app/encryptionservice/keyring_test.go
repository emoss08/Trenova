package encryptionservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

const (
	oldLocalSecret = "the-previous-local-encryption-key-of-32-chars"
	newLocalSecret = "the-current-local-encryption-key-of-32-chars!"
	testKMSKey     = "projects/test/locations/us/keyRings/autokey/cryptoKeys/trenova"
)

type sha256LocalKeyManager struct {
	key []byte
}

func newSHA256LocalKeyManager(secret string) sha256LocalKeyManager {
	sum := sha256.Sum256([]byte(secret))
	return sha256LocalKeyManager{key: sum[:]}
}

func (m sha256LocalKeyManager) WrapKey(
	_ context.Context,
	dek []byte,
	aad AAD,
) (*WrappedKey, error) {
	gcm, err := newGCM(m.key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return &WrappedKey{
		Provider:  localProvider,
		KeyID:     localKeyID,
		Algorithm: localWrappingAlg,
		Nonce:     nonce,
		Bytes:     gcm.Seal(nil, nonce, dek, aad.Bytes()),
	}, nil
}

//
//nolint:gocritic // hugeParam: WrappedKey is fixed by the KeyManager interface.
func (m sha256LocalKeyManager) UnwrapKey(
	_ context.Context,
	wrapped WrappedKey,
	aad AAD,
) ([]byte, error) {
	if wrapped.KeyID != localKeyID {
		return nil, ErrUnknownKeyID
	}
	return openWrappedDEK(m.key, wrapped, aad)
}

func (sha256LocalKeyManager) ActiveKeyID() string { return localKeyID }

func (sha256LocalKeyManager) Provider() string { return localProvider }

func mustLocalKeyring(t *testing.T, active string, retired ...string) LocalKeyManager {
	t.Helper()
	keyring, err := NewLocalKeyring(active, retired)
	require.NoError(t, err)
	return keyring
}

func TestLocalKeyring_SealsUnderAnHKDFKeyThatSHA256CannotOpen(t *testing.T) {
	t.Parallel()

	aad := testAAD()
	sealed, err := NewWithKeyManager(mustLocalKeyring(t, newLocalSecret)).
		EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	env, err := decodeEnvelope(sealed)
	require.NoError(t, err)
	assert.True(t, strings.HasPrefix(env.KeyID, localHKDFKeyIDPrefix))
	assert.NotContains(t, env.KeyID, newLocalSecret)

	_, err = NewWithKeyManager(newSHA256LocalKeyManager(newLocalSecret)).
		DecryptStringWithAAD(sealed, aad)
	require.Error(t, err)
}

func TestLocalKeyring_ReadsEnvelopesSealedUnderTheSHA256Key(t *testing.T) {
	t.Parallel()

	aad := testAAD()
	legacy, err := NewWithKeyManager(newSHA256LocalKeyManager(oldLocalSecret)).
		EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	for name, keyring := range map[string]LocalKeyManager{
		"same key":    mustLocalKeyring(t, oldLocalSecret),
		"retired key": mustLocalKeyring(t, newLocalSecret, oldLocalSecret),
	} {
		plaintext, err := NewWithKeyManager(keyring).DecryptStringWithAAD(legacy, aad)
		require.NoError(t, err, name)
		assert.Equal(t, "client-secret", plaintext, name)
	}

	_, err = NewWithKeyManager(mustLocalKeyring(t, newLocalSecret)).
		DecryptStringWithAAD(legacy, aad)
	require.Error(t, err)
}

func TestLocalKeyring_RotationKeepsRetiredKeysReadableUntilRewrapped(t *testing.T) {
	t.Parallel()

	aad := testAAD()
	sealed, err := NewWithKeyManager(mustLocalKeyring(t, oldLocalSecret)).
		EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	rotated := NewWithKeyManager(mustLocalKeyring(t, newLocalSecret, oldLocalSecret))
	newOnly := NewWithKeyManager(mustLocalKeyring(t, newLocalSecret))

	_, err = newOnly.DecryptStringWithAAD(sealed, aad)
	require.ErrorIs(t, err, ErrUnknownKeyID)
	require.True(t, rotated.NeedsRewrap(sealed))

	rewrapped, changed, err := rotated.RewrapEnvelope(t.Context(), sealed, aad)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, rotated.NeedsRewrap(rewrapped))

	plaintext, err := newOnly.DecryptStringWithAAD(rewrapped, aad)
	require.NoError(t, err)
	assert.Equal(t, "client-secret", plaintext)

	again, changed, err := rotated.RewrapEnvelope(t.Context(), rewrapped, aad)
	require.NoError(t, err)
	assert.False(t, changed)
	assert.Equal(t, rewrapped, again)
}

func TestRewrapEnvelope_RefusesAnEnvelopeSealedForAnotherAAD(t *testing.T) {
	t.Parallel()

	aad := testAAD()
	sealed, err := NewWithKeyManager(mustLocalKeyring(t, oldLocalSecret)).
		EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	other := aad
	other.ResourceID = "documents/other.pdf"
	rotated := NewWithKeyManager(mustLocalKeyring(t, newLocalSecret, oldLocalSecret))
	_, _, err = rotated.RewrapEnvelope(t.Context(), sealed, other)
	require.ErrorIs(t, err, ErrInvalidEnvelope)
}

func TestNewLocalKeyring_RejectsAMissingOrRepeatedKey(t *testing.T) {
	t.Parallel()

	_, err := NewLocalKeyring("  ", nil)
	require.ErrorIs(t, err, ErrLocalKeyRequired)

	_, err = NewLocalKeyring(newLocalSecret, []string{" " + newLocalSecret})
	require.ErrorIs(t, err, ErrDuplicateLocalKey)
}

func failingKMSDialer(context.Context, ...option.ClientOption) (kmsClient, error) {
	return nil, errors.New("could not find default credentials")
}

func fakeKMSDialer(client *fakeKMSClient) kmsDialer {
	return func(context.Context, ...option.ClientOption) (kmsClient, error) {
		return client, nil
	}
}

func encryptionConfig(enc securityconfig.EncryptionConfig) *config.Config {
	return &config.Config{Security: securityconfig.Config{Encryption: enc}}
}

func TestKeyManagerFromConfig_FailsClosed(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		enc  securityconfig.EncryptionConfig
		want error
	}{
		{
			name: "kms client cannot be created",
			enc: securityconfig.EncryptionConfig{
				KeyManager: securityconfig.EncryptionKeyManagerGCPAutokey,
				GCPKMS:     securityconfig.GCPKMSConfig{CryptoKey: testKMSKey},
			},
			want: ErrKMSUnavailable,
		},
		{
			name: "kms key missing",
			enc: securityconfig.EncryptionConfig{
				KeyManager: securityconfig.EncryptionKeyManagerGCPAutokey,
			},
			want: ErrKMSKeyRequired,
		},
		{
			name: "local key missing",
			enc: securityconfig.EncryptionConfig{
				KeyManager: securityconfig.EncryptionKeyManagerLocal,
			},
			want: ErrLocalKeyRequired,
		},
		{
			name: "default manager with no key",
			enc:  securityconfig.EncryptionConfig{},
			want: ErrLocalKeyRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			manager, err := keyManagerFromConfig(encryptionConfig(tt.enc), failingKMSDialer)
			require.ErrorIs(t, err, tt.want)
			assert.Nil(t, manager)
		})
	}
}

func TestKeyManagerFromConfig_KMSReadsEnvelopesSealedByTheLocalKey(t *testing.T) {
	t.Parallel()

	aad := testAAD()
	local, err := NewWithKeyManager(mustLocalKeyring(t, oldLocalSecret)).
		EncryptStringWithAAD("client-secret", aad)
	require.NoError(t, err)

	client := &fakeKMSClient{}
	manager, err := keyManagerFromConfig(encryptionConfig(securityconfig.EncryptionConfig{
		KeyManager:  securityconfig.EncryptionKeyManagerGCPAutokey,
		RetiredKeys: []string{oldLocalSecret},
		GCPKMS:      securityconfig.GCPKMSConfig{CryptoKey: testKMSKey},
	}), fakeKMSDialer(client))
	require.NoError(t, err)
	svc := NewWithKeyManager(manager)

	plaintext, err := svc.DecryptStringWithAAD(local, aad)
	require.NoError(t, err)
	assert.Equal(t, "client-secret", plaintext)
	assert.Nil(t, client.decryptAAD)

	rewrapped, changed, err := svc.RewrapEnvelope(t.Context(), local, aad)
	require.NoError(t, err)
	require.True(t, changed)
	env, err := decodeEnvelope(rewrapped)
	require.NoError(t, err)
	assert.Equal(t, gcpAutokeyProvider, env.Provider)
	assert.Equal(t, testKMSKey, env.KeyID)
	assert.Empty(t, env.WrappedDEKNonce)

	plaintext, err = NewWithKeyManager(NewGCPAutokeyManager(client, testKMSKey, 0)).
		DecryptStringWithAAD(rewrapped, aad)
	require.NoError(t, err)
	assert.Equal(t, "client-secret", plaintext)
}
