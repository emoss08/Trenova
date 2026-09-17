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

func loadFixtureProfile(t *testing.T) (*carrierok.Profile, []byte) {
	t.Helper()

	var envelope struct {
		Items []sonic.NoCopyRawMessage `json:"items"`
	}
	require.NoError(t, sonic.Unmarshal(fixture(t, "profile_818175.json"), &envelope))
	require.Len(t, envelope.Items, 1)

	item := []byte(envelope.Items[0])
	profile, err := carrierok.DecodeProfile(item)
	require.NoError(t, err)
	return profile, item
}

func unixDate(year int, month time.Month, day int) int64 {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Unix()
}

func TestDecodeProfileIdentityAndAuthority(t *testing.T) {
	t.Parallel()

	profile, item := loadFixtureProfile(t)

	assert.JSONEq(t, string(item), string(profile.Raw))
	assert.Equal(t, "818175", profile.Identity.DOTNumber.Value())
	assert.Equal(t, "MC", profile.Identity.DocketPrefix.Value())
	assert.True(t, profile.Identity.DBAFlag.Value())
	assert.InDelta(t, 31.2, profile.Identity.DOTAge.Value(), 1e-9)
	assert.Equal(t, unixDate(2026, time.September, 10), *profile.Identity.SnapshotDate.Unix())

	assert.Equal(t, "ACTIVE", profile.Authority.Common.Value())
	assert.Nil(t, profile.Authority.BrokerPending)
	assert.Nil(t, profile.Authority.AgeBroker)
	assert.Equal(t, int64(2), profile.Authority.TotalRevocations.Value())
	require.Len(t, profile.Authority.History, 2)
	assert.Equal(t, "CONTRACT", profile.Authority.History[1].AuthorityType.Value())
	assert.Equal(t, "REVOKED", profile.Authority.History[1].Action.Value())
	assert.Equal(
		t,
		unixDate(2015, time.August, 3),
		*profile.Authority.History[1].ServedDate.Unix(),
	)
	assert.NotEmpty(t, profile.Authority.History[0].Raw)
}

func TestDecodeProfileFlexibleInsurance(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t)

	assert.InDelta(t, 750000, profile.Insurance.BIPDOnFile.Value(), 1e-9)
	assert.InDelta(t, 750000, profile.Insurance.BIPDRequired.Value(), 1e-9)
	assert.InDelta(t, 100000, profile.Insurance.CargoOnFile.Value(), 1e-9)
	assert.Nil(t, profile.Insurance.CargoRequired)
	assert.Nil(t, profile.Insurance.BondRequired)
	require.NotNil(t, profile.Insurance.BondOnFile)
	assert.Zero(t, profile.Insurance.BondOnFile.Value())
	assert.Nil(t, profile.Insurance.PendingCancelDate)
	assert.Equal(t, unixDate(2024, time.March, 15), *profile.Insurance.LastCanceled.Unix())

	require.Len(t, profile.Insurance.History, 2)
	cargo := profile.Insurance.History[1]
	assert.Equal(t, "CARGO", cargo.Type.Value())
	assert.Equal(t, "NORTHLAND INSURANCE", cargo.Insurer.Value())
	assert.InDelta(t, 100000, cargo.Coverage.Value(), 1e-9)
	assert.Equal(t, int64(1577836800), *cargo.EffectiveDate.Unix())
	assert.Equal(t, unixDate(2024, time.March, 15), *cargo.CancelEffectiveDate.Unix())
	assert.Equal(t, "REPLACED", cargo.CancelMethod.Value())
	assert.InDelta(t, 750000, profile.Insurance.History[0].Coverage.Value(), 1e-9)
}

func TestDecodeProfileSafetyAndBasics(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t)

	assert.InDelta(t, 48, profile.Safety.ISSValue.Value(), 1e-9)
	assert.Equal(t, "Satisfactory", profile.Safety.SafetyRating.Value())
	require.NotNil(t, profile.Safety.OutOfServiceFlag)
	assert.False(t, profile.Safety.OutOfServiceFlag.Value())
	assert.Nil(t, profile.Safety.OutOfServiceDate)

	require.Len(t, profile.Basics, 3)

	unsafe := profile.Basics[carrierok.BasicUnsafeDriving]
	assert.InDelta(t, 1.23, unsafe.Measure.Value(), 1e-9)
	assert.InDelta(t, 0.41, unsafe.Percentile.Value(), 1e-9)
	require.NotNil(t, unsafe.Alert)
	assert.False(t, unsafe.Alert.Value())
	require.NotNil(t, unsafe.RoadsideAlert)
	assert.False(t, unsafe.RoadsideAlert.Value())
	assert.InDelta(t, 0.65, unsafe.InterventionThreshold.Value(), 1e-9)

	hos := profile.Basics[carrierok.BasicHoursOfService]
	assert.True(t, hos.Alert.Value())
	assert.InDelta(t, 0.72, hos.Percentile.Value(), 1e-9)
	assert.Nil(t, hos.ACIndicator)

	maintenance, ok := profile.Basics[carrierok.BasicVehicleMaintenance]
	require.True(t, ok)
	assert.InDelta(t, 5.1, maintenance.Measure.Value(), 1e-9)
	assert.True(t, maintenance.Alert.Value())

	_, ok = profile.Basics[carrierok.BasicControlledSubstance]
	assert.False(t, ok)
}

