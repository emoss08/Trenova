package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedfeaturesservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/managedusageservice"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const managedToken = "0123456789abcdef0123456789abcdef"

type fakeManagedUsage struct {
	since    int64
	readings []managedusageservice.Reading
	err      error
	state    *managedusage.State
}

func (f *fakeManagedUsage) Readings(
	_ context.Context,
	since int64,
) ([]managedusageservice.Reading, error) {
	f.since = since
	return f.readings, f.err
}

func (f *fakeManagedUsage) SetState(_ context.Context, s *managedusage.State) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if f.err != nil {
		return f.err
	}
	f.state = s
	return nil
}

type fakeFeatureRepo struct {
	mu      sync.Mutex
	state   *managedfeatures.State
	changes []*managedfeatures.Change
	err     error
}

func (f *fakeFeatureRepo) Get(context.Context) (*managedfeatures.State, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	if f.state == nil {
		return nil, managedfeatures.ErrNoState
	}
	c := *f.state
	c.Features = slices.Clone(f.state.Features)
	return &c, nil
}

func (f *fakeFeatureRepo) Replace(
	_ context.Context,
	req *managedfeaturesports.ReplaceRequest,
) (*managedfeatures.Replacement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	replacement := managedfeatures.Replace(f.state, req.Features, req.Now)
	if !replacement.Changed() {
		return replacement, nil
	}
	c := *replacement.Current
	c.Features = slices.Clone(replacement.Current.Features)
	f.state = &c
	f.changes = append(f.changes, replacement.Change)
	return replacement, nil
}

func (f *fakeFeatureRepo) History(
	context.Context,
	int,
) (*managedfeatures.State, []*managedfeatures.Change, error) {
	return f.state, f.changes, nil
}

const featuresReceivedAt = int64(1_794_650_400)

type featureRig struct {
	*managedfeaturesservice.Receiver

	gate *managedfeaturesservice.Service
}

func (r featureRig) Enabled(ctx context.Context, key platformcatalog.FeatureKey) (bool, error) {
	return r.gate.Enabled(ctx, key)
}

func newFeatures(
	repo *fakeFeatureRepo,
	observers ...managedfeaturesports.PlanObserver,
) featureRig {
	gate := managedfeaturesservice.New(managedfeaturesservice.Params{
		Repo:   repo,
		Clock:  func() int64 { return featuresReceivedAt },
		Logger: zap.NewNop(),
	})
	return featureRig{
		Receiver: managedfeaturesservice.NewReceiver(managedfeaturesservice.ReceiverParams{
			Features:  gate,
			Observers: observers,
			Logger:    zap.NewNop(),
		}),
		gate: gate,
	}
}

func newManagedHandler(
	usage *fakeManagedUsage,
	features api.ManagedFeatures,
	invites *fakeInvites,
) http.Handler {
	return api.NewManagedHandler(&api.ManagedHandlerParams{
		Usage:    usage,
		Features: features,
		Invites:  invites,
		Token:    managedToken,
		Logger:   zap.NewNop(),
	})
}

type fakeInvites struct {
	got    *instancebootstrapports.FirstAdministratorInvite
	result *instancebootstrapports.FirstAdministratorInviteResult
	err    error
}

func (f *fakeInvites) Install(
	_ context.Context,
	invite *instancebootstrapports.FirstAdministratorInvite,
) (*instancebootstrapports.FirstAdministratorInviteResult, error) {
	f.got = invite
	return f.result, f.err
}

func serveInvite(
	t *testing.T,
	invites *fakeInvites,
	token, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(
		t.Context(),
		http.MethodPut,
		"/managed/v1/first-administrator/invite",
		strings.NewReader(body),
	)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	newManagedHandler(&fakeManagedUsage{}, newFeatures(&fakeFeatureRepo{}), invites).
		ServeHTTP(rec, req)
	return rec
}

func serveManaged(
	t *testing.T,
	usage *fakeManagedUsage,
	method, target, token, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	newManagedHandler(usage, newFeatures(&fakeFeatureRepo{}), &fakeInvites{}).ServeHTTP(rec, req)
	return rec
}

