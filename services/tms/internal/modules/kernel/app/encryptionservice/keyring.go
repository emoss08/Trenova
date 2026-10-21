package encryptionservice

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hkdf"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
)

const (
	localHKDFKeyIDPrefix = "local-hkdf-v2:"
	localKEKInfo         = "trenova/encryption/local-kek/v2"
	localKeyIDInfo       = "trenova/encryption/local-key-id/v2"
	localKEKSalt         = "trenova-envelope-local-kek"
	localKeyIDBytes      = 8
)

var (
	ErrLocalKeyRequired = errors.New(
		"security.encryption.key is required for the local key manager",
	)
	ErrDuplicateLocalKey = errors.New("security.encryption.retiredKeys repeats a configured key")
)

type localKEK struct {
	id  string
	key []byte
}

type LocalKeyManager struct {
	active localKEK
	byID   map[string][]byte
	legacy [][]byte
}

func NewLocalKeyManager(secret string) LocalKeyManager {
	manager, err := NewLocalKeyring(secret, nil)
	if err != nil {
		return LocalKeyManager{}
	}
	return manager
}

func NewLocalKeyring(active string, retired []string) (LocalKeyManager, error) {
	activeSecret := strings.TrimSpace(active)
	if activeSecret == "" {
		return LocalKeyManager{}, ErrLocalKeyRequired
	}

	secrets := make([]string, 0, len(retired)+1)
	secrets = append(secrets, activeSecret)
	for _, secret := range retired {
		trimmed := strings.TrimSpace(secret)
		if trimmed == "" {
			continue
		}
		secrets = append(secrets, trimmed)
	}

	manager := LocalKeyManager{
		byID:   make(map[string][]byte, len(secrets)),
		legacy: make([][]byte, 0, len(secrets)),
	}
	for i, secret := range secrets {
		kek, err := deriveLocalKEK(secret)
		if err != nil {
			return LocalKeyManager{}, err
		}
		if _, exists := manager.byID[kek.id]; exists {
			return LocalKeyManager{}, ErrDuplicateLocalKey
		}
		manager.byID[kek.id] = kek.key
		legacy := sha256.Sum256([]byte(secret))
		manager.legacy = append(manager.legacy, legacy[:])
		if i == 0 {
			manager.active = kek
		}
	}
	return manager, nil
}

func deriveLocalKEK(secret string) (localKEK, error) {
	key, err := hkdf.Key(sha256.New, []byte(secret), []byte(localKEKSalt), localKEKInfo, dekSize)
	if err != nil {
		return localKEK{}, fmt.Errorf("derive local key: %w", err)
	}
	id, err := hkdf.Key(
		sha256.New,
		[]byte(secret),
		[]byte(localKEKSalt),
		localKeyIDInfo,
		localKeyIDBytes,
	)
	if err != nil {
		return localKEK{}, fmt.Errorf("derive local key id: %w", err)
	}
	return localKEK{id: localHKDFKeyIDPrefix + hex.EncodeToString(id), key: key}, nil
}

func (m LocalKeyManager) WrapKey(_ context.Context, dek []byte, aad AAD) (*WrappedKey, error) {
	if len(m.active.key) == 0 {
		return nil, ErrKeyManagerDisabled
	}
	gcm, err := newGCM(m.active.key)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return &WrappedKey{
		Provider:  localProvider,
		KeyID:     m.active.id,
		Algorithm: localWrappingAlg,
		Nonce:     nonce,
		Bytes:     gcm.Seal(nil, nonce, dek, aad.Bytes()),
	}, nil
}

//nolint:gocritic // hugeParam: WrappedKey is fixed by the KeyManager interface.
func (m LocalKeyManager) UnwrapKey(_ context.Context, wrapped WrappedKey, aad AAD) ([]byte, error) {
	if wrapped.Provider != localProvider || wrapped.Algorithm != localWrappingAlg {
		return nil, ErrUnknownKeyID
	}

	if wrapped.KeyID == localKeyID {
		return m.unwrapLegacy(wrapped, aad)
	}

	key, ok := m.byID[wrapped.KeyID]
	if !ok {
		return nil, ErrUnknownKeyID
	}
	return openWrappedDEK(key, wrapped, aad)
}

//nolint:gocritic // hugeParam: WrappedKey is fixed by the KeyManager interface.
func (m LocalKeyManager) unwrapLegacy(wrapped WrappedKey, aad AAD) ([]byte, error) {
	if len(m.legacy) == 0 {
		return nil, ErrUnknownKeyID
	}
	var lastErr error
	for _, key := range m.legacy {
		dek, err := openWrappedDEK(key, wrapped, aad)
		if err == nil {
			return dek, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (m LocalKeyManager) ActiveKeyID() string {
	return m.active.id
}

func (m LocalKeyManager) Provider() string {
	return localProvider
}

//nolint:gocritic // hugeParam: WrappedKey is fixed by the KeyManager interface.
func openWrappedDEK(key []byte, wrapped WrappedKey, aad AAD) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(wrapped.Nonce) != gcm.NonceSize() {
		return nil, ErrCiphertextFormat
	}

	dek, err := gcm.Open(nil, wrapped.Nonce, wrapped.Bytes, aad.Bytes())
	if err != nil {
		return nil, err
	}
	if len(dek) != dekSize {
		return nil, ErrInvalidEnvelope
	}
	return dek, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

type Keyring struct {
	active   KeyManager
	managers []KeyManager
}

func NewKeyring(active KeyManager, readers ...KeyManager) *Keyring {
	managers := make([]KeyManager, 0, len(readers)+1)
	managers = append(managers, active)
	return &Keyring{active: active, managers: append(managers, readers...)}
}

func (k *Keyring) WrapKey(ctx context.Context, dek []byte, aad AAD) (*WrappedKey, error) {
	return k.active.WrapKey(ctx, dek, aad)
}

//nolint:gocritic // hugeParam: WrappedKey is fixed by the KeyManager interface.
func (k *Keyring) UnwrapKey(ctx context.Context, wrapped WrappedKey, aad AAD) ([]byte, error) {
	err := ErrUnknownKeyID
	for _, manager := range k.managers {
		if manager.Provider() != wrapped.Provider {
			continue
		}
		dek, unwrapErr := manager.UnwrapKey(ctx, wrapped, aad)
		if unwrapErr == nil {
			return dek, nil
		}
		if !errors.Is(unwrapErr, ErrUnknownKeyID) || errors.Is(err, ErrUnknownKeyID) {
			err = unwrapErr
		}
	}
	return nil, err
}

func (k *Keyring) ActiveKeyID() string {
	return k.active.ActiveKeyID()
}

func (k *Keyring) Provider() string {
	return k.active.Provider()
}
