package api_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/modules/identity/app/supportaccessservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeSupportLogs struct {
	waiting  []*supportlog.LogRequest
	reported map[pulid.ID]*supportlog.Report
	err      error
}

func (f *fakeSupportLogs) Waiting(context.Context) ([]*supportlog.LogRequest, error) {
	return f.waiting, nil
}

func (f *fakeSupportLogs) Report(_ context.Context, id pulid.ID, report *supportlog.Report) error {
	if f.err != nil {
		return f.err
	}
	if f.reported == nil {
		f.reported = map[pulid.ID]*supportlog.Report{}
	}
	f.reported[id] = report
	return nil
}

type fakeLogPulls struct {
	pulls []*supportaccessservice.LogPullRequest
	err   error
}

func (f *fakeLogPulls) RecordLogPull(_ context.Context, req *supportaccessservice.LogPullRequest) error {
	f.pulls = append(f.pulls, req)
	return f.err
}

func serveLogs(
	t *testing.T,
	logs *fakeSupportLogs,
	pulls *fakeLogPulls,
	method, path, body string,
) *httptest.ResponseRecorder {
	t.Helper()
	handler := api.NewManagedHandler(&api.ManagedHandlerParams{
		Token:       managedToken,
		Logger:      zap.NewNop(),
		SupportLogs: logs,
		LogPulls:    pulls,
	})
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+managedToken)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestManagedLogRequestsListsTheOnesInFlightWithWhoAskedAndTheirState(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew(supportlog.RequestIDPrefix)
	resumed := pulid.MustNew(supportlog.RequestIDPrefix)
	collectingAt := int64(1_800_000_000 - 600)
	logs := &fakeSupportLogs{waiting: []*supportlog.LogRequest{{
		ID:             id,
		RequesterName:  "Ada Lovelace",
		RequesterEmail: "ada@example.com",
		Period:         supportlog.PeriodHour,
		Since:          1_800_000_000 - 3600,
		Until:          1_800_000_000,
		Status:         supportlog.StatusRequested,
		CreatedAt:      1_800_000_000,
	}, {
		ID:             resumed,
		RequesterName:  "Grace Hopper",
		RequesterEmail: "grace@example.com",
		Period:         supportlog.PeriodDay,
		Since:          1_800_000_000 - 86_400,
		Until:          1_800_000_000,
		Status:         supportlog.StatusCollecting,
		BundleID:       "lb_01JTEST",
		CollectingAt:   &collectingAt,
		CreatedAt:      1_800_000_000,
	}}}

	rec := serveLogs(t, logs, &fakeLogPulls{}, http.MethodGet, "/managed/v1/log-requests", "")
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var out struct {
		Requests []struct {
			ID          string  `json:"id"`
			Status      string  `json:"status"`
			BundleID    *string `json:"bundleId"`
			Since       string  `json:"since"`
			Until       string  `json:"until"`
			RequestedAt string  `json:"requestedAt"`
			RequestedBy struct {
				Email string `json:"email"`
				Name  string `json:"name"`
			} `json:"requestedBy"`
		} `json:"requests"`
	}
	require.NoError(t, sonic.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Requests, 2)
	got := out.Requests[0]
	assert.Equal(t, id.String(), got.ID)
	assert.Equal(t, "requested", got.Status)
	assert.Nil(t, got.BundleID, "a request with no bundle yet carries no bundleId")
	assert.NotContains(t, rec.Body.String(), `"bundleId":""`)
	assert.Equal(t, "2027-01-15T07:00:00Z", got.Since)
	assert.Equal(t, "2027-01-15T08:00:00Z", got.Until)
	assert.Equal(t, "2027-01-15T08:00:00Z", got.RequestedAt)
	assert.Equal(t, "ada@example.com", got.RequestedBy.Email)
	assert.Equal(t, "Ada Lovelace", got.RequestedBy.Name)

	again := out.Requests[1]
	assert.Equal(t, resumed.String(), again.ID)
	assert.Equal(t, "collecting", again.Status, "a restarted operator picks a collecting request up again")
	require.NotNil(t, again.BundleID)
	assert.Equal(t, "lb_01JTEST", *again.BundleID)
	assert.Equal(t, "grace@example.com", again.RequestedBy.Email)
}

func TestManagedLogRequestsAnswersAnEmptyListAsAList(t *testing.T) {
	t.Parallel()

	rec := serveLogs(t, &fakeSupportLogs{}, &fakeLogPulls{}, http.MethodGet, "/managed/v1/log-requests", "")
	require.Equal(t, http.StatusOK, rec.Code)
	assert.JSONEq(t, `{"requests":[]}`, rec.Body.String())
}

