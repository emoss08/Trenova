package seedaccountcli

import (
	"bytes"
	"testing"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountport"
	"github.com/emoss08/trenova/internal/cloud/seedaccounts/seedaccountservice"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleReport() *seedaccountservice.Report {
	orgID := pulid.MustNew("org_")

	return &seedaccountservice.Report{
		Users: []*seedaccountservice.UserReport{
			{
				Footprint: &seedaccountport.UserFootprint{
					ID:           pulid.MustNew("usr_"),
					Username:     "admin",
					EmailAddress: "admin@trenova.app",
					Status:       domaintypes.StatusActive,
					PasswordHash: "secret-hash",
					Memberships:  []seedaccountport.Membership{{ID: pulid.MustNew("mem_"), OrganizationID: orgID}},
					References: []seedaccountport.Reference{
						{Table: "audit_entries", Column: "user_id", Rows: seedaccountport.ReferenceCap},
					},
				},
				Provenance: []seedaccountservice.Provenance{seedaccountservice.ProvenanceSeedTracking},
				Action:     seedaccountservice.UserActionRetire,
			},
			{
				Footprint: &seedaccountport.UserFootprint{
					ID:           pulid.MustNew("usr_"),
					Username:     "admin-transport",
					EmailAddress: "admin.transport@trenova.app",
					Status:       domaintypes.StatusActive,
				},
				Action: seedaccountservice.UserActionUnproven,
			},
		},
		Organizations: []*seedaccountservice.OrganizationReport{
			{
				Footprint: &seedaccountport.OrganizationFootprint{ID: orgID, Name: "Trenova Logistics", ScacCode: "TRNV"},
				Action:    seedaccountservice.OrganizationActionReview,
				Reasons:   []string{"hosts the instance system user"},
			},
		},
		Errors: []string{},
	}
}

func TestResolveMode(t *testing.T) {
	t.Parallel()

	apply, err := resolveMode(false, false)
	require.NoError(t, err)
	assert.False(t, apply)

	apply, err = resolveMode(true, false)
	require.NoError(t, err)
	assert.True(t, apply)

	_, err = resolveMode(true, true)
	require.ErrorIs(t, err, errApplyAndDryRun)
}

func TestRenderText(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, render(&out, sampleReport(), outputText))
	text := out.String()

	assert.Contains(t, text, "Dry run. Nothing was changed")
	assert.Contains(t, text, "admin <admin@trenova.app>")
	assert.Contains(t, text, "seed_created_entities row for AdminAccount")
	assert.Contains(t, text, "kept because of audit_entries.user_id (1000+)")
	assert.Contains(t, text, "skipped: nothing proves the seed created it")
	assert.Contains(t, text, "kept for manual review:")
	assert.Contains(t, text, "hosts the instance system user")
	assert.NotContains(t, text, "secret-hash")
}

func TestRenderJSONNeverIncludesThePasswordHash(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	require.NoError(t, render(&out, sampleReport(), outputJSON))
	assert.NotContains(t, out.String(), "secret-hash")

	var decoded seedaccountservice.Report
	require.NoError(t, sonic.Unmarshal(out.Bytes(), &decoded))
	require.Len(t, decoded.Users, 2)
	assert.Equal(t, "admin", decoded.Users[0].Footprint.Username)
}
