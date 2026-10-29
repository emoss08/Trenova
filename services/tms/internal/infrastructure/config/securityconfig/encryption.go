package securityconfig

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrEncryptionKeyIsInsecure = errors.New(
		"encryption key contains insecure default value",
	)
	ErrProductionEncryptionModeRequired = errors.New(
		"production and staging require security.encryption.mode=envelope",
	)
	ErrProductionKMSRequired = errors.New(
		"production and staging require security.encryption.keyManager=gcp-autokey, or keyManager=local with security.encryption.allowLocalKeyManagerInProduction",
	)
	ErrProductionLocalEncryptionKeyRequired = errors.New(
		"security.encryption.allowLocalKeyManagerInProduction requires security.encryption.key of at least 32 characters",
	)
	ErrProductionRecoveryCodePepperRequired = errors.New(
		"security.mfa.recoveryCodePepper must be set to a random value of at least 32 characters in production",
	)
	ErrProductionGCPKMSConfigRequired = errors.New(
		"production and staging require a GCP KMS crypto key resource",
	)
	ErrProductionSessionCookieRequired = errors.New(
		"production and staging require secure, httpOnly, sameSite=strict session cookies",
	)
)

const (
	EncryptionModeEnvelope = "envelope"
	EncryptionModeDisabled = "disabled"

	EncryptionKeyManagerLocal      = "local"
	EncryptionKeyManagerGCPAutokey = "gcp-autokey"
	EncryptionKeyManagerDisabled   = "disabled"
)

type EncryptionConfig struct {
	Mode                             string       `mapstructure:"mode"                             validate:"omitempty,oneof=envelope disabled"`
	KeyManager                       string       `mapstructure:"keyManager"                       validate:"omitempty,oneof=local gcp-autokey disabled"`
	Key                              string       `mapstructure:"key"                              validate:"omitempty,min=32"`
	RetiredKeys                      []string     `mapstructure:"retiredKeys"                      validate:"omitempty,dive,required"`
	AllowLocalKeyManagerInProduction bool         `mapstructure:"allowLocalKeyManagerInProduction"`
	GCPKMS                           GCPKMSConfig `mapstructure:"gcpKms"`
}

type GCPKMSConfig struct {
	CryptoKey       string        `mapstructure:"cryptoKey"`
	KeyResource     string        `mapstructure:"keyResource"`
	CredentialsMode string        `mapstructure:"credentialsMode" validate:"omitempty,oneof=adc workload-identity credentials-file"`
	CredentialsFile string        `mapstructure:"credentialsFile"`
	Timeout         time.Duration `mapstructure:"timeout"`
	RetryAttempts   int           `mapstructure:"retryAttempts"   validate:"omitempty,min=1,max=10"`
}

func (c *EncryptionConfig) validateProductionKeyManager() error {
	keyManager := strings.ToLower(strings.TrimSpace(c.KeyManager))
	if keyManager == EncryptionKeyManagerLocal && c.AllowLocalKeyManagerInProduction {
		return validateProductionLocalKey(c.Key)
	}
	if keyManager != EncryptionKeyManagerGCPAutokey {
		return ErrProductionKMSRequired
	}

	gcpKey := strings.TrimSpace(c.GCPKMS.CryptoKey)
	if gcpKey == "" {
		gcpKey = strings.TrimSpace(c.GCPKMS.KeyResource)
	}
	if gcpKey == "" {
		return ErrProductionGCPKMSConfigRequired
	}

	return nil
}

const minProductionLocalEncryptionKeyLength = 32

func validateProductionRecoveryCodePepper(pepper string) error {
	trimmed := strings.TrimSpace(pepper)
	if len(trimmed) < minProductionLocalEncryptionKeyLength {
		return ErrProductionRecoveryCodePepperRequired
	}
	lowered := strings.ToLower(trimmed)
	for _, placeholder := range InsecureDefaultValues {
		if strings.Contains(lowered, placeholder) {
			return ErrProductionRecoveryCodePepperRequired
		}
	}
	return nil
}

func validateProductionLocalKey(key string) error {
	trimmed := strings.TrimSpace(key)
	if len(trimmed) < minProductionLocalEncryptionKeyLength {
		return ErrProductionLocalEncryptionKeyRequired
	}

	lowered := strings.ToLower(trimmed)
	for _, placeholder := range InsecureDefaultValues {
		if strings.Contains(lowered, placeholder) {
			return ErrEncryptionKeyIsInsecure
		}
	}

	return nil
}
