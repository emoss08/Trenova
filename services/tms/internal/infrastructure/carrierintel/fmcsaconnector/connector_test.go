package fmcsaconnector

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

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/shared/fmcsa"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testWebKey = "qc_web_key_0123456789SECRET"
	basePath   = "/qc/services"
)

type callLog struct {
	mu    sync.Mutex
	calls []services.CarrierIntelCall
}

func (l *callLog) record(_ context.Context, call services.CarrierIntelCall) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.calls = append(l.calls, call)
}

func (l *callLog) all() []services.CarrierIntelCall {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]services.CarrierIntelCall(nil), l.calls...)
}

type keyRecordingLimiter struct {
	mu   sync.Mutex
	keys []string
}

func (l *keyRecordingLimiter) Acquire(_ context.Context, bucket restx.Bucket) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.keys = append(l.keys, bucket.Key)
	return nil
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

func compositeRoutes() map[string]string {
	return map[string]string{
		basePath + "/carriers/818175":                          "carrier.json",
		basePath + "/carriers/818175/basics":                   "basics.json",
		basePath + "/carriers/818175/cargo-carried":            "cargo_carried.json",
		basePath + "/carriers/818175/operation-classification": "operation_classification.json",
		basePath + "/carriers/818175/oos":                      "oos.json",
		basePath + "/carriers/818175/docket-numbers":           "docket_numbers.json",
		basePath + "/carriers/818175/authority":                "authority.json",
		basePath + "/carriers/docket-number/277621":            "carriers_list.json",
		basePath + "/carriers/name/sandbox":                    "carriers_list.json",
	}
}

func routeHandler(t *testing.T, overrides map[string]http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	routes := compositeRoutes()
	return func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, testWebKey, r.URL.Query().Get("webKey"))
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

func testConnector() *Connector {
	connector := New(&config.Config{})
	connector.extraOptions = []fmcsa.Option{fmcsa.WithRetry(restx.RetryConfig{})}
	return connector
}

type boundClient struct {
	client *Client
	log    *callLog
}

func bindTestClient(t *testing.T, limiter restx.Limiter, handler http.HandlerFunc) boundClient {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	log := &callLog{}
	bound, err := testConnector().Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{
			integration.ConfigKeyCarrierIntelWebKey:  testWebKey,
			integration.ConfigKeyCarrierIntelBaseURL: server.URL + basePath + "/",
		},
		Limiter:  limiter,
		Recorder: log.record,
	})
	require.NoError(t, err)

	client, ok := bound.(*Client)
	require.True(t, ok)
	return boundClient{client: client, log: log}
}

func TestConnector_DescriptorAndPriceBook(t *testing.T) {
	t.Parallel()

	connector := New(&config.Config{})
	assert.Equal(t, integration.TypeFMCSAQCMobile, connector.IntegrationType())

	descriptor := connector.Descriptor()
	assert.True(t, descriptor.Capabilities.Has(carrierintel.CapabilityLookupFMCSA))
	assert.True(t, descriptor.Capabilities.Has(carrierintel.CapabilitySnapshotMonitoring))
	assert.False(t, descriptor.Capabilities.Has(carrierintel.CapabilityNativeMonitoring))
	assert.False(t, descriptor.Capabilities.Has(carrierintel.CapabilityLookupFull))
	assert.Equal(
		t,
		carrierintel.EnrollmentModeSnapshotDiff,
		descriptor.Capabilities.MonitoringMode(),
	)
	assert.NotContains(t, descriptor.Sections, carrierintel.SectionNetwork)

	prices := connector.PriceBook()
	assert.Empty(t, prices)
	assert.Equal(
		t,
		carrierintel.BillingModelFree,
		prices.Price(carrierintel.EndpointComposite).Model,
	)
}

func TestConnector_BindAndClientInterfaces(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, nil, routeHandler(t, nil))
	var client services.CarrierIntelClient = bound.client
	_, isMonitor := client.(services.CarrierIntelNativeMonitor)
	_, isEquipment := client.(services.CarrierIntelEquipmentLookup)
	_, isSearcher := client.(services.CarrierIntelSearcher)
	assert.False(t, isMonitor)
	assert.False(t, isEquipment)
	assert.True(t, isSearcher)
	assert.False(t, client.IsSandbox())
	assert.Equal(t, integration.TypeFMCSAQCMobile, client.Provider())
}

func TestConnector_BindValidation(t *testing.T) {
	t.Parallel()

	_, err := New(&config.Config{}).Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{},
	})
	require.Error(t, err)
	assert.Equal(t, "FMCSA QCMobile web key is required", err.Error())

	_, err = New(&config.Config{}).Bind(&services.CarrierIntelBindParams{
		Config: map[string]string{
			integration.ConfigKeyCarrierIntelWebKey:  testWebKey,
			integration.ConfigKeyCarrierIntelBaseURL: "http://mobile.fmcsa.dot.gov/qc/services",
		},
	})
	require.Error(t, err)
	assert.Equal(t, "Base URL must use HTTPS", err.Error())
}

