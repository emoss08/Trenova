package carrierokconnector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func fixtureProfile(t *testing.T) *carrierok.Profile {
	t.Helper()
	obj, err := jsonflex.DecodeObject(fixture(t, "profile_818175.json"))
	require.NoError(t, err)
	items, ok := obj.Array("items")
	require.True(t, ok)
	require.Len(t, items, 1)
	profile, err := carrierok.DecodeProfile(items[0])
	require.NoError(t, err)
	return profile
}

func unixDate(t *testing.T, value string) int64 {
	t.Helper()
	parsed, err := time.ParseInLocation("2006-01-02", value, time.UTC)
	require.NoError(t, err)
	return parsed.Unix()
}

func TestNormalizeProfile_Identity(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t))
	require.NotNil(t, profile.Identity)

	identity := profile.Identity
	assert.Equal(t, "818175", identity.DOTNumber)
	assert.Equal(t, "MC", identity.DocketPrefix)
	assert.Equal(t, "277621", identity.DocketNumber)
	assert.Equal(t, "818175-MC277621", identity.ProviderRef())
	assert.Equal(t, "SANDBOX FREIGHT LINES INC", identity.LegalName)
	assert.Equal(t, "SANDBOX FREIGHT", identity.DBAName)
	assert.Equal(t, "364123456", identity.EIN)
	assert.Equal(t, "ACTIVE", identity.USDOTStatus)
	assert.Equal(t, "CARRIER", identity.EntityType)
	assert.Equal(t, "Interstate", identity.CarrierOperation)
	require.NotNil(t, identity.DOTAddedAt)
	assert.Equal(t, unixDate(t, "1994-06-01"), *identity.DOTAddedAt)
	require.NotNil(t, identity.DOTAgeDays)
	assert.Equal(t, 11396, *identity.DOTAgeDays)

	require.NotNil(t, identity.PhysicalAddress)
	assert.Equal(t, "100 MAIN ST", identity.PhysicalAddress.Line1)
	assert.Equal(t, "CHICAGO", identity.PhysicalAddress.City)
	assert.Equal(t, "IL", identity.PhysicalAddress.State)
	assert.Equal(t, "60601", identity.PhysicalAddress.PostalCode)
	assert.Equal(t, "US", identity.PhysicalAddress.Country)
	require.NotNil(t, identity.PhysicalAddress.Undelivered)
	assert.False(t, *identity.PhysicalAddress.Undelivered)
	require.NotNil(t, identity.MailingAddress)
	assert.Equal(t, "PO BOX 42", identity.MailingAddress.Line1)
}

