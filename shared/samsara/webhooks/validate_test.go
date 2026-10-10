package webhooks

import (
	"strings"
	"testing"

	samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"
	"github.com/stretchr/testify/require"
)

func TestValidateCreateRequest(t *testing.T) {
	t.Parallel()

	badVersion := Version("2019-01-01")
	goodVersion := Version20240227
	events := []EventType{"RouteStopArrival", "NotARealEvent"}
	tooManyHeaders := make([]CustomHeader, 6)
	for i := range tooManyHeaders {
		tooManyHeaders[i] = CustomHeader{Key: "k", Value: "v"}
	}
	emptyHeader := []CustomHeader{{Key: "", Value: "v"}}
	longKeyHeader := []CustomHeader{{Key: strings.Repeat("k", 101), Value: "v"}}

	valid := CreateRequest{Name: "TMS", Url: "https://tms.example.com/hooks/samsara/x/"}

	tests := []struct {
		name    string
		mutate  func(*CreateRequest)
		wantErr error
	}{
		{name: "valid", mutate: func(*CreateRequest) {}},
		{
			name:    "missing name",
			mutate:  func(r *CreateRequest) { r.Name = " " },
			wantErr: ErrWebhookNameRequired,
		},
		{
			name:    "long name",
			mutate:  func(r *CreateRequest) { r.Name = strings.Repeat("n", 256) },
			wantErr: ErrWebhookNameTooLong,
		},
		{
			name:    "missing url",
			mutate:  func(r *CreateRequest) { r.Url = "" },
			wantErr: ErrWebhookURLRequired,
		},
		{
			name:    "relative url",
			mutate:  func(r *CreateRequest) { r.Url = "/hooks" },
			wantErr: ErrWebhookURLInvalid,
		},
		{
			name:    "ftp url",
			mutate:  func(r *CreateRequest) { r.Url = "ftp://example.com/hooks" },
			wantErr: ErrWebhookURLInvalid,
		},
		{
			name: "long url",
			mutate: func(r *CreateRequest) {
				r.Url = "https://example.com/" + strings.Repeat("a", 2048)
			},
			wantErr: ErrWebhookURLTooLong,
		},
		{
			name:    "bad version",
			mutate:  func(r *CreateRequest) { r.Version = &badVersion },
			wantErr: ErrWebhookVersionInvalid,
		},
		{name: "good version", mutate: func(r *CreateRequest) { r.Version = &goodVersion }},
		{
			name:    "bad event type",
			mutate:  func(r *CreateRequest) { r.EventTypes = &events },
			wantErr: ErrWebhookEventInvalid,
		},
		{
			name:    "too many headers",
			mutate:  func(r *CreateRequest) { r.CustomHeaders = &tooManyHeaders },
			wantErr: ErrCustomHeadersTooMany,
		},
		{
			name:    "empty header key",
			mutate:  func(r *CreateRequest) { r.CustomHeaders = &emptyHeader },
			wantErr: ErrCustomHeaderInvalid,
		},
		{
			name:    "long header key",
			mutate:  func(r *CreateRequest) { r.CustomHeaders = &longKeyHeader },
			wantErr: ErrCustomHeaderKeyTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := valid
			tt.mutate(&req)
			err := ValidateCreateRequest(req)
			if tt.wantErr == nil {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestValidateUpdateRequest(t *testing.T) {
	t.Parallel()

	blank := ""
	badURL := "not a url"
	badVersion := samsaraspec.WebhooksPatchWebhookRequestBodyVersion("1999-01-01")

	require.NoError(t, ValidateUpdateRequest(UpdateRequest{}))
	require.ErrorIs(t, ValidateUpdateRequest(UpdateRequest{Name: &blank}), ErrWebhookNameRequired)
	require.ErrorIs(t, ValidateUpdateRequest(UpdateRequest{Url: &badURL}), ErrWebhookURLInvalid)
	require.ErrorIs(
		t,
		ValidateUpdateRequest(UpdateRequest{Version: &badVersion}),
		ErrWebhookVersionInvalid,
	)
}
