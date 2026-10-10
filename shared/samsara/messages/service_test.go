package messages

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/bytedance/sonic"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.List(t.Context(), ListParams{DurationMs: -1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDurationInvalid)
}

func TestListQueryAndPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodGet, req.Method)
			assert.Equal(t, "/v1/fleet/messages", req.Path)
			assert.Equal(t, "100", req.Query.Get("endMs"))
			assert.Equal(t, "200", req.Query.Get("durationMs"))
			return nil
		}},
	)

	_, err := svc.List(t.Context(), ListParams{
		EndMs:      100,
		DurationMs: 200,
	})
	require.NoError(t, err)
}

func TestCreateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Create(t.Context(), CreateRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTextRequired)

	_, err = svc.Create(t.Context(), CreateRequest{Text: "hello"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrDriverIDsRequired)

	_, err = svc.Create(t.Context(), CreateRequest{
		Text:      strings.Repeat("x", 2501),
		DriverIDs: []string{"1654973"},
	})
	require.ErrorIs(t, err, ErrTextTooLong)

	for _, bad := range []string{"drv-1", "", " ", "-5", "+5", "0", "12.5", "1e6", "99999999999999999999"} {
		_, err = svc.Create(t.Context(), CreateRequest{Text: "hello", DriverIDs: []string{bad}})
		require.ErrorIs(t, err, ErrDriverIDInvalid, "id %q", bad)
	}
}

func TestCreatePathAndResponse(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "/v1/fleet/messages", req.Path)
			body, ok := req.Body.(createRequestBody)
			require.True(t, ok)
			encoded, err := sonic.Marshal(body)
			require.NoError(t, err)
			assert.JSONEq(
				t,
				`{"driverIds":[1654973,281474977075805],"text":"Hello"}`,
				string(encoded),
			)
			out := req.Out.(*CreateResponse)
			*out = CreateResponse{}
			return nil
		}},
	)

	_, err := svc.Create(t.Context(), CreateRequest{
		Text:      "Hello",
		DriverIDs: []string{"1654973", " 281474977075805 ", "1654973"},
	})
	require.NoError(t, err)
}
