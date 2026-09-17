package carrierokconnector

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	fullProfileFixture = "profile_265752.json"
	liteProfileFixture = "profile_lite_265752.json"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	require.NoError(t, err)
	return data
}

func fixtureProfile(t *testing.T, name string) *carrierok.Profile {
	t.Helper()
	profile, err := carrierok.DecodeProfile(fixture(t, name))
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

	profile := normalizeProfile(fixtureProfile(t, fullProfileFixture))
	require.NotNil(t, profile.Identity)

	identity := profile.Identity
	assert.Equal(t, "265752", identity.DOTNumber)
	assert.Equal(t, "MC", identity.DocketPrefix)
	assert.Equal(t, "179059", identity.DocketNumber)
	assert.Equal(t, "265752-MC179059", identity.ProviderRef())
	assert.Equal(t, "FEDEX GROUND PACKAGE SYSTEM INC", identity.LegalName)
	assert.Equal(t, "FEDEX GROUND", identity.DBAName)
	assert.Equal(t, "341441019", identity.EIN)
	assert.Equal(t, "INACTIVE", identity.USDOTStatus)
	assert.Equal(t, "Broker,Carrier,Freight Forwarder,Shipper", identity.EntityType)
	assert.Equal(t, "interstate", identity.CarrierOperation)
	require.NotNil(t, identity.DOTAddedAt)
	assert.Equal(t, unixDate(t, "1985-10-25"), *identity.DOTAddedAt)
	require.NotNil(t, identity.DOTAgeDays)
	assert.Equal(t, 14936, *identity.DOTAgeDays)

	require.NotNil(t, identity.PhysicalAddress)
	assert.Equal(t, "3660 HACKS CROSS RD BLDG F 2ND FLR", identity.PhysicalAddress.Line1)
	assert.Equal(t, "MEMPHIS", identity.PhysicalAddress.City)
	assert.Equal(t, "TN", identity.PhysicalAddress.State)
	assert.Equal(t, "38125", identity.PhysicalAddress.PostalCode)
	assert.Equal(t, "US", identity.PhysicalAddress.Country)
	require.NotNil(t, identity.PhysicalAddress.Undelivered)
	assert.False(t, *identity.PhysicalAddress.Undelivered)
	require.NotNil(t, identity.MailingAddress)
	assert.Equal(t, "1000 FEDEX DRIVE", identity.MailingAddress.Line1)
	assert.Equal(t, "CORAOPOLIS", identity.MailingAddress.City)
	require.NotNil(t, identity.MailingAddress.Undelivered)
}

func TestNormalizeProfile_Authority(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t, fullProfileFixture))
	require.NotNil(t, profile.Authority)

	authority := profile.Authority
	require.NotNil(t, authority.Common)
	assert.Equal(t, carrierintel.AuthorityStatusInactive, authority.Common.Status)
	assert.False(t, authority.Common.Pending)
	assert.False(t, authority.Common.UnderReview)
	assert.False(t, authority.Common.RevocationPending)
	require.NotNil(t, authority.Common.AgeDays)
	assert.Equal(t, 6882, *authority.Common.AgeDays)
	require.NotNil(t, authority.Common.GrantedAt)
	assert.Equal(t, unixDate(t, "2005-12-14"), *authority.Common.GrantedAt)
	require.NotNil(t, authority.Broker)
	assert.Equal(t, carrierintel.AuthorityStatusInactive, authority.Broker.Status)
	require.NotNil(t, authority.Broker.AgeDays)
	assert.Equal(t, 6735, *authority.Broker.AgeDays)
	require.NotNil(t, authority.Broker.GrantedAt)
	assert.Equal(t, unixDate(t, "2006-05-10"), *authority.Broker.GrantedAt)
	assert.Nil(t, authority.OldestActiveAgeDays())
	require.NotNil(t, authority.TotalRevocations)
	assert.Equal(t, 5, *authority.TotalRevocations)
	require.NotNil(t, authority.LastRevocationAt)
	assert.Equal(t, unixDate(t, "2024-10-17"), *authority.LastRevocationAt)

	require.Len(t, authority.History, 10)
	for idx := 1; idx < len(authority.History); idx++ {
		assert.GreaterOrEqual(
			t,
			*authority.History[idx-1].ServedAt,
			*authority.History[idx].ServedAt,
		)
	}
	assert.Equal(t, carrierintel.AuthorityHistoryEntry{
		AuthorityType: "PROPERTY BROKER",
		Action:        "REVOKED",
		ServedAt:      new(unixDate(t, "2024-10-17")),
	}, authority.History[0])
	assert.Contains(t, authority.History, carrierintel.AuthorityHistoryEntry{
		AuthorityType: "PROPERTY BROKER",
		Action:        "GRANTED",
		ServedAt:      new(unixDate(t, "2006-05-10")),
	})
	assert.Equal(t, carrierintel.AuthorityHistoryEntry{
		AuthorityType: "MOTOR PROPERTY CONTRACT CARRIER",
		Action:        "GRANTED",
		ServedAt:      new(unixDate(t, "1985-01-24")),
	}, authority.History[len(authority.History)-1])
}