func TestDecodeProfileInspectionsFleetAndOperations(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t)

	assert.Equal(t, int64(120), profile.Inspections.Driver.Value())
	assert.Equal(t, int64(17), profile.Inspections.VehicleOutOfService.Value())
	assert.InDelta(t, 2.5, profile.Inspections.DriverOutOfServicePct.Value(), 1e-9)
	assert.InDelta(t, 22.26, profile.Inspections.NationalAvgOOSVehicle.Value(), 1e-9)
	assert.False(t, profile.Inspections.OOSAlertHazmat.Value())

	assert.Equal(t, int64(4), profile.Crashes.Total.Value())
	assert.Equal(t, int64(1), profile.Crashes.Injuries.Value())

	assert.Equal(t, int64(85), profile.Fleet.TotalPowerUnits.Value())
	assert.JSONEq(t, `{"tractors":85,"trailers":150}`, string(profile.Fleet.EquipmentSummary))
	require.Len(t, profile.Fleet.Equipment, 2)
	assert.Equal(t, int64(2018), profile.Fleet.Equipment[0].Year.Value())
	assert.Equal(t, "TRAILER", profile.Fleet.Equipment[1].UnitType.Value())
	assert.Equal(t, int64(2019), profile.Fleet.Equipment[1].Year.Value())
	assert.Nil(t, profile.Fleet.Equipment[1].PlateNumber)

	assert.Nil(t, profile.Contacts.Fax)
	assert.Nil(t, profile.Contacts.SecondaryContact)
	assert.Equal(t, "JANE DOE", profile.Contacts.PrimaryContact.Value())

	assert.Equal(t, "60601", profile.Addresses.Physical.ZipCode.Value())
	assert.False(t, profile.Addresses.Physical.Undeliverable.Value())
	assert.Nil(t, profile.Addresses.Mailing.Full)
	assert.Equal(t, "PO BOX 42", profile.Addresses.Mailing.Street.Value())

	assert.Equal(
		t,
		[]string{"General Freight", "Refrigerated Food"},
		profile.Operations.CargoCarried,
	)
	assert.Equal(
		t,
		[]string{"Authorized For Hire", "Private Property"},
		profile.Operations.OperationClassification,
	)
	assert.True(t, profile.Operations.SmartWay.Value())
	assert.True(t, profile.Operations.CARBTRU.Value())
	require.NotNil(t, profile.Operations.PHMSA)
	assert.False(t, profile.Operations.PHMSA.Value())
	assert.Equal(t, int64(9500000), profile.Operations.MCS150Mileage.Value())
	assert.Equal(t, int64(2024), profile.Operations.MCS150Year.Value())
}

func TestDecodeProfileHistoryNetworkLoads(t *testing.T) {
	t.Parallel()

	profile, _ := loadFixtureProfile(t)

	assert.Nil(t, profile.ChangeHistory.NameLastChanged)
	assert.Equal(t, int64(2), profile.ChangeHistory.EmailChangeCount.Value())
	assert.Equal(t, unixDate(2026, time.January, 1), *profile.ChangeHistory.PhoneLastChanged.Unix())
	assert.Equal(t, int64(1767225600), *profile.ChangeHistory.ContactLastChanged.Unix())
	assert.Nil(t, profile.ChangeHistory.AddressLastChanged)

	assert.Equal(t, "YELLOW", profile.Network.IndicatorContact.Value())
	assert.Equal(t, int64(2), profile.Network.PhysicalAddressCount.Value())
	assert.Nil(t, profile.Network.DUNSCount)
	assert.Equal(t, []carrierok.NetworkLink{
		{
			Kind:      carrierok.NetworkLinkPhysicalAddress,
			DOTNumber: "3456789",
			LegalName: "SHADOW HAULING LLC",
			Value:     "100 MAIN ST, CHICAGO, IL 60601",
			Status:    "INACTIVE",
		},
		{Kind: carrierok.NetworkLinkTelephone, Value: "3125550100"},
		{
			Kind:      carrierok.NetworkLinkEquipment,
			DOTNumber: "2233445",
			LegalName: "OTHER CARRIER",
			Value:     "1XKYD49X0JJ123456",
			Status:    "ACTIVE",
		},
	}, profile.Network.Links)

	assert.Equal(t, int64(1532), profile.Loads.Total.Value())
	assert.InDelta(t, 97.91, profile.Loads.FTLPercentage.Value(), 1e-9)
	require.Len(t, profile.Loads.PreferredLanes, 2)
	assert.Equal(t, int64(210), profile.Loads.PreferredLanes[0].Loads.Value())
	assert.Equal(t, int64(98), profile.Loads.PreferredLanes[1].Loads.Value())
	assert.Equal(t, "DALLAS", profile.Loads.PreferredLanes[0].DestinationCity.Value())

	assert.Equal(t, "YELLOW", profile.Benchmarks.InspectedPowerUnitsRatio.Value())
	assert.Equal(t, "1.02", profile.Benchmarks.PowerUnitMileageRatio.Value())
}

func TestDecodeProfileAcceptsCorrectBasicSpellingAndRejectsNonObject(t *testing.T) {
	t.Parallel()

	profile, err := carrierok.DecodeProfile(
		[]byte(
			`{"basic_percentile_vehicle_maintenance":"0.5","basic_alert_vehicle_maintenance":"Y"}`,
		),
	)
	require.NoError(t, err)
	score := profile.Basics[carrierok.BasicVehicleMaintenance]
	assert.InDelta(t, 0.5, score.Percentile.Value(), 1e-9)
	assert.True(t, score.Alert.Value())

	_, err = carrierok.DecodeProfile([]byte(`[1,2]`))
	require.ErrorIs(t, err, carrierok.ErrUnexpectedPayload)
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
