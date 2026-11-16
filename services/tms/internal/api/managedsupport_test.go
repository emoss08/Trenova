package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type fakeSupportAccess struct {
	set    *supportaccess.KeySet
	held   *supportaccess.KeySet
	grants []*supportaccessports.LiveGrant
}

func (f *fakeSupportAccess) LiveGrants(context.Context) ([]*supportaccessports.LiveGrant, error) {
	return f.grants, nil
}

func (f *fakeSupportAccess) SetKeys(_ context.Context, keys *supportaccess.KeySet) error {
	if err := keys.Normalize(); err != nil {
		multiErr := errortypes.NewMultiError()
		multiErr.Add("keys", errortypes.ErrInvalid, err.Error())
		return multiErr
	}
	f.set = keys
	return nil
}

func (f *fakeSupportAccess) KeySet(context.Context) (*supportaccess.KeySet, error) {
	return f.held, nil
}

const supportKeyX = "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"

func serveSupport(t *testing.T, support *fakeSupportAccess, method, body, token string) *httptest.ResponseRecorder {
	t.Helper()
	handler := api.NewManagedHandler(&api.ManagedHandlerParams{
		Token:         managedToken,
		Logger:        zap.NewNop(),
		SupportAccess: support,
	})
	req := httptest.NewRequest(method, "/managed/v1/support-access", strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestManagedSupportAccessStoresTheDeliveredKeys(t *testing.T) {
	t.Parallel()

	support := &fakeSupportAccess{}
	body := `{"instanceId":"inst_01JTEST","staffDomain":"Trenova.app","keys":[` +
		`{"kid":"cp-1","kty":"OKP","crv":"Ed25519","alg":"EdDSA","use":"sig","x":"` + supportKeyX + `"}]}`

	rec := serveSupport(t, support, http.MethodPut, body, managedToken)

	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.NotNil(t, support.set)
	assert.Equal(t, "inst_01JTEST", support.set.InstanceID)
	assert.Equal(t, "trenova.app", support.set.StaffDomain)
	require.Len(t, support.set.Keys, 1)
	assert.Equal(t, "cp-1", support.set.Keys[0].KeyID)
}

func TestManagedSupportAccessRefusesWhatIsNotAnEd25519KeySet(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"an unknown field": `{"instanceId":"i","staffDomain":"trenova.app","keys":[],"extra":1}`,
		"no keys":          `{"instanceId":"inst_1","staffDomain":"trenova.app","keys":[]}`,
		"an RSA key": `{"instanceId":"inst_1","staffDomain":"trenova.app","keys":[` +
			`{"kid":"k","kty":"RSA","crv":"","alg":"RS256","use":"sig","x":"` + supportKeyX + `"}]}`,
		"a short key": `{"instanceId":"inst_1","staffDomain":"trenova.app","keys":[` +
			`{"kid":"k","kty":"OKP","crv":"Ed25519","alg":"EdDSA","use":"sig","x":"AAAA"}]}`,
		"a domain that is not a host": `{"instanceId":"inst_1","staffDomain":"trenova","keys":[` +
			`{"kid":"k","kty":"OKP","crv":"Ed25519","alg":"EdDSA","use":"sig","x":"` + supportKeyX + `"}]}`,
	} {
		support := &fakeSupportAccess{}
		rec := serveSupport(t, support, http.MethodPut, body, managedToken)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
		assert.Nil(t, support.set, name)
	}
}

func TestManagedSupportAccessRefusesAMissingToken(t *testing.T) {
	t.Parallel()

	support := &fakeSupportAccess{}
	rec := serveSupport(t, support, http.MethodPut, `{}`, "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Nil(t, support.set)
}

func serveSupportGrants(t *testing.T, support *fakeSupportAccess, token string) *httptest.ResponseRecorder {
	t.Helper()
	handler := api.NewManagedHandler(&api.ManagedHandlerParams{
		Token:         managedToken,
		Logger:        zap.NewNop(),
		SupportAccess: support,
	})
	req := httptest.NewRequest(http.MethodGet, "/managed/v1/support-access/grants", nil)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestManagedSupportGrantsListsTheLiveGrantsInTheContractsShape(t *testing.T) {
	t.Parallel()

	acme, globex := pulid.MustNew("org_"), pulid.MustNew("org_")
	readOnly, full := pulid.MustNew("sag_"), pulid.MustNew("sag_")
	support := &fakeSupportAccess{grants: []*supportaccessports.LiveGrant{
		{
			Grant: &supportaccess.Grant{
				ID:             readOnly,
				OrganizationID: acme,
				StaffEmail:     "dana@trenova.app",
				Scope:          supportaccess.ScopeReadOnly,
				ExpiresAt:      1_800_000_000,
				Reason:         "Looking into a stuck invoice",
			},
			OrganizationName: "Acme Freight",
		},
		{
			Grant: &supportaccess.Grant{
				ID:             full,
				OrganizationID: globex,
				StaffEmail:     "lee@trenova.app",
				Scope:          supportaccess.ScopeFull,
				ExpiresAt:      1_800_003_600,
			},
			OrganizationName: "Globex Logistics",
		},
	}}

	rec := serveSupportGrants(t, support, managedToken)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"grants": [
		{"id": "`+readOnly.String()+`", "organizationId": "`+acme.String()+`",
		 "organizationName": "Acme Freight", "staffEmail": "dana@trenova.app",
		 "scope": "read_only", "expiresAt": "2027-01-15T08:00:00Z"},
		{"id": "`+full.String()+`", "organizationId": "`+globex.String()+`",
		 "organizationName": "Globex Logistics", "staffEmail": "lee@trenova.app",
		 "scope": "full", "expiresAt": "2027-01-15T09:00:00Z"}
	]}`, rec.Body.String())
	assert.NotContains(t, rec.Body.String(), "stuck invoice", "the reason stays on the instance")
}

func TestManagedSupportGrantsAnswersAnEmptyListWhenNoneIsLive(t *testing.T) {
	t.Parallel()

	rec := serveSupportGrants(t, &fakeSupportAccess{}, managedToken)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"grants": []}`, rec.Body.String())
}

func TestManagedSupportGrantsRefusesAWrongToken(t *testing.T) {
	t.Parallel()

	rec := serveSupportGrants(t, &fakeSupportAccess{}, "not-the-token")

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.NotContains(t, rec.Body.String(), "grants")
}

type recordedLifecycle struct {
	hooks []fx.Hook
}

func (l *recordedLifecycle) Append(hook fx.Hook) {
	l.hooks = append(l.hooks, hook)
}

func TestManagedSupportGrantsAreNotServedWhenManagedModeIsOff(t *testing.T) {
	t.Parallel()

	off := &recordedLifecycle{}
	api.StartManagedServer(api.ManagedServerParams{
		Config: &managedconfig.Config{Enabled: false},
		Logger: zap.NewNop(),
		LC:     off,
	})
	assert.Empty(t, off.hooks, "a self-hosted instance opens no operator listener, so no grants route")

	on := &recordedLifecycle{}
	api.StartManagedServer(api.ManagedServerParams{
		Config: &managedconfig.Config{Enabled: true, Token: managedToken},
		Logger: zap.NewNop(),
		LC:     on,
	})
	assert.Len(t, on.hooks, 1, "a managed instance opens the listener the grants route lives on")
}

func TestManagedSupportAccessAnswersTheKeyIDsItHolds(t *testing.T) {
	t.Parallel()

	support := &fakeSupportAccess{held: &supportaccess.KeySet{
		InstanceID:  "inst_1",
		StaffDomain: "trenova.app",
		Keys:        []supportaccess.SigningKey{{KeyID: "cp-1", X: supportKeyX}},
		ReceivedAt:  1_800_000_000,
	}}
	rec := serveSupport(t, support, http.MethodGet, "", managedToken)
	require.Equal(t, http.StatusOK, rec.Code)

	var out map[string]any
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &out))
	assert.Equal(t, []any{"cp-1"}, out["keyIds"])
	assert.Equal(t, "inst_1", out["instanceId"])
	assert.NotContains(t, rec.Body.String(), supportKeyX, "the answer names keys, it never carries one")
}
