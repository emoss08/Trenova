package logs

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplane"
	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/credential"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

type staticToken string

func (s staticToken) Token(context.Context) (string, error)   { return string(s), nil }
func (s staticToken) Refresh(context.Context) (string, error) { return string(s), nil }

type relayWorld struct {
	cp      *controlplanetest.Server
	client  *controlplane.Client
	tms     *tmstest.Server
	tmsAPI  *tms.Client
	src     *fakeSource
	objects *fakeObjects
	dir     string
}

func newRelayWorld(t *testing.T) *relayWorld {
	t.Helper()
	cp := controlplanetest.New(t, controlplanetest.Spec(uuid.New()))
	bundle := cp.Bundle(time.Hour)
	ident, err := bundle.Identity()
	require.NoError(t, err)
	tr := &credential.Transport{}
	require.NoError(t, tr.Use(&ident))
	client, err := controlplane.New(cp.URL, &http.Client{Transport: tr, Timeout: 10 * time.Second})
	require.NoError(t, err)
	server := tmstest.New(t, "managed-token")
	api, err := tms.New(tms.Params{URL: server.URL, Tokens: staticToken("managed-token")})
	require.NoError(t, err)
	w := &relayWorld{
		cp: cp, client: client, tms: server, tmsAPI: api, src: newSource(), objects: newObjects(),
		dir: t.TempDir(),
	}
	fill(w.src, 10)
	return w
}

func (w *relayWorld) service(t *testing.T) *Service {
	t.Helper()
	svc := newService(t, w.src, w.objects, w.client, w.tmsAPI, DefaultCaps, w.dir)
	(&world{svc: svc}).start(t)
	return svc
}

func dayRequest(id string) tms.LogRequest {
	return tms.LogRequest{
		ID: id, Since: t0, Until: t0.Add(24 * time.Hour),
		RequestedBy: tms.LogRequester{Email: "dana@acme.example", Name: "Dana Whitfield"},
		RequestedAt: t0.Add(25 * time.Hour),
	}
}

func TestARequestMadeInTrenovaIsRegisteredCollectedStoredAndSent(t *testing.T) {
	t.Parallel()
	w := newRelayWorld(t)
	svc := w.service(t)
	w.tms.AddLogRequest(dayRequest("lrq_01"))

	require.NoError(t, svc.Relay(t.Context()))
	require.Eventually(t, func() bool {
		return w.tms.LogRequestStatus("lrq_01") == tms.LogRequestSent
	}, 30*time.Second, 20*time.Millisecond)
	require.NoError(t, svc.Relay(t.Context()))

	updates := w.tms.LogUpdates()
	require.Len(t, updates, 2)
	assert.Equal(t, tms.LogRequestCollecting, updates[0].Update.Status)
	bundle, err := uuid.Parse(updates[1].Update.BundleID)
	require.NoError(t, err)
	assert.Equal(t, updates[0].Update.BundleID, updates[1].Update.BundleID)
	assert.Equal(t, 1, w.cp.LogBundleRegistrations())
	b, ok := w.cp.LogBundle(bundle)
	require.True(t, ok)
	require.NotNil(t, b.Registration)
	assert.Equal(t, "lrq_01", b.Registration.RequestId)
	assert.Equal(t, "Dana Whitfield", *b.Registration.RequestedBy.Name)
	require.Len(t, b.Updates, 2)
	assert.Equal(t, instanceapi.LogBundleCollecting, b.Updates[0].Status)
	assert.Equal(t, instanceapi.LogBundleStored, b.Updates[1].Status)
	_, stored := w.objects.get("support/logs/" + bundle.String() + ".zip")
	assert.True(t, stored)
	assert.Empty(t, scratchFiles(t, w.dir))
}

