package encryptionservice

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	kms "cloud.google.com/go/kms/apiv1"
	kmspb "cloud.google.com/go/kms/apiv1/kmspb"
	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/pulid"
	gax "github.com/googleapis/gax-go/v2"
	"go.uber.org/fx"
	"google.golang.org/api/option"
)

var (
	ErrValueRequired      = errors.New("value is required")
	ErrCiphertextFormat   = errors.New("invalid ciphertext format")
	ErrCiphertextShort    = errors.New("ciphertext too short")
	ErrInvalidEnvelope    = errors.New("invalid encryption envelope")
	ErrKeyManagerDisabled = errors.New("encryption key manager is disabled")
	ErrUnknownKeyID       = errors.New("unknown encryption key id")
)

type Params struct {
	fx.In

	Config *config.Config
}

type Service struct {
	keyManager KeyManager
}

type Purpose string

const (
	PurposeDocument                    Purpose = "document"
	PurposeDocumentUploadSession       Purpose = "document_upload_session"
	PurposeIAMOIDCClientSecret         Purpose = "iam_oidc_client_secret" // #nosec G101 -- AAD label, not a credential.
	PurposeEDICommunicationProfile     Purpose = "edi_communication_profile"
	PurposeEDICommunicationProfileItem Purpose = "edi_communication_profile_item"

	CryptoModeEnvelopeV1 = "envelope_v1"

	envelopePrefix       = "trenova-envelope:v1:"
	envelopeVersion      = 1
	dekSize              = 32
	payloadAlgorithm     = "AES-256-GCM"
	localWrappingAlg     = "LOCAL-AES-256-GCM"
	gcpKMSWrappingAlg    = "GCP-CLOUD-KMS"
	localKeyID           = "local-sha256-v1"
	localProvider        = "local"
	gcpAutokeyProvider   = "gcp-autokey"
	disabledProvider     = "disabled"
	defaultKMSOpTimeout  = 10 * time.Second
	defaultRetryAttempts = 3
)

type AAD struct {
	Purpose        Purpose
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	ResourceID     string
}

type KeyManager interface {
	WrapKey(context.Context, []byte, AAD) (*WrappedKey, error)
	UnwrapKey(context.Context, WrappedKey, AAD) ([]byte, error)
	ActiveKeyID() string
	Provider() string
}

type WrappedKey struct {
	Provider  string
	KeyID     string
	Algorithm string
	Nonce     []byte
	Bytes     []byte
}

type envelope struct {
	Version           int    `json:"version"`
	Provider          string `json:"provider"`
	KeyID             string `json:"keyId"`
	WrappedDEK        string `json:"wrappedDek"`
	WrappedDEKNonce   string `json:"wrappedDekNonce,omitempty"`
	WrappingAlgorithm string `json:"wrappingAlgorithm"`
	PayloadAlgorithm  string `json:"payloadAlgorithm"`
	AADHash           string `json:"aadHash"`
	Nonce             string `json:"nonce"`
	Ciphertext        string `json:"ciphertext"`
}

type kmsClient interface {
	Encrypt(context.Context, *kmspb.EncryptRequest, ...gax.CallOption) (*kmspb.EncryptResponse, error)
	Decrypt(context.Context, *kmspb.DecryptRequest, ...gax.CallOption) (*kmspb.DecryptResponse, error)
	Close() error
}

func New(p Params) *Service {
	return &Service{keyManager: keyManagerFromConfig(p.Config)}
}

func NewWithKeyManager(keyManager KeyManager) *Service {
	return &Service{keyManager: keyManager}
}

func (s *Service) EncryptString(value string) (string, error) {
	return s.EncryptStringWithAAD(value, AAD{})
}

func (s *Service) EncryptStringWithAAD(value string, aad AAD) (string, error) {
	plaintext := strings.TrimSpace(value)
	if plaintext == "" {
		return "", ErrValueRequired
	}

	encrypted, err := s.EncryptBytesWithAAD([]byte(plaintext), aad)
	if err != nil {
		return "", err
	}

	return encrypted, nil
}

func (s *Service) EncryptBytesWithAAD(plaintext []byte, aad AAD) (string, error) {
	return s.EncryptBytesWithAADContext(context.Background(), plaintext, aad)
}