func TestConnector_TestConnection(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler http.HandlerFunc
		message string
	}{
		{
			name: "not found accepted",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, fixture(t, "not_found.json"))
			},
		},
		{
			name: "rejected key",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(
					w,
					http.StatusOK,
					[]byte(`{"content":"Webkey not found or not authorized"}`),
				)
			},
			message: "FMCSA rejected the web key",
		},
		{
			name: "unavailable",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusInternalServerError, []byte(`{}`))
			},
			message: "Could not reach FMCSA QCMobile",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					assert.Equal(t, basePath+"/carriers/"+testDOTNumber, r.URL.Path)
					tc.handler(w, r)
				}),
			)
			t.Cleanup(server.Close)

			err := testConnector().TestConnection(t.Context(), map[string]string{
				integration.ConfigKeyCarrierIntelWebKey:  testWebKey,
				integration.ConfigKeyCarrierIntelBaseURL: server.URL + basePath,
			})
			if tc.message == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Equal(t, tc.message, err.Error())
			assert.NotContains(t, err.Error(), "SECRET")
		})
	}
}

func TestClient_LookupByDOT(t *testing.T) {
	t.Parallel()

	limiter := &keyRecordingLimiter{}
	bound := bindTestClient(t, limiter, routeHandler(t, nil))

	result, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{DOTNumber: "818175"},
		Depth:      carrierintel.LookupDepthFull,
	})
	require.NoError(t, err)
	assert.Equal(t, carrierintel.EndpointComposite, result.Endpoint)
	assert.Equal(t, carrierintel.LookupDepthFMCSA, result.Depth)
	assert.Equal(t, "818175-MC277621", result.ProviderRef)
	assert.Contains(t, string(result.Raw), `"basics"`)
	assert.Nil(t, result.SourceAsOf)

	calls := bound.log.all()
	require.Len(t, calls, 1)
	assert.Equal(t, carrierintel.EndpointComposite, calls[0].Endpoint)
	assert.Equal(t, "818175", calls[0].DOTNumber)
	assert.True(t, calls[0].Found)
	assert.Equal(t, 1, calls[0].Units)
	assert.Equal(t, http.StatusOK, calls[0].StatusCode)

	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	require.NotEmpty(t, limiter.keys)
	for _, key := range limiter.keys {
		assert.True(t, strings.HasPrefix(key, "fmcsa:"))
		assert.True(t, strings.HasSuffix(key, ":qcmobile"))
		assert.NotContains(t, key, "SECRET")
	}
}

func TestClient_LookupByDocketResolvesDOT(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, nil, routeHandler(t, nil))

	result, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{DocketNumber: "MC-277621"},
	})
	require.NoError(t, err)
	assert.Equal(t, "818175", result.Profile.DOTNumber())

	calls := bound.log.all()
	require.Len(t, calls, 1)
	assert.Equal(t, "818175", calls[0].DOTNumber)
}

func TestClient_LookupRejectsUnsupportedIdentifiers(t *testing.T) {
	t.Parallel()

	var hits atomic.Int32
	bound := bindTestClient(t, nil, func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	_, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{Company: "Sandbox"},
	})
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorUnsupported))

	_, err = bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{})
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorInvalidRequest))
	assert.Equal(t, int32(0), hits.Load())
	assert.Empty(t, bound.log.all())
}

func TestClient_LookupErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		handler http.HandlerFunc
		kind    services.CarrierIntelErrorKind
		retry   time.Duration
	}{
		{
			name: "not found",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusOK, fixture(t, "not_found.json"))
			},
			kind: services.CarrierIntelErrorNotFound,
		},
		{
			name: "unauthorized",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusUnauthorized, []byte(`{"content":"Webkey not authorized"}`))
			},
			kind: services.CarrierIntelErrorUnauthorized,
		},
		{
			name: "rate limited",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Retry-After", "30")
				writeJSON(w, http.StatusTooManyRequests, []byte(`{}`))
			},
			kind:  services.CarrierIntelErrorRateLimited,
			retry: 30 * time.Second,
		},
		{
			name: "server error",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeJSON(w, http.StatusBadGateway, []byte(`{}`))
			},
			kind: services.CarrierIntelErrorUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			bound := bindTestClient(t, nil, routeHandler(t, map[string]http.HandlerFunc{
				basePath + "/carriers/818175": tc.handler,
			}))

			_, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
				Identifier: services.CarrierIntelIdentifier{DOTNumber: "818175"},
			})
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "SECRET")

			var providerErr *services.CarrierIntelProviderError
			require.ErrorAs(t, err, &providerErr)
			assert.Equal(t, tc.kind, providerErr.Kind)
			assert.Equal(t, tc.retry, providerErr.RetryAfter)

			calls := bound.log.all()
			require.Len(t, calls, 1)
			assert.False(t, calls[0].Found)
		})
	}
}