func TestNormalizeProfile_Insurance(t *testing.T) {
	t.Parallel()

	insurance := normalizeProfile(fixtureProfile(t, fullProfileFixture)).Insurance
	require.NotNil(t, insurance)
	require.NotNil(t, insurance.BIPDOnFile)
	assert.True(t, insurance.BIPDOnFile.IsZero())
	require.NotNil(t, insurance.BIPDRequired)
	assert.Equal(t, "5000000", insurance.BIPDRequired.String())
	require.NotNil(t, insurance.BondRequired)
	assert.Equal(t, "75000", insurance.BondRequired.String())
	require.NotNil(t, insurance.CargoRequired)
	assert.True(t, insurance.CargoRequired.IsZero())
	assert.Nil(t, insurance.CargoOnFile)
	assert.Nil(t, insurance.PendingCancelAt)
	require.NotNil(t, insurance.CancelCount)
	assert.Equal(t, 5, *insurance.CancelCount)
	require.NotNil(t, insurance.LastCanceledAt)
	assert.Equal(t, unixDate(t, "2025-06-23"), *insurance.LastCanceledAt)

	require.Len(t, insurance.Filings, 4)
	cargo := insurance.Filings[0]
	assert.Equal(t, carrierintel.InsuranceFilingTypeCargo, cargo.Type)
	assert.Equal(t, "A", cargo.Status)
	assert.Equal(t, "OLD REPUBLIC INSURANCE COMPANY", cargo.InsurerName)
	assert.Equal(t, "MWE 315778", cargo.PolicyNumber)
	require.NotNil(t, cargo.Coverage)
	assert.Equal(t, "5000", cargo.Coverage.String())
	assert.Equal(t, unixDate(t, "2020-10-01"), *cargo.EffectiveAt)
	assert.Nil(t, cargo.CancelEffectiveAt)

	surety := insurance.Filings[1]
	assert.Equal(t, carrierintel.InsuranceFilingTypeBond, surety.Type)
	assert.Equal(t, "H", surety.Status)
	assert.Equal(t, "CANCELLED", surety.CancelMethod)
	assert.Equal(t, unixDate(t, "2025-06-23"), *surety.CancelEffectiveAt)

	bipd := insurance.Filings[2]
	assert.Equal(t, carrierintel.InsuranceFilingTypeBIPD, bipd.Type)
	assert.Equal(t, "5000000", bipd.Coverage.String())
	assert.Equal(t, "LIBERTY MUTUAL INSURANCE CO.", bipd.InsurerName)
}

