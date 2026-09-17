package carrierok_test

import (
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fullProfileFixture = "profile_265752.json"
	liteProfileFixture = "profile_lite_265752.json"
)

func loadFixtureProfile(t *testing.T, name string) (*carrierok.Profile, []byte) {
	t.Helper()

	item := fixture(t, name)
	profile, err := carrierok.DecodeProfile(item)
	require.NoError(t, err)
	return profile, item
}

func unixDate(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

func TestDecodeProfileIdentityAndAuthority(t *testing.T) {
	t.Parallel()

	profile, item := loadFixtureProfile(t, fullProfileFixture)

	assert.JSONEq(t, string(item), string(profile.Raw))
	assert.Equal(t, "265752", profile.Identity.DOTNumber.Value())
	assert.Equal(t, "MC", profile.Identity.DocketPrefix.Value())
	assert.Equal(t, "179059", profile.Identity.DocketNumber.Value())
	assert.Equal(t, "119133494", profile.Identity.DUNS.Value())
	assert.True(t, profile.Identity.DBAFlag.Value())
	assert.Equal(t, int64(14936), profile.Identity.DOTAgeDays.Value())
	assert.Equal(t, unixDate(1985, time.October, 25), *profile.Identity.AddedDate.Unix())
	assert.Equal(t, unixDate(2026, time.September, 16), *profile.Identity.SnapshotDate.Unix())

	authority := profile.Authority
	assert.Equal(t, "Inactive", authority.Common.Value())
	require.NotNil(t, authority.BrokerPending)
	assert.False(t, authority.BrokerPending.Value())
	require.NotNil(t, authority.CommonRevocation)
	assert.False(t, authority.CommonRevocation.Value())
	assert.Equal(t, int64(6882), authority.AgeCommonDays.Value())
	assert.Equal(t, int64(6735), authority.AgeBrokerDays.Value())
	assert.Equal(t, unixDate(2006, time.May, 10), *authority.StartBroker.Unix())
	assert.Equal(t, int64(5), authority.TotalRevocations.Value())
	assert.Equal(t, int64(699), authority.DaysSinceLastRevocation.Value())
	assert.Equal(t, unixDate(2024, time.October, 17), *authority.LastRevocationDate.Unix())

	require.Len(t, authority.History, 5)
	broker := authority.History[0]
	assert.Equal(t, "Property Broker", broker.AuthorityType.Value())
	assert.Equal(t, "Granted", broker.OriginalAction.Value())
	assert.Equal(t, unixDate(2006, time.May, 10), *broker.OriginalServedDate.Unix())
	assert.Equal(t, "Revoked", broker.DispositionAction.Value())
	assert.Equal(t, unixDate(2024, time.October, 17), *broker.DispositionServedDate.Unix())
	assert.NotEmpty(t, broker.Raw)
}

func TestDecodeProfileInsurance(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t, fullProfileFixture)

	insurance := profile.Insurance
	assert.Zero(t, insurance.BIPDOnFile.Value())
	require.NotNil(t, insurance.BIPDOnFile)
	assert.InDelta(t, 5_000_000, insurance.BIPDRequired.Value(), 1e-9)
	assert.InDelta(t, 75_000, insurance.BondRequired.Value(), 1e-9)
	assert.Nil(t, insurance.CargoOnFile)
	require.NotNil(t, insurance.Indicator)
	assert.False(t, insurance.Indicator.Value())
	assert.Equal(t, int64(5), insurance.CancelCount.Value())
	assert.Equal(t, unixDate(2025, time.June, 23), *insurance.LastCanceled.Unix())
	assert.Nil(t, insurance.PendingCancelDate)

	require.Len(t, insurance.History, 4)
	cargo := insurance.History[0]
	assert.Equal(t, "A", cargo.Status.Value())
	assert.Equal(t, "34", cargo.FormCode.Value())
	assert.Equal(t, "CARGO", cargo.Type.Value())
	assert.Equal(t, "OLD REPUBLIC INSURANCE COMPANY", cargo.Insurer.Value())
	assert.Equal(t, "MWE 315778", cargo.PolicyNumber.Value())
	assert.InDelta(t, 5000, cargo.Coverage.Value(), 1e-9)
	assert.Zero(t, cargo.UnderlyingLimit.Value())
	assert.Equal(t, unixDate(2020, time.October, 1), *cargo.EffectiveDate.Unix())
	assert.Equal(t, unixDate(2021, time.October, 1), *cargo.NextRenewalDate.Unix())
	assert.Nil(t, cargo.CancelEffectiveDate)
	assert.Nil(t, cargo.CancelMethod)

	bipd := insurance.History[2]
	assert.Equal(t, "H", bipd.Status.Value())
	assert.Equal(t, "BIPD", bipd.Type.Value())
	assert.InDelta(t, 5_000_000, bipd.Coverage.Value(), 1e-9)
	assert.Equal(t, "CANCELLED", bipd.CancelMethod.Value())
	assert.Equal(t, unixDate(2024, time.November, 30), *bipd.CancelEffectiveDate.Unix())
}