func TestClient_SearchAndAutocomplete(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, nil, routeHandler(t, map[string]http.HandlerFunc{
		basePath + "/carriers/name/nobody": func(w http.ResponseWriter, _ *http.Request) {
			writeJSON(w, http.StatusOK, []byte(`{"content":[]}`))
		},
	}))

	result, err := bound.client.Search(t.Context(), &services.CarrierIntelSearchRequest{
		CompanyName: "sandbox",
		Limit:       25,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	assert.Equal(t, 2, result.Total)
	assert.Equal(t, "818175", result.Items[0].ProviderRef)
	assert.Equal(t, "ACTIVE", result.Items[0].Profile.Identity.USDOTStatus)
	assert.Equal(t, "INACTIVE", result.Items[1].Profile.Identity.USDOTStatus)
	assert.Nil(t, result.Items[0].Profile.Safety)

	empty, err := bound.client.Search(
		t.Context(),
		&services.CarrierIntelSearchRequest{Query: "nobody"},
	)
	require.NoError(t, err)
	assert.Empty(t, empty.Items)

	_, err = bound.client.Search(t.Context(), &services.CarrierIntelSearchRequest{EIN: "364123456"})
	assert.True(t, services.IsCarrierIntelErrorKind(err, services.CarrierIntelErrorUnsupported))

	suggestions, err := bound.client.Autocomplete(t.Context(), "sandbox", 5)
	require.NoError(t, err)
	require.Len(t, suggestions, 2)
	assert.Equal(t, "SANDBOX & SONS/TRUCKING", suggestions[1].LegalName)

	calls := bound.log.all()
	require.Len(t, calls, 3)
	assert.Equal(t, carrierintel.EndpointSearch, calls[0].Endpoint)
	assert.Equal(t, carrierintel.EndpointSearch, calls[1].Endpoint)
	assert.False(t, calls[1].Found)
	assert.Equal(t, carrierintel.EndpointAutocomplete, calls[2].Endpoint)
}

func TestNormalizeComposite(t *testing.T) {
	t.Parallel()

	bound := bindTestClient(t, nil, routeHandler(t, nil))
	result, err := bound.client.Lookup(t.Context(), &services.CarrierIntelLookupRequest{
		Identifier: services.CarrierIntelIdentifier{DOTNumber: "818175"},
	})
	require.NoError(t, err)
	profile := result.Profile

	assert.Equal(t, []carrierintel.Section{
		carrierintel.SectionIdentity,
		carrierintel.SectionAuthority,
		carrierintel.SectionInsurance,
		carrierintel.SectionSafety,
		carrierintel.SectionBasics,
		carrierintel.SectionInspections,
		carrierintel.SectionCrashes,
		carrierintel.SectionFleet,
		carrierintel.SectionOperations,
	}, profile.Coverage)

	identity := profile.Identity
	require.NotNil(t, identity)
	assert.Equal(t, "818175", identity.DOTNumber)
	assert.Equal(t, "MC", identity.DocketPrefix)
	assert.Equal(t, "277621", identity.DocketNumber)
	assert.Equal(t, "ACTIVE", identity.USDOTStatus)
	assert.Equal(t, "364123456", identity.EIN)
	assert.Empty(t, identity.DBAName)
	assert.Equal(t, "Interstate", identity.CarrierOperation)
	require.NotNil(t, identity.PhysicalAddress)
	assert.Equal(t, "100 MAIN ST", identity.PhysicalAddress.Line1)
	assert.Equal(t, "60601", identity.PhysicalAddress.PostalCode)

	authority := profile.Authority
	require.NotNil(t, authority)
	assert.Equal(t, carrierintel.AuthorityStatusActive, authority.Common.Status)
	assert.Equal(t, carrierintel.AuthorityStatusInactive, authority.Contract.Status)
	assert.Equal(t, carrierintel.AuthorityStatusNone, authority.Broker.Status)

	insurance := profile.Insurance
	require.NotNil(t, insurance)
	assert.Equal(t, "750000", insurance.BIPDOnFile.String())
	assert.Equal(t, "750000", insurance.BIPDRequired.String())
	assert.Equal(t, "100000", insurance.CargoOnFile.String())
	assert.Nil(t, insurance.CargoRequired)
	assert.True(t, insurance.BondOnFile.IsZero())
	require.NotNil(t, insurance.BondRequired)
	assert.True(t, insurance.BondRequired.IsZero())

	safety := profile.Safety
	require.NotNil(t, safety)
	assert.Equal(t, carrierintel.SafetyRatingSatisfactory, safety.Rating)
	require.NotNil(t, safety.RatingDate)
	assert.Equal(
		t,
		time.Date(2019, time.November, 4, 0, 0, 0, 0, time.UTC).Unix(),
		*safety.RatingDate,
	)
	assert.Nil(t, safety.ISSValue)
	require.NotNil(t, safety.OutOfServiceOrder)
	assert.False(t, *safety.OutOfServiceOrder)
	assert.Equal(t, "C", safety.LatestReviewType)

	require.Len(t, profile.Basics, 2)
	hos := profile.Basic(worker.BasicHOSCompliance)
	require.NotNil(t, hos)
	require.NotNil(t, hos.Percentile)
	assert.InDelta(t, 72.4, *hos.Percentile, 0.0001)
	require.NotNil(t, hos.Threshold)
	assert.InDelta(t, 65, *hos.Threshold, 0.0001)
	assert.True(t, hos.Alert)
	assert.True(t, hos.RoadsideAlert)
	assert.False(t, hos.ACIndicator)

	unsafe := profile.Basic(worker.BasicUnsafeDriving)
	require.NotNil(t, unsafe)
	assert.Nil(t, unsafe.Percentile)
	assert.False(t, unsafe.Alert)
	require.NotNil(t, unsafe.Measure)
	assert.InDelta(t, 1.23, *unsafe.Measure, 0.0001)

	inspections := profile.Inspections
	require.NotNil(t, inspections)
	assert.Equal(t, 120, *inspections.Driver)
	assert.Equal(t, 17, *inspections.VehicleOOS)
	assert.InDelta(t, 21.25, *inspections.VehicleOOSRate, 0.0001)
	assert.InDelta(t, 20.72, *inspections.NationalVehicleOOS, 0.0001)
	assert.Nil(t, inspections.Total)

	require.NotNil(t, profile.Crashes)
	assert.Equal(t, 4, *profile.Crashes.Total)
	assert.Equal(t, 1, *profile.Crashes.Injury)
	require.NotNil(t, profile.Fleet)
	assert.Equal(t, 85, *profile.Fleet.PowerUnits)
	assert.Equal(t, 92, *profile.Fleet.Drivers)

	require.NotNil(t, profile.Operations)
	assert.Equal(
		t,
		[]string{"General Freight", "Refrigerated Food"},
		profile.Operations.CargoCarried,
	)
	assert.Equal(t, []string{"Authorized For Hire"}, profile.Operations.Classification)
}

func TestOutOfServiceOrder(t *testing.T) {
	t.Parallel()

	blocked, err := fmcsa.DecodeCarrier(
		[]byte(`{"dotNumber": 1, "allowedToOperate": "N", "statusCode": "I"}`),
	)
	require.NoError(t, err)
	allowed, err := fmcsa.DecodeCarrier([]byte(`{"dotNumber": 1, "allowedToOperate": "Y"}`))
	require.NoError(t, err)

	active := []fmcsa.OOSEntry{{Status: nil}}
	noEntries := &carrierExtras{complete: true}

	order := outOfServiceOrder(blocked, &carrierExtras{complete: true, oos: active})
	require.NotNil(t, order)
	assert.True(t, *order)

	order = outOfServiceOrder(allowed, &carrierExtras{complete: true, oos: active})
	require.NotNil(t, order)
	assert.False(t, *order)

	order = outOfServiceOrder(blocked, noEntries)
	require.NotNil(t, order)
	assert.False(t, *order)

	assert.Nil(t, outOfServiceOrder(blocked, &carrierExtras{}))
}

func TestCSABasicFromLabel(t *testing.T) {
	t.Parallel()

	cases := map[string]worker.CSABasic{
		"Unsafe Driving":                 worker.BasicUnsafeDriving,
		"HOS Compliance":                 worker.BasicHOSCompliance,
		"Hours-of-Service Compliance":    worker.BasicHOSCompliance,
		"Driver Fitness":                 worker.BasicDriverFitness,
		"Drugs/Alcohol":                  worker.BasicControlledSubstances,
		"Controlled Substances/Alcohol":  worker.BasicControlledSubstances,
		"Vehicle Maint.":                 worker.BasicVehicleMaintenance,
		"HM Compliance":                  worker.BasicHazmatCompliance,
		"Hazardous Materials Compliance": worker.BasicHazmatCompliance,
		"Crash Indicator":                worker.BasicCrashIndicator,
	}
	for label, want := range cases {
		got, ok := csaBasicFromLabel(label)
		assert.True(t, ok, label)
		assert.Equal(t, want, got, label)
	}

	_, ok := csaBasicFromLabel("Insurance/Other")
	assert.False(t, ok)
}
