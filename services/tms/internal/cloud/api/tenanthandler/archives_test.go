package tenanthandler_test

import (
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/app/tenantlifecycle"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	commandID   = "7d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"
	fakeAccess  = "AKFAKEACCESS"
	fakeSecret  = "fake-secret-for-tests"
	keyMaterial = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
)

func target(kind, access, prefix string) string {
	return `{"instanceId":"` + instanceID + `","commandId":"` + commandID + `","kind":"` + kind +
		`","access":"` + access + `","endpoint":"https://objects.test","region":"us-east-1",` +
		`"bucket":"archives","prefix":"` + prefix + `","accessKeyId":"` + fakeAccess +
		`","secretAccessKey":"` + fakeSecret + `","archiveKeys":[{"id":"region-1","key":"` +
		keyMaterial + `","current":true}]}`
}

func TestStartingAnArchivePassesItsTargetAndAnswersTheCommand(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	prefix := tenantarchive.CommandPrefix(instanceID, commandID)
	rec := serve(
		t,
		l,
		http.MethodPost,
		"/managed/v1/tenants/"+instanceID+"/archives",
		`{"commandId":"`+commandID+`","purpose":"final","target":`+target(
			"archive",
			"write",
			prefix,
		)+`}`,
	)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	require.NotNil(t, l.archive)
	assert.Equal(t, "final", string(l.archive.Purpose))
	assert.Equal(t, fakeSecret, l.archive.Access.Credentials.SecretKey)
	require.Len(t, l.archive.Access.Keys, 1)
	assert.Equal(t, "0123456789abcdef0123456789abcdef", string(l.archive.Access.Keys[0].Material))
	assert.True(t, l.archive.Access.Keys[0].Current)
	assert.NotContains(t, rec.Body.String(), fakeSecret)
	assert.NotContains(t, rec.Body.String(), fakeAccess)
	assert.Contains(t, rec.Body.String(), `"state":"queued"`)
}

func TestATargetForAnotherCommandOrAccessIsRefused(t *testing.T) {
	t.Parallel()

	prefix := tenantarchive.CommandPrefix(instanceID, commandID)
	for _, body := range []string{
		`{"commandId":"` + commandID + `","purpose":"final","target":` + target("archive", "read", prefix) + `}`,
		`{"commandId":"` + commandID + `","purpose":"final","target":` + target("restore", "write", prefix) + `}`,
		`{"commandId":"8d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10","purpose":"final","target":` +
			target("archive", "write", prefix) + `}`,
		`{"commandId":"` + commandID + `","purpose":"final","target":` +
			strings.Replace(target("archive", "write", prefix), keyMaterial, "%%%", 1) + `}`,
	} {
		l := &lifecycle{shared: true}
		rec := serve(t, l, http.MethodPost, "/managed/v1/tenants/"+instanceID+"/archives", body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, body)
		assert.Nil(t, l.archive)
		assert.NotContains(t, rec.Body.String(), fakeSecret)
	}
}

func TestArchiveRoutesMapTheirRefusals(t *testing.T) {
	t.Parallel()

	prefix := tenantarchive.CommandPrefix(instanceID, commandID)
	body := `{"commandId":"` + commandID + `","purpose":"final","target":` + target(
		"archive",
		"write",
		prefix,
	) + `}`
	for _, tc := range []struct {
		err  error
		code int
	}{
		{tenantlifecycle.ErrTenantBusy, http.StatusLocked},
		{tenantlifecycle.ErrTenantState, http.StatusConflict},
		{tenantlifecycle.ErrCommandConflict, http.StatusConflict},
		{tenantlifecycle.ErrPurposeInvalid, http.StatusBadRequest},
		{tenantarchive.ErrLocationInvalid, http.StatusBadRequest},
		{cloudtenantports.ErrNotFound, http.StatusNotFound},
		{errors.New("wrapped " + fakeSecret), http.StatusInternalServerError},
	} {
		l := &lifecycle{shared: true, err: tc.err}
		rec := serve(t, l, http.MethodPost, "/managed/v1/tenants/"+instanceID+"/archives", body)
		assert.Equal(t, tc.code, rec.Code, tc.err.Error())
		assert.NotContains(t, rec.Body.String(), fakeSecret)
	}
}

