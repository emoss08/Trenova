package telemetry_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/emoss08/trenova/operator/internal/telemetry"
)

type authority struct {
	key  *ecdsa.PrivateKey
	cert *x509.Certificate
	pem  []byte
}

func newAuthority(t *testing.T) authority {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "k3s-server-ca"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	require.NoError(t, err)
	cert, err := x509.ParseCertificate(der)
	require.NoError(t, err)
	return authority{key: key, cert: cert, pem: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})}
}

func (a authority) serving(t *testing.T) tls.Certificate {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "node-1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}, a.cert, &key.PublicKey, a.key)
	require.NoError(t, err)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
}

type kubelet struct {
	addr   string
	mu     sync.Mutex
	tokens []string
	status int
	body   []byte
}

const summaryJSON = `{"node":{"nodeName":"node-1","fs":{"capacityBytes":107374182400,"usedBytes":10737418240},` +
	`"runtime":{"imageFs":{"capacityBytes":536870912000,"availableBytes":493921239040,"usedBytes":6442450944}}},` +
	`"pods":[{"podRef":{"name":"trenova-loki-0","namespace":"trenova"},` +
	`"volume":[{"name":"storage","pvcRef":{"name":"trenova-loki","namespace":"trenova"},"usedBytes":2147483648}]}]}`

func newKubelet(t *testing.T, ca authority) *kubelet {
	t.Helper()
	k := &kubelet{status: http.StatusOK, body: []byte(summaryJSON)}
	srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		k.mu.Lock()
		k.tokens = append(k.tokens, r.Header.Get("Authorization"))
		status, body := k.status, k.body
		k.mu.Unlock()
		if r.URL.Path != "/stats/summary" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	srv.TLS = &tls.Config{MinVersion: tls.VersionTLS12, Certificates: []tls.Certificate{ca.serving(t)}}
	srv.StartTLS()
	t.Cleanup(srv.Close)
	k.addr = srv.Listener.Addr().String()
	return k
}

func (k *kubelet) seen() []string {
	k.mu.Lock()
	defer k.mu.Unlock()
	return append([]string(nil), k.tokens...)
}

func tokenFile(t *testing.T, token string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "token")
	require.NoError(t, os.WriteFile(path, []byte(token+"\n"), 0o600))
	return path
}

func summaries(t *testing.T, ca []byte, token string) *telemetry.KubeletSummaries {
	t.Helper()
	k, err := telemetry.NewKubeletSummaries(telemetry.KubeletParams{CA: ca, TokenFile: token})
	require.NoError(t, err)
	return k
}

func TestTheKubeletIsReadVerifiedWithTheClustersCA(t *testing.T) {
	t.Parallel()
	ca := newAuthority(t)
	k := newKubelet(t, ca)

	got, err := summaries(t, ca.pem, tokenFile(t, "sa-token-1")).Summary(t.Context(), k.addr)

	require.NoError(t, err)
	require.Equal(t, uint64(10<<30), *got.Node.FS.UsedBytes)
	require.Equal(t, uint64(100<<30), *got.Node.FS.CapacityBytes)
	require.Equal(t, uint64(500<<30), *got.Node.DataFS().CapacityBytes)
	require.Equal(t, uint64(40<<30), *got.Node.DataFS().Taken(),
		"the image filesystem's own usedBytes counts only the images")
	require.Equal(t, []string{"Bearer sa-token-1"}, k.seen())
}

func TestTheTokenIsReadAgainOnEveryCall(t *testing.T) {
	t.Parallel()
	ca := newAuthority(t)
	k := newKubelet(t, ca)
	path := tokenFile(t, "sa-token-1")
	reader := summaries(t, ca.pem, path)

	_, err := reader.Summary(t.Context(), k.addr)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, []byte("sa-token-2"), 0o600))
	_, err = reader.Summary(t.Context(), k.addr)
	require.NoError(t, err)

	require.Equal(t, []string{"Bearer sa-token-1", "Bearer sa-token-2"}, k.seen())
}

func TestAKubeletSignedByAnotherCAIsNotRead(t *testing.T) {
	t.Parallel()
	k := newKubelet(t, newAuthority(t))

	_, err := summaries(t, newAuthority(t).pem, tokenFile(t, "sa-token-1")).Summary(t.Context(), k.addr)

	require.Error(t, err)
	var unknown x509.UnknownAuthorityError
	require.ErrorAs(t, err, &unknown)
	require.Empty(t, k.seen(), "nothing, the token least of all, reaches a kubelet that does not verify")
}

func TestARefusedOrOversizedSummaryIsAnError(t *testing.T) {
	t.Parallel()
	ca := newAuthority(t)
	k := newKubelet(t, ca)
	reader := summaries(t, ca.pem, tokenFile(t, "sa-token-1"))

	k.mu.Lock()
	k.status = http.StatusForbidden
	k.mu.Unlock()
	_, err := reader.Summary(t.Context(), k.addr)
	require.ErrorContains(t, err, "answered 403")

	k.mu.Lock()
	k.status, k.body = http.StatusOK, make([]byte, 9<<20)
	k.mu.Unlock()
	_, err = reader.Summary(t.Context(), k.addr)
	require.ErrorContains(t, err, "larger than")
}

func TestTheSamplerLeavesDiskOutWhenTheKubeletCannotBeRead(t *testing.T) {
	t.Parallel()
	cases := map[string]func(*testing.T, authority, *kubelet) []byte{
		"another CA": func(t *testing.T, _ authority, _ *kubelet) []byte { return newAuthority(t).pem },
		"forbidden": func(_ *testing.T, ca authority, k *kubelet) []byte {
			k.mu.Lock()
			k.status = http.StatusForbidden
			k.mu.Unlock()
			return ca.pem
		},
		"read": func(_ *testing.T, ca authority, _ *kubelet) []byte { return ca.pem },
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ca := newAuthority(t)
			k := newKubelet(t, ca)
			trust := setup(t, ca, k)
			host, port, err := net.SplitHostPort(k.addr)
			require.NoError(t, err)
			p, err := strconv.Atoi(port)
			require.NoError(t, err)
			n := machineNode()
			n.Status.Addresses = []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: host}}
			n.Status.DaemonEndpoints.KubeletEndpoint.Port = int32(p)
			objs := standardObjects()
			objs[len(objs)-1] = n
			w := newWorld(t, "", objs...)
			sampler := telemetry.NewSampler(&telemetry.SamplerParams{
				Kube:      w.kube,
				Metrics:   w.metrics,
				Parts:     w.checker,
				Summaries: summaries(t, trust, tokenFile(t, "sa-token-1")),
				Data:      w.data,
				Namespace: ns,
				Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
			})

			sample := sampler.Sample(t.Context(), minute, params())

			require.Equal(t, int64(4000), *sample.Machine.CpuCapacityMillicores)
			if name == "read" {
				require.Equal(t, 40*gib, *sample.Machine.DiskBytes)
				require.Equal(t, 500*gib, *sample.Machine.DiskCapacityBytes)
				require.Equal(t, 2*gib, *sample.Data.LogsBytes)
				return
			}
			require.Nil(t, sample.Machine.DiskBytes)
			require.Nil(t, sample.Machine.DiskCapacityBytes)
			require.Nil(t, sample.Data.LogsBytes)
		})
	}
}

func TestTheKubeletReaderRefusesAnEmptyCABundle(t *testing.T) {
	t.Parallel()
	_, err := telemetry.NewKubeletSummaries(telemetry.KubeletParams{CA: []byte("nothing"), TokenFile: "/token"})
	require.Error(t, err)
}
