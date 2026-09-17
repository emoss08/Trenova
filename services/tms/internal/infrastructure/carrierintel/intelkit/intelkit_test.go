package intelkit

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/restx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errSentinelNotFound = errors.New("no record")

func testMapper() ErrorMapper {
	return ErrorMapper{
		Provider: integration.TypeCarrierOK,
		NotFound: func(err error) bool { return errors.Is(err, errSentinelNotFound) },
	}
}

func TestErrorMapper_Kinds(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		err    error
		kind   services.CarrierIntelErrorKind
		status int
		retry  time.Duration
	}{
		{
			name:   "401",
			err:    &restx.APIError{StatusCode: http.StatusUnauthorized},
			kind:   services.CarrierIntelErrorUnauthorized,
			status: 401,
		},
		{
			name:   "403",
			err:    &restx.APIError{StatusCode: http.StatusForbidden},
			kind:   services.CarrierIntelErrorUnauthorized,
			status: 403,
		},
		{
			name:   "402",
			err:    &restx.APIError{StatusCode: http.StatusPaymentRequired},
			kind:   services.CarrierIntelErrorPaymentRequired,
			status: 402,
		},
		{
			name:   "404",
			err:    &restx.APIError{StatusCode: http.StatusNotFound},
			kind:   services.CarrierIntelErrorNotFound,
			status: 404,
		},
		{
			name: "sentinel",
			err:  fmt.Errorf("lookup: %w", errSentinelNotFound),
			kind: services.CarrierIntelErrorNotFound,
		},
		{
			name: "429",
			err: &restx.APIError{
				StatusCode: http.StatusTooManyRequests,
				RetryAfter: 5 * time.Second,
			},
			kind:   services.CarrierIntelErrorRateLimited,
			status: 429,
			retry:  5 * time.Second,
		},
		{
			name:   "400",
			err:    &restx.APIError{StatusCode: http.StatusBadRequest},
			kind:   services.CarrierIntelErrorInvalidRequest,
			status: 400,
		},
		{
			name:   "422",
			err:    &restx.APIError{StatusCode: http.StatusUnprocessableEntity},
			kind:   services.CarrierIntelErrorInvalidRequest,
			status: 422,
		},
		{
			name:   "500",
			err:    &restx.APIError{StatusCode: http.StatusInternalServerError},
			kind:   services.CarrierIntelErrorUnavailable,
			status: 500,
		},
		{
			name: "transport",
			err: &restx.TransportError{
				Method: "GET",
				Path:   "/v2/profile",
				Err:    context.DeadlineExceeded,
			},
			kind: services.CarrierIntelErrorUnavailable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			mapped := testMapper().Map(tc.err)
			var providerErr *services.CarrierIntelProviderError
			require.ErrorAs(t, mapped, &providerErr)
			assert.Equal(t, tc.kind, providerErr.Kind)
			assert.Equal(t, tc.status, providerErr.StatusCode)
			assert.Equal(t, tc.retry, providerErr.RetryAfter)
			assert.Equal(t, integration.TypeCarrierOK, providerErr.Provider)
			assert.NotEmpty(t, providerErr.Message)
			assert.ErrorIs(t, mapped, tc.err)
		})
	}
}

func TestErrorMapper_PassThrough(t *testing.T) {
	t.Parallel()

	mapper := testMapper()
	assert.NoError(t, mapper.Map(nil))

	limited := &restx.RateLimitedError{RetryAfter: time.Second, Key: "k"}
	assert.Same(t, limited, mapper.Map(limited))

	assert.Equal(t, context.Canceled, mapper.Map(context.Canceled))

	already := mapper.InvalidRequest("bad")
	assert.Same(t, already, mapper.Map(already))
}

func TestErrorMapper_MessagesNeverEchoVendorText(t *testing.T) {
	t.Parallel()

	mapped := testMapper().Map(&restx.APIError{
		StatusCode: http.StatusUnauthorized,
		Message:    "key sk_live_SECRET is invalid",
	})
	var providerErr *services.CarrierIntelProviderError
	require.ErrorAs(t, mapped, &providerErr)
	assert.NotContains(t, providerErr.Error(), "SECRET")
}

func TestRecorder_RecordsStatusAndUnits(t *testing.T) {
	t.Parallel()

	var calls []services.CarrierIntelCall
	recorder := NewRecorder(func(_ context.Context, call services.CarrierIntelCall) {
		calls = append(calls, call)
	})

	mapper := testMapper()
	recorder.Record(t.Context(), &CallOutcome{
		Endpoint: carrierintel.EndpointProfileFull,
		Started:  time.Now(),
		Found:    true,
	})
	notFound := mapper.Map(errSentinelNotFound)
	recorder.Record(t.Context(), &CallOutcome{
		Endpoint: carrierintel.EndpointProfileFull,
		Started:  time.Now(),
		RawErr:   errSentinelNotFound,
		Mapped:   notFound,
	})
	recorder.Record(t.Context(), &CallOutcome{
		Endpoint: carrierintel.EndpointMonitorAdd,
		Started:  time.Now(),
		Status:   http.StatusMultiStatus,
		Units:    3,
		Found:    true,
	})

	require.Len(t, calls, 3)
	assert.Equal(t, http.StatusOK, calls[0].StatusCode)
	assert.Equal(t, 1, calls[0].Units)
	assert.Equal(t, http.StatusOK, calls[1].StatusCode)
	assert.Equal(t, notFound, calls[1].Err)
	assert.Equal(t, http.StatusMultiStatus, calls[2].StatusCode)
	assert.Equal(t, 3, calls[2].Units)

	NewRecorder(nil).Record(t.Context(), &CallOutcome{Started: time.Now()})
}