func TestARegisteredRequestIsNotRegisteredOrUploadedAgainAfterARestart(t *testing.T) {
	t.Parallel()
	w := newRelayWorld(t)
	first := w.service(t)
	w.tms.AddLogRequest(dayRequest("lrq_02"))
	require.NoError(t, first.Relay(t.Context()))
	require.Eventually(t, func() bool {
		return w.tms.LogRequestStatus("lrq_02") == tms.LogRequestSent
	}, 30*time.Second, 20*time.Millisecond)
	w.tms.LoseLogRequestStatus("lrq_02")

	restarted := w.service(t)
	require.Eventually(t, func() bool {
		_ = restarted.Relay(t.Context())
		return w.tms.LogRequestStatus("lrq_02") == tms.LogRequestSent
	}, 30*time.Second, 50*time.Millisecond)

	assert.Equal(t, 1, w.objects.putCount(), "the stored bundle was found, not made again")
	ids := map[string]bool{}
	for _, u := range w.tms.LogUpdates() {
		if u.Update.BundleID != "" {
			ids[u.Update.BundleID] = true
		}
	}
	assert.Len(t, ids, 1, "the control plane answered the same bundle for the same request")
}

func TestARequestForTooLongAPeriodFailsWithoutRegistering(t *testing.T) {
	t.Parallel()
	w := newRelayWorld(t)
	svc := w.service(t)
	req := dayRequest("lrq_03")
	req.Until = req.Since.Add(8 * 24 * time.Hour)
	w.tms.AddLogRequest(req)

	require.NoError(t, svc.Relay(t.Context()))

	assert.Equal(t, tms.LogRequestFailed, w.tms.LogRequestStatus("lrq_03"))
	assert.Contains(t, w.tms.LogUpdates()[0].Update.Message, "7 days")
	assert.Zero(t, w.cp.LogBundleRegistrations())
}

func TestARequestTheControlPlaneCannotTakeFailsAndIsNotRetried(t *testing.T) {
	t.Parallel()
	w := newRelayWorld(t)
	svc := w.service(t)
	w.tms.AddLogRequest(dayRequest("lrq_05"))
	w.tms.AddLogRequest(dayRequest("lrq_06"))
	w.cp.RefuseLogBundlesNext(2, "")

	require.NoError(t, svc.Relay(t.Context()))
	require.NoError(t, svc.Relay(t.Context()))

	assert.Equal(t, tms.LogRequestFailed, w.tms.LogRequestStatus("lrq_05"))
	assert.Equal(t, tms.LogRequestFailed, w.tms.LogRequestStatus("lrq_06"))
	updates := w.tms.LogUpdates()
	require.Len(t, updates, 2, "each is told once and never registered again")
	assert.Equal(t, conflictMessage, updates[0].Update.Message)
	assert.Equal(t, 2, w.cp.LogBundleRegistrations())
	assert.Zero(t, w.src.calls, "nothing is collected")
	assert.Zero(t, w.objects.putCount())
}

func TestARequestFailsWithTheControlPlanesSentence(t *testing.T) {
	t.Parallel()
	w := newRelayWorld(t)
	svc := w.service(t)
	const reason = "This instance has used its 10 log bundles for today. Try again tomorrow."
	w.cp.RefuseLogBundlesNext(1, reason)
	w.tms.AddLogRequest(dayRequest("lrq_07"))

	require.NoError(t, svc.Relay(t.Context()))

	updates := w.tms.LogUpdates()
	require.Len(t, updates, 1)
	assert.Equal(t, tms.LogRequestFailed, updates[0].Update.Status)
	assert.Equal(t, reason, updates[0].Update.Message)
}

func TestARequestFailsWhenLogKeepingIsOff(t *testing.T) {
	t.Parallel()
	w := newRelayWorld(t)
	svc := newService(t, nil, w.objects, w.client, w.tmsAPI, DefaultCaps, w.dir)
	w.tms.AddLogRequest(dayRequest("lrq_04"))

	require.NoError(t, svc.Relay(t.Context()))

	assert.Equal(t, tms.LogRequestFailed, w.tms.LogRequestStatus("lrq_04"))
	assert.Equal(t, NotKeptMessage, w.tms.LogUpdates()[0].Update.Message)
	assert.Zero(t, w.cp.LogBundleRegistrations())
}