func TestNormalizeProfile_SafetyAndBasics(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t, fullProfileFixture))
	safety := profile.Safety
	require.NotNil(t, safety)
	assert.Equal(t, carrierintel.SafetyRatingSatisfactory, safety.Rating)
	require.NotNil(t, safety.RatingDate)
	assert.Equal(t, unixDate(t, "1996-09-25"), *safety.RatingDate)
	require.NotNil(t, safety.ISSValue)
	assert.Equal(t, 69, *safety.ISSValue)
	assert.Equal(t, "OPTIONAL", safety.ISSRecommendation)
	assert.Equal(t, carrierintel.RiskLevelHigh, safety.RiskScore)
	require.NotNil(t, safety.RiskProbability)
	assert.InDelta(t, 0.4433, *safety.RiskProbability, 0.0001)
	require.NotNil(t, safety.SafetyScore)
	assert.Zero(t, *safety.SafetyScore)
	require.NotNil(t, safety.OutOfServiceOrder)
	assert.False(t, *safety.OutOfServiceOrder)
	assert.Nil(t, safety.OutOfServiceAt)
	assert.Equal(t, "Compliance Review", safety.LatestReviewType)
	assert.Equal(t, unixDate(t, "1996-03-05"), *safety.LatestReviewAt)

	require.Len(t, profile.Basics, len(worker.AllCSABasics()))
	for idx := range profile.Basics {
		basic := &profile.Basics[idx]
		assert.False(t, basic.Alert, basic.Basic)
		assert.Nil(t, basic.Percentile, basic.Basic)
		assert.Nil(t, basic.Threshold, basic.Basic)
	}

	unsafe := profile.Basic(worker.BasicUnsafeDriving)
	require.NotNil(t, unsafe)
	require.NotNil(t, unsafe.Measure)
	assert.InDelta(t, 0.13, *unsafe.Measure, 0.0001)
	require.NotNil(t, unsafe.Violations)
	assert.Zero(t, *unsafe.Violations)
	require.NotNil(t, unsafe.OOSViolations)
	require.NotNil(t, unsafe.MeasuredAt)
	assert.Equal(t, unixDate(t, "2024-12-31"), *unsafe.MeasuredAt)

	maintenance := profile.Basic(worker.BasicVehicleMaintenance)
	require.NotNil(t, maintenance)
	require.NotNil(t, maintenance.Measure)
	assert.InDelta(t, 3.11, *maintenance.Measure, 0.0001)

	crash := profile.Basic(worker.BasicCrashIndicator)
	require.NotNil(t, crash)
	assert.Nil(t, crash.Measure)
	assert.Nil(t, crash.Violations)
	assert.Nil(t, crash.MeasuredAt)
}

func TestNormalizeProfile_BasicAlerts(t *testing.T) {
	t.Parallel()

	sdk, err := carrierok.DecodeProfile([]byte(`{
		"dot_number": "42",
		"basic_alert_hours_of_service": true,
		"violations_hours_of_service": "9",
		"basic_alert_vehicle_maintence": false
	}`))
	require.NoError(t, err)
	profile := normalizeProfile(sdk)

	hos := profile.Basic(worker.BasicHOSCompliance)
	require.NotNil(t, hos)
	assert.True(t, hos.Alert)
	assert.Equal(t, 9, *hos.Violations)
	require.NotNil(t, profile.Basic(worker.BasicVehicleMaintenance))
	assert.Nil(t, profile.Basic(worker.BasicDriverFitness))
}

