package escrow_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/escrow"
	"github.com/emoss08/trenova/operator/internal/fakecp"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

const (
	namespace = "trenova"
	name      = "trenova-escrow"
)

type stubCP struct {
	mu     sync.Mutex
	calls  int
	values instanceapi.InstanceSecrets
	err    error
}

func (s *stubCP) Secrets(context.Context) (instanceapi.InstanceSecrets, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls++
	return s.values, s.err
}

func data(v instanceapi.InstanceSecrets) map[string][]byte {
	return map[string][]byte{
		escrow.KeyEncryptionKey:         []byte(v.EncryptionKey),
		escrow.KeyRLSScopeKeyID:         []byte(v.RlsScopeKeyId),
		escrow.KeyRLSScopeKey:           []byte(v.RlsScopeKey),
		escrow.KeyMFARecoveryCodePepper: []byte(v.MfaRecoveryCodePepper),
	}
}

func other() instanceapi.InstanceSecrets {
	return instanceapi.InstanceSecrets{
		EncryptionKey:         strings.Repeat("e", 40),
		RlsScopeKeyId:         "k1",
		RlsScopeKey:           "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		MfaRecoveryCodePepper: strings.Repeat("p", 40),
	}
}

func secret(d map[string][]byte) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
		Data:       d,
	}
}

