//go:build integration

package customerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile"
	"github.com/emoss08/trenova/internal/modules/parties/domain/trackingprofile/trackingprofiledb"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func (f *customerFixture) withTracking(
	t *testing.T,
	tenant customerTenant,
	customerID pulid.ID,
) *customer.Customer {
	t.Helper()

	entity, err := f.repo.GetByID(tenant.ctx, customerports.GetCustomerByIDRequest{
		ID:         customerID,
		TenantInfo: tenant.info,
		CustomerFilterOptions: customerports.CustomerFilterOptions{
			IncludeTrackingProfile: true,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, entity.TrackingProfile)

	return entity
}

func TestGetByIDReadsDefaultTrackingSettingsUntilTheyAreSaved(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "TRK1"))
	require.NoError(t, err)
	assert.Nil(t, created.TrackingProfile, "a create without settings writes none")

	stored := f.withTracking(t, f.a, created.ID)

	assert.True(t, stored.TrackingProfile.ID.IsNil())
	assert.Equal(t, created.ID, stored.TrackingProfile.CustomerID)
	assert.True(t, stored.TrackingProfile.LinkEnabled)
	assert.Equal(t, trackingprofile.DefaultFields(), stored.TrackingProfile.VisibleFields)
	assert.Equal(t, trackingprofile.DefaultLinkTTLDays, stored.TrackingProfile.LinkTTLDays)
}

func TestUpdateSavesTheTrackingSettingsWithTheCustomer(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "TRK2"))
	require.NoError(t, err)

	stored := f.withTracking(t, f.a, created.ID)
	enabled := int64(1_800_000_000)
	stored.Name = "Tracked Customer"
	stored.TrackingProfile.VisibleFields = []trackingprofile.Field{
		trackingprofile.FieldStatus,
		trackingprofile.FieldProofOfDelivery,
	}
	stored.TrackingProfile.LinkTTLDays = 14
	stored.TrackingProfile.NotifyPickedUp = true
	stored.TrackingProfile.NotificationsEnabledAt = &enabled
	stored.TrackingProfile.Recipients = "ops@example.com"

	updated, err := f.repo.Update(f.a.ctx, stored)
	require.NoError(t, err)
	assert.Equal(t, "Tracked Customer", updated.Name)
	require.False(t, updated.TrackingProfile.ID.IsNil())
	profileID := updated.TrackingProfile.ID

	read := f.withTracking(t, f.a, created.ID)
	assert.Equal(t, profileID, read.TrackingProfile.ID)
	assert.Equal(t, []trackingprofile.Field{
		trackingprofile.FieldStatus,
		trackingprofile.FieldProofOfDelivery,
	}, read.TrackingProfile.VisibleFields)
	assert.Equal(t, int16(14), read.TrackingProfile.LinkTTLDays)
	assert.True(t, read.TrackingProfile.NotifyPickedUp)
	assert.Equal(t, enabled, *read.TrackingProfile.NotificationsEnabledAt)
	assert.Equal(t, "ops@example.com", read.TrackingProfile.Recipients)

	read.TrackingProfile.NotifyPickedUp = false
	read.TrackingProfile.NotificationsEnabledAt = nil
	read.TrackingProfile.LinkEnabled = false
	_, err = f.repo.Update(f.a.ctx, read)
	require.NoError(t, err)

	again := f.withTracking(t, f.a, created.ID)
	assert.Equal(t, profileID, again.TrackingProfile.ID, "a second save updates the same row")
	assert.False(t, again.TrackingProfile.LinkEnabled)
	assert.False(t, again.TrackingProfile.NotifyPickedUp)
	assert.Nil(t, again.TrackingProfile.NotificationsEnabledAt)

	ctp := trackingprofiledb.Profiles
	count, err := ctp.Where(ctp.CustomerID.Eq(created.ID)).Count(f.a.ctx, f.admin)
	require.NoError(t, err)
	assert.Equal(t, int64(1), count)
}

func TestUpdateWithoutTrackingSettingsLeavesThemAlone(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "TRK3"))
	require.NoError(t, err)

	stored := f.withTracking(t, f.a, created.ID)
	stored.TrackingProfile.LinkEnabled = false
	stored.TrackingProfile.IncludeLink = false
	saved, err := f.repo.Update(f.a.ctx, stored)
	require.NoError(t, err)

	saved.TrackingProfile = nil
	saved.Name = "Renamed Without Tracking"
	_, err = f.repo.Update(f.a.ctx, saved)
	require.NoError(t, err)

	read := f.withTracking(t, f.a, created.ID)
	assert.False(t, read.TrackingProfile.LinkEnabled, "false is stored, not the column default")
	assert.False(t, read.TrackingProfile.IncludeLink)
}

func TestTheDatabaseRefusesTrackingRulesWithoutAStartTime(t *testing.T) {
	f := newCustomerFixture(t)
	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "TRK4"))
	require.NoError(t, err)

	stored := f.withTracking(t, f.a, created.ID)
	stored.TrackingProfile.NotifyDelivered = true
	stored.TrackingProfile.NotificationsEnabledAt = nil

	_, err = f.repo.Update(f.a.ctx, stored)
	require.Error(t, err)
}

func TestTrackingProfileReaderDefaultsAndStaysInsideTheTenant(t *testing.T) {
	f := newCustomerFixture(t)
	reader := NewTrackingProfileReader(Params{DB: f.conn, Logger: zap.NewNop()})
	created, err := f.repo.Create(f.a.ctx, newCustomer(f.a, "TRK5"))
	require.NoError(t, err)

	defaults, err := reader.GetByCustomer(f.a.ctx, f.a.info, created.ID)
	require.NoError(t, err)
	assert.True(t, defaults.ID.IsNil())
	assert.True(t, defaults.LinkEnabled)

	stored := f.withTracking(t, f.a, created.ID)
	stored.TrackingProfile.LinkTTLDays = 7
	_, err = f.repo.Update(f.a.ctx, stored)
	require.NoError(t, err)

	saved, err := reader.GetByCustomer(f.a.ctx, f.a.info, created.ID)
	require.NoError(t, err)
	assert.False(t, saved.ID.IsNil())
	assert.Equal(t, int16(7), saved.LinkTTLDays)

	other, err := reader.GetByCustomer(f.b.ctx, f.b.info, created.ID)
	require.NoError(t, err)
	assert.True(t, other.ID.IsNil(), "another tenant never sees the saved settings")
	assert.Equal(t, trackingprofile.DefaultLinkTTLDays, other.LinkTTLDays)
}