func TestNormalizeProfile_AuthorityAndInsurance(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t))
	require.NotNil(t, profile.Authority)

	authority := profile.Authority
	require.NotNil(t, authority.Common)
	assert.Equal(t, carrierintel.AuthorityStatusActive, authority.Common.Status)
	assert.False(t, authority.Common.Pending)
	assert.False(t, authority.Common.RevocationPending)
	require.NotNil(t, authority.Common.AgeDays)
	assert.Equal(t, 11200, *authority.Common.AgeDays)
	require.NotNil(t, authority.Common.GrantedAt)
	assert.Equal(t, unixDate(t, "1995-02-14"), *authority.Common.GrantedAt)
	require.NotNil(t, authority.Contract)
	assert.Equal(t, carrierintel.AuthorityStatusInactive, authority.Contract.Status)
	require.NotNil(t, authority.Broker)
	assert.Equal(t, carrierintel.AuthorityStatusNone, authority.Broker.Status)
	require.NotNil(t, authority.TotalRevocations)
	assert.Equal(t, 2, *authority.TotalRevocations)
	require.NotNil(t, authority.LastRevocationAt)
	assert.Equal(t, unixDate(t, "2015-08-03"), *authority.LastRevocationAt)
	require.Len(t, authority.History, 2)
	assert.Equal(t, "COMMON", authority.History[0].AuthorityType)
	assert.Equal(t, "REVOKED", authority.History[1].Action)

	insurance := profile.Insurance
	require.NotNil(t, insurance)
	require.NotNil(t, insurance.BIPDOnFile)
	assert.Equal(t, "750000", insurance.BIPDOnFile.String())
	require.NotNil(t, insurance.BIPDRequired)
	assert.Equal(t, "750000", insurance.BIPDRequired.String())
	require.NotNil(t, insurance.CargoOnFile)
	assert.Equal(t, "100000", insurance.CargoOnFile.String())
	assert.Nil(t, insurance.CargoRequired)
	require.NotNil(t, insurance.BondOnFile)
	assert.True(t, insurance.BondOnFile.IsZero())
	assert.Nil(t, insurance.BondRequired)
	assert.Nil(t, insurance.PendingCancelAt)
	require.NotNil(t, insurance.CancelCount)
	assert.Equal(t, 1, *insurance.CancelCount)
	require.NotNil(t, insurance.LastCanceledAt)
	assert.Equal(t, unixDate(t, "2024-03-15"), *insurance.LastCanceledAt)
	require.Len(t, insurance.Filings, 2)
	assert.Equal(t, carrierintel.InsuranceFilingTypeBIPD, insurance.Filings[0].Type)
	assert.Equal(t, "GREAT WEST CASUALTY COMPANY", insurance.Filings[0].InsurerName)
	assert.Equal(t, carrierintel.InsuranceFilingTypeCargo, insurance.Filings[1].Type)
	assert.Equal(t, "NORTHLAND INSURANCE", insurance.Filings[1].InsurerName)
	require.NotNil(t, insurance.Filings[1].Coverage)
	assert.Equal(t, "100000", insurance.Filings[1].Coverage.String())
	assert.Equal(t, "REPLACED", insurance.Filings[1].CancelMethod)
	require.NotNil(t, insurance.Filings[1].CancelEffectiveAt)
}

func TestNormalizeProfile_SafetyAndBasics(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t))
	safety := profile.Safety
	require.NotNil(t, safety)
	assert.Equal(t, carrierintel.SafetyRatingSatisfactory, safety.Rating)
	require.NotNil(t, safety.ISSValue)
	assert.Equal(t, 48, *safety.ISSValue)
	assert.Equal(t, "Optional", safety.ISSRecommendation)
	require.NotNil(t, safety.RiskProbability)
	assert.InDelta(t, 0.087, *safety.RiskProbability, 0.0001)
	require.NotNil(t, safety.SafetyScore)
	assert.InDelta(t, 92, *safety.SafetyScore, 0.0001)
	require.NotNil(t, safety.OutOfServiceOrder)
	assert.False(t, *safety.OutOfServiceOrder)
	assert.Nil(t, safety.OutOfServiceAt)
	assert.Equal(t, "Compliance Review", safety.LatestReviewType)

	require.Len(t, profile.Basics, 3)
	unsafe := profile.Basic(worker.BasicUnsafeDriving)
	require.NotNil(t, unsafe)
	require.NotNil(t, unsafe.Percentile)
	assert.InDelta(t, 41, *unsafe.Percentile, 0.0001)
	require.NotNil(t, unsafe.Threshold)
	assert.InDelta(t, 65, *unsafe.Threshold, 0.0001)
	assert.False(t, unsafe.Alert)

	hos := profile.Basic(worker.BasicHOSCompliance)
	require.NotNil(t, hos)
	require.NotNil(t, hos.Percentile)
	assert.InDelta(t, 72, *hos.Percentile, 0.0001)
	assert.True(t, hos.Alert)

	maintenance := profile.Basic(worker.BasicVehicleMaintenance)
	require.NotNil(t, maintenance)
	require.NotNil(t, maintenance.Percentile)
	assert.InDelta(t, 83, *maintenance.Percentile, 0.0001)
	require.NotNil(t, maintenance.Measure)
	assert.InDelta(t, 5.1, *maintenance.Measure, 0.0001)
	assert.True(t, maintenance.Alert)
}

