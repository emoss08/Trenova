package cloudtenant_test

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const instanceID = "5d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"

func entry() *cloudtenant.Desired {
	return &cloudtenant.Desired{
		InstanceID:       instanceID,
		Revision:         3,
		Lifecycle:        cloudtenant.LifecycleActive,
		OrganizationName: "Acme Freight",
		Plan: cloudtenant.Plan{
			Key:                 "shared",
			Name:                "Shared",
			BackupRetentionDays: 14,
			Support:             "email",
		},
		Addons:     []string{"ACC"},
		Features:   []string{"tms.accounting"},
		AIProvider: cloudtenant.AIProviderCustomer,
		Allowances: []cloudtenant.Allowance{
			{Meter: "shipments", AppMeter: "shipments.total", Unit: "count", Quantity: 100},
		},
		FirstAdministrator: cloudtenant.FirstAdministrator{
			Name:             "Dana Ortiz",
			Email:            "dana@acme.test",
			OrganizationName: "Acme Freight",
		},
		Account:  cloudtenant.Account{PlanURL: "https://console.test/plan", FinalBackupDays: 30},
		Commands: []cloudtenant.Command{},
	}
}

func TestAWellFormedEntryHasNoRefusal(t *testing.T) {
	t.Parallel()

	assert.Nil(t, entry().Refusal())
}

func TestAnEntryIsAddressedByItsOwnInstance(t *testing.T) {
	t.Parallel()

	e := entry()
	require.NoError(t, e.CheckAddress(instanceID))
	require.ErrorIs(
		t,
		e.CheckAddress("6d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"),
		cloudtenant.ErrInstanceIDMismatch,
	)
	require.ErrorIs(
		t,
		e.CheckAddress(strings.ToUpper(instanceID)),
		cloudtenant.ErrInstanceIDMismatch,
	)

	e.Revision = 0
	require.ErrorIs(t, e.CheckAddress(instanceID), cloudtenant.ErrRevisionInvalid)
}

func TestAnInvalidEntryIsRefusedWithEveryProblemAndNoName(t *testing.T) {
	t.Parallel()

	e := entry()
	e.Lifecycle = "paused"
	e.AIProvider = "nobody"
	e.FirstAdministrator.Email = ""
	e.FirstAdministrator.Invite = &cloudtenant.Invite{TokenSHA256: "nothex"}
	e.Account.FinalBackupDays = 0

	refusal := e.Refusal()
	require.NotNil(t, refusal)
	assert.Equal(t, int64(3), refusal.Revision)
	assert.Equal(t, cloudtenant.RefusalInvalid, refusal.Code)
	for _, want := range []string{"lifecycle", "aiProvider", "firstAdministrator.email",
		"firstAdministrator.invite", "finalBackupDays"} {
		assert.Contains(t, refusal.Reason, want)
	}
	assert.NotContains(t, refusal.Reason, "Dana")
	assert.LessOrEqual(t, len([]rune(refusal.Reason)), 500)
}

func TestARefusalReasonIsCutToTheWireLimit(t *testing.T) {
	t.Parallel()

	refusal := cloudtenant.Invalid(1, strings.Repeat("x", 900))
	assert.Len(t, []rune(refusal.Reason), 500)
}

func TestTheDigestFollowsTheEntry(t *testing.T) {
	t.Parallel()

	a, err := entry().Digest()
	require.NoError(t, err)
	b, err := entry().Digest()
	require.NoError(t, err)
	assert.Equal(t, a, b)

	changed := entry()
	ends := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	changed.Account.EndsAt = &ends
	c, err := changed.Digest()
	require.NoError(t, err)
	assert.NotEqual(
		t,
		a,
		c,
		"a plan end composed at serve time changes the entry without a new revision",
	)
}

func TestTheIdentityOfAMadeTenantIsFixed(t *testing.T) {
	t.Parallel()

	held := entry()
	same := entry()
	same.Revision = 9
	same.Features = nil
	assert.True(t, same.SameIdentity(held))

	renamed := entry()
	renamed.OrganizationName = "Acme Logistics"
	assert.False(t, renamed.SameIdentity(held))

	otherAdmin := entry()
	otherAdmin.FirstAdministrator.Email = "someone@acme.test"
	assert.False(t, otherAdmin.SameIdentity(held))
}

func TestStatusReportsRemovalAsApplyingAndCarriesTheRefusal(t *testing.T) {
	t.Parallel()

	tenant := &cloudtenant.Tenant{
		InstanceID:      instanceID,
		OrganizationID:  pulid.MustNew("org_"),
		State:           cloudtenant.StateRemoving,
		AppliedRevision: 4,
	}
	tenant.Refuse(cloudtenant.Invalid(5, "no"))

	status := tenant.Status(nil)
	assert.Equal(t, cloudtenant.StateApplying, status.State)
	require.NotNil(t, status.AppliedRevision)
	assert.Equal(t, int64(4), *status.AppliedRevision)
	require.NotNil(t, status.Refused)
	assert.Equal(t, int64(5), status.Refused.Revision)

	tenant.Refuse(nil)
	tenant.AppliedRevision = 0
	status = tenant.Status(nil)
	assert.Nil(t, status.Refused)
	assert.Nil(t, status.AppliedRevision)
}

func TestOnlyAFinalArchiveLetsATenantBeRemoved(t *testing.T) {
	t.Parallel()

	tenant := &cloudtenant.Tenant{LastArchiveCommandID: "cmd", LastArchivePurpose: "nightly"}
	assert.False(t, tenant.HasFinalArchive())
	tenant.LastArchivePurpose = cloudtenant.ArchivePurposeFinal
	assert.True(t, tenant.HasFinalArchive())
}

func TestArchiveCommandsAreFound(t *testing.T) {
	t.Parallel()

	e := entry()
	e.Commands = []cloudtenant.Command{
		{Kind: "restart", ID: "7d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10"},
		{
			Kind:    cloudtenant.CommandArchive,
			ID:      "8d1a3c1e-8c1f-4a39-9a43-5b0d7c4e2f10",
			Purpose: "final",
		},
	}
	require.Len(t, e.ArchiveCommands(), 1)
	assert.Nil(t, e.Refusal())
}