func (s *Service) EncryptBytesWithAADContext(
	ctx context.Context,
	plaintext []byte,
	aad AAD,
) (string, error) {
	if len(plaintext) == 0 {
		return "", ErrValueRequired
	}
	if s.keyManager == nil {
		return "", ErrKeyManagerDisabled
	}

	dek := make([]byte, dekSize)
	if _, err := io.ReadFull(rand.Reader, dek); err != nil {
		return "", err
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, aad.Bytes())

	wrapped, err := s.keyManager.WrapKey(ctx, dek, aad)
	if err != nil {
		return "", err
	}

	env := envelope{
		Version:           envelopeVersion,
		Provider:          wrapped.Provider,
		KeyID:             wrapped.KeyID,
		WrappedDEK:        base64.StdEncoding.EncodeToString(wrapped.Bytes),
		WrappedDEKNonce:   base64.StdEncoding.EncodeToString(wrapped.Nonce),
		WrappingAlgorithm: wrapped.Algorithm,
		PayloadAlgorithm:  payloadAlgorithm,
		AADHash:           aad.Hash(),
		Nonce:             base64.StdEncoding.EncodeToString(nonce),
		Ciphertext:        base64.StdEncoding.EncodeToString(ciphertext),
	}

	payload, err := sonic.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshal encryption envelope: %w", err)
	}

	return envelopePrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func (s *Service) DecryptString(value string) (string, error) {
	plaintext, err := s.DecryptBytesWithAAD(value, AAD{})
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Service) DecryptStringWithAAD(value string, aad AAD) (string, error) {
	plaintext, err := s.DecryptBytesWithAAD(value, aad)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (s *Service) DecryptBytesWithAAD(value string, aad AAD) ([]byte, error) {
	return s.DecryptBytesWithAADContext(context.Background(), value, aad)
}

func (s *Service) DecryptBytesWithAADContext(
	ctx context.Context,
	value string,
	aad AAD,
) ([]byte, error) {
	ciphertext := strings.TrimSpace(value)
	if ciphertext == "" {
		return nil, ErrValueRequired
	}
	if !IsEnvelope(ciphertext) {
		return nil, ErrInvalidEnvelope
	}
	if s.keyManager == nil {
		return nil, ErrKeyManagerDisabled
	}

	env, err := decodeEnvelope(ciphertext)
	if err != nil {
		return nil, err
	}
	if env.AADHash != aad.Hash() {
		return nil, ErrInvalidEnvelope
	}

	wrapped, err := env.wrappedKey()
	if err != nil {
		return nil, err
	}
	dek, err := s.keyManager.UnwrapKey(ctx, wrapped, aad)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(env.Nonce)
	if err != nil {
		return nil, ErrCiphertextFormat
	}
	payload, err := base64.StdEncoding.DecodeString(env.Ciphertext)
	if err != nil {
		return nil, ErrCiphertextFormat
	}

	block, err := aes.NewCipher(dek)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, ErrCiphertextFormat
	}

	plaintext, err := gcm.Open(nil, nonce, payload, aad.Bytes())
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func (s *Service) RewrapEnvelopeWithAAD(value string, aad AAD) (string, error) {
	return s.RewrapEnvelopeWithAADContext(context.Background(), value, aad)
}

func (s *Service) RewrapEnvelopeWithAADContext(
	ctx context.Context,
	value string,
	aad AAD,
) (string, error) {
	ciphertext := strings.TrimSpace(value)
	if ciphertext == "" {
		return "", ErrValueRequired
	}
	if !IsEnvelope(ciphertext) {
		return "", ErrInvalidEnvelope
	}
	if s.keyManager == nil {
		return "", ErrKeyManagerDisabled
	}

	env, err := decodeEnvelope(ciphertext)
	if err != nil {
		return "", err
	}
	wrapped, err := env.wrappedKey()
	if err != nil {
		return "", err
	}
	dek, err := s.keyManager.UnwrapKey(ctx, wrapped, aad)
	if err != nil {
		return "", err
	}
	nextWrapped, err := s.keyManager.WrapKey(ctx, dek, aad)
	if err != nil {
		return "", err
	}

	env.Provider = nextWrapped.Provider
	env.KeyID = nextWrapped.KeyID
	env.WrappedDEK = base64.StdEncoding.EncodeToString(nextWrapped.Bytes)
	env.WrappedDEKNonce = base64.StdEncoding.EncodeToString(nextWrapped.Nonce)
	env.WrappingAlgorithm = nextWrapped.Algorithm
	env.AADHash = aad.Hash()

	payload, err := sonic.Marshal(env)
	if err != nil {
		return "", fmt.Errorf("marshal encryption envelope: %w", err)
	}

	return envelopePrefix + base64.StdEncoding.EncodeToString(payload), nil
}

func IsEnvelope(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), envelopePrefix)
}

