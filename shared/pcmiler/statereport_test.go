package pcmiler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/stretchr/testify/require"
)

func TestBuildStopUsesGivenCountry(t *testing.T) {
	t.Parallel()

	stop := buildStop(Stop{
		City:    "Toronto",
		State:   "ON",
		Country: " ca ",
	}, "NA", 0, 2)

	address, ok := stop["Address"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "CA", address["Country"])
}

func TestBuildStopDefaultsCountryToUS(t *testing.T) {
	t.Parallel()

	stop := buildStop(Stop{City: "Dallas", State: "TX"}, "NA", 0, 2)

	address, ok := stop["Address"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "US", address["Country"])
}

func reportTypesForRoute(t *testing.T, opts RouteOptions) []map[string]any {
	t.Helper()

	payload := buildRouteReportsPayload([]RouteRequest{
		{
			RouteID: "route-1",
			Stops: []Stop{
				{City: "Philadelphia", State: "PA"},
				{City: "Pittsburgh", State: "PA"},
			},
			Options: opts,
		},
	})

	routes, ok := payload["ReportRoutes"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, routes, 1)

	reportTypes, ok := routes[0]["ReportTypes"].([]map[string]any)
	require.True(t, ok)
	return reportTypes
}

func TestBuildRouteReportsPayloadRequestsStateReportWhenEnabled(t *testing.T) {
	t.Parallel()

	reportTypes := reportTypesForRoute(t, RouteOptions{StateReport: true})

	require.Len(t, reportTypes, 2)
	require.Equal(t, mileageReportTypeValue, reportTypes[0]["__type"])
	require.Equal(t, false, reportTypes[0]["TimeInSeconds"])
	require.Equal(t, stateReportTypeValue, reportTypes[1]["__type"])
}

func TestBuildRouteReportsPayloadOmitsStateReportByDefault(t *testing.T) {
	t.Parallel()

	reportTypes := reportTypesForRoute(t, RouteOptions{})

	require.Len(t, reportTypes, 1)
	require.Equal(t, mileageReportTypeValue, reportTypes[0]["__type"])
}

func TestParseStCntry(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		value       string
		wantCountry string
		wantCode    string
		wantOK      bool
	}{
		{name: "country and code", value: "US_TX", wantCountry: "US", wantCode: "TX", wantOK: true},
		{name: "canadian province", value: "CA_ON", wantCountry: "CA", wantCode: "ON", wantOK: true},
		{name: "bare code defaults to US", value: "TX", wantCountry: "US", wantCode: "TX", wantOK: true},
		{name: "lowercase and padded", value: " us_nj ", wantCountry: "US", wantCode: "NJ", wantOK: true},
		{name: "blank", value: "   ", wantOK: false},
		{name: "total line upper", value: "TOTAL", wantOK: false},
		{name: "total line mixed", value: "Total", wantOK: false},
		{name: "missing code", value: "US_", wantOK: false},
		{name: "missing country", value: "_TX", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			country, code, ok := parseStCntry(tt.value)

			require.Equal(t, tt.wantOK, ok)
			require.Equal(t, tt.wantCountry, country)
			require.Equal(t, tt.wantCode, code)
		})
	}
}

func TestFlexFloatUnmarshalAcceptsNumbersStringsAndNull(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    float64
		wantErr bool
	}{
		{name: "number", input: `12.5`, want: 12.5},
		{name: "integer", input: `7`, want: 7},
		{name: "quoted number", input: `"88.25"`, want: 88.25},
		{name: "quoted with thousands separator", input: `"1,043.9"`, want: 1043.9},
		{name: "quoted padded", input: `"  3.5 "`, want: 3.5},
		{name: "null", input: `null`, want: 0},
		{name: "empty string", input: `""`, want: 0},
		{name: "garbage string", input: `"abc"`, wantErr: true},
		{name: "object", input: `{}`, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var value flexFloat
			err := value.UnmarshalJSON([]byte(tt.input))

			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.InDelta(t, tt.want, float64(value), 0.0001)
		})
	}
}

func loadStateReportFixture(t *testing.T) []routeReport {
	t.Helper()

	raw, err := os.ReadFile(filepath.Join("testdata", "state_report_response.json"))
	require.NoError(t, err)

	var reports []routeReport
	require.NoError(t, sonic.Unmarshal(raw, &reports))
	return reports
}