func TestDecodeProfileSafetyAndBasics(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t, fullProfileFixture)

	safety := profile.Safety
	assert.Equal(t, int64(69), safety.ISSValue.Value())
	assert.Equal(t, "High", safety.RiskScore.Value())
	assert.InDelta(t, 0.44334646087254104, safety.RiskScoreProbability.Value(), 1e-12)
	assert.Equal(t, "Satisfactory", safety.SafetyRating.Value())
	assert.Equal(t, unixDate(1996, time.September, 25), *safety.SafetyRatingDate.Unix())
	require.NotNil(t, safety.OutOfServiceFlag)
	assert.False(t, safety.OutOfServiceFlag.Value())
	assert.True(t, safety.IndicatorCarrierSafety.Value())

	require.Len(t, profile.Basics, len(carrierok.BasicCategories()))

	unsafe := profile.Basics[carrierok.BasicUnsafeDriving]
	assert.InDelta(t, 0.13, unsafe.Measure.Value(), 1e-9)
	require.NotNil(t, unsafe.ACIndicator)
	assert.False(t, unsafe.ACIndicator.Value())
	assert.Equal(t, unixDate(2024, time.December, 31), *unsafe.MeasuredAt.Unix())
	require.NotNil(t, unsafe.Alert)
	assert.False(t, unsafe.Alert.Value())
	require.NotNil(t, unsafe.Violations)
	assert.Zero(t, unsafe.Violations.Value())
	require.NotNil(t, unsafe.OOSViolations)

	maintenance := profile.Basics[carrierok.BasicVehicleMaintenance]
	assert.InDelta(t, 3.11, maintenance.Measure.Value(), 1e-9)
	require.NotNil(t, maintenance.Alert)

	crash := profile.Basics[carrierok.BasicCrashIndicator]
	assert.Nil(t, crash.Measure)
	assert.Nil(t, crash.MeasuredAt)
	require.NotNil(t, crash.Alert)
	assert.Nil(t, crash.Violations)
}

func TestDecodeProfileBasicAlertsAndHistoryOrdering(t *testing.T) {
	t.Parallel()

	profile, err := carrierok.DecodeProfile([]byte(`{
		"basic_alert_hours_of_service": true,
		"violations_hours_of_service": "12",
		"violations_oos_hours_of_service": "3",
		"basic_alert_vehicle_maintence": true,
		"basic_history": [
			{"snapshot_date": "2025-01-31", "basic_measure_hours_of_service": 4.2},
			{"snapshot_date": "2025-03-31", "basic_measure_hours_of_service": 6.8,
				"basic_ac_indicator_hours_of_service": true},
			{"snapshot_date": "2025-02-28", "basic_measure_hours_of_service": 5.1}
		]
	}`))
	require.NoError(t, err)

	hos := profile.Basics[carrierok.BasicHoursOfService]
	assert.True(t, hos.Alert.Value())
	assert.Equal(t, int64(12), hos.Violations.Value())
	assert.Equal(t, int64(3), hos.OOSViolations.Value())
	assert.InDelta(t, 6.8, hos.Measure.Value(), 1e-9)
	assert.True(t, hos.ACIndicator.Value())
	assert.Equal(t, unixDate(2025, time.March, 31), *hos.MeasuredAt.Unix())

	assert.True(t, profile.Basics[carrierok.BasicVehicleMaintenance].Alert.Value())
	_, ok := profile.Basics[carrierok.BasicControlledSubstance]
	assert.False(t, ok)

	_, err = carrierok.DecodeProfile([]byte(`[1,2]`))
	require.ErrorIs(t, err, carrierok.ErrUnexpectedPayload)
}