func CryptoModeForCiphertext(value string) string {
	if IsEnvelope(value) {
		return CryptoModeEnvelopeV1
	}
	return ""
}

func (a AAD) Bytes() []byte {
	parts := []string{
		"v=1",
		"purpose=" + string(a.Purpose),
		"org=" + a.OrganizationID.String(),
		"bu=" + a.BusinessUnitID.String(),
		"resource=" + strings.TrimSpace(a.ResourceID),
	}
	return []byte(strings.Join(parts, "\n"))
}

func (a AAD) Hash() string {
	sum := sha256.Sum256(a.Bytes())
	return base64.StdEncoding.EncodeToString(sum[:])
}

func decodeEnvelope(ciphertext string) (envelope, error) {
	var env envelope
	payload, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ciphertext, envelopePrefix))
	if err != nil {
		return env, ErrCiphertextFormat
	}
	if err = sonic.Unmarshal(payload, &env); err != nil {
		return env, ErrCiphertextFormat
	}
	if env.Version != envelopeVersion ||
		env.Provider == "" ||
		env.KeyID == "" ||
		env.WrappedDEK == "" ||
		env.WrappingAlgorithm == "" ||
		env.PayloadAlgorithm != payloadAlgorithm ||
		env.AADHash == "" ||
		env.Nonce == "" ||
		env.Ciphertext == "" {
		return env, ErrInvalidEnvelope
	}
	return env, nil
}

func (e envelope) wrappedKey() (WrappedKey, error) {
	wrappedDEK, err := base64.StdEncoding.DecodeString(e.WrappedDEK)
	if err != nil {
		return WrappedKey{}, ErrCiphertextFormat
	}
	var nonce []byte
	if e.WrappedDEKNonce != "" {
		nonce, err = base64.StdEncoding.DecodeString(e.WrappedDEKNonce)
		if err != nil {
			return WrappedKey{}, ErrCiphertextFormat
		}
	}
	return WrappedKey{
		Provider:  e.Provider,
		KeyID:     e.KeyID,
		Algorithm: e.WrappingAlgorithm,
		Nonce:     nonce,
		Bytes:     wrappedDEK,
	}, nil
}

func keyManagerFromConfig(cfg *config.Config) KeyManager {
	if cfg == nil {
		return DisabledKeyManager{}
	}

	enc := cfg.Security.Encryption
	manager := strings.ToLower(strings.TrimSpace(enc.KeyManager))
	if manager == "" {
		manager = config.EncryptionKeyManagerLocal
	}

	switch manager {
	case config.EncryptionKeyManagerGCPAutokey:
		return newGCPAutokeyManager(enc.GCPKMS)
	case config.EncryptionKeyManagerDisabled:
		return DisabledKeyManager{}
	default:
		return NewLocalKeyManager(enc.Key)
	}
}

type LocalKeyManager struct {
	key []byte
}

func NewLocalKeyManager(secret string) LocalKeyManager {
	sum := sha256.Sum256([]byte(strings.TrimSpace(secret)))
	return LocalKeyManager{key: sum[:]}
}

func (m LocalKeyManager) WrapKey(_ context.Context, dek []byte, aad AAD) (*WrappedKey, error) {
	if len(m.key) == 0 {
		return nil, ErrKeyManagerDisabled
	}
	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return &WrappedKey{
		Provider:  localProvider,
		KeyID:     m.ActiveKeyID(),
		Algorithm: localWrappingAlg,
		Nonce:     nonce,
		Bytes:     gcm.Seal(nil, nonce, dek, aad.Bytes()),
	}, nil
}

