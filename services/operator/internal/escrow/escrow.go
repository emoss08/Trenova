package escrow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"regexp"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	KeyEncryptionKey         = "encryptionKey"
	KeyRLSScopeKeyID         = "rlsScopeKeyId"
	KeyRLSScopeKey           = "rlsScopeKey"
	KeyMFARecoveryCodePepper = "mfaRecoveryCodePepper"

	minSecretLength  = 32
	maxSecretLength  = 128
	minScopeKeyBytes = 32
)

var (
	ErrInvalid = errors.New("the escrowed secrets are not valid")

	scopeKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
)

var Keys = []string{
	KeyEncryptionKey,
	KeyRLSScopeKeyID,
	KeyRLSScopeKey,
	KeyMFARecoveryCodePepper,
}

type ControlPlane interface {
	Secrets(ctx context.Context) (instanceapi.InstanceSecrets, error)
}

func Validate(v instanceapi.InstanceSecrets) error {
	var errs []error
	if err := text(KeyEncryptionKey, v.EncryptionKey); err != nil {
		errs = append(errs, err)
	}
	if !scopeKeyIDPattern.MatchString(v.RlsScopeKeyId) {
		errs = append(errs, fmt.Errorf("%w: %s must be 1 to 32 of A-Z, a-z, 0-9, _ and -",
			ErrInvalid, KeyRLSScopeKeyID))
	}
	if err := scopeKey(v.RlsScopeKey); err != nil {
		errs = append(errs, err)
	}
	if err := text(KeyMFARecoveryCodePepper, v.MfaRecoveryCodePepper); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

func text(key, v string) error {
	switch {
	case v == "":
		return fmt.Errorf("%w: %s is missing", ErrInvalid, key)
	case len(v) < minSecretLength || len(v) > maxSecretLength:
		return fmt.Errorf("%w: %s must be %d to %d characters",
			ErrInvalid, key, minSecretLength, maxSecretLength)
	}
	return nil
}

func scopeKey(v string) error {
	if v == "" {
		return fmt.Errorf("%w: %s is missing", ErrInvalid, KeyRLSScopeKey)
	}
	raw, err := base64.StdEncoding.DecodeString(v)
	if err != nil || len(raw) < minScopeKeyBytes || len(v) > maxSecretLength {
		return fmt.Errorf("%w: %s must be standard base64 of at least %d bytes",
			ErrInvalid, KeyRLSScopeKey, minScopeKeyBytes)
	}
	return nil
}

func FromData(data map[string][]byte) instanceapi.InstanceSecrets {
	return instanceapi.InstanceSecrets{
		EncryptionKey:         string(data[KeyEncryptionKey]),
		RlsScopeKeyId:         string(data[KeyRLSScopeKeyID]),
		RlsScopeKey:           string(data[KeyRLSScopeKey]),
		MfaRecoveryCodePepper: string(data[KeyMFARecoveryCodePepper]),
	}
}

func ToData(existing map[string][]byte, v instanceapi.InstanceSecrets) map[string][]byte {
	out := make(map[string][]byte, len(existing)+4)
	maps.Copy(out, existing)
	out[KeyEncryptionKey] = []byte(v.EncryptionKey)
	out[KeyRLSScopeKeyID] = []byte(v.RlsScopeKeyId)
	out[KeyRLSScopeKey] = []byte(v.RlsScopeKey)
	out[KeyMFARecoveryCodePepper] = []byte(v.MfaRecoveryCodePepper)
	return out
}
