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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrValueRequired      = errors.New("value is required")
	ErrCiphertextFormat   = errors.New("invalid ciphertext format")
	ErrCiphertextShort    = errors.New("ciphertext too short")
	ErrInvalidEnvelope    = errors.New("invalid encryption envelope")
	ErrKeyManagerDisabled = errors.New("encryption key manager is disabled")
	ErrUnknownKeyID       = errors.New("unknown encryption key id")
	ErrKeyRejected        = errors.New("the wrapped key was refused")
	ErrKMSKeyRequired     = errors.New(
		"security.encryption.gcpKms.cryptoKey is required for the gcp-autokey key manager",
	)
	ErrKMSUnavailable = errors.New("the GCP KMS client could not be created")
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
	PurposeCapturePage                 Purpose = "capture_page"
	PurposeIAMOIDCClientSecret         Purpose = "iam_oidc_client_secret" // #nosec G101 -- AAD label, not a credential.
	PurposeEDICommunicationProfile     Purpose = "edi_communication_profile"
	PurposeEDICommunicationProfileItem Purpose = "edi_communication_profile_item"
	PurposeIntegrationSecret           Purpose = "integration_secret"          // #nosec G101 -- AAD label, not a credential.
	PurposeAgentExtensionSecret        Purpose = "agent_extension_secret"      // #nosec G101 -- AAD label, not a credential.
	PurposeAccountingConnectionToken   Purpose = "accounting_connection_token" // #nosec G101 -- AAD label, not a credential.
	PurposeAccountingAppSecret         Purpose = "accounting_app_secret"       // #nosec G101 -- AAD label, not a credential.
	PurposeAccountingPendingGrant      Purpose = "accounting_pending_grant"    // #nosec G101 -- AAD label, not a credential.
	PurposeAccountingWebhookState      Purpose = "accounting_webhook_state"
	PurposeAccountingOAuthVerifier     Purpose = "accounting_oauth_verifier"
	PurposeAccountingClientKey         Purpose = "accounting_client_key"        // #nosec G101 -- AAD label, not a credential.
	PurposeMFATOTPSecret               Purpose = "mfa_totp_secret"              // #nosec G101 -- AAD label, not a credential.
	PurposeAIProviderCredential        Purpose = "ai_provider_credential"       // #nosec G101 -- AAD label, not a credential.
	PurposeAIProviderDraftCredential   Purpose = "ai_provider_draft_credential" // #nosec G101 -- AAD label, not a credential.
	PurposeInboundMailboxSecret        Purpose = "inbound_mailbox_secret"       // #nosec G101 -- AAD label, not a credential.

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
	Encrypt(
		context.Context,
		*kmspb.EncryptRequest,
		...gax.CallOption,
	) (*kmspb.EncryptResponse, error)
	Decrypt(
		context.Context,
		*kmspb.DecryptRequest,
		...gax.CallOption,
	) (*kmspb.DecryptResponse, error)
	Close() error
}

