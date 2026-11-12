package recordshareservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/recordshare"
	"github.com/emoss08/trenova/internal/modules/agent/ports/recordshareports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInput(users ...pulid.ID) *shareInput {
	return &shareInput{userIDs: users, path: "/billing/invoices?item=inv_1"}
}

// A share names at least one teammate, never the sharer, at most the cap, a
// note within its limit and a link inside the app.
func TestValidateShareInput_RefusesWhatCannotBeShared(t *testing.T) {
	t.Parallel()

	me, dana := pulid.MustNew("usr_"), pulid.MustNew("usr_")
	require.NoError(t, validateShareInput(validInput(dana), me))

	crowd := make([]pulid.ID, 0, recordshare.MaxRecipients+1)
	for range recordshare.MaxRecipients + 1 {
		crowd = append(crowd, pulid.MustNew("usr_"))
	}
	outside := validInput(dana)
	outside.path = "https://evil.example/billing"
	wordy := validInput(dana)
	wordy.note = strings.Repeat("n", recordshare.MaxNoteLength+1)

	for name, input := range map[string]*shareInput{
		"nobody":             validInput(),
		"too many":           validInput(crowd...),
		"the sharer":         validInput(me),
		"a link out":         outside,
		"a note too long":    wordy,
		"a missing teammate": validInput(pulid.Nil),
	} {
		err := validateShareInput(input, me)
		require.Error(t, err, name)
	}
}

// One share row per recipient, about the record, from the sharer, opening
// where the sharer was.
func TestBuildShares_RecordsOneShareForEachRecipient(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	plan := &sharePlan{
		record: &recordshareports.Record{
			Kind:  recordkind.Kind("shipment"),
			ID:    pulid.MustNew("shp_"),
			Label: "SEED-SHP-010",
		},
		sharer:     &tenant.User{ID: pulid.MustNew("usr_"), Name: "Marcus Bell"},
		recipients: []*tenant.User{{ID: pulid.MustNew("usr_")}, {ID: pulid.MustNew("usr_")}},
		input:      shareInput{note: "Look at the delivery", path: "/desk/r/shipment/shp_1"},
	}

	shares, err := buildShares(tenantInfo, plan)

	require.NoError(t, err)
	require.Len(t, shares, 2)
	for i, share := range shares {
		assert.Equal(t, plan.recipients[i].ID, share.SharedWithID)
		assert.Equal(t, plan.sharer.ID, share.SharedByID)
		assert.Equal(t, plan.record.ID, share.RecordID)
		assert.Equal(t, recordkind.Kind("shipment"), share.RecordKind)
		assert.Equal(t, "/desk/r/shipment/shp_1", share.Path)
		assert.Equal(t, "Look at the delivery", share.Note)
		assert.Equal(t, 1, share.ShareCount)
		assert.Equal(t, tenantInfo.OrgID, share.OrganizationID)
	}
}

func TestEmailStatus_SaysHowManyEmailsWent(t *testing.T) {
	t.Parallel()

	assert.Equal(t, recordshareports.EmailNotConfigured, emailStatus(false, 0, 2))
	assert.Equal(t, recordshareports.EmailQueued, emailStatus(true, 2, 2))
	assert.Equal(t, recordshareports.EmailPartial, emailStatus(true, 1, 2))
	assert.Equal(t, recordshareports.EmailFailed, emailStatus(true, 0, 2))
}

// The link only opens for an active member who can sign in.
func TestIsShareableUser_IsAnActiveMemberWhoCanSignIn(t *testing.T) {
	t.Parallel()

	active := &tenant.User{Status: domaintypes.StatusActive, Username: "dana"}
	assert.True(t, isShareableUser(active))
	assert.False(
		t,
		isShareableUser(&tenant.User{Status: domaintypes.StatusInactive, Username: "dana"}),
	)
	assert.False(
		t,
		isShareableUser(
			&tenant.User{Status: domaintypes.StatusActive, Username: "dana", IsLocked: true},
		),
	)
	assert.False(
		t,
		isShareableUser(&tenant.User{Status: domaintypes.StatusActive, Username: systemUsername}),
	)
}
