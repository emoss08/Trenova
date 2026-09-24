package agentextension_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckSearchQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		want  error
	}{
		{name: "regulation question", query: "FMCSA ELD malfunction 8 day rule"},
		{name: "year range is not a phone number", query: "HOS changes 2019 - 2020 short haul"},
		{name: "section number", query: "49 CFR 395.8 record of duty status"},
		{name: "too short", query: " a ", want: agentextension.ErrQueryTooShort},
		{name: "too long", query: strings.Repeat("rules ", 80), want: agentextension.ErrQueryTooLong},
		{
			name:  "record id",
			query: "status of shp_01J8ZK3M4N5P6Q7R8S9T0V1W2X late delivery",
			want:  agentextension.ErrQueryHasRecordID,
		},
		{
			name:  "email",
			query: "dispatch@acme-logistics.com carrier complaint",
			want:  agentextension.ErrQueryHasEmail,
		},
		{name: "phone", query: "who owns (555) 123-4567 trucking", want: agentextension.ErrQueryHasPhone},
		{name: "intl phone", query: "+1 312 555 0199 broker", want: agentextension.ErrQueryHasPhone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := agentextension.CheckSearchQuery(tt.query)
			if tt.want == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.want)
		})
	}
}

func TestParsePageURL(t *testing.T) {
	t.Parallel()

	parsed, err := agentextension.ParsePageURL(" https://www.fmcsa.dot.gov/regulations/hours-of-service ")
	require.NoError(t, err)
	assert.Equal(t, "www.fmcsa.dot.gov", parsed.Hostname())

	for _, raw := range []string{
		"",
		"ftp://example.com/file",
		"javascript:alert(1)",
		"https://user:pass@example.com/",
		"/relative/path",
		"https://" + strings.Repeat("a", agentextension.MaxPageURLLength),
	} {
		_, err = agentextension.ParsePageURL(raw)
		assert.ErrorIs(t, err, agentextension.ErrPageURLInvalid, raw)
	}
}

func TestIsOfficialSite(t *testing.T) {
	t.Parallel()

	for _, host := range []string{"fmcsa.dot.gov", "www.ecfr.gov", "tc.gc.ca", "www.sct.gob.mx", "army.mil"} {
		assert.True(t, agentextension.IsOfficialSite(host), host)
	}
	for _, host := range []string{"", "govtrucking.com", "eld-vendor.com", "gov.example.org", "gc.ca.example.com"} {
		assert.False(t, agentextension.IsOfficialSite(host), host)
	}
}

func TestSiteOfDropsWWW(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "fmcsa.dot.gov", agentextension.SiteOf("https://WWW.FMCSA.dot.gov/x"))
	assert.Empty(t, agentextension.SiteOf("::"))
}