func TestDecodeProfileInspectionsFleetAndOperations(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t, fullProfileFixture)

	inspections := profile.Inspections
	assert.Equal(t, int64(15015), inspections.Total.Value())
	assert.Equal(t, int64(14947), inspections.Driver.Value())
	assert.Equal(t, int64(1662), inspections.VehicleOutOfService.Value())
	assert.InDelta(t, 0.019, inspections.DriverOutOfServiceRate.Value(), 1e-9)
	assert.InDelta(t, 0.183, inspections.VehicleOutOfServiceRate.Value(), 1e-9)
	assert.InDelta(t, 0.2226, inspections.NationalAvgOOSVehicle.Value(), 1e-9)
	assert.False(t, inspections.OOSAlertHazmat.Value())

	assert.Equal(t, int64(123), profile.Crashes.Total.Value())
	assert.Equal(t, int64(47), profile.Crashes.Injuries.Value())
	assert.Equal(t, unixDate(2026, time.August, 24), *profile.Crashes.LastCrashDate.Unix())

	fleet := profile.Fleet
	assert.Equal(t, int64(101844), fleet.TotalPowerUnits.Value())
	assert.Equal(t, int64(30014), fleet.TotalDriversCDL.Value())
	assert.Equal(t, int64(18363), fleet.TermLeasedTractors.Value())
	assert.Equal(t, int64(79704), fleet.OwnedTrailers.Value())
	assert.Nil(t, fleet.OwnedTractors)
	assert.Empty(t, fleet.Equipment)

	assert.Nil(t, profile.Contacts.Cellphone)
	assert.Equal(t, "4128595045", profile.Contacts.Fax.Value())
	assert.Equal(t, "CLARENCE DOZIER", profile.Contacts.PrimaryContact.Value())

	physical := profile.Addresses.Physical
	assert.Equal(t, "38125", physical.ZipCode.Value())
	assert.Equal(t, "US", physical.CountryCode.Value())
	require.NotNil(t, physical.Undeliverable)
	assert.False(t, physical.Undeliverable.Value())
	assert.Equal(t, "1000 FEDEX DRIVE", profile.Addresses.Mailing.Street.Value())
	assert.Equal(t, "US", profile.Addresses.Mailing.CountryCode.Value())

	operations := profile.Operations
	assert.Equal(t, []string{"General Freight", "Other"}, operations.CargoCarried)
	assert.Equal(t, []string{"Authorized For Hire"}, operations.OperationClassification)
	assert.True(t, operations.SmartWay.Value())
	assert.True(t, operations.HazardousMaterial.Value())
	require.NotNil(t, operations.CARBTRU)
	assert.False(t, operations.CARBTRU.Value())
	assert.Equal(t, int64(3_279_390_000), operations.MCS150Mileage.Value())
	assert.Equal(t, int64(2024), operations.MCS150Year.Value())
	assert.Equal(t, "CT CORPORATION SYSTEM", operations.BOC3CompanyName.Value())

	boc3 := profile.RiskFactor("boc3_on_file")
	require.NotNil(t, boc3)
	assert.True(t, *boc3)
	assert.Nil(t, profile.RiskFactor("not_a_flag"))
}

func TestDecodeProfileHistoryNetworkAndBenchmarks(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t, fullProfileFixture)

	history := profile.ChangeHistory
	assert.Zero(t, history.NameChangeCount.Value())
	assert.Nil(t, history.NameLastChanged)
	assert.Equal(t, int64(2), history.EmailChangeCount.Value())
	assert.Equal(t, unixDate(2019, time.June, 27), *history.PhoneLastChanged.Unix())
	assert.Equal(t, int64(3), history.AddressChangeCount.Value())
	assert.Equal(t, unixDate(2022, time.December, 8), *history.ContactLastChanged.Unix())

	network := profile.Network
	assert.True(t, network.IndicatorContact.Value())
	assert.False(t, network.IndicatorEquipment.Value())
	assert.Equal(t, int64(4), network.MailingAddressCount.Value())
	assert.Equal(t, int64(1), network.FaxNumberCount.Value())
	assert.Zero(t, network.PowerUnitsCount.Value())
	assert.Equal(t, []carrierok.NetworkLink{
		{Kind: carrierok.NetworkLinkPhysicalAddress, DOTNumber: "86876"},
		{Kind: carrierok.NetworkLinkMailingAddress, DOTNumber: "1964900"},
		{Kind: carrierok.NetworkLinkMailingAddress, DOTNumber: "4356378"},
		{Kind: carrierok.NetworkLinkMailingAddress, DOTNumber: "598081"},
		{Kind: carrierok.NetworkLinkMailingAddress, DOTNumber: "86876"},
		{Kind: carrierok.NetworkLinkTelephone, DOTNumber: "86876"},
		{Kind: carrierok.NetworkLinkTelephone, DOTNumber: "598081"},
		{Kind: carrierok.NetworkLinkFax, DOTNumber: "598081"},
		{Kind: carrierok.NetworkLinkEmail, DOTNumber: "598081"},
		{Kind: carrierok.NetworkLinkEIN, DOTNumber: "3859536"},
		{Kind: carrierok.NetworkLinkDUNS, DOTNumber: "3859536"},
	}, network.Links)

	assert.Empty(t, profile.Lanes.PreferredStates)

	benchmarks := profile.Benchmarks
	require.NotNil(t, benchmarks.IndicatorIndustry)
	assert.False(t, benchmarks.IndicatorIndustry.Value())
	require.NotNil(t, benchmarks.PowerUnitMileageRatio)
	assert.False(t, benchmarks.PowerUnitMileageRatio.Value())
}

