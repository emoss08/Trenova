package fmcsa_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/fmcsa"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWebKey = "qc_web_key_0123456789SECRET"
	basePath   = "/qc/services"
)

type recordingLimiter struct {
	mu   sync.Mutex
	keys []string
}

func (l *recordingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = append(l.keys, bucket.Key)
	return nil
}

func (l *recordingLimiter) recorded() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.keys...)
}

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func writeJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}

func newTestClient(
	t *testing.T,
	handler http.HandlerFunc,
	opts ...fmcsa.Option,
) *fmcsa.Client {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testWebKey, r.URL.Query().Get("webKey"))
		handler(w, r)
	}))
	t.Cleanup(server.Close)

	options := append([]fmcsa.Option{
		fmcsa.WithBaseURL(server.URL + basePath),
		fmcsa.WithTimeout(5 * time.Second),
		fmcsa.WithRetry(restx.RetryConfig{}),
	}, opts...)

	client, err := fmcsa.New(testWebKey, options...)
	require.NoError(t, err)
	return client
}

func assertNoWebKey(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	assert.NotContains(t, err.Error(), testWebKey)
	assert.NotContains(t, err.Error(), "0123456789SECRET")
}

func compositeHandler(t *testing.T, overrides map[string]http.HandlerFunc) http.HandlerFunc {
	t.Helper()

	routes := map[string]string{
		basePath + "/carriers/818175":                          "carrier.json",
		basePath + "/carriers/818175/basics":                   "basics.json",
		basePath + "/carriers/818175/cargo-carried":            "cargo_carried.json",
		basePath + "/carriers/818175/operation-classification": "operation_classification.json",
		basePath + "/carriers/818175/oos":                      "oos.json",
		basePath + "/carriers/818175/docket-numbers":           "docket_numbers.json",
		basePath + "/carriers/818175/authority":                "authority.json",
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if override, ok := overrides[r.URL.Path]; ok {
			override(w, r)
			return
		}
		name, ok := routes[r.URL.Path]
		if !ok {
			writeJSON(w, http.StatusNotFound, []byte(`{"content":"Not found"}`))
			return
		}
		writeJSON(w, http.StatusOK, fixture(t, name))
	}
}

func TestNewRequiresWebKey(t *testing.T) {
	t.Parallel()

	_, err := fmcsa.New(" ")
	require.ErrorIs(t, err, fmcsa.ErrWebKeyRequired)
}

func TestCarrierByDOT(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, basePath+"/carriers/818175", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		writeJSON(w, http.StatusOK, fixture(t, "carrier.json"))
	})

	carrier, err := client.CarrierByDOT(t.Context(), " 818175 ")
	require.NoError(t, err)

	assert.Equal(t, "818175", carrier.DOTNumber.Value())
	assert.Equal(t, "SANDBOX FREIGHT LINES INC", carrier.LegalName.Value())
	assert.Nil(t, carrier.DBAName)
	assert.True(t, carrier.AllowedToOperate.Value())
	assert.True(t, carrier.BIPDInsuranceRequired.Value())
	assert.False(t, carrier.BondInsuranceRequired.Value())
	assert.InDelta(t, 750, carrier.BIPDInsuranceOnFile.Value(), 1e-9)
	assert.Equal(t, "A", carrier.CarrierOperationCode.Value())
	assert.Equal(t, "Interstate", carrier.CarrierOperationDesc.Value())
	assert.Equal(t, "364123456", carrier.EIN.Value())
	assert.InDelta(t, 6.67, carrier.DriverOOSRateNationalAverage.Value(), 1e-9)
	assert.Nil(t, carrier.ISSScore)
	assert.Nil(t, carrier.OOSDate)
	assert.False(t, carrier.MCS150Outdated.Value())
	assert.Equal(t, "S", carrier.SafetyRating.Value())
	assert.Equal(
		t,
		time.Date(2019, time.November, 4, 0, 0, 0, 0, time.UTC).Unix(),
		*carrier.SafetyRatingDate.Unix(),
	)
	assert.Equal(t, int64(85), carrier.TotalPowerUnits.Value())
	assert.Equal(t, "2009-2010", carrier.OOSRateNationalAverageYear.Value())
	assert.Contains(t, string(carrier.Raw), `"legalName"`)
	assert.NotContains(t, string(carrier.Raw), `"_links"`)

	bipd := fmcsa.InsuranceDollars(carrier.BIPDInsuranceOnFile)
	require.NotNil(t, bipd)
	assert.Equal(t, "750000", bipd.String())
	assert.Equal(t, "100000", fmcsa.InsuranceDollars(carrier.CargoInsuranceOnFile).String())
	assert.Nil(t, fmcsa.InsuranceDollars(nil))
}

func TestCarrierByDOTValidation(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	for _, dot := range []string{"", "81A", "../1", "1/2"} {
		_, err := client.CarrierByDOT(t.Context(), dot)
		require.ErrorIs(t, err, fmcsa.ErrInvalidArgument, dot)
	}
	_, err := client.CarriersByName(t.Context(), "  ", 0, 0)
	require.ErrorIs(t, err, fmcsa.ErrInvalidArgument)
	_, err = client.CarriersByDocket(t.Context(), "MC-")
	require.ErrorIs(t, err, fmcsa.ErrInvalidArgument)
	assert.Equal(t, int32(0), calls.Load())
}

