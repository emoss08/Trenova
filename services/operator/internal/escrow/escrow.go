package escrow

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"maps"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

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

	labelManagedBy = "app.kubernetes.io/managed-by"
	labelPartOf    = "app.kubernetes.io/part-of"
	labelComponent = "app.kubernetes.io/component"
	annotationNote = "trenova.app/escrow"
	managedBy      = "trenova-operator"
	note           = "the instance's keys held in escrow by the Trenova Cloud control plane; " +
		"restored from it when missing, never a Helm resource"
)

var (
	ErrInvalid = errors.New("the escrowed secrets are not valid")

	scopeKeyIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,32}$`)
)

type ControlPlane interface {
	Secrets(ctx context.Context) (instanceapi.InstanceSecrets, error)
}

type Store struct {
	client    kubernetes.Interface
	cp        ControlPlane
	namespace string
	name      string
	logger    *slog.Logger
}

type Params struct {
	Client       kubernetes.Interface
	ControlPlane ControlPlane
	Namespace    string
	Name         string
	Logger       *slog.Logger
}

func New(p *Params) *Store {
	return &Store{
		client:    p.Client,
		cp:        p.ControlPlane,
		namespace: p.Namespace,
		name:      p.Name,
		logger:    p.Logger,
	}
}

func (s *Store) Ensure(ctx context.Context) error {
	secrets := s.client.CoreV1().Secrets(s.namespace)
	sec, err := secrets.Get(ctx, s.name, metav1.GetOptions{})
	missing := apierrors.IsNotFound(err)
	if err != nil && !missing {
		return fmt.Errorf("read the escrow Secret %s/%s: %w", s.namespace, s.name, err)
	}
	if !missing {
		problem := Validate(fromData(sec.Data))
		if problem == nil {
			return nil
		}
		s.logger.Warn(
			"the escrow Secret is incomplete or invalid; restoring it from the control plane",
			"secret", s.namespace+"/"+s.name,
			"reason", problem.Error(),
		)
	}
	fetched, err := s.cp.Secrets(ctx)
	if err != nil {
		return fmt.Errorf("fetch the escrowed secrets: %w", err)
	}
	if err = Validate(fetched); err != nil {
		return fmt.Errorf("the control plane's escrowed secrets: %w", err)
	}
	if missing {
		_, err = secrets.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        s.name,
				Namespace:   s.namespace,
				Labels:      labels(nil),
				Annotations: annotations(nil),
			},
			Type: corev1.SecretTypeOpaque,
			Data: toData(nil, fetched),
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create the escrow Secret %s/%s: %w", s.namespace, s.name, err)
		}
		s.logger.Info("escrow Secret created", "secret", s.namespace+"/"+s.name)
		return nil
	}
	next := sec.DeepCopy()
	next.Labels = labels(next.Labels)
	next.Annotations = annotations(next.Annotations)
	next.Data = toData(next.Data, fetched)
	if _, err = secrets.Update(ctx, next, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update the escrow Secret %s/%s: %w", s.namespace, s.name, err)
	}
	s.logger.Info("escrow Secret restored", "secret", s.namespace+"/"+s.name)
	return nil
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

func fromData(data map[string][]byte) instanceapi.InstanceSecrets {
	return instanceapi.InstanceSecrets{
		EncryptionKey:         string(data[KeyEncryptionKey]),
		RlsScopeKeyId:         string(data[KeyRLSScopeKeyID]),
		RlsScopeKey:           string(data[KeyRLSScopeKey]),
		MfaRecoveryCodePepper: string(data[KeyMFARecoveryCodePepper]),
	}
}

func toData(existing map[string][]byte, v instanceapi.InstanceSecrets) map[string][]byte {
	out := make(map[string][]byte, len(existing)+4)
	maps.Copy(out, existing)
	out[KeyEncryptionKey] = []byte(v.EncryptionKey)
	out[KeyRLSScopeKeyID] = []byte(v.RlsScopeKeyId)
	out[KeyRLSScopeKey] = []byte(v.RlsScopeKey)
	out[KeyMFARecoveryCodePepper] = []byte(v.MfaRecoveryCodePepper)
	return out
}

func labels(existing map[string]string) map[string]string {
	out := maps.Clone(existing)
	if out == nil {
		out = map[string]string{}
	}
	out[labelManagedBy] = managedBy
	out[labelPartOf] = "trenova"
	out[labelComponent] = "escrow"
	return out
}

func annotations(existing map[string]string) map[string]string {
	out := maps.Clone(existing)
	if out == nil {
		out = map[string]string{}
	}
	out[annotationNote] = note
	return out
}