func TestManagedLogRequestReportIsPassedOn(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew(supportlog.RequestIDPrefix)
	logs := &fakeSupportLogs{}
	rec := serveLogs(t, logs, &fakeLogPulls{}, http.MethodPut, "/managed/v1/log-requests/"+id.String(),
		`{"status":"sent","bundleId":"lb_01JTEST"}`)

	require.Equal(t, http.StatusNoContent, rec.Code, rec.Body.String())
	require.Contains(t, logs.reported, id)
	assert.Equal(t, supportlog.StatusSent, logs.reported[id].Status)
	assert.Equal(t, "lb_01JTEST", logs.reported[id].BundleID)
}

func TestManagedLogRequestReportAnswersEachOutcome(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew(supportlog.RequestIDPrefix)
	path := "/managed/v1/log-requests/" + id.String()
	_, invalidStatus := (&supportlog.LogRequest{Status: supportlog.StatusRequested}).
		Apply(&supportlog.Report{Status: supportlog.StatusRequested}, 0)
	require.Error(t, invalidStatus)
	cases := map[string]struct {
		path   string
		body   string
		err    error
		status int
	}{
		"an unknown field":       {path: path, body: `{"status":"sent","extra":1}`, status: http.StatusBadRequest},
		"not JSON":               {path: path, body: `status=sent`, status: http.StatusBadRequest},
		"an id of another kind":  {path: "/managed/v1/log-requests/usr_01JTESTTESTTESTTESTTESTTEST", body: `{"status":"sent"}`, status: http.StatusNotFound},
		"no such request":        {path: path, body: `{"status":"sent"}`, err: supportlog.ErrRequestNotFound, status: http.StatusNotFound},
		"already finished":       {path: path, body: `{"status":"sent"}`, err: supportlog.ErrFinished, status: http.StatusConflict},
		"another bundle":         {path: path, body: `{"status":"sent","bundleId":"b"}`, err: supportlog.ErrBundleMismatch, status: http.StatusConflict},
		"changed meanwhile":      {path: path, body: `{"status":"collecting"}`, err: supportlog.ErrChangedMeanwhile, status: http.StatusConflict},
		"a status it cannot set": {path: path, body: `{"status":"requested"}`, err: invalidStatus, status: http.StatusBadRequest},
	}
	for name, tc := range cases {
		logs := &fakeSupportLogs{err: tc.err}
		rec := serveLogs(t, logs, &fakeLogPulls{}, http.MethodPut, tc.path, tc.body)
		assert.Equal(t, tc.status, rec.Code, name)
	}
}

func TestManagedLogPullAnswers200UnderALiveGrant(t *testing.T) {
	t.Parallel()

	pulls := &fakeLogPulls{}
	rec := serveLogs(t, &fakeSupportLogs{}, pulls, http.MethodPost, "/managed/v1/support/log-pulls",
		`{"email":"jane.doe@trenova.app","kind":"bundle"}`)

	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	require.Len(t, pulls.pulls, 1)
	assert.Equal(t, "jane.doe@trenova.app", pulls.pulls[0].StaffEmail)
	assert.Equal(t, supportlog.PullBundle, pulls.pulls[0].Kind)
}

func TestManagedLogPullRefusalsLookTheSame(t *testing.T) {
	t.Parallel()

	var bodies []string
	for _, reason := range []supportaccess.RefusalReason{
		supportaccess.RefusalNoGrant,
		supportaccess.RefusalStaffEmail,
		supportaccess.RefusalNotConfigured,
	} {
		pulls := &fakeLogPulls{err: &supportaccess.RefusalError{Reason: reason, Detail: "for the logs only"}}
		rec := serveLogs(t, &fakeSupportLogs{}, pulls, http.MethodPost, "/managed/v1/support/log-pulls",
			`{"email":"jane.doe@trenova.app","kind":"lines"}`)
		assert.Equal(t, http.StatusForbidden, rec.Code, string(reason))
		assert.NotContains(t, rec.Body.String(), string(reason))
		assert.NotContains(t, rec.Body.String(), "for the logs only")
		bodies = append(bodies, rec.Body.String())
	}
	assert.Equal(t, bodies[0], bodies[1])
	assert.Equal(t, bodies[0], bodies[2])
}

func TestManagedLogPullRefusesABodyItCannotRead(t *testing.T) {
	t.Parallel()

	for name, body := range map[string]string{
		"an unknown kind":  `{"email":"jane.doe@trenova.app","kind":"everything"}`,
		"no email":         `{"kind":"lines"}`,
		"an unknown field": `{"email":"jane.doe@trenova.app","kind":"lines","org":"x"}`,
	} {
		pulls := &fakeLogPulls{}
		rec := serveLogs(t, &fakeSupportLogs{}, pulls, http.MethodPost, "/managed/v1/support/log-pulls", body)
		assert.Equal(t, http.StatusBadRequest, rec.Code, name)
		assert.Empty(t, pulls.pulls, name)
	}
}