func TestCarrierByDOTNotFound(t *testing.T) {
	t.Parallel()

	tests := map[string]func(w http.ResponseWriter){
		"null content": func(w http.ResponseWriter) {
			writeJSON(w, http.StatusOK, fixture(t, "not_found.json"))
		},
		"empty array": func(w http.ResponseWriter) {
			writeJSON(w, http.StatusOK, []byte(`{"content":[]}`))
		},
		"status 404": func(w http.ResponseWriter) {
			writeJSON(w, http.StatusNotFound, []byte(`{"content":"No carrier found"}`))
		},
	}

	for name, respond := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
				respond(w)
			})

			carrier, err := client.CarrierByDOT(t.Context(), "1")
			assert.Nil(t, carrier)
			assert.True(t, fmcsa.IsNotFound(err))
			assertNoWebKey(t, err)
		})
	}
}

func TestCarriersByDocket(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, basePath+"/carriers/docket-number/277621", r.URL.Path)
		writeJSON(w, http.StatusOK, fixture(t, "carriers_list.json"))
	})

	carriers, err := client.CarriersByDocket(t.Context(), "MC-277621")
	require.NoError(t, err)
	require.Len(t, carriers, 2)
	assert.Equal(t, "818175", carriers[0].DOTNumber.Value())
	assert.False(t, carriers[1].AllowedToOperate.Value())
}

func TestCarriersByNameEscapesPath(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(
			t,
			basePath+"/carriers/name/SANDBOX%20&%20SONS%2FTRUCKING",
			r.URL.EscapedPath(),
		)
		assert.Equal(t, "10", r.URL.Query().Get("start"))
		assert.Equal(t, "25", r.URL.Query().Get("size"))
		writeJSON(w, http.StatusOK, fixture(t, "carriers_list.json"))
	})

	carriers, err := client.CarriersByName(t.Context(), "SANDBOX & SONS/TRUCKING", 10, 25)
	require.NoError(t, err)
	require.Len(t, carriers, 2)
	assert.Equal(t, "SANDBOX & SONS/TRUCKING", carriers[1].LegalName.Value())
}

func TestBasics(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, basePath+"/carriers/818175/basics", r.URL.Path)
		writeJSON(w, http.StatusOK, fixture(t, "basics.json"))
	})

	basics, err := client.Basics(t.Context(), "818175")
	require.NoError(t, err)
	require.Len(t, basics, 2)

	hos := basics[0]
	assert.Equal(t, "111", hos.ID.Value())
	assert.InDelta(t, 72.4, hos.Percentile.Value(), 1e-9)
	assert.Equal(t, "HOS", hos.Code.Value())
	assert.Equal(t, "Hours-of-Service Compliance", hos.ShortDescription.Value())
	assert.True(t, hos.ExceededInterventionThreshold.Value())
	assert.True(t, hos.OnRoadPerformanceThresholdViolation.Value())
	assert.False(t, hos.SeriousViolationFromInvestigation12M.Value())
	assert.Equal(t, int64(21), hos.TotalViolations.Value())
	assert.Equal(
		t,
		time.Date(2026, time.August, 29, 4, 0, 0, 0, time.UTC).Unix(),
		*hos.RunDate.Unix(),
	)

	unsafe := basics[1]
	assert.Nil(t, unsafe.Percentile)
	assert.Nil(t, unsafe.ExceededInterventionThreshold)
	assert.Equal(t, "UNSAFE_DRIVING", unsafe.CodeMCMIS.Value())
	assert.Equal(t, int64(1788048000), *unsafe.RunDate.Unix())
}

func TestSubresources(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, compositeHandler(t, nil))

	cargo, err := client.CargoCarried(t.Context(), "818175")
	require.NoError(t, err)
	assert.Equal(t, []string{"General Freight", "Refrigerated Food"}, cargo)

	operations, err := client.OperationClassification(t.Context(), "818175")
	require.NoError(t, err)
	assert.Equal(t, []string{"Authorized For Hire"}, operations)

	oos, err := client.OOS(t.Context(), "818175")
	require.NoError(t, err)
	require.Len(t, oos, 2)
	assert.Equal(t, "Imminent Hazard", oos[0].Reason.Value())
	assert.Equal(t, "Rescinded", oos[0].Status.Value())
	assert.Equal(t, "Failure to pay fine", oos[1].Reason.Value())
	assert.Equal(
		t,
		time.Date(2021, time.April, 10, 0, 0, 0, 0, time.UTC).Unix(),
		*oos[1].Date.Unix(),
	)

	dockets, err := client.DocketNumbers(t.Context(), "818175")
	require.NoError(t, err)
	require.Len(t, dockets, 1)
	assert.Equal(t, "277621", dockets[0].DocketNumber.Value())
	assert.Equal(t, "MC", dockets[0].Prefix.Value())

	authorities, err := client.Authority(t.Context(), "818175")
	require.NoError(t, err)
	require.Len(t, authorities, 1)
	assert.True(t, authorities[0].AuthorizedForProperty.Value())
	assert.False(t, authorities[0].AuthorizedForBroker.Value())
	assert.Equal(t, "A", authorities[0].CommonAuthorityStatus.Value())
}