func (m LocalKeyManager) UnwrapKey(_ context.Context, wrapped WrappedKey, aad AAD) ([]byte, error) {
	if wrapped.Provider != localProvider ||
		wrapped.KeyID != m.ActiveKeyID() ||
		wrapped.Algorithm != localWrappingAlg {
		return nil, ErrUnknownKeyID
	}

	block, err := aes.NewCipher(m.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
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

func (m LocalKeyManager) ActiveKeyID() string {
	return localKeyID
}

func (m LocalKeyManager) Provider() string {
	return localProvider
}

type DisabledKeyManager struct{}

func (DisabledKeyManager) WrapKey(context.Context, []byte, AAD) (*WrappedKey, error) {
	return nil, ErrKeyManagerDisabled
}

func (DisabledKeyManager) UnwrapKey(context.Context, WrappedKey, AAD) ([]byte, error) {
	return nil, ErrKeyManagerDisabled
}

func (DisabledKeyManager) ActiveKeyID() string {
	return ""
}

func (DisabledKeyManager) Provider() string {
	return disabledProvider
}

type GCPAutokeyManager struct {
	client    kmsClient
	activeKey string
	timeout   time.Duration
}

func NewGCPAutokeyManager(client kmsClient, activeKey string, timeout time.Duration) *GCPAutokeyManager {
	if timeout <= 0 {
		timeout = defaultKMSOpTimeout
	}
	return &GCPAutokeyManager{
		client:    client,
		activeKey: strings.TrimSpace(activeKey),
		timeout:   timeout,
	}
}

func (m *GCPAutokeyManager) WrapKey(ctx context.Context, dek []byte, aad AAD) (*WrappedKey, error) {
	if m == nil || m.client == nil || m.activeKey == "" {
		return nil, ErrKeyManagerDisabled
	}

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	resp, err := m.client.Encrypt(ctx, &kmspb.EncryptRequest{
		Name:                        m.activeKey,
		Plaintext:                   dek,
		AdditionalAuthenticatedData: aad.Bytes(),
	})
	if err != nil {
		return nil, err
	}

	return &WrappedKey{
		Provider:  gcpAutokeyProvider,
		KeyID:     m.activeKey,
		Algorithm: gcpKMSWrappingAlg,
		Bytes:     resp.Ciphertext,
	}, nil
}

func (m *GCPAutokeyManager) UnwrapKey(ctx context.Context, wrapped WrappedKey, aad AAD) ([]byte, error) {
	if m == nil || m.client == nil {
		return nil, ErrKeyManagerDisabled
	}
	if wrapped.Provider != gcpAutokeyProvider || wrapped.Algorithm != gcpKMSWrappingAlg {
		return nil, ErrUnknownKeyID
	}

	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	resp, err := m.client.Decrypt(ctx, &kmspb.DecryptRequest{
		Name:                        wrapped.KeyID,
		Ciphertext:                  wrapped.Bytes,
		AdditionalAuthenticatedData: aad.Bytes(),
	})
	if err != nil {
		return nil, err
	}
	if len(resp.Plaintext) != dekSize {
		return nil, ErrInvalidEnvelope
	}
	return resp.Plaintext, nil
}

func (m *GCPAutokeyManager) ActiveKeyID() string {
	if m == nil {
		return ""
	}
	return m.activeKey
}

func (m *GCPAutokeyManager) Provider() string {
	return gcpAutokeyProvider
}

func newGCPAutokeyManager(cfg config.GCPKMSConfig) KeyManager {
	activeKey := cfg.CryptoKey
	if strings.TrimSpace(activeKey) == "" {
		activeKey = cfg.KeyResource
	}
	activeKey = strings.TrimSpace(activeKey)
	if activeKey == "" {
		return DisabledKeyManager{}
	}

	ctx := context.Background()
	opts := make([]option.ClientOption, 0, 1)
	if cfg.CredentialsFile != "" {
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := kms.NewKeyManagementClient(ctx, opts...)
	if err != nil {
		return DisabledKeyManager{}
	}

	return NewGCPAutokeyManager(client, activeKey, cfg.Timeout)
}