func TestNormalizeProfile_OperationalSections(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t, fullProfileFixture))

	inspections := profile.Inspections
	require.NotNil(t, inspections)
	assert.Equal(t, 15015, *inspections.Total)
	assert.Equal(t, 9083, *inspections.Vehicle)
	assert.Equal(t, 286, *inspections.DriverOOS)
	require.NotNil(t, inspections.DriverOOSRate)
	assert.InDelta(t, 1.9, *inspections.DriverOOSRate, 0.0001)
	require.NotNil(t, inspections.VehicleOOSRate)
	assert.InDelta(t, 18.3, *inspections.VehicleOOSRate, 0.0001)
	require.NotNil(t, inspections.NationalVehicleOOS)
	assert.InDelta(t, 22.26, *inspections.NationalVehicleOOS, 0.0001)
	require.NotNil(t, inspections.NationalDriverOOSRate)
	assert.InDelta(t, 6.67, *inspections.NationalDriverOOSRate, 0.0001)
	require.NotNil(t, inspections.NationalHazmatOOSRate)
	assert.InDelta(t, 4.44, *inspections.NationalHazmatOOSRate, 0.0001)
	assert.Nil(t, inspections.LastInspectionAt)

	require.NotNil(t, profile.Crashes)
	assert.Equal(t, 123, *profile.Crashes.Total)
	assert.Equal(t, 1, *profile.Crashes.Fatal)
	assert.Equal(t, 75, *profile.Crashes.Tow)
	assert.Equal(t, unixDate(t, "2026-08-24"), *profile.Crashes.LastCrashAt)

	fleet := profile.Fleet
	require.NotNil(t, fleet)
	assert.Equal(t, 101844, *fleet.PowerUnits)
	assert.Equal(t, 127917, *fleet.Drivers)
	assert.Equal(t, 30014, *fleet.CDLDrivers)
	assert.Equal(t, 18363, *fleet.TermLeasedTractors)
	assert.Equal(t, 79704, *fleet.OwnedTrailers)
	assert.Nil(t, fleet.OwnedTractors)
	assert.Empty(t, profile.Equipment)

	contacts := profile.Contacts
	require.NotNil(t, contacts)
	assert.Equal(t, "4127478482", contacts.Phone)
	assert.Equal(t, "4128595045", contacts.Fax)
	assert.Empty(t, contacts.Cellphone)
	assert.Equal(t, "safety@fedex.com", contacts.Email)
	assert.Equal(t, "CLARENCE DOZIER", contacts.PrimaryContact)

	operations := profile.Operations
	require.NotNil(t, operations)
	assert.Equal(t, []string{"Authorized For Hire"}, operations.Classification)
	assert.Equal(t, []string{"General Freight", "Other"}, operations.CargoCarried)
	require.NotNil(t, operations.HazmatCarrier)
	assert.True(t, *operations.HazmatCarrier)
	require.NotNil(t, operations.SmartWay)
	assert.True(t, *operations.SmartWay)
	require.NotNil(t, operations.CARBCompliant)
	assert.False(t, *operations.CARBCompliant)
	require.NotNil(t, operations.MCS150Mileage)
	assert.Equal(t, int64(3_279_390_000), *operations.MCS150Mileage)
	assert.Equal(t, unixDate(t, "2024-12-30"), *operations.MCS150At)
	require.NotNil(t, operations.BOC3OnFile)
	assert.True(t, *operations.BOC3OnFile)
	assert.Equal(t, "CT CORPORATION SYSTEM", operations.BOC3Agent)

	history := profile.ChangeHistory
	require.NotNil(t, history)
	assert.Equal(t, 0, *history.NameChanges)
	assert.Equal(t, 2, *history.EmailChanges)
	assert.Equal(t, 1, *history.PhoneChanges)
	assert.Equal(t, 3, *history.AddressChanges)
	assert.Equal(t, 9, *history.ContactChanges)
	assert.Equal(t, 6, history.TotalChanges())
	assert.Equal(t, unixDate(t, "2026-01-23"), *history.LatestChangeAt())

	benchmarks := profile.Benchmarks
	require.NotNil(t, benchmarks)
	require.NotNil(t, benchmarks.AnyAnomaly)
	assert.False(t, *benchmarks.AnyAnomaly)
	require.NotNil(t, benchmarks.InspectedUnitsAnomaly)
	assert.False(t, *benchmarks.InspectedUnitsAnomaly)
	require.NotNil(t, benchmarks.PowerUnitMileageAnomaly)
	assert.False(t, *benchmarks.PowerUnitMileageAnomaly)

	assert.Nil(t, profile.Lanes)
}

func TestNormalizeProfile_NetworkLinksCarryOtherCarriersDOTs(t *testing.T) {
	t.Parallel()

	network := normalizeProfile(fixtureProfile(t, fullProfileFixture)).Network
	require.NotNil(t, network)

	assert.Equal(t, []carrierintel.NetworkLink{
		{Kind: carrierintel.NetworkKindAddress, DOTNumber: "86876"},
		{Kind: carrierintel.NetworkKindAddress, DOTNumber: "1964900"},
		{Kind: carrierintel.NetworkKindAddress, DOTNumber: "4356378"},
		{Kind: carrierintel.NetworkKindAddress, DOTNumber: "598081"},
		{Kind: carrierintel.NetworkKindPhone, DOTNumber: "86876"},
		{Kind: carrierintel.NetworkKindPhone, DOTNumber: "598081"},
		{Kind: carrierintel.NetworkKindEmail, DOTNumber: "598081"},
		{Kind: carrierintel.NetworkKindEIN, DOTNumber: "3859536"},
	}, network.Links)

	assert.Equal(t, 4, network.SharedCount(carrierintel.NetworkKindAddress))
	assert.Equal(t, 2, network.SharedCount(carrierintel.NetworkKindPhone))
	assert.Equal(t, 1, network.SharedCount(carrierintel.NetworkKindEmail))
	assert.Equal(t, 1, network.SharedCount(carrierintel.NetworkKindEIN))
	require.NotNil(t, network.SharedEquipment)
	assert.Zero(t, *network.SharedEquipment)
}