func TestRestoreAndCommandStateRoutes(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	prefix := "tenants/" + instanceID + "/nightly/2026-11-19/"
	rec := serve(t, l, http.MethodPost, "/managed/v1/tenants/"+instanceID+"/restores",
		`{"commandId":"`+commandID+`","expectSha256":"`+strings.Repeat("ab", 32)+`","target":`+
			target("restore", "read", prefix)+`}`)
	require.Equal(t, http.StatusAccepted, rec.Code, rec.Body.String())
	require.NotNil(t, l.restore)
	assert.Equal(t, prefix, l.restore.Access.Location.Prefix)

	for _, path := range []string{"archives", "restores", "imports"} {
		rec = serve(
			t,
			l,
			http.MethodGet,
			"/managed/v1/tenants/"+instanceID+"/"+path+"/"+commandID,
			"",
		)
		require.Equal(t, http.StatusOK, rec.Code, path)
		assert.Contains(t, rec.Body.String(), `"state":"running"`)
	}
	missing := &lifecycle{shared: true, err: cloudtenantports.ErrCommandNotFound}
	rec = serve(
		t,
		missing,
		http.MethodGet,
		"/managed/v1/tenants/"+instanceID+"/archives/"+commandID,
		"",
	)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestApplyTakesAnImportTargetApartFromTheEntry(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	prefix := tenantarchive.CommandPrefix(instanceID, "9d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10")
	rec := serve(t, l, http.MethodPut, "/managed/v1/tenants/"+instanceID,
		`{"instanceId":"`+instanceID+`","revision":1,"commands":[{"kind":"import","id":"`+commandID+
			`","location":"s3://archives/`+prefix+`","expectSha256":"`+strings.Repeat("ab", 32)+
			`"}],"importTarget":`+target("import", "read", prefix)+`}`)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, l.source)
	assert.Equal(t, fakeSecret, l.source.Credentials.SecretKey)
	body, err := sonic.Marshal(l.applied)
	require.NoError(t, err)
	assert.NotContains(t, string(body), fakeSecret, "the entry the server stores carries no secret")
	assert.NotContains(t, string(body), keyMaterial)
}

func nightlyBody(prefix string) string {
	return `{"endpoint":"https://objects.test","region":"us-east-1","bucket":"archives",` +
		`"prefix":"` + prefix + `","accessKeyId":"` + fakeAccess + `","secretAccessKey":"` +
		fakeSecret + `","archiveKeys":[{"id":"region-1","key":"` + keyMaterial +
		`","current":true}]}`
}

func TestTheNightlyStorageGrantIsTaken(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	rec := serve(
		t,
		l,
		http.MethodPut,
		"/managed/v1/tenants/archive-storage",
		nightlyBody("tenants/"),
	)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.NotNil(t, l.grant)
	assert.Equal(t, "archives", l.grant.Location.Bucket)
	assert.Equal(t, "tenants/", l.grant.Location.Prefix)
	assert.NotContains(t, rec.Body.String(), fakeSecret)
}

func TestARefusedNightlyStorageIsAnswered400NamingTheRuleNotTheCredential(t *testing.T) {
	t.Parallel()

	for name, cause := range map[string]error{
		"prefix": tenantarchive.ErrNightlyPrefix,
		"endpoint": tenantarchive.Refuse(tenantarchive.RefusalLocation,
			"the archive endpoint's host is not in this server's managed.archive.allowedEndpoints"),
		"http": tenantarchive.ErrLocationInvalid,
	} {
		l := &lifecycle{shared: true, err: cause}
		rec := serve(t, l, http.MethodPut, "/managed/v1/tenants/archive-storage",
			nightlyBody("tenants/"))
		require.Equal(t, http.StatusBadRequest, rec.Code, name)
		assert.Contains(t, rec.Body.String(), cause.Error(), name)
		assert.NotContains(t, rec.Body.String(), fakeSecret, name)
		assert.NotContains(t, rec.Body.String(), fakeAccess, name)
	}
}

func TestTheNightlyStorageIsRemoved(t *testing.T) {
	t.Parallel()

	l := &lifecycle{shared: true}
	rec := serve(t, l, http.MethodDelete, "/managed/v1/tenants/archive-storage", "")
	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	assert.Equal(t, 1, l.removed)

	failing := &lifecycle{shared: true, err: errors.New("the database is down")}
	rec = serve(t, failing, http.MethodDelete, "/managed/v1/tenants/archive-storage", "")
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "database")
}

func TestArchiveRoutesAreRefusedOnADedicatedInstance(t *testing.T) {
	t.Parallel()

	l := &lifecycle{}
	for _, route := range []struct{ method, target string }{
		{http.MethodPost, "/managed/v1/tenants/" + instanceID + "/archives"},
		{http.MethodGet, "/managed/v1/tenants/" + instanceID + "/archives/" + commandID},
		{http.MethodPost, "/managed/v1/tenants/" + instanceID + "/restores"},
		{http.MethodPut, "/managed/v1/tenants/archive-storage"},
		{http.MethodDelete, "/managed/v1/tenants/archive-storage"},
	} {
		rec := serve(t, l, route.method, route.target, "{}")
		assert.Equal(t, http.StatusConflict, rec.Code, route.method+" "+route.target)
	}
	assert.Zero(t, l.removed, "a dedicated instance never removes anything")
}