func TestLimiterKeyPrefix(t *testing.T) {
	t.Parallel()

	prefix := LimiterKeyPrefix("carrierok", " sk_live_abc ")
	assert.Equal(t, LimiterKeyPrefix("carrierok", "sk_live_abc"), prefix)
	assert.Len(t, prefix, len("carrierok:")+16)
	assert.NotContains(t, prefix, "sk_live")
	assert.NotEqual(t, prefix, LimiterKeyPrefix("carrierok", "sk_live_abd"))
}

func TestConversions(t *testing.T) {
	t.Parallel()

	fraction := jsonflex.Float(0.41)
	whole := jsonflex.Float(72.4)
	negative := jsonflex.Float(-1)
	assert.InDelta(t, 41, *Percent(&fraction), 0.0001)
	assert.InDelta(t, 72.4, *Percent(&whole), 0.0001)
	assert.Nil(t, Percent(&negative))
	assert.Nil(t, Percent(nil))
	assert.Nil(t, NonNegativeFloat(&negative))

	rounded := jsonflex.Float(47.6)
	assert.Equal(t, 48, *IntFromFloat(&rounded))

	one := jsonflex.Int(1)
	two := jsonflex.Int(2)
	assert.Equal(t, 3, *SumInts(&one, nil, &two))
	assert.Nil(t, SumInts(nil, nil))

	for value, want := range map[string]bool{
		"Y": true, "N": false, "": false, "NONE": false, "2026-01-01": true, "true": true, "0": false,
	} {
		text := jsonflex.String(value)
		assert.Equal(t, want, Truthy(&text), value)
	}
	assert.False(t, Truthy(nil))
}

func TestVocabulary(t *testing.T) {
	t.Parallel()

	assert.Equal(t, carrierintel.AuthorityStatusActive, AuthorityStatus("active"))
	assert.Equal(t, carrierintel.AuthorityStatusActive, AuthorityStatus("A"))
	assert.Equal(t, carrierintel.AuthorityStatusInactive, AuthorityStatus("I"))
	assert.Equal(t, carrierintel.AuthorityStatusNone, AuthorityStatus(""))
	assert.Equal(t, carrierintel.AuthorityStatusRevoked, AuthorityStatus("Revoked"))
	assert.Equal(t, carrierintel.AuthorityStatusUnknown, AuthorityStatus("pending"))

	assert.Equal(t, carrierintel.SafetyRatingConditional, SafetyRating("c"))
	assert.Equal(t, carrierintel.SafetyRatingUnsatisfactory, SafetyRating("Unsatisfactory"))
	assert.Equal(t, carrierintel.SafetyRatingNotRated, SafetyRating("None"))

	assert.Equal(t, "ACTIVE", USDOTStatus("A"))
	assert.Equal(t, "INACTIVE", USDOTStatus("inactive"))
	assert.Equal(t, "OUT-OF-SERVICE", USDOTStatus("out-of-service"))
	assert.Empty(t, USDOTStatus(" "))

	for input, want := range map[string][2]string{
		"MC277621":  {"MC", "277621"},
		"mc-277621": {"MC", "277621"},
		"277621":    {"", "277621"},
		"FF 123":    {"FF", "123"},
		"MX#0042":   {"MX", "0042"},
		"no digits": {"", ""},
		"":          {"", ""},
	} {
		prefix, number := SplitDocket(input)
		assert.Equal(t, want[0], prefix, input)
		assert.Equal(t, want[1], number, input)
	}

	assert.Equal(t, "MC", DocketPrefixOrDefault(" "))
	assert.Equal(t, "FF", DocketPrefixOrDefault("ff"))

	assert.Equal(t, []string{"A", "B", "C"}, SplitList([]string{"A; B", " ", "C", "A"}))
	assert.Nil(t, SplitList([]string{" ; "}))

	assert.Nil(t, Address(AddressParts{Country: "US"}))
	address := Address(AddressParts{Full: "1 MAIN ST, X", State: "il"})
	require.NotNil(t, address)
	assert.Equal(t, "1 MAIN ST, X", address.Line1)
	assert.Equal(t, "IL", address.State)
}

func TestUserError(t *testing.T) {
	t.Parallel()

	cause := errors.New("status 401 body")
	err := NewUserError("CarrierOk rejected the API key", cause)
	assert.Equal(t, "CarrierOk rejected the API key", err.Error())
	assert.ErrorIs(t, err, cause)
}