func TestNormalizeProfile_NetworkExcludesOwnDOTAndFallsBackToCounts(t *testing.T) {
	t.Parallel()

	sdk, err := carrierok.DecodeProfile([]byte(`{
		"dot_number": "100",
		"network_graph_physical_address": ["100", "200"],
		"network_graph_equipment_ext": ["300"],
		"network_graph_count_telephone_numbers": "2",
		"network_graph_count_cellphone_numbers": "1"
	}`))
	require.NoError(t, err)
	network := normalizeProfile(sdk).Network
	require.NotNil(t, network)

	assert.Equal(t, []carrierintel.NetworkLink{
		{Kind: carrierintel.NetworkKindAddress, DOTNumber: "200"},
		{Kind: carrierintel.NetworkKindEquipment, DOTNumber: "300"},
	}, network.Links)
	assert.Equal(t, 1, network.SharedCount(carrierintel.NetworkKindAddress))
	assert.Equal(t, 1, network.SharedCount(carrierintel.NetworkKindEquipment))
	assert.Equal(t, 3, network.SharedCount(carrierintel.NetworkKindPhone))
	assert.Nil(t, network.SharedEmails)
}

func TestNormalizeProfile_PreferredStates(t *testing.T) {
	t.Parallel()

	sdk, err := carrierok.DecodeProfile([]byte(`{
		"dot_number": "42",
		"preferred_lanes": [{"state": "tx"}, {"state": ""}],
		"preferred_states": "IL, TX"
	}`))
	require.NoError(t, err)
	profile := normalizeProfile(sdk)

	require.NotNil(t, profile.Lanes)
	assert.Equal(t, []string{"TX", "IL"}, profile.Lanes.PreferredStates)
	assert.Nil(t, profile.Lanes.TotalLoads)
	assert.Empty(t, profile.Lanes.Preferred)
	assert.True(t, profile.Covers(carrierintel.SectionLanes))
}