func TestComposite(t *testing.T) {
	t.Parallel()

	limiter := &recordingLimiter{}
	var requests atomic.Int32
	handler := compositeHandler(t, map[string]http.HandlerFunc{
		basePath + "/carriers/818175/oos": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusNotFound, []byte(`{"content":"No OOS records"}`))
		},
		basePath + "/carriers/818175/docket-numbers": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []byte(`{"content":[]}`))
		},
	})
	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		handler(w, r)
	}, fmcsa.WithLimiter(limiter, "fmcsa:org_1"))

	composite, err := client.Composite(t.Context(), "818175")
	require.NoError(t, err)

	assert.Equal(t, int32(7), requests.Load())
	keys := limiter.recorded()
	require.Len(t, keys, 7)
	for _, key := range keys {
		assert.Equal(t, "fmcsa:org_1:qcmobile", key)
	}

	assert.Equal(t, "SANDBOX FREIGHT LINES INC", composite.Carrier.LegalName.Value())
	assert.Len(t, composite.Basics, 2)
	assert.Equal(t, []string{"General Freight", "Refrigerated Food"}, composite.Cargo)
	assert.Equal(t, []string{"Authorized For Hire"}, composite.Operations)
	assert.Empty(t, composite.OOS)
	assert.NotNil(t, composite.OOS)
	assert.Empty(t, composite.Dockets)
	require.Len(t, composite.Authorities, 1)

	var document map[string]sonic.NoCopyRawMessage
	require.NoError(t, sonic.Unmarshal(composite.Raw, &document))
	for _, key := range []string{
		"carrier",
		"basics",
		"cargoCarried",
		"operationClassification",
		"oos",
		"docketNumbers",
		"authority",
	} {
		_, ok := document[key]
		assert.True(t, ok, key)
	}
	assert.Equal(t, "null", string(document["oos"]))
	assert.Equal(t, "null", string(document["docketNumbers"]))
	assert.Contains(t, string(document["carrier"]), "SANDBOX FREIGHT LINES INC")
	assert.Contains(t, string(document["basics"]), "Hours-of-Service Compliance")
}

func TestCompositeCarrierNotFoundSkipsSubresources(t *testing.T) {
	t.Parallel()

	var requests atomic.Int32
	client := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writeJSON(w, http.StatusOK, fixture(t, "not_found.json"))
	})

	composite, err := client.Composite(t.Context(), "818175")
	assert.Nil(t, composite)
	assert.True(t, fmcsa.IsNotFound(err))
	assert.Equal(t, int32(1), requests.Load())
}

func TestCompositePropagatesSubresourceFailure(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, compositeHandler(t, map[string]http.HandlerFunc{
		basePath + "/carriers/818175/basics": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusInternalServerError, []byte(`{"content":"boom"}`))
		},
	}))

	composite, err := client.Composite(t.Context(), "818175")
	assert.Nil(t, composite)
	require.Error(t, err)
	assert.False(t, fmcsa.IsNotFound(err))
	assert.True(t, restx.IsStatus(err, http.StatusInternalServerError))
	assertNoWebKey(t, err)
}

func TestUnauthorizedNeverLeaksWebKey(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := sonic.Marshal(map[string]string{
			"content": "Webkey " + r.URL.Query().Get("webKey") + " is not authorized",
		})
		writeJSON(w, http.StatusUnauthorized, body)
	})

	_, err := client.CarrierByDOT(t.Context(), "818175")
	assert.True(t, fmcsa.IsUnauthorized(err))
	assertNoWebKey(t, err)

	var apiErr *restx.APIError
	require.ErrorAs(t, err, &apiErr)
	assert.NotContains(t, string(apiErr.Body), testWebKey)
	assert.Contains(t, apiErr.Message, "REDACTED")
}

func TestContentStringCredentialMessage(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		body, _ := sonic.Marshal(map[string]string{
			"content": "Webkey not found: " + r.URL.Query().Get("webKey"),
		})
		writeJSON(w, http.StatusOK, body)
	})

	_, err := client.CarrierByDOT(t.Context(), "818175")
	assert.True(t, fmcsa.IsUnauthorized(err))
	assert.False(t, fmcsa.IsNotFound(err))
	assertNoWebKey(t, err)
}

func TestTransportErrorNeverLeaksWebKey(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	baseURL := server.URL + basePath
	server.Close()

	client, err := fmcsa.New(
		testWebKey,
		fmcsa.WithBaseURL(baseURL),
		fmcsa.WithRetry(restx.RetryConfig{}),
	)
	require.NoError(t, err)

	_, err = client.CarriersByName(t.Context(), "sandbox", 0, 0)
	assertNoWebKey(t, err)
	assert.False(t, strings.Contains(err.Error(), "webKey="))

	var transportErr *restx.TransportError
	require.ErrorAs(t, err, &transportErr)
}
