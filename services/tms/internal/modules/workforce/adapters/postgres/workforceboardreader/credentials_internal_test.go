package workforceboardreader

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func credentialEnding(ends time.Time, renewal int32) *credentialRow {
	at := ends.Unix()

	return &credentialRow{Status: "Active", ExpiresAt: &at, Renewal: renewal}
}

func TestCredentialStatusIsTheWorkforcePagesWordAndTone(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, time.November, 23, 19, 0, 0, 0, time.UTC)
	cases := []struct {
		row  *credentialRow
		word string
		tone boardcatalog.Tone
	}{
		{credentialEnding(now.AddDate(0, 0, -8), 30), "Expired", boardcatalog.ToneBad},
		{credentialEnding(now.AddDate(0, 0, 8), 30), "Expiring soon", boardcatalog.ToneWarn},
		{credentialEnding(now.AddDate(0, 3, 0), 30), "Valid", boardcatalog.ToneOK},
		{&credentialRow{Status: "Archived"}, "Archived", ""},
	}
	for _, tc := range cases {
		word, tone := credentialStatus(context.Background(), tc.row, now.Unix())
		assert.Equal(t, tc.word, word)
		assert.Equal(t, tc.tone, tone)
	}
}

func TestExpiryReadsWholeDaysAgainstTheRenewalWindow(t *testing.T) {
	t.Parallel()

	zone, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	now := time.Date(2026, time.November, 23, 14, 48, 0, 0, zone)
	req := &boardreaderports.RecordRequest{Now: now.Unix(), Location: zone}

	left := expiry(context.Background(),
		credentialEnding(time.Date(2026, time.December, 1, 19, 3, 0, 0, zone), 30), req)
	assert.Equal(t, boardcatalog.Days(8), left.Value)
	assert.Equal(t, int64(30*86_400), left.Window.Span)

	unwindowed := expiry(context.Background(),
		credentialEnding(time.Date(2026, time.December, 1, 19, 3, 0, 0, zone), 0), req)
	assert.Equal(t, int64(defaultRenewalDays*86_400), unwindowed.Window.Span,
		"a type with no renewal window is read against thirty days")
}