func serveFeatures(
	t *testing.T,
	features api.ManagedFeatures,
	method, token, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequestWithContext(
		t.Context(),
		method,
		"/managed/v1/features",
		strings.NewReader(body),
	)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	newManagedHandler(&fakeManagedUsage{}, features, &fakeInvites{}).ServeHTTP(rec, req)
	return rec
}

func TestManagedSetFeaturesEnablesTheListedFeatures(t *testing.T) {
	t.Parallel()

	repo := &fakeFeatureRepo{}
	features := newFeatures(repo)
	on, err := features.Enabled(t.Context(), platformcatalog.FeatureAccounting)
	require.NoError(t, err)
	require.False(t, on)

	rec := serveFeatures(t, features, http.MethodPut, managedToken,
		`{"features": ["tms.accounting", "ai.desk"]}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Empty(t, rec.Body.String())
	require.NotNil(t, repo.state)
	assert.Equal(t, []platformcatalog.FeatureKey{"ai.desk", "tms.accounting"}, repo.state.Features)

	on, err = features.Enabled(t.Context(), platformcatalog.FeatureAccounting)
	require.NoError(t, err)
	assert.True(t, on)
}

func TestManagedSetFeaturesAcceptsAnEmptyList(t *testing.T) {
	t.Parallel()

	repo := &fakeFeatureRepo{}
	rec := serveFeatures(t, newFeatures(repo), http.MethodPut, managedToken, `{"features": []}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.NotNil(t, repo.state)
	assert.Equal(t, []platformcatalog.FeatureKey{}, repo.state.Features)
}

func TestManagedSetFeaturesRefusesAMissingToken(t *testing.T) {
	t.Parallel()

	repo := &fakeFeatureRepo{}
	rec := serveFeatures(t, newFeatures(repo), http.MethodPut, "", `{"features": ["ai.desk"]}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
	assert.Nil(t, repo.state)
}

func TestManagedSetFeaturesRefusesABodyThatIsNotAFeatureList(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"not JSON":           `{`,
		"no features":        `{}`,
		"null features":      `{"features": null}`,
		"features not array": `{"features": "ai.desk"}`,
		"keys not strings":   `{"features": [1, 2]}`,
		"an array":           `["ai.desk"]`,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			repo := &fakeFeatureRepo{}
			rec := serveFeatures(t, newFeatures(repo), http.MethodPut, managedToken, body)
			assert.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
			assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
			assert.Nil(t, repo.state)
		})
	}
}

func TestManagedSetFeaturesHidesAStoreFailure(t *testing.T) {
	t.Parallel()

	repo := &fakeFeatureRepo{err: errors.New("pq: connection refused at 10.0.0.5")}
	rec := serveFeatures(t, newFeatures(repo), http.MethodPut, managedToken,
		`{"features": ["ai.desk"]}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "10.0.0.5")
}

func TestManagedFeaturesAnswersWhatIsHeld(t *testing.T) {
	t.Parallel()

	features := newFeatures(&fakeFeatureRepo{})
	rec := serveFeatures(t, features, http.MethodPut, managedToken,
		`{"features": ["tms.accounting", "tms.core", "ai.desk"]}`)
	require.Equal(t, http.StatusNoContent, rec.Code)

	rec = serveFeatures(t, features, http.MethodGet, managedToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	assert.JSONEq(t,
		`{"features":["ai.desk","tms.accounting"],"receivedAt":"2026-11-14T10:00:00Z"}`,
		rec.Body.String())
}

func TestManagedFeaturesAnswersAnEmptyListBeforeAnyIsSet(t *testing.T) {
	t.Parallel()

	rec := serveFeatures(t, newFeatures(&fakeFeatureRepo{}), http.MethodGet, managedToken, "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.JSONEq(t, `{"features":[]}`, rec.Body.String())
}

func TestManagedFeaturesRefusesAMissingToken(t *testing.T) {
	t.Parallel()

	rec := serveFeatures(t, newFeatures(&fakeFeatureRepo{}), http.MethodGet, "", "")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestManagedRefusesAWrongToken(t *testing.T) {
	t.Parallel()

	for _, token := range []string{"", "wrong", managedToken + "x"} {
		rec := serveManaged(t, &fakeManagedUsage{}, http.MethodGet,
			"/managed/v1/readings?since=2026-11-05T10:00:00Z", token, "")
		assert.Equal(t, http.StatusUnauthorized, rec.Code)
		assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
	}
}

func TestManagedReadings(t *testing.T) {
	t.Parallel()

	usage := &fakeManagedUsage{readings: []managedusageservice.Reading{
		{Meter: platformcatalog.MeterShipmentsTotal, Hour: 1_793_872_800, Quantity: 7},
	}}
	rec := serveManaged(t, usage, http.MethodGet,
		"/managed/v1/readings?since=2026-11-05T10:00:00Z", managedToken, "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, int64(1_793_872_800), usage.since)
	assert.JSONEq(t,
		`{"readings":[{"meter":"shipments.total","hour":"2026-11-05T10:00:00Z","quantity":7}]}`,
		rec.Body.String())
}

func TestManagedReadingsRefuseABadSince(t *testing.T) {
	t.Parallel()

	rec := serveManaged(t, &fakeManagedUsage{}, http.MethodGet,
		"/managed/v1/readings?since=yesterday", managedToken, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serveManaged(t, &fakeManagedUsage{err: managedusageservice.ErrSinceTooOld},
		http.MethodGet, "/managed/v1/readings?since=2026-01-01T00:00:00Z", managedToken, "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serveManaged(t, &fakeManagedUsage{err: errors.New("database down")},
		http.MethodGet, "/managed/v1/readings?since=2026-11-05T10:00:00Z", managedToken, "")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "database down")
}

func TestManagedSetState(t *testing.T) {
	t.Parallel()

	usage := &fakeManagedUsage{}
	rec := serveManaged(t, usage, http.MethodPut, "/managed/v1/state", managedToken, `{
		"window": {"start": "2026-11-01T00:00:00Z", "end": "2026-12-01T00:00:00Z"},
		"overage": true,
		"meters": [{"meter": "shipments.total", "unit": "count", "quantity": 12,
			"allowance": 500, "hardCap": false, "stopped": false}]
	}`)
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.NotNil(t, usage.state)
	assert.True(t, usage.state.HasWindow())
	assert.True(t, usage.state.Overage)
	m, ok := usage.state.Meter(platformcatalog.MeterShipmentsTotal)
	require.True(t, ok)
	assert.Equal(t, int64(500), m.Allowance)
}

func TestManagedSetStateWithoutAWindow(t *testing.T) {
	t.Parallel()

	usage := &fakeManagedUsage{}
	rec := serveManaged(t, usage, http.MethodPut, "/managed/v1/state", managedToken,
		`{"overage": false}`)
	require.Equal(t, http.StatusNoContent, rec.Code)
	assert.False(t, usage.state.HasWindow())
	assert.NotNil(t, usage.state.Meters)
}

func TestManagedSetStateRefusesAnInvalidBody(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		body string
	}{
		{name: "not JSON", body: `{`},
		{
			name: "an unknown unit",
			body: `{"meters":[{"meter":"shipments.total","unit":"minutes"}]}`,
		},
		{
			name: "a window that ends first",
			body: `{"window":{"start":"2026-12-01T00:00:00Z","end":"2026-11-01T00:00:00Z"}}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			usage := &fakeManagedUsage{}
			rec := serveManaged(
				t,
				usage,
				http.MethodPut,
				"/managed/v1/state",
				managedToken,
				tt.body,
			)
			assert.Equal(t, http.StatusBadRequest, rec.Code)
			assert.Nil(t, usage.state)
		})
	}
}

func TestManagedSetStateHidesAStoreFailure(t *testing.T) {
	t.Parallel()

	rec := serveManaged(t, &fakeManagedUsage{err: errors.New("database down")},
		http.MethodPut, "/managed/v1/state", managedToken, `{"overage": true}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "database down")
	var p map[string]any
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &p))
	assert.InDelta(t, float64(http.StatusInternalServerError), p["status"], 0)
}

func TestManagedRejectsOtherRoutes(t *testing.T) {
	t.Parallel()

	rec := serveManaged(t, &fakeManagedUsage{}, http.MethodPost, "/managed/v1/state",
		managedToken, `{}`)
	assert.Equal(t, http.StatusMethodNotAllowed, rec.Code)
}

const inviteHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestManagedInviteInstallsAndAnswers(t *testing.T) {
	t.Parallel()

	invites := &fakeInvites{result: &instancebootstrapports.FirstAdministratorInviteResult{
		InviteTokenSHA256: inviteHash,
	}}
	rec := serveInvite(t, invites, managedToken,
		`{"tokenSha256":"`+inviteHash+`","expiresAt":"2026-11-14T10:00:00Z"}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.NotNil(t, invites.got)
	assert.Equal(t, inviteHash, invites.got.TokenSHA256)
	assert.Equal(t, int64(1_794_650_400), invites.got.ExpiresAt)

	var answer map[string]any
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &answer))
	assert.Equal(t, map[string]any{"setUp": false, "inviteTokenSha256": inviteHash}, answer)
}

func TestManagedInviteReportsASetUpAdministrator(t *testing.T) {
	t.Parallel()

	invites := &fakeInvites{
		result: &instancebootstrapports.FirstAdministratorInviteResult{SetUp: true},
	}
	rec := serveInvite(t, invites, managedToken,
		`{"tokenSha256":"`+inviteHash+`","expiresAt":"2026-11-14T10:00:00Z"}`)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"setUp":true}`, rec.Body.String())
}

func TestManagedInviteRefuses(t *testing.T) {
	t.Parallel()

	valid := `{"tokenSha256":"` + inviteHash + `","expiresAt":"2026-11-14T10:00:00Z"}`
	invalid := errortypes.NewMultiError()
	invalid.Add("expiresAt", errortypes.ErrInvalid, "expiresAt must be in the future")

	tests := map[string]struct {
		token  string
		body   string
		err    error
		status int
	}{
		"no token":    {"", valid, nil, http.StatusUnauthorized},
		"wrong token": {"nope", valid, nil, http.StatusUnauthorized},
		"not json":    {managedToken, "{", nil, http.StatusBadRequest},
		"unknown field": {
			managedToken,
			`{"tokenSha256":"x","expiresAt":"2026-11-14T10:00:00Z","token":"raw"}`,
			nil,
			http.StatusBadRequest,
		},
		"no expiry": {
			managedToken,
			`{"tokenSha256":"` + inviteHash + `"}`,
			nil,
			http.StatusBadRequest,
		},
		"not rfc 3339": {
			managedToken,
			`{"tokenSha256":"` + inviteHash + `","expiresAt":"tomorrow"}`,
			nil,
			http.StatusBadRequest,
		},
		"invalid": {managedToken, valid, invalid, http.StatusBadRequest},
		"not bootstrapped": {
			managedToken,
			valid,
			instancebootstrapports.ErrNotBootstrapped,
			http.StatusConflict,
		},
		"store failure": {
			managedToken,
			valid,
			errors.New("pq: connection refused at 10.0.0.5"),
			http.StatusInternalServerError,
		},
		"oversized body": {
			managedToken,
			`{"tokenSha256":"` + strings.Repeat("a", 5000) + `"}`,
			nil,
			http.StatusRequestEntityTooLarge,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			invites := &fakeInvites{
				result: &instancebootstrapports.FirstAdministratorInviteResult{},
				err:    tt.err,
			}
			rec := serveInvite(t, invites, tt.token, tt.body)
			assert.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, "application/problem+json", rec.Header().Get("Content-Type"))
			assert.NotContains(t, rec.Body.String(), "10.0.0.5")
		})
	}
}

type failingPlanObserver struct{ calls int }

func (o *failingPlanObserver) PlanChanged(
	context.Context,
	*managedfeatures.Replacement,
) error {
	o.calls++
	return errors.New("notifications are down")
}

func TestManagedSetFeaturesAnswers204AndKeepsTheChangeWhenNotifyingFails(t *testing.T) {
	t.Parallel()

	repo := &fakeFeatureRepo{state: &managedfeatures.State{
		ID:       managedfeatures.CurrentStateID,
		Features: []platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
	}}
	observer := &failingPlanObserver{}
	rec := serveFeatures(t, newFeatures(repo, observer), http.MethodPut, managedToken,
		`{"features": []}`)

	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Equal(t, 1, observer.calls)
	require.Len(t, repo.changes, 1)
	assert.Equal(t,
		[]platformcatalog.FeatureKey{"tms.dispatch_intelligence"},
		repo.changes[0].Removed,
	)
}