func New(p Params) (*Service, error) {
	keyManager, err := keyManagerFromConfig(p.Config, dialKMS)
	if err != nil {
		return nil, err
	}
	return &Service{keyManager: keyManager}, nil
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

func (s *Service) RewrapEnvelope(
	ctx context.Context,
	value string,
	aad AAD,
) (rewrapped string, changed bool, err error) {
	ciphertext := strings.TrimSpace(value)
	if ciphertext == "" {
		return "", false, ErrValueRequired
	}
	if !IsEnvelope(ciphertext) {
		return "", false, ErrInvalidEnvelope
	}
	if s.keyManager == nil {
		return "", false, ErrKeyManagerDisabled
	}

	env, err := decodeEnvelope(ciphertext)
	if err != nil {
		return "", false, err
	}
	if env.AADHash != aad.Hash() {
		return "", false, ErrInvalidEnvelope
	}
	if env.Provider == s.keyManager.Provider() && env.KeyID == s.keyManager.ActiveKeyID() {
		return value, false, nil
	}

	wrapped, err := env.wrappedKey()
	if err != nil {
		return "", false, err
	}
	dek, err := s.keyManager.UnwrapKey(ctx, wrapped, aad)
	if err != nil {
		return "", false, err
	}
	nextWrapped, err := s.keyManager.WrapKey(ctx, dek, aad)
	if err != nil {
		return "", false, err
	}

	env.Provider = nextWrapped.Provider
	env.KeyID = nextWrapped.KeyID
	env.WrappedDEK = base64.StdEncoding.EncodeToString(nextWrapped.Bytes)
	env.WrappedDEKNonce = ""
	if len(nextWrapped.Nonce) > 0 {
		env.WrappedDEKNonce = base64.StdEncoding.EncodeToString(nextWrapped.Nonce)
	}
	env.WrappingAlgorithm = nextWrapped.Algorithm

	payload, err := sonic.Marshal(env)
	if err != nil {
		return "", false, fmt.Errorf("marshal encryption envelope: %w", err)
	}

	return envelopePrefix + base64.StdEncoding.EncodeToString(payload), true, nil
}

func (s *Service) NeedsRewrap(value string) bool {
	ciphertext := strings.TrimSpace(value)
	if s.keyManager == nil || !IsEnvelope(ciphertext) {
		return false
	}
	env, err := decodeEnvelope(ciphertext)
	if err != nil {
		return false
	}
	return env.Provider != s.keyManager.Provider() || env.KeyID != s.keyManager.ActiveKeyID()
}

func (s *Service) DecryptStringWithLegacy(
	value string,
	aad AAD,
) (plaintext string, legacy bool, err error) {
	plaintext, err = s.DecryptStringWithAAD(value, aad)
	if err == nil {
		return plaintext, false, nil
	}
	if aad.Hash() == (AAD{}).Hash() || !EnvelopeHasAAD(value, AAD{}) {
		return "", false, err
	}

	unbound, legacyErr := s.DecryptString(value)
	if legacyErr != nil {
		return "", false, err
	}
	return unbound, true, nil
}

func (s *Service) DecryptAndRebind(
	value string,
	aad AAD,
) (plaintext, rebound string, err error) {
	plaintext, legacy, err := s.DecryptStringWithLegacy(value, aad)
	if err != nil || !legacy {
		return plaintext, "", err
	}

	rebound, err = s.EncryptStringWithAAD(plaintext, aad)
	if err != nil {
		return plaintext, "", err
	}
	return plaintext, rebound, nil
}

func EnvelopeHasAAD(value string, aad AAD) bool {
	ciphertext := strings.TrimSpace(value)
	if !IsEnvelope(ciphertext) {
		return false
	}
	env, err := decodeEnvelope(ciphertext)
	if err != nil {
		return false
	}
	return env.AADHash == aad.Hash()
}

func AIProviderCredentialAAD(orgID, buID, providerID pulid.ID) AAD {
	return AAD{
		Purpose:        PurposeAIProviderCredential,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ResourceID:     providerID.String(),
	}
}

func AIProviderDraftCredentialAAD(orgID, buID, userID pulid.ID) AAD {
	return AAD{
		Purpose:        PurposeAIProviderDraftCredential,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ResourceID:     userID.String(),
	}
}

func InboundMailboxSecretAAD(orgID, buID, mailboxID pulid.ID, field string) AAD {
	return AAD{
		Purpose:        PurposeInboundMailboxSecret,
		OrganizationID: orgID,
		BusinessUnitID: buID,
		ResourceID:     mailboxID.String() + ":" + field,
	}
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

type kmsDialer func(context.Context, ...option.ClientOption) (kmsClient, error)

func dialKMS(ctx context.Context, opts ...option.ClientOption) (kmsClient, error) {
	return kms.NewKeyManagementClient(ctx, opts...)
}

func keyManagerFromConfig(cfg *config.Config, dial kmsDialer) (KeyManager, error) {
	if cfg == nil {
		return DisabledKeyManager{}, nil
	}

	enc := cfg.Security.Encryption
	manager := strings.ToLower(strings.TrimSpace(enc.KeyManager))
	if manager == "" {
		manager = config.EncryptionKeyManagerLocal
	}

	switch manager {
	case config.EncryptionKeyManagerGCPAutokey:
		active, err := newGCPAutokeyManager(&enc.GCPKMS, dial)
		if err != nil {
			return nil, err
		}
		reader, ok, err := localReader(&enc)
		if err != nil {
			return nil, err
		}
		if !ok {
			return active, nil
		}
		return NewKeyring(active, reader), nil
	case config.EncryptionKeyManagerDisabled:
		return DisabledKeyManager{}, nil
	default:
		keyring, err := NewLocalKeyring(enc.Key, enc.RetiredKeys)
		if err != nil {
			return nil, err
		}
		return keyring, nil
	}
}

func localReader(enc *config.EncryptionConfig) (KeyManager, bool, error) {
	secrets := make([]string, 0, len(enc.RetiredKeys)+1)
	if strings.TrimSpace(enc.Key) != "" {
		secrets = append(secrets, enc.Key)
	}
	for _, secret := range enc.RetiredKeys {
		if strings.TrimSpace(secret) != "" {
			secrets = append(secrets, secret)
		}
	}
	if len(secrets) == 0 {
		return nil, false, nil
	}
	reader, err := NewLocalKeyring(secrets[0], secrets[1:])
	if err != nil {
		return nil, false, err
	}
	return reader, true, nil
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

func NewGCPAutokeyManager(
	client kmsClient,
	activeKey string,
	timeout time.Duration,
) *GCPAutokeyManager {
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
		Bytes:     resp.GetCiphertext(),
	}, nil
}

// which three implementations and both call sites share
//
//nolint:gocritic // hugeParam: WrappedKey is fixed by the KeyManager interface,
func (m *GCPAutokeyManager) UnwrapKey(
	ctx context.Context,
	wrapped WrappedKey,
	aad AAD,
) ([]byte, error) {
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
		return nil, kmsUnwrapError(err)
	}
	if len(resp.GetPlaintext()) != dekSize {
		return nil, ErrInvalidEnvelope
	}
	return resp.GetPlaintext(), nil
}

func kmsUnwrapError(err error) error {
	switch status.Code(err) {
	case codes.InvalidArgument, codes.NotFound, codes.FailedPrecondition:
		return fmt.Errorf("%w: %w", ErrKeyRejected, err)
	default:
		return err
	}
}

func IsKeyRefused(err error) bool {
	return errors.Is(err, ErrKeyRejected) ||
		errors.Is(err, ErrUnknownKeyID) ||
		errors.Is(err, ErrInvalidEnvelope) ||
		errors.Is(err, ErrCiphertextFormat)
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

func newGCPAutokeyManager(cfg *config.GCPKMSConfig, dial kmsDialer) (KeyManager, error) {
	activeKey := cfg.CryptoKey
	if strings.TrimSpace(activeKey) == "" {
		activeKey = cfg.KeyResource
	}
	activeKey = strings.TrimSpace(activeKey)
	if activeKey == "" {
		return nil, ErrKMSKeyRequired
	}

	opts := make([]option.ClientOption, 0, 1)
	if cfg.CredentialsFile != "" {
		//nolint:staticcheck // SA1019: the replacement pins the credential type at the
		// call site, which would break deployments using authorized_user, impersonated
		// or workload-identity credentials. Needs a config field first.
		opts = append(opts, option.WithCredentialsFile(cfg.CredentialsFile))
	}

	client, err := dial(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrKMSUnavailable, err)
	}

	return NewGCPAutokeyManager(client, activeKey, cfg.Timeout), nil
}
