package integrationservice

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPCMilerConnectionTesterUsesRouteReportsForCurrentDataVersion(t *testing.T) {
	t.Parallel()

	var calledPCMVersion bool
	var calledRouteReports bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Service.svc/pcmversion":
			calledPCMVersion = true
			http.Error(w, `"This type of request is not allowed for Trial Keys."`, http.StatusMethodNotAllowed)
		case "/Service.svc/route/routeReports":
			calledRouteReports = true
			require.Equal(t, "Current", r.URL.Query().Get("dataVersion"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"__type": "MileageReport:http://pcmiler.alk.com/APIs/v1.0",
					"RouteID": "connection-test",
					"ReportLines": [{"TMiles": "42.1"}]
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := (&pcmilerConnectionTester{}).Test(t.Context(), map[string]string{
		"apiKey":      "valid-trial-key",
		"baseUrl":     server.URL + "/Service.svc",
		"dataVersion": "Current",
	})

	require.NoError(t, err)
	require.True(t, calledRouteReports)
	require.False(t, calledPCMVersion)
}

func TestPCMilerConnectionTesterIgnoresRoutingPolicyFields(t *testing.T) {
	t.Parallel()

	var calledPCMVersion bool
	var calledRouteReports bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/Service.svc/pcmversion":
			calledPCMVersion = true
			http.Error(w, `"This type of request is not allowed for Trial Keys."`, http.StatusMethodNotAllowed)
		case "/Service.svc/route/routeReports":
			calledRouteReports = true
			require.Equal(t, "Current", r.URL.Query().Get("dataVersion"))
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[
				{
					"__type": "MileageReport:http://pcmiler.alk.com/APIs/v1.0",
					"RouteID": "connection-test",
					"ReportLines": [{"TMiles": "42.1"}]
				}
			]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	err := (&pcmilerConnectionTester{}).Test(t.Context(), map[string]string{
		"apiKey":      "valid-trial-key",
		"baseUrl":     server.URL + "/Service.svc",
		"dataVersion": "PCM36",
	})

	require.NoError(t, err)
	require.True(t, calledRouteReports)
	require.False(t, calledPCMVersion)
}
