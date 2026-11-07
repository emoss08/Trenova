package firstadmininviteservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/passwordreset"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/dbtest"
	"github.com/emoss08/trenova/internal/testutil/securityaudittest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const (
	testNow   = int64(1_790_000_000)
	inviteTTL = 7 * 24 * time.Hour
	hashA     = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	hashB     = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

type fakeRecords struct {
	record *instancebootstrap.InstanceBootstrap
}

func (f *fakeRecords) GetRecord(context.Context) (*instancebootstrap.InstanceBootstrap, error) {
	if f.record == nil {
		return nil, keel.ErrNotFound
	}
	return f.record, nil
}

type fakeUsers struct {
	user *tenant.User
}

func (f *fakeUsers) FindByIDForLogin(_ context.Context, id pulid.ID) (*tenant.User, error) {
	if f.user == nil || f.user.ID != id {
		return nil, errortypes.NewNotFoundError("User not found")
	}
	return f.user, nil
}

type fakeTokens struct {
	rows        []*passwordreset.PasswordResetToken
	invalidated []pulid.ID
	createErr   error
}

func (f *fakeTokens) FindByUserAndHash(
	_ context.Context,
	userID pulid.ID,
	hash string,
) (*passwordreset.PasswordResetToken, error) {
	for _, row := range f.rows {
		if row.UserID == userID && row.TokenHash == hash {
			return row, nil
		}
	}
	return nil, errortypes.NewNotFoundError("Password reset token not found")
}

func (f *fakeTokens) InvalidateOutstanding(_ context.Context, userID pulid.ID, now int64) error {
	f.invalidated = append(f.invalidated, userID)
	for _, row := range f.rows {
		if row.UserID == userID && row.UsedAt == nil && row.InvalidatedAt == nil {
			row.InvalidatedAt = &now
		}
	}
	return nil
}

func (f *fakeTokens) Create(_ context.Context, token *passwordreset.PasswordResetToken) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.rows = append(f.rows, token)
	return nil
}

type deps struct {
	records *fakeRecords
	users   *fakeUsers
	tokens  *fakeTokens
	auditor *securityaudittest.Recorder
	svc     *Service
}

func newDeps(t *testing.T, admin *tenant.User) *deps {
	t.Helper()

	d := &deps{
		records: &fakeRecords{},
		users:   &fakeUsers{user: admin},
		tokens:  &fakeTokens{},
		auditor: &securityaudittest.Recorder{},
	}
	if admin != nil {
		d.records.record = &instancebootstrap.InstanceBootstrap{
			AdminUserID:    admin.ID,
			OrganizationID: admin.CurrentOrganizationID,
			BusinessUnitID: admin.BusinessUnitID,
		}
	}
	d.svc = &Service{
		db:         dbtest.NopConnection{},
		bootstraps: d.records,
		users:      d.users,
		tokens:     d.tokens,
		inviteTTL:  inviteTTL,
		auditor:    d.auditor,
		now:        func() int64 { return testNow },
		l:          zap.NewNop(),
	}
	return d
}

func awaitingAdmin() *tenant.User {
	return &tenant.User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		EmailAddress:          "dana@acme.example",
	}
}

func invite(hash string, expiresIn time.Duration) *instancebootstrapports.FirstAdministratorInvite {
	return &instancebootstrapports.FirstAdministratorInvite{
		TokenSHA256: hash,
		ExpiresAt:   testNow + int64(expiresIn.Seconds()),
	}
}

func TestInstallIssuesAnInviteForTheFirstAdministrator(t *testing.T) {
	t.Parallel()

	admin := awaitingAdmin()
	d := newDeps(t, admin)

	result, err := d.svc.Install(t.Context(), invite(hashA, 24*time.Hour))
	require.NoError(t, err)
	assert.False(t, result.SetUp)
	assert.Equal(t, hashA, result.InviteTokenSHA256)

	require.Len(t, d.tokens.rows, 1)
	row := d.tokens.rows[0]
	assert.Equal(t, admin.ID, row.UserID)
	assert.Equal(t, admin.BusinessUnitID, row.BusinessUnitID)
	assert.Equal(t, admin.CurrentOrganizationID, row.OrganizationID)
	assert.Equal(t, hashA, row.TokenHash)
	assert.Equal(t, passwordreset.PasswordResetPurposeInvite, row.Purpose)
	assert.Equal(t, testNow+int64((24*time.Hour).Seconds()), row.ExpiresAt)
	assert.Equal(t, []pulid.ID{admin.ID}, d.tokens.invalidated)

	changes := d.auditor.Changes()
	require.Len(t, changes, 1)
	assert.Equal(t, permission.ResourceUser, changes[0].Resource)
	assert.Equal(t, admin.ID.String(), changes[0].ResourceID)
	assert.Equal(t, permissionports.SystemAuditActor(), changes[0].Actor)
	assert.Equal(t, admin.CurrentOrganizationID, changes[0].OrganizationID)
}

