package tenanthandler_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/api/tenanthandler"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const instanceID = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"

type lifecycle struct {
	shared   bool
	applied  *cloudtenant.Desired
	source   *tenantarchive.Access
	path     string
	readings *tenantlifecycle.ReadingsRequest
	usage    []tenantlifecycle.TenantUsage
	archive  *tenantlifecycle.ArchiveRequest
	restore  *tenantlifecycle.RestoreRequest
	grant    *tenantarchive.Access
	err      error
}

func (l *lifecycle) Shared() bool { return l.shared }

func (l *lifecycle) Apply(
	_ context.Context,
	id string,
	d *cloudtenant.Desired,
	source *tenantarchive.Access,
) (*cloudtenant.Status, error) {
	l.path, l.applied, l.source = id, d, source
	if l.err != nil {
		return nil, l.err
	}
	revision := d.Revision
	return &cloudtenant.Status{
		InstanceID:      id,
		AppliedRevision: &revision,
		State:           cloudtenant.StateActive,
	}, nil
}

func (l *lifecycle) List(context.Context, string, int) (*tenantlifecycle.StatusPage, error) {
	return &tenantlifecycle.StatusPage{Tenants: []*cloudtenant.Status{
		{InstanceID: instanceID, State: cloudtenant.StateSuspended},
	}}, l.err
}

func (l *lifecycle) Readings(
	_ context.Context,
	req *tenantlifecycle.ReadingsRequest,
) (*tenantlifecycle.ReadingsPage, error) {
	l.readings = req
	return &tenantlifecycle.ReadingsPage{Tenants: []tenantlifecycle.TenantReadings{}}, l.err
}

func (l *lifecycle) SetUsage(
	_ context.Context,
	entries []tenantlifecycle.TenantUsage,
) (*tenantlifecycle.UsageAnswer, error) {
	l.usage = entries
	return &tenantlifecycle.UsageAnswer{
		Applied: len(entries),
		Refused: []tenantlifecycle.UsageRefused{},
	}, l.err
}

func (l *lifecycle) StartArchive(
	_ context.Context,
	id string,
	req *tenantlifecycle.ArchiveRequest,
) (*tenantcommand.View, error) {
	l.path, l.archive = id, req
	if l.err != nil {
		return nil, l.err
	}
	return &tenantcommand.View{
		CommandID:  req.CommandID,
		InstanceID: id,
		Kind:       tenantcommand.KindArchive,
		Purpose:    string(req.Purpose),
		State:      tenantcommand.StateQueued,
		Location:   req.Access.Location.String(),
	}, nil
}

func (l *lifecycle) StartRestore(
	_ context.Context,
	id string,
	req *tenantlifecycle.RestoreRequest,
) (*tenantcommand.View, error) {
	l.path, l.restore = id, req
	if l.err != nil {
		return nil, l.err
	}
	return &tenantcommand.View{
		CommandID:  req.CommandID,
		InstanceID: id,
		Kind:       tenantcommand.KindRestore,
		State:      tenantcommand.StateQueued,
	}, nil
}

func (l *lifecycle) Command(
	_ context.Context,
	id, commandID string,
	kind tenantcommand.Kind,
) (*tenantcommand.View, error) {
	l.path = id
	if l.err != nil {
		return nil, l.err
	}
	return &tenantcommand.View{CommandID: commandID, InstanceID: id, Kind: kind,
		State: tenantcommand.StateRunning}, nil
}

func (l *lifecycle) SetNightlyGrant(
	_ context.Context,
	access *tenantarchive.Access,
	_ time.Time,
) error {
	l.grant = access
	return l.err
}

func serve(t *testing.T, l *lifecycle, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	mux := http.NewServeMux()
	tenanthandler.NewHandler(l, zap.NewNop()).RouteManaged(mux)
	req := httptest.NewRequestWithContext(t.Context(), method, target, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	return rec
}

func TestTenantRoutesAreRefusedOnADedicatedInstance(t *testing.T) {
	t.Parallel()

	l := &lifecycle{}
	for _, route := range []struct{ method, target string }{
		{http.MethodGet, "/managed/v1/tenants"},
		{http.MethodGet, "/managed/v1/tenants/readings?from=2026-11-20T00:00:00Z&to=2026-11-20T00:00:00Z"},
		{http.MethodPut, "/managed/v1/tenants/usage"},
		{http.MethodPut, "/managed/v1/tenants/" + instanceID},
	} {
		rec := serve(t, l, route.method, route.target, "{}")
		assert.Equal(t, http.StatusConflict, rec.Code, route.target)
		assert.Contains(t, rec.Body.String(), "shared server")
	}
}

func TestApplyTakesTheEntryAndAnswersTheTenantsState(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	rec := serve(
		t,
		l,
		http.MethodPut,
		"/managed/v1/tenants/"+instanceID,
		`{"instanceId":"`+instanceID+`","revision":7,"lifecycle":"active","organizationName":"Acme",`+
			`"futureField":true}`,
	)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	assert.Equal(t, instanceID, l.path)
	assert.Equal(t, int64(7), l.applied.Revision)

	var status map[string]any
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &status))
	assert.Equal(t, map[string]any{
		"instanceId":      instanceID,
		"appliedRevision": float64(7),
		"state":           "active",
	}, status)
}

func TestApplyAnswers400ForAnEntryForAnotherInstance(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true, err: cloudtenant.ErrInstanceIDMismatch}
	rec := serve(t, l, http.MethodPut, "/managed/v1/tenants/"+instanceID, `{"revision":1}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, l, http.MethodPut, "/managed/v1/tenants/"+instanceID, `not json`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestApplyHidesAnInternalFailure(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true, err: errors.New("db down: secret detail")}
	rec := serve(t, l, http.MethodPut, "/managed/v1/tenants/"+instanceID, `{}`)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "secret detail")
}

func TestListAndReadingsAreBounded(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	rec := serve(t, l, http.MethodGet, "/managed/v1/tenants?limit=1000", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, l, http.MethodGet, "/managed/v1/tenants?limit=50", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"state":"suspended"`)

	rec = serve(t, l, http.MethodGet, "/managed/v1/tenants/readings?from=yesterday&to=now", "")
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(
		t,
		l,
		http.MethodGet,
		"/managed/v1/tenants/readings?from=2026-11-20T00:00:00Z&to=2026-11-20T03:00:00Z&after="+instanceID,
		"",
	)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, instanceID, l.readings.After)
	assert.Equal(t, int64(3*3600), l.readings.To-l.readings.From)
}

func TestUsageNeedsItsTenantsList(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	rec := serve(t, l, http.MethodPut, "/managed/v1/tenants/usage", `{}`)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = serve(t, l, http.MethodPut, "/managed/v1/tenants/usage",
		`{"tenants":[{"instanceId":"`+instanceID+`","overage":false,"meters":[]}]}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, l.usage, 1)
	assert.JSONEq(t, `{"applied":1,"refused":[]}`, rec.Body.String())
}