func TestDecodeProfilePreferredStates(t *testing.T) {
	t.Parallel()

	profile, err := carrierok.DecodeProfile([]byte(`{
		"preferred_lanes": [{"state": "TX"}, {"state": ""}, {"state": "IL"}],
		"preferred_states": "IL, GA"
	}`))
	require.NoError(t, err)
	assert.Equal(t, []string{"TX", "IL", "GA"}, profile.Lanes.PreferredStates)
}

func TestDecodeLiteProfile(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t, liteProfileFixture)

	assert.Equal(t, "265752-MC179059", profile.ProfileID())
	assert.Equal(t, int64(6882), profile.Authority.AgeCommonDays.Value())
	assert.Equal(t, "High", profile.Safety.RiskScore.Value())
	assert.Nil(t, profile.Safety.RiskScoreProbability)
	assert.Nil(t, profile.Identity.DOTAgeDays)
	assert.Nil(t, profile.Insurance.BIPDRequired)
	assert.Empty(t, profile.Authority.History)
	assert.Empty(t, profile.Insurance.History)
	assert.Empty(t, profile.Basics)
	assert.Empty(t, profile.Network.Links)
	assert.Nil(t, profile.ChangeHistory.EmailChangeCount)
	require.NotNil(t, profile.RiskFactor("boc3_on_file"))
}

func TestProfileSonicRoundTrip(t *testing.T) {
	t.Parallel()

	var profile carrierok.Profile
	require.NoError(t, sonic.Unmarshal([]byte(`{"dot_number":"42","total_drivers":"7"}`), &profile))
	assert.Equal(t, "42", profile.Identity.DOTNumber.Value())
	assert.Equal(t, int64(7), profile.Fleet.TotalDrivers.Value())

	encoded, err := sonic.Marshal(&profile)
	require.NoError(t, err)
	assert.JSONEq(t, `{"dot_number":"42","total_drivers":"7"}`, string(encoded))

	var value jsonflex.Int
	require.NoError(t, sonic.Unmarshal([]byte(`"7"`), &value))
	assert.Equal(t, int64(7), value.Value())
}

func TestBasicVendorKeys(t *testing.T) {
	t.Parallel()

	assert.Equal(
		t,
		"vehicle_maintence",
		carrierok.BasicVendorKey(carrierok.BasicVehicleMaintenance),
	)
	assert.Equal(t, "unsafe_driving", carrierok.BasicVendorKey(carrierok.BasicUnsafeDriving))

	category, ok := carrierok.BasicCategoryForVendorKey("vehicle_maintence")
	require.True(t, ok)
	assert.Equal(t, carrierok.BasicVehicleMaintenance, category)

	_, ok = carrierok.BasicCategoryForVendorKey("total")
	assert.False(t, ok)
}

func TestProfileIDHelpers(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "568253-MC277621", carrierok.ProfileID("568253", "MC", "277621"))
	assert.Equal(t, "568253-MC277621", carrierok.ProfileID(" 568253 ", "mc", "MC277621"))
	assert.Equal(t, "568253-FF12", carrierok.ProfileID("568253", "FF", "12"))
	assert.Equal(t, "568253", carrierok.ProfileID("568253", "MC", ""))

	dot, docket := carrierok.SplitProfileID("568253-MC277621")
	assert.Equal(t, "568253", dot)
	assert.Equal(t, "MC277621", docket)

	dot, docket = carrierok.SplitProfileID("568253")
	assert.Equal(t, "568253", dot)
	assert.Empty(t, docket)
}
