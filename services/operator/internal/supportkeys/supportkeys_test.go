package supportkeys_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/instanceapi"
	"github.com/emoss08/trenova/operator/internal/supportkeys"
	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

const (
	token  = "managed-token"
	keyOne = "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"
	keyTwo = "pbcSVoRVgLNjqhYk7oC_UBN35UQTEvLIx4uYJhn8Ebo"
)

func key(kid, x string) instanceapi.SupportKey {
	return instanceapi.SupportKey{Kid: kid, Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig", X: x}
}

func support(keys ...instanceapi.SupportKey) *instanceapi.SpecSupport {
	return &instanceapi.SpecSupport{
		InstanceId:  "0b7d3f4e-8f7a-4a52-9d61-2f3c8e1a5b90",
		StaffDomain: "trenova.app",
		Keys:        keys,
	}
}

type rig struct {
	tms   *tmstest.Server
	relay *supportkeys.Relay
	logs  *bytes.Buffer
}

func newRig(t *testing.T) *rig {
	t.Helper()
	srv := tmstest.New(t, token)
	kube := fake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "trenova-secrets", Namespace: "trenova"},
		Data:       map[string][]byte{tms.TokenKey: []byte(token)},
	})
	client, err := tms.New(tms.Params{
		URL: srv.URL + "/",
		Tokens: tms.NewSecretTokens(tms.SecretTokensParams{
			Client: kube, Namespace: "trenova", Name: "trenova-secrets",
		}),
	})
	require.NoError(t, err)
	logs := &bytes.Buffer{}
	return &rig{
		tms:   srv,
		relay: supportkeys.New(client, slog.New(slog.NewTextHandler(logs, nil))),
		logs:  logs,
	}
}

func TestDeliversTheKeysWhenTheTMSHoldsNone(t *testing.T) {
	t.Parallel()
	r := newRig(t)

	require.NoError(t, r.relay.Sync(t.Context(), support(key("cp-1", keyOne))))

	held := r.tms.SupportKeys()
	require.NotNil(t, held)
	assert.Equal(t, "0b7d3f4e-8f7a-4a52-9d61-2f3c8e1a5b90", held.InstanceID)
	assert.Equal(t, "trenova.app", held.StaffDomain)
	require.Len(t, held.Keys, 1)
	assert.Equal(t, key("cp-1", keyOne).X, held.Keys[0].X)
	assert.Contains(t, r.logs.String(), "cp-1", "the log names the key ids")
	assert.NotContains(t, r.logs.String(), keyOne, "the log never carries a key")
}

func TestDeliversNothingWhenTheKeyIDsMatch(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	desired := support(key("cp-1", keyOne), key("cp-2", keyTwo))
	require.NoError(t, r.relay.Sync(t.Context(), desired))

	reordered := support(key("cp-2", keyTwo), key("cp-1", keyOne))
	require.NoError(t, r.relay.Sync(t.Context(), reordered))

	assert.Len(t, r.tms.SupportKeyPuts(), 1, "the same set of key ids, in any order, is not sent again")
}

func TestDeliversNothingWithoutSupportInTheSpec(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	r.tms.HoldSupportKeys(tms.SupportKeys{
		InstanceID:  "held",
		StaffDomain: "trenova.app",
		Keys:        []tms.SupportKey{{Kid: "old", Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig", X: keyOne}},
	})

	require.NoError(t, r.relay.Sync(t.Context(), nil))

	assert.Empty(t, r.tms.SupportKeyPuts())
	held := r.tms.SupportKeys()
	require.NotNil(t, held, "the TMS keeps what it holds")
	assert.Equal(t, "held", held.InstanceID)
}

func TestARotatedKeySetReplacesTheOldOne(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	require.NoError(t, r.relay.Sync(t.Context(), support(key("cp-1", keyOne))))

	require.NoError(t, r.relay.Sync(t.Context(), support(key("cp-2", keyTwo))))

	puts := r.tms.SupportKeyPuts()
	require.Len(t, puts, 2)
	held := r.tms.SupportKeys()
	require.Len(t, held.Keys, 1)
	assert.Equal(t, "cp-2", held.Keys[0].Kid)
}

func TestAChangedInstanceOrDomainIsDeliveredAgain(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	desired := support(key("cp-1", keyOne))
	require.NoError(t, r.relay.Sync(t.Context(), desired))

	moved := support(key("cp-1", keyOne))
	moved.StaffDomain = "support.trenova.app"
	require.NoError(t, r.relay.Sync(t.Context(), moved))

	moved.InstanceId = "5f1c2d3e-4b5a-4c6d-8e7f-9a0b1c2d3e4f"
	require.NoError(t, r.relay.Sync(t.Context(), moved))

	assert.Len(t, r.tms.SupportKeyPuts(), 3)
}

func TestAKeySetTheTMSRefusesIsAnErrorNamingOnlyTheKeyIDs(t *testing.T) {
	t.Parallel()
	r := newRig(t)
	bad := support(instanceapi.SupportKey{Kid: "cp-rsa", Kty: "RSA", Crv: "", Alg: "RS256", Use: "sig", X: keyOne})

	err := r.relay.Sync(t.Context(), bad)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cp-rsa")
	assert.False(t, strings.Contains(err.Error(), keyOne), "an error never carries a key")
	assert.Nil(t, r.tms.SupportKeys())
}