func TestInstallIsIdempotentByHash(t *testing.T) {
	t.Parallel()

	d := newDeps(t, awaitingAdmin())
	for range 3 {
		result, err := d.svc.Install(t.Context(), invite(hashA, 24*time.Hour))
		require.NoError(t, err)
		assert.Equal(t, hashA, result.InviteTokenSHA256)
	}

	assert.Len(t, d.tokens.rows, 1)
	assert.Len(t, d.auditor.Changes(), 1)
}

func TestInstallCapsTheExpiryAtTheInviteTTL(t *testing.T) {
	t.Parallel()

	d := newDeps(t, awaitingAdmin())
	_, err := d.svc.Install(t.Context(), invite(hashA, 30*24*time.Hour))
	require.NoError(t, err)

	require.Len(t, d.tokens.rows, 1)
	assert.Equal(t, testNow+int64(inviteTTL.Seconds()), d.tokens.rows[0].ExpiresAt)
}

func TestANewInviteReplacesTheOneBefore(t *testing.T) {
	t.Parallel()

	d := newDeps(t, awaitingAdmin())
	_, err := d.svc.Install(t.Context(), invite(hashA, time.Hour))
	require.NoError(t, err)
	result, err := d.svc.Install(t.Context(), invite(hashB, time.Hour))
	require.NoError(t, err)
	assert.Equal(t, hashB, result.InviteTokenSHA256)

	require.Len(t, d.tokens.rows, 2)
	assert.NotNil(t, d.tokens.rows[0].InvalidatedAt)
	assert.True(t, d.tokens.rows[1].IsRedeemable(testNow))

	again, err := d.svc.Install(t.Context(), invite(hashA, time.Hour))
	require.NoError(t, err)
	assert.Empty(t, again.InviteTokenSHA256, "a replaced invite is never made live again")
	assert.False(t, again.SetUp)
	assert.Len(t, d.tokens.rows, 2)
}

func TestAUsedOrExpiredInviteIsReportedAndNeverRecreated(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*passwordreset.PasswordResetToken){
		"used":    func(row *passwordreset.PasswordResetToken) { used := testNow; row.UsedAt = &used },
		"expired": func(row *passwordreset.PasswordResetToken) { row.ExpiresAt = testNow },
	}
	for name, spoil := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			admin := awaitingAdmin()
			d := newDeps(t, admin)
			row := &passwordreset.PasswordResetToken{
				UserID: admin.ID, TokenHash: hashA, ExpiresAt: testNow + 60,
				Purpose: passwordreset.PasswordResetPurposeInvite,
			}
			spoil(row)
			d.tokens.rows = []*passwordreset.PasswordResetToken{row}

			result, err := d.svc.Install(t.Context(), invite(hashA, time.Hour))
			require.NoError(t, err)
			assert.False(t, result.SetUp)
			assert.Empty(t, result.InviteTokenSHA256)
			assert.Len(t, d.tokens.rows, 1)
			assert.Empty(t, d.auditor.Changes())
		})
	}
}

func TestInstallReportsAnAdministratorWhoIsSetUp(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*tenant.User){
		"chose a password": func(u *tenant.User) { at := testNow; u.PasswordChangedAt = &at },
		"signed in":        func(u *tenant.User) { at := testNow; u.LastLoginAt = &at },
	}
	for name, setUp := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			admin := awaitingAdmin()
			setUp(admin)
			d := newDeps(t, admin)

			result, err := d.svc.Install(t.Context(), invite(hashA, time.Hour))
			require.NoError(t, err)
			assert.True(t, result.SetUp)
			assert.Empty(t, result.InviteTokenSHA256)
			assert.Empty(t, d.tokens.rows)
			assert.Empty(t, d.auditor.Changes())
		})
	}
}

func TestInstallBeforeTheBootstrapIsRefused(t *testing.T) {
	t.Parallel()

	d := newDeps(t, nil)
	_, err := d.svc.Install(t.Context(), invite(hashA, time.Hour))
	require.ErrorIs(t, err, instancebootstrapports.ErrNotBootstrapped)
}

func TestInstallValidatesTheInvite(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		invite *instancebootstrapports.FirstAdministratorInvite
		field  string
	}{
		"missing":    {nil, fieldTokenSHA256},
		"short hash": {invite("abc", time.Hour), fieldTokenSHA256},
		"uppercase hash": {
			invite("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA", time.Hour),
			fieldTokenSHA256,
		},
		"expired":     {invite(hashA, 0), fieldExpiresAt},
		"in the past": {invite(hashA, -time.Hour), fieldExpiresAt},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			d := newDeps(t, awaitingAdmin())
			_, err := d.svc.Install(t.Context(), tt.invite)
			var multiErr *errortypes.MultiError
			require.ErrorAs(t, err, &multiErr)
			require.Len(t, multiErr.Errors, 1)
			assert.Equal(t, tt.field, multiErr.Errors[0].Field)
			assert.Empty(t, d.tokens.rows)
		})
	}
}

func TestInstallSurfacesAStoreFailure(t *testing.T) {
	t.Parallel()

	d := newDeps(t, awaitingAdmin())
	d.tokens.createErr = errors.New("the database is gone")
	_, err := d.svc.Install(t.Context(), invite(hashA, time.Hour))
	require.Error(t, err)
	assert.Empty(t, d.auditor.Changes())
}
