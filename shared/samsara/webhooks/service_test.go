package webhooks

import (
	"context"
	"net/http"
	"testing"

	"github.com/emoss08/trenova/shared/samsara/internal/httpx"
	"github.com/emoss08/trenova/shared/samsara/internal/httpxtest"
	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Create(t.Context(), CreateRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookNameRequired)

	_, err = svc.Create(t.Context(), CreateRequest{Name: "hook-1"})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookURLRequired)
}

func TestCreateDecodesWebhookResponse(t *testing.T) {
	t.Parallel()

	events := []samsaraspec.WebhooksGetWebhookResponseBodyEventTypes{"RouteStopArrival"}
	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPost, req.Method)
			assert.Equal(t, "/webhooks", req.Path)

			out := req.Out.(*Webhook)
			*out = Webhook{
				Id:         "wh-1",
				Name:       "route-updates",
				Url:        "https://example.com/webhook",
				Version:    "2021-06-09",
				EventTypes: &events,
			}
			return nil
		}},
	)

	out, err := svc.Create(t.Context(), CreateRequest{
		Name: "route-updates",
		Url:  "https://example.com/webhook",
	})
	require.NoError(t, err)
	assert.Equal(t, "wh-1", out.Id)
	assert.Equal(t, "route-updates", out.Name)
	assert.Equal(t, "https://example.com/webhook", out.Url)
	require.NotNil(t, out.EventTypes)
	require.Len(t, *out.EventTypes, 1)
	assert.Equal(
		t,
		samsaraspec.WebhooksGetWebhookResponseBodyEventTypes("RouteStopArrival"),
		(*out.EventTypes)[0],
	)
}

func TestUpdateValidation(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, _ httpx.Request) error {
			return nil
		}},
	)

	_, err := svc.Update(t.Context(), "  ", UpdateRequest{})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrWebhookIDRequired)
}

func TestUpdateUsesWebhookIDInPath(t *testing.T) {
	t.Parallel()

	svc := NewService(
		&httpxtest.MockRequester{DoFunc: func(_ context.Context, req httpx.Request) error {
			assert.Equal(t, http.MethodPatch, req.Method)
			assert.Equal(t, "/webhooks/wh-2", req.Path)

			out := req.Out.(*Webhook)
			*out = Webhook{Id: "wh-2", Name: "updated", Url: "https://example.com/new"}
			return nil
		}},
	)

	updated, err := svc.Update(t.Context(), "wh-2", UpdateRequest{Name: strPtr("updated")})
	require.NoError(t, err)
	assert.Equal(t, "wh-2", updated.Id)
	assert.Equal(t, "updated", updated.Name)
}

func strPtr(value string) *string {
	return &value
}