func TestParseMileageResponseGroupsStateReportByRoute_ShapeAssumedFromTrimbleDocsNotLiveConfirmed(
	t *testing.T,
) {
	t.Parallel()

	results := parseMileageResponse(loadStateReportFixture(t))

	require.Len(t, results, 2)

	require.Equal(t, "route-2", results[0].RouteID)
	require.InDelta(t, 1344.75, results[0].Distance, 0.0001)
	require.Equal(t, []string{"Border crossing on route"}, results[0].Warnings)
	require.Equal(t, []JurisdictionDistance{
		{Country: "US", Code: "PA", Distance: 212.7, Toll: 150.4, Ferry: 0},
		{Country: "US", Code: "NY", Distance: 88.15, Toll: 0, Ferry: 0},
		{Country: "CA", Code: "ON", Distance: 1043.9, Toll: 0, Ferry: 2.5},
	}, results[0].JurisdictionDistances)

	require.Equal(t, "route-1", results[1].RouteID)
	require.InDelta(t, 97.3, results[1].Distance, 0.0001)
	require.Empty(t, results[1].Warnings)
	require.Equal(t, []JurisdictionDistance{
		{Country: "US", Code: "NJ", Distance: 40.1, Toll: 12.0, Ferry: 0},
		{Country: "US", Code: "PA", Distance: 57.2, Toll: 0, Ferry: 0},
	}, results[1].JurisdictionDistances)
}

func TestParseMileageResponseStateReportWithoutMileageReportYieldsZeroDistance(t *testing.T) {
	t.Parallel()

	results := parseMileageResponse([]routeReport{
		{
			Type:    "StateReport:http://pcmiler.alk.com/APIs/v1.0",
			RouteID: "route-3",
			StateReportLines: []stateReportLine{
				{StCntry: "US_OK", Total: 120},
			},
		},
	})

	require.Len(t, results, 1)
	require.Equal(t, "route-3", results[0].RouteID)
	require.Zero(t, results[0].Distance)
	require.Nil(t, results[0].Warnings)
	require.Equal(t, []JurisdictionDistance{
		{Country: "US", Code: "OK", Distance: 120},
	}, results[0].JurisdictionDistances)
}

func TestParseMileageResponseMileageReportOnlyLeavesJurisdictionsEmpty(t *testing.T) {
	t.Parallel()

	results := parseMileageResponse([]routeReport{
		{
			Type:        "MileageReport:http://pcmiler.alk.com/APIs/v1.0",
			RouteID:     "route-4",
			ReportLines: []reportLine{{TMiles: "10"}},
		},
	})

	require.Len(t, results, 1)
	require.InDelta(t, 10.0, results[0].Distance, 0.0001)
	require.Empty(t, results[0].JurisdictionDistances)
}

func TestMileageSurfacesStateReportJurisdictions_ShapeAssumedFromTrimbleDocsNotLiveConfirmed(
	t *testing.T,
) {
	t.Parallel()

	var requestedReportTypes []string
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
			return
		}

		var payload struct {
			ReportRoutes []struct {
				RouteID     string `json:"RouteId"`
				ReportTypes []struct {
					Type string `json:"__type"`
				} `json:"ReportTypes"`
			} `json:"ReportRoutes"`
		}
		if err = sonic.Unmarshal(body, &payload); err != nil {
			t.Errorf("decode request payload: %v", err)
			return
		}
		if len(payload.ReportRoutes) != 1 {
			t.Errorf("expected one route, got %d", len(payload.ReportRoutes))
			return
		}
		for _, reportType := range payload.ReportRoutes[0].ReportTypes {
			requestedReportTypes = append(requestedReportTypes, reportType.Type)
		}

		raw, err := os.ReadFile(filepath.Join("testdata", "state_report_response.json"))
		if err != nil {
			t.Errorf("read fixture: %v", err)
			return
		}
		_, _ = w.Write(raw)
	})

	results, err := client.Mileage(t.Context(), []RouteRequest{
		{
			RouteID: "route-1",
			Stops: []Stop{
				{City: "Newark", State: "NJ"},
				{City: "Philadelphia", State: "PA"},
			},
			Options: RouteOptions{StateReport: true},
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{mileageReportTypeValue, stateReportTypeValue}, requestedReportTypes)

	require.Len(t, results, 2)
	require.Equal(t, "route-1", results[1].RouteID)
	require.InDelta(t, 97.3, results[1].Distance, 0.0001)
	require.Equal(t, []JurisdictionDistance{
		{Country: "US", Code: "NJ", Distance: 40.1, Toll: 12.0},
		{Country: "US", Code: "PA", Distance: 57.2},
	}, results[1].JurisdictionDistances)
}
