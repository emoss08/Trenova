package credential

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	BundleKey       = "enrolment.json"
	keyCertificate  = "tls.crt"
	keyPrivateKey   = "tls.key"
	keyCA           = "ca.crt"
	keyAPIURL       = "apiUrl"
	keyInstanceID   = "instanceId"
	keyEnrolledFrom = "enrolledFrom"
)

type Store struct {
	client    kubernetes.Interface
	namespace string
	enrolment string
	identity  string
}

type StoreParams struct {
	Client    kubernetes.Interface
	Namespace string
	Enrolment string
	Identity  string
}

func NewStore(p StoreParams) Store {
	return Store{
		client:    p.Client,
		namespace: p.Namespace,
		enrolment: p.Enrolment,
		identity:  p.Identity,
	}
}

func (s Store) Bundle(ctx context.Context) (Bundle, bool, error) {
	sec, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, s.enrolment, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return Bundle{}, false, nil
	}
	if err != nil {
		return Bundle{}, false, fmt.Errorf("read the enrolment Secret %s: %w", s.enrolment, err)
	}
	body, ok := sec.Data[BundleKey]
	if !ok {
		return Bundle{}, false, fmt.Errorf(
			"%w: Secret %s has no %s",
			ErrBundleInvalid,
			s.enrolment,
			BundleKey,
		)
	}
	b, err := ParseBundle(body)
	if err != nil {
		return Bundle{}, false, err
	}
	return b, true, nil
}

func (s Store) Load(ctx context.Context) (Identity, bool, error) {
	sec, err := s.client.CoreV1().Secrets(s.namespace).Get(ctx, s.identity, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return Identity{}, false, nil
	}
	if err != nil {
		return Identity{}, false, fmt.Errorf("read the identity Secret %s: %w", s.identity, err)
	}
	id, err := uuid.Parse(string(sec.Data[keyInstanceID]))
	if err != nil {
		return Identity{}, false, fmt.Errorf("%w: instanceId: %w", ErrIdentityInvalid, err)
	}
	ident, err := NewIdentity(&IdentityParams{
		InstanceID:   id,
		APIURL:       string(sec.Data[keyAPIURL]),
		CertPEM:      sec.Data[keyCertificate],
		KeyPEM:       sec.Data[keyPrivateKey],
		CAPEM:        sec.Data[keyCA],
		EnrolledFrom: string(sec.Data[keyEnrolledFrom]),
	})
	if err != nil {
		return Identity{}, false, fmt.Errorf("%w: %w", ErrIdentityInvalid, err)
	}
	return ident, true, nil
}

func (s Store) Save(ctx context.Context, ident *Identity) error {
	data := map[string][]byte{
		keyInstanceID:   []byte(ident.InstanceID.String()),
		keyAPIURL:       []byte(ident.APIURL),
		keyCertificate:  ident.CertPEM,
		keyPrivateKey:   ident.KeyPEM,
		keyCA:           ident.CAPEM,
		keyEnrolledFrom: []byte(ident.EnrolledFrom),
	}
	secrets := s.client.CoreV1().Secrets(s.namespace)
	sec, err := secrets.Get(ctx, s.identity, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = secrets.Create(ctx, &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.identity,
				Namespace: s.namespace,
				Labels:    map[string]string{"app.kubernetes.io/managed-by": "trenova-operator"},
			},
			Type: corev1.SecretTypeOpaque,
			Data: data,
		}, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("create the identity Secret %s: %w", s.identity, err)
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("read the identity Secret %s: %w", s.identity, err)
	}
	sec.Data = data
	if _, err = secrets.Update(ctx, sec, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update the identity Secret %s: %w", s.identity, err)
	}
	return nil
}