func TestNormalizeProfile_CoverageIsHonest(t *testing.T) {
	t.Parallel()

	full := normalizeProfile(fixtureProfile(t, fullProfileFixture))
	assert.Equal(t, []carrierintel.Section{
		carrierintel.SectionIdentity,
		carrierintel.SectionAuthority,
		carrierintel.SectionInsurance,
		carrierintel.SectionSafety,
		carrierintel.SectionBasics,
		carrierintel.SectionInspections,
		carrierintel.SectionCrashes,
		carrierintel.SectionFleet,
		carrierintel.SectionContacts,
		carrierintel.SectionOperations,
		carrierintel.SectionChangeHistory,
		carrierintel.SectionNetwork,
		carrierintel.SectionBenchmarks,
	}, full.Coverage)

	lite := normalizeProfile(fixtureProfile(t, liteProfileFixture))
	assert.Equal(t, []carrierintel.Section{
		carrierintel.SectionIdentity,
		carrierintel.SectionAuthority,
		carrierintel.SectionInsurance,
		carrierintel.SectionSafety,
		carrierintel.SectionInspections,
		carrierintel.SectionFleet,
		carrierintel.SectionContacts,
		carrierintel.SectionOperations,
		carrierintel.SectionBenchmarks,
	}, lite.Coverage)
	assert.Empty(t, lite.Basics)
	assert.Nil(t, lite.Network)
	assert.Nil(t, lite.ChangeHistory)
	assert.Nil(t, lite.Crashes)
	assert.Nil(t, lite.Identity.DOTAgeDays)
	assert.Equal(t, 6882, *lite.Authority.Common.AgeDays)
	assert.Equal(t, carrierintel.RiskLevelHigh, lite.Safety.RiskScore)
	assert.Nil(t, lite.Safety.RiskProbability)
	assert.Nil(t, lite.Insurance.BIPDRequired)
	assert.Empty(t, lite.Insurance.Filings)
	assert.Empty(t, lite.Authority.History)
	require.NotNil(t, lite.Operations.BOC3OnFile)
	assert.True(t, *lite.Operations.BOC3OnFile)

	minimal, err := carrierok.DecodeProfile(
		[]byte(`{"dot_number": 42, "legal_name": "LITE CO", "authority_common": "A"}`),
	)
	require.NoError(t, err)
	profile := normalizeProfile(minimal)
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
		"authority_common_revocation": true,
		"authority_contract": "A",
		"authority_contract_revocation": true,
		"authority_broker": "r",
		"authority_broker_pending": true,
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

func TestFullProfileRuleMessages(t *testing.T) {
	t.Parallel()

	profile := normalizeProfile(fixtureProfile(t, fullProfileFixture))
	findings := carrierintel.EvaluateFindings(&carrierintel.EvaluateInput{
		Profile: profile,
		Now:     unixDate(t, "2026-09-16"),
		Subject: carrierintel.SubjectTypeCarrier,
	})

	messages := make(map[carrierintel.RuleCode]string, len(findings))
	for idx := range findings {
		messages[findings[idx].Code] = findings[idx].Message
	}
	assert.Equal(
		t,
		"BIPD liability coverage on file ($0) is below the required $5,000,000",
		messages[carrierintel.RuleInsuranceBIPDBelow],
	)
	assert.Equal(
		t,
		"Shares identifiers with other USDOT numbers: address (4), phone (2), email (1), ein (1)",
		messages[carrierintel.RuleFraudNetworkSharing],
	)
	assert.NotContains(t, messages, carrierintel.RuleIdentityNewEntrant)
	assert.NotContains(t, messages, carrierintel.RuleInsurancePendingCancel)
	assert.NotContains(t, messages, carrierintel.RuleInspectionsOOSHigh)
}

func TestFilingType(t *testing.T) {
	t.Parallel()

	assert.Equal(t, carrierintel.InsuranceFilingTypeBIPD, filingType("BIPD"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeBIPD, filingType("BIPD/PRIMARY"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeCargo, filingType("CARGO"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeBond, filingType("SURETY"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeBond, filingType("BROKER TRUST FUND"))
	assert.Equal(t, carrierintel.InsuranceFilingTypeOther, filingType("workers comp"))
}

func TestVendorFieldPath(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"safety_rating_desc":                "safety.rating",
		"insurance_bipd_on_file":            "insurance.bipdOnFile",
		"authority_common":                  "authority.common.status",
		"authority_start_broker":            "authority.broker.grantedAt",
		"usdot_status":                      "identity.usdotStatus",
		"out_of_service_flag":               "safety.outOfServiceOrder",
		"iss_value":                         "safety.issValue",
		"legal_name":                        "identity.legalName",
		"telephone_number":                  "contacts.phone",
		"email_address":                     "contacts.email",
		"physical_address_city":             "identity.physicalAddress",
		"physical_address_iso_country_code": "identity.physicalAddress",
		"undeliverable_mailing_address":     "identity.mailingAddress",
		"mailing_address":                   "identity.mailingAddress",
		"basic_measure_hours_of_service":    "basics.HOSCompliance.measure",
		"basic_alert_vehicle_maintence":     "basics.VehicleMaintenance.alert",
		"basic_ac_indicator_unsafe_driving": "basics.UnsafeDriving.acIndicator",
		"violations_oos_driver_fitness":     "basics.DriverFitness.oosViolations",
		"violations_crash_indicator":        "basics.CrashIndicator.violations",
		"network_graph_mailing_address":     "network.links",
		"network_graph_count_fax_numbers":   "network.sharedPhones",
		"preferred_states":                  "lanes.preferredStates",
		"violations_total":                  "",
		"indicator_insurance":               "",
		"started_monitoring_at":             "",
	}
	for field, want := range cases {
		assert.Equal(t, want, vendorFieldPath(field), field)
	}
}

func TestVendorFieldPath_MatchesFlattenedProfile(t *testing.T) {
	t.Parallel()

	flat, err := carrierintel.FlattenProfile(
		normalizeProfile(fixtureProfile(t, fullProfileFixture)),
	)
	require.NoError(t, err)

	for _, field := range []string{
		"safety_rating_desc", "insurance_bipd_on_file", "authority_common", "usdot_status",
		"out_of_service_flag", "iss_value", "legal_name", "telephone_number", "email_address",
		"physical_address_street", "total_power_units", "crashes_total",
		"basic_measure_unsafe_driving", "basic_alert_crash_indicator",
		"violations_oos_hours_of_service", "network_graph_count_ein", "mcs150_date",
	} {
		path := vendorFieldPath(field)
		require.NotEmpty(t, path, field)
		_, present := flat[path]
		assert.True(t, present, "%s -> %s missing from flattened profile", field, path)
	}
}
