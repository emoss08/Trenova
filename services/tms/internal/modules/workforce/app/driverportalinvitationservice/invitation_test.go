package driverportalinvitationservice

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/capabilitytest"
	"github.com/emoss08/trenova/internal/testutil/mocks/portalaccessportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func assetOperationsOrgRepo(
	t *testing.T,
	tenantInfo pagination.TenantInfo,
	enabled bool,
) *tenantportsmock.MockOrganizationRepository {
	t.Helper()

	return capabilitytest.OrgRepo(t, capabilitytest.OrgRepoParams{
		TenantInfo:             tenantInfo,
		Lock:                   tenantports.CapabilityLockNone,
		BrokerageEnabled:       true,
		AssetOperationsEnabled: enabled,
	})
}

func TestInviteResult_GivesTheLinkOnlyWhenTheEmailDidNotGo(t *testing.T) {
	t.Parallel()

	const link = "https://dash.example.com/invite/token"

	emailed := inviteResult(nil, link, true)
	assert.Nil(t, emailed.InviteURL, "the link is the driver's once it is in their inbox")
	assert.True(t, emailed.EmailSent)

	handed := inviteResult(nil, link, false)
	require.NotNil(t, handed.InviteURL, "with no email the link is the only way to reach them")
	assert.Equal(t, link, *handed.InviteURL)
	assert.False(t, handed.EmailSent)
}

func TestInviteAuditComment_SaysWhenTheLinkWentToTheInviter(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"Portal invitation sent to driver@example.com",
		inviteAuditComment("driver@example.com", true),
	)
	assert.Contains(
		t,
		inviteAuditComment("driver@example.com", false),
		"the link was given to the inviter",
	)
}

func TestResolveInviteEmail_KeepsAnInvitationOnTheDriversAddress(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		override string
		onFile   string
		want     string
		refused  bool
	}{
		{name: "the address on file", onFile: "Driver@Example.com", want: "driver@example.com"},
		{
			name:     "the same address typed",
			override: " driver@example.com ",
			onFile:   "Driver@Example.com",
			want:     "driver@example.com",
		},
		{
			name:     "a different address",
			override: "someone.else@example.com",
			onFile:   "driver@example.com",
			refused:  true,
		},
		{
			name:     "an address for a driver with none on file",
			override: "New.Driver@Example.com",
			want:     "new.driver@example.com",
		},
		{name: "no address anywhere", refused: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := resolveInviteEmail(tt.override, tt.onFile)

			if tt.refused {
				require.Error(t, err)
				assert.Empty(t, got)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestInviteWorker_RefusedWhenAssetOperationsDisabled(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	svc := &Service{
		l:          zap.NewNop(),
		orgRepo:    assetOperationsOrgRepo(t, tenantInfo, false),
		portalRepo: portalaccessportsmock.NewMockPortalAccessRepository(t),
	}

	result, err := svc.InviteWorker(
		t.Context(),
		&InviteWorkerRequest{TenantInfo: tenantInfo, WorkerID: pulid.MustNew("wrk_")},
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	assert.Nil(t, result)
	require.Error(t, err)

	validationErr := new(errortypes.Error)
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "feature", validationErr.Field)
	assert.Equal(t, errortypes.ErrInvalidOperation, validationErr.Code)
	assert.Equal(
		t,
		"Driver portal access requires asset operations. Enable asset operations "+
			"for this organization before inviting drivers",
		validationErr.Message,
	)
}

func TestInviteWorker_ProceedsWhenAssetOperationsEnabled(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}
	workerID := pulid.MustNew("wrk_")
	lookupErr := errors.New("worker lookup failed")

	portalRepo := portalaccessportsmock.NewMockPortalAccessRepository(t)
	portalRepo.EXPECT().
		GetWorkerForPortalManagement(mock.Anything, tenantInfo, workerID).
		Return(nil, lookupErr).
		Once()

	svc := &Service{
		l:          zap.NewNop(),
		orgRepo:    assetOperationsOrgRepo(t, tenantInfo, true),
		portalRepo: portalRepo,
	}

	result, err := svc.InviteWorker(
		t.Context(),
		&InviteWorkerRequest{TenantInfo: tenantInfo, WorkerID: workerID},
		&permissionports.RequestActor{UserID: pulid.MustNew("usr_")},
	)

	assert.Nil(t, result)
	require.ErrorIs(t, err, lookupErr)
}