func TestEnsure(t *testing.T) {
	t.Parallel()
	incomplete := data(other())
	delete(incomplete, escrow.KeyRLSScopeKey)
	incomplete["unrelated"] = []byte("kept")
	invalid := data(other())
	invalid[escrow.KeyEncryptionKey] = []byte("short")
	short := fakecp.Secrets()
	short.EncryptionKey = "too-short-to-encrypt"

	cases := []struct {
		name     string
		existing *corev1.Secret
		values   instanceapi.InstanceSecrets
		cpErr    error
		calls    int
		want     instanceapi.InstanceSecrets
		check    func(t *testing.T, err error)
		kept     bool
		noSecret bool
	}{
		{
			name:  "missing: fetched and created",
			calls: 1,
			want:  fakecp.Secrets(),
		},
		{
			name:     "present and valid: not fetched",
			existing: secret(data(other())),
			calls:    0,
			want:     other(),
		},
		{
			name:     "incomplete: fetched again and restored",
			existing: secret(incomplete),
			calls:    1,
			want:     fakecp.Secrets(),
			kept:     true,
		},
		{
			name:     "present but invalid: fetched again and restored",
			existing: secret(invalid),
			calls:    1,
			want:     fakecp.Secrets(),
		},
		{
			name:   "the control plane's values are invalid: refused",
			values: short,
			calls:  1,
			check: func(t *testing.T, err error) {
				require.ErrorIs(t, err, escrow.ErrInvalid)
				require.NotContains(t, err.Error(), short.EncryptionKey)
			},
			noSecret: true,
		},
		{
			name:  "rate limited",
			cpErr: &controlplane.RetryLaterError{Reason: "rate limited", RetryAfter: time.Second},
			calls: 1,
			check: func(t *testing.T, err error) {
				var later *controlplane.RetryLaterError
				require.ErrorAs(t, err, &later)
			},
			noSecret: true,
		},
		{
			name:  "certificate refused",
			cpErr: controlplane.ErrUnauthenticated,
			calls: 1,
			check: func(t *testing.T, err error) {
				require.ErrorIs(t, err, controlplane.ErrUnauthenticated)
			},
			noSecret: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			kube := fake.NewClientset()
			if tc.existing != nil {
				_, err := kube.CoreV1().Secrets(namespace).
					Create(t.Context(), tc.existing.DeepCopy(), metav1.CreateOptions{})
				require.NoError(t, err)
			}
			values := tc.values
			if values == (instanceapi.InstanceSecrets{}) {
				values = fakecp.Secrets()
			}
			cp := &stubCP{values: values, err: tc.cpErr}
			store := escrow.New(&escrow.Params{
				Client:       kube,
				ControlPlane: cp,
				Namespace:    namespace,
				Name:         name,
				Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			err := store.Ensure(t.Context())
			require.Equal(t, tc.calls, cp.calls)
			sec, getErr := kube.CoreV1().
				Secrets(namespace).
				Get(t.Context(), name, metav1.GetOptions{})
			if tc.check != nil {
				tc.check(t, err)
				if tc.noSecret {
					require.True(t, apierrors.IsNotFound(getErr))
				}
				return
			}
			require.NoError(t, err)
			require.NoError(t, getErr)
			require.Equal(t, data(tc.want), withoutUnrelated(sec.Data))
			if tc.kept {
				require.Equal(t, "kept", string(sec.Data["unrelated"]))
			}
			if tc.calls > 0 {
				require.Equal(t, "trenova-operator", sec.Labels["app.kubernetes.io/managed-by"])
				require.Equal(t, "escrow", sec.Labels["app.kubernetes.io/component"])
				require.NotEmpty(t, sec.Annotations["trenova.app/escrow"])
				require.Empty(t, sec.Annotations["meta.helm.sh/release-name"])
			}

			require.NoError(t, store.Ensure(t.Context()))
			require.Equal(t, tc.calls, cp.calls, "a complete Secret is never fetched again")
		})
	}
}

func withoutUnrelated(d map[string][]byte) map[string][]byte {
	out := map[string][]byte{}
	for k, v := range d {
		if k != "unrelated" {
			out[k] = v
		}
	}
	return out
}

func TestEnsureReportsAKubernetesFailure(t *testing.T) {
	t.Parallel()
	kube := fake.NewClientset()
	cp := &stubCP{values: fakecp.Secrets()}
	store := escrow.New(&escrow.Params{
		Client:       kube,
		ControlPlane: cp,
		Namespace:    "missing-namespace",
		Name:         name,
		Logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	kube.PrependReactor(
		"create",
		"secrets",
		func(k8stesting.Action) (bool, runtime.Object, error) {
			return true, nil, errors.New("namespace missing-namespace not found")
		},
	)
	err := store.Ensure(t.Context())
	require.ErrorContains(t, err, "create the escrow Secret")
	require.NotContains(t, err.Error(), fakecp.Secrets().EncryptionKey)
}

func TestValidate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(*instanceapi.InstanceSecrets)
		field  string
	}{
		{name: "valid", mutate: func(*instanceapi.InstanceSecrets) {}},
		{
			name:   "encryption key missing",
			mutate: func(v *instanceapi.InstanceSecrets) { v.EncryptionKey = "" },
			field:  escrow.KeyEncryptionKey,
		},
		{
			name:   "encryption key short",
			mutate: func(v *instanceapi.InstanceSecrets) { v.EncryptionKey = strings.Repeat("a", 31) },
			field:  escrow.KeyEncryptionKey,
		},
		{
			name:   "encryption key long",
			mutate: func(v *instanceapi.InstanceSecrets) { v.EncryptionKey = strings.Repeat("a", 129) },
			field:  escrow.KeyEncryptionKey,
		},
		{
			name:   "scope key ID missing",
			mutate: func(v *instanceapi.InstanceSecrets) { v.RlsScopeKeyId = "" },
			field:  escrow.KeyRLSScopeKeyID,
		},
		{
			name:   "scope key ID with a dot",
			mutate: func(v *instanceapi.InstanceSecrets) { v.RlsScopeKeyId = "k.1" },
			field:  escrow.KeyRLSScopeKeyID,
		},
		{
			name:   "scope key ID too long",
			mutate: func(v *instanceapi.InstanceSecrets) { v.RlsScopeKeyId = strings.Repeat("k", 33) },
			field:  escrow.KeyRLSScopeKeyID,
		},
		{
			name:   "scope key not base64",
			mutate: func(v *instanceapi.InstanceSecrets) { v.RlsScopeKey = strings.Repeat("!", 44) },
			field:  escrow.KeyRLSScopeKey,
		},
		{
			name: "scope key URL base64",
			mutate: func(v *instanceapi.InstanceSecrets) {
				v.RlsScopeKey = "____________________________________________"
			},
			field: escrow.KeyRLSScopeKey,
		},
		{
			name:   "scope key of 31 bytes",
			mutate: func(v *instanceapi.InstanceSecrets) { v.RlsScopeKey = strings.Repeat("A", 40) + "AA==" },
			field:  escrow.KeyRLSScopeKey,
		},
		{
			name:   "pepper short",
			mutate: func(v *instanceapi.InstanceSecrets) { v.MfaRecoveryCodePepper = "pepper" },
			field:  escrow.KeyMFARecoveryCodePepper,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			v := fakecp.Secrets()
			tc.mutate(&v)
			err := escrow.Validate(v)
			if tc.field == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, escrow.ErrInvalid)
			require.ErrorContains(t, err, tc.field)
		})
	}
}