func TestNormalizeProfile_OperationalSections(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t))

	require.NotNil(t, profile.Inspections)
	require.NotNil(t, profile.Inspections.Total)
	assert.Equal(t, 212, *profile.Inspections.Total)
	require.NotNil(t, profile.Inspections.DriverOOSRate)
	assert.InDelta(t, 2.5, *profile.Inspections.DriverOOSRate, 0.0001)
	require.NotNil(t, profile.Inspections.NationalVehicleOOS)
	assert.InDelta(t, 22.26, *profile.Inspections.NationalVehicleOOS, 0.0001)

	require.NotNil(t, profile.Crashes)
	assert.Equal(t, 4, *profile.Crashes.Total)
	assert.Equal(t, 3, *profile.Crashes.Tow)

	require.NotNil(t, profile.Fleet)
	assert.Equal(t, 85, *profile.Fleet.PowerUnits)
	assert.Equal(t, 90, *profile.Fleet.CDLDrivers)

	require.Len(t, profile.Equipment, 2)
	assert.Equal(t, carrierintel.UnitTypeTractor, profile.Equipment[0].UnitType)
	assert.Equal(t, "P123456", profile.Equipment[0].PlateNumber)
	assert.Equal(t, 2018, *profile.Equipment[0].Year)
	assert.Equal(t, carrierintel.UnitTypeTrailer, profile.Equipment[1].UnitType)

	require.NotNil(t, profile.Contacts)
	assert.Equal(t, "(312) 555-0100", profile.Contacts.Phone)
	assert.Equal(t, "dispatch@sandboxfreight.example", profile.Contacts.Email)
	assert.Equal(t, "JANE DOE", profile.Contacts.PrimaryContact)

	operations := profile.Operations
	require.NotNil(t, operations)
	assert.Equal(t, []string{"Authorized For Hire", "Private Property"}, operations.Classification)
	assert.Equal(t, []string{"General Freight", "Refrigerated Food"}, operations.CargoCarried)
	require.NotNil(t, operations.HazmatCarrier)
	assert.False(t, *operations.HazmatCarrier)
	require.NotNil(t, operations.SmartWay)
	assert.True(t, *operations.SmartWay)
	require.NotNil(t, operations.CARBCompliant)
	assert.True(t, *operations.CARBCompliant)
	require.NotNil(t, operations.MCS150Mileage)
	assert.Equal(t, int64(9_500_000), *operations.MCS150Mileage)
	require.NotNil(t, operations.BOC3OnFile)
	assert.True(t, *operations.BOC3OnFile)
	assert.Equal(t, "PROCESS AGENTS INC", operations.BOC3Agent)

	require.NotNil(t, profile.ChangeHistory)
	assert.Equal(t, 2, *profile.ChangeHistory.EmailChanges)
	assert.Equal(t, 3, profile.ChangeHistory.TotalChanges())

	network := profile.Network
	require.NotNil(t, network)
	assert.Equal(t, 2, network.SharedCount(carrierintel.NetworkKindAddress))
	assert.Equal(t, 1, network.SharedCount(carrierintel.NetworkKindPhone))
	assert.Equal(t, 3, network.SharedCount(carrierintel.NetworkKindEquipment))
	require.Len(t, network.Links, 3)
	assert.Equal(t, carrierintel.NetworkKindAddress, network.Links[0].Kind)
	assert.Equal(t, "3456789", network.Links[0].DOTNumber)

	lanes := profile.Lanes
	require.NotNil(t, lanes)
	assert.Equal(t, 1532, *lanes.TotalLoads)
	assert.InDelta(t, 97.91, *lanes.FTLPercent, 0.0001)
	require.Len(t, lanes.Preferred, 2)
	assert.Equal(t, 98, *lanes.Preferred[1].Loads)

	benchmarks := profile.Benchmarks
	require.NotNil(t, benchmarks)
	require.NotNil(t, benchmarks.AnyAnomaly)
	assert.False(t, *benchmarks.AnyAnomaly)
	require.NotNil(t, benchmarks.InspectedUnitsAnomaly)
	assert.True(t, *benchmarks.InspectedUnitsAnomaly)
	assert.Nil(t, benchmarks.PowerUnitMileageAnomaly)
}

func TestNormalizeProfile_CoverageIsHonest(t *testing.T) {
	t.Parallel()

	full := normalizeProfile(fixtureProfile(t))
	assert.Equal(t, carrierintel.AllSections(), full.Coverage)

	lite, err := carrierok.DecodeProfile(
		[]byte(`{"dot_number": 42, "legal_name": "LITE CO", "authority_common": "A"}`),
	)
	require.NoError(t, err)
	profile := normalizeProfile(lite)
	assert.Equal(
		t,
		[]carrierintel.Section{carrierintel.SectionIdentity, carrierintel.SectionAuthority},
		profile.Coverage,
	)
	assert.Nil(t, profile.Insurance)
	assert.Nil(t, profile.Safety)
	assert.Empty(t, profile.Basics)
}

