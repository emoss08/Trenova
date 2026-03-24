package messages

import (
	"context"
	"net/http"
	"testing"

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
}

func TestCreatePathAndResponse(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "/v1/fleet/messages", req.Path)
			out := req.Out.(*CreateResponse)
			*out = CreateResponse{}
			return nil
		}},
	)

	_, err := svc.Create(t.Context(), CreateRequest{
		Text:      "Hello",
		DriverIds: []float32{1},
	})
	require.NoError(t, err)
}
