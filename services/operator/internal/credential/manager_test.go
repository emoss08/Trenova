package credential_test

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
)

const namespace = "trenova-system"

type harness struct {
	cp        *controlplanetest.Server
	kube      kubernetes.Interface
	store     credential.Store
	transport *credential.Transport
	client    *controlplane.Client
	now       time.Time
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	cp := controlplanetest.New(t, controlplanetest.Spec(uuid.New()))
	kube := fake.NewClientset()
	transport := &credential.Transport{}
	client, err := controlplane.New(cp.URL, &http.Client{Transport: transport, Timeout: 10 * time.Second})
	require.NoError(t, err)
	return &harness{
		cp:        cp,
		kube:      kube,
		store:     credential.NewStore(credential.StoreParams{Client: kube, Namespace: namespace, Enrolment: "enrolment", Identity: "identity"}),
		transport: transport,
		client:    client,
		now:       time.Now(),
	}
}

func (h *harness) installBundle(t *testing.T, b credential.Bundle) {
	t.Helper()
	body, err := sonic.Marshal(b)
	require.NoError(t, err)
	secrets := h.kube.CoreV1().Secrets(namespace)
	sec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "enrolment", Namespace: namespace},
		Data:       map[string][]byte{credential.BundleKey: body},
	}
	if _, err = secrets.Get(t.Context(), "enrolment", metav1.GetOptions{}); err == nil {
		_, err = secrets.Update(t.Context(), sec, metav1.UpdateOptions{})
	} else {
		_, err = secrets.Create(t.Context(), sec, metav1.CreateOptions{})
	}
	require.NoError(t, err)
}

func (h *harness) manager() *credential.Manager {
	return credential.NewManager(&credential.ManagerParams{
		Store:     h.store,
		Transport: h.transport,
		Renewer:   h.client,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:       func() time.Time { return h.now },
	})
}

func (h *harness) works(ctx context.Context) error {
	_, _, err := h.client.Spec(ctx, "")
	return err
}

func TestEnrolsOntoItsOwnKeyAndRetiresTheBundle(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	bundle := h.cp.Bundle(7 * 24 * time.Hour)
	h.installBundle(t, bundle)

	m := h.manager()
	require.NoError(t, m.Start(t.Context()))
	require.Equal(t, 1, h.cp.Renewals())
	cur := m.Current()
	require.NotEqual(t, bundle.Certificate, string(cur.CertPEM))
	require.NoError(t, h.works(t.Context()))

	stored, ok, err := h.store.Load(t.Context())
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, cur.Fingerprint(), stored.Fingerprint())

	first, err := bundle.Identity()
	require.NoError(t, err)
	require.Equal(t, first.Fingerprint(), stored.EnrolledFrom)
	require.NoError(t, h.transport.Use(&first))
	require.ErrorIs(t, h.works(t.Context()), controlplane.ErrUnauthenticated)
}

func TestRestartKeepsTheStoredIdentity(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	require.NoError(t, h.manager().Start(t.Context()))

	again := h.manager()
	require.NoError(t, again.Start(t.Context()))
	require.Equal(t, 1, h.cp.Renewals())
	require.NoError(t, h.works(t.Context()))
}

func TestRenewsWhenAThirdOfItsLifeIsLeft(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	m := h.manager()
	require.NoError(t, m.Start(t.Context()))
	old := m.Current()

	h.now = old.RenewAt().Add(-time.Minute)
	require.NoError(t, m.RenewIfDue(t.Context()))
	require.Equal(t, 1, h.cp.Renewals())

	h.now = old.RenewAt().Add(time.Minute)
	require.NoError(t, m.RenewIfDue(t.Context()))
	require.Equal(t, 2, h.cp.Renewals())
	renewed := m.Current()
	require.NotEqual(t, old.Fingerprint(), renewed.Fingerprint())
	require.Equal(t, old.EnrolledFrom, renewed.EnrolledFrom)
	require.NoError(t, h.works(t.Context()))

	require.NoError(t, h.transport.Use(&old))
	require.ErrorIs(t, h.works(t.Context()), controlplane.ErrUnauthenticated)
}

func TestTakesANewBundleAfterStaffReenrol(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	m := h.manager()
	require.NoError(t, m.Start(t.Context()))

	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	require.ErrorIs(t, h.works(t.Context()), controlplane.ErrUnauthenticated)
	require.NoError(t, m.Recover(t.Context()))
	require.NoError(t, h.works(t.Context()))

	require.ErrorIs(t, m.Recover(t.Context()), credential.ErrReenrolmentNeeded)
}

func TestStartPrefersANewerBundleOverTheStoredIdentity(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	require.NoError(t, h.manager().Start(t.Context()))

	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	m := h.manager()
	require.NoError(t, m.Start(t.Context()))
	require.Equal(t, 2, h.cp.Renewals())
	require.NoError(t, h.works(t.Context()))
}

func TestRefusesToStartWithoutCredentials(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	require.ErrorIs(t, h.manager().Start(t.Context()), credential.ErrNotEnrolled)

	h.installBundle(t, h.cp.Bundle(time.Hour))
	h.now = time.Now().Add(2 * time.Hour)
	require.ErrorIs(t, h.manager().Start(t.Context()), credential.ErrBundleExpired)
}

func TestRevokedWithoutANewBundleNeedsReenrolment(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.installBundle(t, h.cp.Bundle(7*24*time.Hour))
	m := h.manager()
	require.NoError(t, m.Start(t.Context()))

	h.cp.RevokeAll()
	require.ErrorIs(t, h.works(t.Context()), controlplane.ErrUnauthenticated)
	require.ErrorIs(t, m.Recover(t.Context()), credential.ErrReenrolmentNeeded)
}