func TestNormalizeProfile_AuthorityVocabulary(t *testing.T) {
	t.Parallel()

	raw := []byte(`{
		"dot_number": 7,
		"authority_common": "I",
		"authority_common_revocation": "Y",
		"authority_contract": "A",
		"authority_contract_revocation": "2026-10-01",
		"authority_broker": "r",
		"authority_broker_pending": "Y",
		"authority_broker_review": "Y",
		"docket": "FF123456",
		"risk_score": "Very High"
	}`)
	sdk, err := carrierok.DecodeProfile(raw)
	require.NoError(t, err)
	profile := normalizeProfile(sdk)

	assert.Equal(t, carrierintel.AuthorityStatusRevoked, profile.Authority.Common.Status)
	assert.False(t, profile.Authority.Common.RevocationPending)
	assert.Equal(t, carrierintel.AuthorityStatusActive, profile.Authority.Contract.Status)
	assert.True(t, profile.Authority.Contract.RevocationPending)
	assert.Equal(t, carrierintel.AuthorityStatusRevoked, profile.Authority.Broker.Status)
	assert.True(t, profile.Authority.Broker.Pending)
	assert.True(t, profile.Authority.Broker.UnderReview)
	assert.Equal(t, "FF", profile.Identity.DocketPrefix)
	assert.Equal(t, "123456", profile.Identity.DocketNumber)
	require.NotNil(t, profile.Safety)
	assert.Equal(t, carrierintel.RiskLevelVeryHigh, profile.Safety.RiskScore)
}

func TestFilingType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, carrierintel.InsuranceFilingTypeBIPD, filingType("BI&PD"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeBIPD, filingType("Auto Liability"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeCargo, filingType("cargo"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeBond, filingType("Surety"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeBond, filingType("BROKER TRUST FUND"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeOther, filingType("workers comp"))
}

func TestVendorFieldPath(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"safety_rating_desc":                     "safety.rating",
		"insurance_bipd_on_file":                 "insurance.bipdOnFile",
		"authority_common":                       "authority.common.status",
		"usdot_status":                           "identity.usdotStatus",
		"out_of_service_flag":                    "safety.outOfServiceOrder",
		"iss_value":                              "safety.issValue",
		"legal_name":                             "identity.legalName",
		"telephone_number":                       "contacts.phone",
		"email_address":                          "contacts.email",
		"physical_address_city":                  "identity.physicalAddress",
		"mailing_address":                        "identity.mailingAddress",
		"basic_percentile_hours_of_service":      "basics.HOSCompliance.percentile",
		"basic_alert_vehicle_maintence":          "basics.VehicleMaintenance.alert",
		"basic_roadside_alert_unsafe_driving":    "basics.UnsafeDriving.roadsideAlert",
		"intervention_threshold_crash_indicator": "basics.CrashIndicator.threshold",
		"indicator_insurance":                    "",
		"started_monitoring_at":                  "",
	}
	for field, want := range cases {
		assert.Equal(t, want, vendorFieldPath(field), field)
	}
}

func TestVendorFieldPath_MatchesFlattenedProfile(t *testing.T) {
	t.Parallel()

	flat, err := carrierintel.FlattenProfile(normalizeProfile(fixtureProfile(t)))
	require.NoError(t, err)

	for _, field := range []string{
		"safety_rating_desc", "insurance_bipd_on_file", "authority_common", "usdot_status",
		"out_of_service_flag", "iss_value", "legal_name", "telephone_number", "email_address",
		"physical_address_street", "total_power_units", "crashes_total",
		"basic_percentile_unsafe_driving", "network_graph_count_equipment", "mcs150_date",
	} {
		path := vendorFieldPath(field)
		require.NotEmpty(t, path, field)
		_, present := flat[path]
		assert.True(t, present, "%s -> %s missing from flattened profile", field, path)
	}
}
