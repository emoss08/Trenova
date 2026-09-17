package carrierok

import (
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/stringutils"
)

type BasicCategory string

const (
	BasicUnsafeDriving       BasicCategory = "unsafe_driving"
	BasicHoursOfService      BasicCategory = "hours_of_service"
	BasicVehicleMaintenance  BasicCategory = "vehicle_maintenance"
	BasicControlledSubstance BasicCategory = "controlled_substance"
	BasicDriverFitness       BasicCategory = "driver_fitness"
	BasicHazardousMaterials  BasicCategory = "hazardous_materials"
	BasicCrashIndicator      BasicCategory = "crash_indicator"
)

const vehicleMaintenanceVendorKey = "vehicle_maintence"

type NetworkLinkKind string

const (
	NetworkLinkPhysicalAddress NetworkLinkKind = "physical_address"
	NetworkLinkMailingAddress  NetworkLinkKind = "mailing_address"
	NetworkLinkTelephone       NetworkLinkKind = "telephone_number"
	NetworkLinkCellphone       NetworkLinkKind = "cellphone_number"
	NetworkLinkFax             NetworkLinkKind = "fax_number"
	NetworkLinkEmail           NetworkLinkKind = "email"
	NetworkLinkEIN             NetworkLinkKind = "ein"
	NetworkLinkDUNS            NetworkLinkKind = "duns"
	NetworkLinkEquipment       NetworkLinkKind = "equipment"
	NetworkLinkEquipmentExt    NetworkLinkKind = "equipment_ext"
)

type Profile struct {
	Identity      Identity
	Authority     Authority
	Insurance     Insurance
	Safety        Safety
	Basics        map[BasicCategory]BasicScore
	Inspections   Inspections
	Crashes       Crashes
	Fleet         Fleet
	Contacts      Contacts
	Addresses     Addresses
	Operations    Operations
	ChangeHistory ChangeHistory
	Network       Network
	Lanes         Lanes
	Benchmarks    Benchmarks
	RiskFactors   map[string]bool
	Raw           []byte
}

type Identity struct {
	DOTNumber        *jsonflex.String
	DocketPrefix     *jsonflex.String
	DocketNumber     *jsonflex.String
	Docket           *jsonflex.String
	DocID            *jsonflex.String
	EIN              *jsonflex.String
	DUNS             *jsonflex.String
	LegalName        *jsonflex.String
	DBAName          *jsonflex.String
	DBAFlag          *jsonflex.Bool
	EntityType       *jsonflex.String
	CarrierOperation *jsonflex.String
	OrganizationType *jsonflex.String
	USDOTStatus      *jsonflex.String
	AddedDate        *jsonflex.Time
	DOTAgeDays       *jsonflex.Int
	SnapshotDate     *jsonflex.Time
}

type Authority struct {
	Common                  *jsonflex.String
	Contract                *jsonflex.String
	Broker                  *jsonflex.String
	CommonPending           *jsonflex.Bool
	ContractPending         *jsonflex.Bool
	BrokerPending           *jsonflex.Bool
	CommonReview            *jsonflex.Bool
	ContractReview          *jsonflex.Bool
	BrokerReview            *jsonflex.Bool
	CommonRevocation        *jsonflex.Bool
	ContractRevocation      *jsonflex.Bool
	BrokerRevocation        *jsonflex.Bool
	AgeCommonDays           *jsonflex.Int
	AgeContractDays         *jsonflex.Int
	AgeBrokerDays           *jsonflex.Int
	StartCommon             *jsonflex.Time
	StartContract           *jsonflex.Time
	StartBroker             *jsonflex.Time
	TotalRevocations        *jsonflex.Int
	DaysSinceLastRevocation *jsonflex.Int
	LastRevocationDate      *jsonflex.Time
	History                 []AuthorityEvent
}

type AuthorityEvent struct {
	AuthorityType         *jsonflex.String
	OriginalAction        *jsonflex.String
	OriginalServedDate    *jsonflex.Time
	DispositionAction     *jsonflex.String
	DispositionServedDate *jsonflex.Time
	Raw                   []byte
}

type Insurance struct {
	BIPDOnFile        *jsonflex.Float
	BIPDRequired      *jsonflex.Float
	CargoOnFile       *jsonflex.Float
	CargoRequired     *jsonflex.Float
	BondOnFile        *jsonflex.Float
	BondRequired      *jsonflex.Float
	Indicator         *jsonflex.Bool
	CancelCount       *jsonflex.Int
	LastCanceled      *jsonflex.Time
	PendingCancelDate *jsonflex.Time
	History           []InsurancePolicy
}

type InsurancePolicy struct {
	Status              *jsonflex.String
	FormCode            *jsonflex.String
	Type                *jsonflex.String
	Insurer             *jsonflex.String
	PolicyNumber        *jsonflex.String
	Coverage            *jsonflex.Float
	UnderlyingLimit     *jsonflex.Float
	EffectiveDate       *jsonflex.Time
	NextRenewalDate     *jsonflex.Time
	CancelEffectiveDate *jsonflex.Time
	CancelMethod        *jsonflex.String
	Raw                 []byte
}

type Safety struct {
	ISSValue                *jsonflex.Int
	ISSRecommendation       *jsonflex.String
	ISSRecommendationReason *jsonflex.String
	RiskScore               *jsonflex.String
	RiskScoreProbability    *jsonflex.Float
	SafetyScore             *jsonflex.Float
	SafetyRating            *jsonflex.String
	SafetyRatingDate        *jsonflex.Time
	LatestReviewType        *jsonflex.String
	LatestReviewDate        *jsonflex.Time
	IndicatorCarrierSafety  *jsonflex.Bool
	OutOfServiceFlag        *jsonflex.Bool
	OutOfServiceDate        *jsonflex.Time
}

type BasicScore struct {
	Measure       *jsonflex.Float
	ACIndicator   *jsonflex.Bool
	MeasuredAt    *jsonflex.Time
	Alert         *jsonflex.Bool
	Violations    *jsonflex.Int
	OOSViolations *jsonflex.Int
}

type Inspections struct {
	ViolationsTotal         *jsonflex.Int
	Total                   *jsonflex.Int
	Driver                  *jsonflex.Int
	Vehicle                 *jsonflex.Int
	Hazmat                  *jsonflex.Int
	DriverOutOfService      *jsonflex.Int
	VehicleOutOfService     *jsonflex.Int
	HazmatOutOfService      *jsonflex.Int
	DriverOutOfServiceRate  *jsonflex.Float
	VehicleOutOfServiceRate *jsonflex.Float
	HazmatOutOfServiceRate  *jsonflex.Float
	NationalAvgOOSDriver    *jsonflex.Float
	NationalAvgOOSVehicle   *jsonflex.Float
	NationalAvgOOSHazmat    *jsonflex.Float
	OOSAlertDriver          *jsonflex.Bool
	OOSAlertVehicle         *jsonflex.Bool
	OOSAlertHazmat          *jsonflex.Bool
}

type Crashes struct {
	Total         *jsonflex.Int
	Fatalities    *jsonflex.Int
	Injuries      *jsonflex.Int
	TowAway       *jsonflex.Int
	LastCrashDate *jsonflex.Time
}

type Fleet struct {
	TotalPowerUnits    *jsonflex.Int
	TotalDrivers       *jsonflex.Int
	TotalDriversCDL    *jsonflex.Int
	OwnedTractors      *jsonflex.Int
	TermLeasedTractors *jsonflex.Int
	OwnedTrailers      *jsonflex.Int
	TermLeasedTrailers *jsonflex.Int
	TotalTrailers      *jsonflex.Int
	TotalTrucks        *jsonflex.Int
	Equipment          []Equipment
}

type Equipment struct {
	VIN         *jsonflex.String
	UnitType    *jsonflex.String
	Make        *jsonflex.String
	Model       *jsonflex.String
	Year        *jsonflex.Int
	PlateNumber *jsonflex.String
	PlateState  *jsonflex.String
	UnitNumber  *jsonflex.String
	Raw         []byte
}

type Contacts struct {
	Telephone        *jsonflex.String
	Cellphone        *jsonflex.String
	Fax              *jsonflex.String
	Email            *jsonflex.String
	EmailDomain      *jsonflex.String
	PrimaryContact   *jsonflex.String
	SecondaryContact *jsonflex.String
}

type Addresses struct {
	Physical Address
	Mailing  Address
}

type Address struct {
	Full          *jsonflex.String
	Street        *jsonflex.String
	City          *jsonflex.String
	State         *jsonflex.String
	ZipCode       *jsonflex.String
	CountryCode   *jsonflex.String
	Undeliverable *jsonflex.Bool
}

type Operations struct {
	CargoCarried            []string
	OperationClassification []string
	HazardousMaterial       *jsonflex.Bool
	SmartWay                *jsonflex.Bool
	CARBTRU                 *jsonflex.Bool
	PHMSA                   *jsonflex.Bool
	MCS150Date              *jsonflex.Time
	MCS150Year              *jsonflex.Int
	MCS150Mileage           *jsonflex.Int
	BOC3CompanyName         *jsonflex.String
}

type ChangeHistory struct {
	NameChangeCount    *jsonflex.Int
	NameLastChanged    *jsonflex.Time
	EmailChangeCount   *jsonflex.Int
	EmailLastChanged   *jsonflex.Time
	PhoneChangeCount   *jsonflex.Int
	PhoneLastChanged   *jsonflex.Time
	AddressChangeCount *jsonflex.Int
	AddressLastChanged *jsonflex.Time
	ContactChangeCount *jsonflex.Int
	ContactLastChanged *jsonflex.Time
}

type Network struct {
	IndicatorContact     *jsonflex.Bool
	IndicatorEquipment   *jsonflex.Bool
	PhysicalAddressCount *jsonflex.Int
	MailingAddressCount  *jsonflex.Int
	TelephoneNumberCount *jsonflex.Int
	CellphoneNumberCount *jsonflex.Int
	FaxNumberCount       *jsonflex.Int
	EmailAddressCount    *jsonflex.Int
	EINCount             *jsonflex.Int
	DUNSCount            *jsonflex.Int
	PowerUnitsCount      *jsonflex.Int
	TrailersCount        *jsonflex.Int
	Links                []NetworkLink
}

type NetworkLink struct {
	Kind      NetworkLinkKind
	DOTNumber string
}

type Lanes struct {
	PreferredStates []string
}

type Benchmarks struct {
	IndicatorIndustry        *jsonflex.Bool
	InspectionMileageRatio   *jsonflex.Bool
	InspectedPowerUnitsRatio *jsonflex.Bool
	PowerUnitMileageRatio    *jsonflex.Bool
}

func DecodeProfile(data []byte) (*Profile, error) {
	profile := new(Profile)
	if err := profile.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return profile, nil
}

func (p *Profile) UnmarshalJSON(data []byte) error {
	raw := slices.Clone(data)
	obj, err := jsonflex.DecodeObject(raw)
	if err != nil {
		return fmt.Errorf("%w: profile: %w", ErrUnexpectedPayload, err)
	}

	*p = Profile{
		Identity:      decodeIdentity(obj),
		Authority:     decodeAuthority(obj),
		Insurance:     decodeInsurance(obj),
		Safety:        decodeSafety(obj),
		Basics:        decodeBasics(obj),
		Inspections:   decodeInspections(obj),
		Crashes:       decodeCrashes(obj),
		Fleet:         decodeFleet(obj),
		Contacts:      decodeContacts(obj),
		Addresses:     decodeAddresses(obj),
		Operations:    decodeOperations(obj),
		ChangeHistory: decodeChangeHistory(obj),
		Network:       decodeNetwork(obj),
		Lanes:         decodeLanes(obj),
		Benchmarks:    decodeBenchmarks(obj),
		RiskFactors:   decodeRiskFactors(obj),
		Raw:           raw,
	}
	return nil
}

func (p *Profile) MarshalJSON() ([]byte, error) {
	if p == nil || len(p.Raw) == 0 {
		return []byte("null"), nil
	}
	return slices.Clone(p.Raw), nil
}

func (p *Profile) ProfileID() string {
	if p == nil {
		return ""
	}
	return ProfileID(
		p.Identity.DOTNumber.Value(),
		p.Identity.DocketPrefix.Value(),
		p.Identity.DocketNumber.Value(),
	)
}

func (p *Profile) RiskFactor(name string) *bool {
	if p == nil {
		return nil
	}
	value, ok := p.RiskFactors[name]
	if !ok {
		return nil
	}
	return &value
}

func decodeIdentity(obj jsonflex.Object) Identity {
	return Identity{
		DOTNumber:        obj.String("dot_number"),
		DocketPrefix:     obj.String("docket_prefix"),
		DocketNumber:     obj.String("docket_number"),
		Docket:           obj.String("docket"),
		DocID:            obj.String("doc_id"),
		EIN:              obj.String("ein"),
		DUNS:             obj.String("duns"),
		LegalName:        obj.String("legal_name"),
		DBAName:          obj.String("dba_name"),
		DBAFlag:          obj.Bool("dba_flag"),
		EntityType:       obj.String("entity_type_desc"),
		CarrierOperation: obj.String("carrier_operation_desc"),
		OrganizationType: obj.String("organization_type_desc"),
		USDOTStatus:      obj.String("usdot_status"),
		AddedDate:        obj.Time("added_date"),
		DOTAgeDays:       obj.Int("dot_age"),
		SnapshotDate:     obj.Time("snapshot_date"),
	}
}

func decodeAuthority(obj jsonflex.Object) Authority {
	return Authority{
		Common:                  obj.String("authority_common"),
		Contract:                obj.String("authority_contract"),
		Broker:                  obj.String("authority_broker"),
		CommonPending:           obj.Bool("authority_common_pending"),
		ContractPending:         obj.Bool("authority_contract_pending"),
		BrokerPending:           obj.Bool("authority_broker_pending"),
		CommonReview:            obj.Bool("authority_common_review"),
		ContractReview:          obj.Bool("authority_contract_review"),
		BrokerReview:            obj.Bool("authority_broker_review"),
		CommonRevocation:        obj.Bool("authority_common_revocation"),
		ContractRevocation:      obj.Bool("authority_contract_revocation"),
		BrokerRevocation:        obj.Bool("authority_broker_revocation"),
		AgeCommonDays:           obj.Int("authority_age_common"),
		AgeContractDays:         obj.Int("authority_age_contract"),
		AgeBrokerDays:           obj.Int("authority_age_broker"),
		StartCommon:             obj.Time("authority_start_common"),
		StartContract:           obj.Time("authority_start_contract"),
		StartBroker:             obj.Time("authority_start_broker"),
		TotalRevocations:        obj.Int("total_revocations"),
		DaysSinceLastRevocation: obj.Int("days_since_last_revocation"),
		LastRevocationDate:      obj.Time("last_revocation_date"),
		History:                 decodeObjects(obj, "authority_history", decodeAuthorityEvent),
	}
}

func decodeAuthorityEvent(item jsonflex.Object, raw []byte) AuthorityEvent {
	return AuthorityEvent{
		AuthorityType:         item.String("authority_type_desc"),
		OriginalAction:        item.String("original_action_desc"),
		OriginalServedDate:    item.Time("original_served_date"),
		DispositionAction:     item.String("dispensation_action_desc"),
		DispositionServedDate: item.Time("dispensation_served_date"),
		Raw:                   raw,
	}
}

func decodeInsurance(obj jsonflex.Object) Insurance {
	return Insurance{
		BIPDOnFile:        obj.Float("insurance_bipd_on_file"),
		BIPDRequired:      obj.Float("insurance_bipd_required"),
		CargoOnFile:       obj.Float("insurance_cargo_on_file"),
		CargoRequired:     obj.Float("insurance_cargo_required"),
		BondOnFile:        obj.Float("insurance_bond_on_file"),
		BondRequired:      obj.Float("insurance_bond_required"),
		Indicator:         obj.Bool("indicator_insurance"),
		CancelCount:       obj.Int("insurance_cancel_count"),
		LastCanceled:      obj.Time("insurance_last_canceled"),
		PendingCancelDate: obj.Time("insurance_pending_cancel_date"),
		History:           decodeObjects(obj, "insurance_history", decodeInsurancePolicy),
	}
}

func decodeInsurancePolicy(item jsonflex.Object, raw []byte) InsurancePolicy {
	return InsurancePolicy{
		Status:              item.String("insurance_status"),
		FormCode:            item.String("insurance_form_code"),
		Type:                item.String("insurance_type_code"),
		Insurer:             item.String("insurance_carrier"),
		PolicyNumber:        item.String("policy_number"),
		Coverage:            item.Float("minimum_coverage_amount"),
		UnderlyingLimit:     item.Float("underlying_limit_amount"),
		EffectiveDate:       item.Time("effective_date"),
		NextRenewalDate:     item.Time("next_renewal_date"),
		CancelEffectiveDate: item.Time("cancel_effective_date"),
		CancelMethod:        item.String("cancel_method"),
		Raw:                 raw,
	}
}

func decodeSafety(obj jsonflex.Object) Safety {
	return Safety{
		ISSValue:                obj.Int("iss_value"),
		ISSRecommendation:       obj.String("iss_recommendation"),
		ISSRecommendationReason: obj.String("iss_recommendation_reason"),
		RiskScore:               obj.String("risk_score"),
		RiskScoreProbability:    obj.Float("risk_score_probability"),
		SafetyScore:             obj.Float("safety_score"),
		SafetyRating:            obj.String("safety_rating_desc"),
		SafetyRatingDate:        obj.Time("safety_rating_date"),
		LatestReviewType:        obj.String("latest_review_type_desc"),
		LatestReviewDate:        obj.Time("latest_review_date"),
		IndicatorCarrierSafety:  obj.Bool("indicator_carrier_safety"),
		OutOfServiceFlag:        obj.Bool("out_of_service_flag"),
		OutOfServiceDate:        obj.Time("out_of_service_date"),
	}
}

func BasicCategories() []BasicCategory {
	return []BasicCategory{
		BasicUnsafeDriving,
		BasicHoursOfService,
		BasicVehicleMaintenance,
		BasicControlledSubstance,
		BasicDriverFitness,
		BasicHazardousMaterials,
		BasicCrashIndicator,
	}
}

func BasicVendorKey(category BasicCategory) string {
	if category == BasicVehicleMaintenance {
		return vehicleMaintenanceVendorKey
	}
	return string(category)
}

func BasicCategoryForVendorKey(key string) (BasicCategory, bool) {
	for _, category := range BasicCategories() {
		if BasicVendorKey(category) == key {
			return category, true
		}
	}
	return "", false
}

func decodeBasics(obj jsonflex.Object) map[BasicCategory]BasicScore {
	latest := latestBasicSnapshot(obj)
	categories := BasicCategories()
	basics := make(map[BasicCategory]BasicScore, len(categories))
	for _, category := range categories {
		key := BasicVendorKey(category)
		score := BasicScore{
			Alert:         obj.Bool("basic_alert_" + key),
			Violations:    obj.Int("violations_" + key),
			OOSViolations: obj.Int("violations_oos_" + key),
		}
		if latest != nil {
			score.Measure = latest.Float("basic_measure_" + key)
			score.ACIndicator = latest.Bool("basic_ac_indicator_" + key)
			if score.Measure != nil || score.ACIndicator != nil {
				score.MeasuredAt = latest.Time("snapshot_date")
			}
		}
		if score != (BasicScore{}) {
			basics[category] = score
		}
	}
	return basics
}

func latestBasicSnapshot(obj jsonflex.Object) jsonflex.Object {
	var (
		latest   jsonflex.Object
		latestAt int64
	)
	for _, entry := range decodeObjects(obj, "basic_history", basicSnapshot) {
		at := entry.Time("snapshot_date").Unix()
		if at == nil {
			continue
		}
		if latest == nil || *at > latestAt {
			latest = entry
			latestAt = *at
		}
	}
	return latest
}

func basicSnapshot(item jsonflex.Object, _ []byte) jsonflex.Object {
	return item
}

func decodeInspections(obj jsonflex.Object) Inspections {
	return Inspections{
		ViolationsTotal:         obj.Int("violations_total"),
		Total:                   obj.Int("inspections_total"),
		Driver:                  obj.Int("inspections_driver"),
		Vehicle:                 obj.Int("inspections_vehicle"),
		Hazmat:                  obj.Int("inspections_hazmat"),
		DriverOutOfService:      obj.Int("inspections_driver_out_of_service"),
		VehicleOutOfService:     obj.Int("inspections_vehicle_out_of_service"),
		HazmatOutOfService:      obj.Int("inspections_hazmat_out_of_service"),
		DriverOutOfServiceRate:  obj.Float("inspections_driver_out_of_service_pct"),
		VehicleOutOfServiceRate: obj.Float("inspections_vehicle_out_of_service_pct"),
		HazmatOutOfServiceRate:  obj.Float("inspections_hazmat_out_of_service_pct"),
		NationalAvgOOSDriver:    obj.Float("natl_avg_oos_driver"),
		NationalAvgOOSVehicle:   obj.Float("natl_avg_oos_vehicle"),
		NationalAvgOOSHazmat:    obj.Float("natl_avg_oos_hazmat"),
		OOSAlertDriver:          obj.Bool("oos_alert_driver"),
		OOSAlertVehicle:         obj.Bool("oos_alert_vehicle"),
		OOSAlertHazmat:          obj.Bool("oos_alert_hazmat"),
	}
}

func decodeCrashes(obj jsonflex.Object) Crashes {
	return Crashes{
		Total:         obj.Int("crashes_total"),
		Fatalities:    obj.Int("crash_fatalities"),
		Injuries:      obj.Int("crash_injuries"),
		TowAway:       obj.Int("crashes_tow_away"),
		LastCrashDate: obj.Time("last_crash_date"),
	}
}

func decodeFleet(obj jsonflex.Object) Fleet {
	return Fleet{
		TotalPowerUnits:    obj.Int("total_power_units"),
		TotalDrivers:       obj.Int("total_drivers"),
		TotalDriversCDL:    obj.Int("total_drivers_cdl"),
		OwnedTractors:      obj.Int("owned_tractors"),
		TermLeasedTractors: obj.Int("term_leased_tractors"),
		OwnedTrailers:      obj.Int("owned_trailers"),
		TermLeasedTrailers: obj.Int("term_leased_trailers"),
		TotalTrailers:      obj.Int("total_trailers"),
		TotalTrucks:        obj.Int("total_trucks"),
		Equipment:          decodeObjects(obj, "equipment_history", decodeEquipment),
	}
}

func decodeEquipment(item jsonflex.Object, raw []byte) Equipment {
	return Equipment{
		VIN:         item.String("vin"),
		UnitType:    item.String("unit_type"),
		Make:        item.String("make"),
		Model:       item.String("model"),
		Year:        item.Int("year"),
		PlateNumber: item.String("plate_number"),
		PlateState:  item.String("plate_state"),
		UnitNumber:  item.String("unit_number"),
		Raw:         raw,
	}
}

func decodeContacts(obj jsonflex.Object) Contacts {
	return Contacts{
		Telephone:        obj.String("telephone_number"),
		Cellphone:        obj.String("cellphone_number"),
		Fax:              obj.String("fax_number"),
		Email:            obj.String("email_address"),
		EmailDomain:      obj.String("email_domain"),
		PrimaryContact:   obj.String("company_contact_primary"),
		SecondaryContact: obj.String("company_contact_secondary"),
	}
}

func decodeAddresses(obj jsonflex.Object) Addresses {
	return Addresses{
		Physical: decodeAddress(obj, "physical_address"),
		Mailing:  decodeAddress(obj, "mailing_address"),
	}
}

func decodeAddress(obj jsonflex.Object, prefix string) Address {
	return Address{
		Full:          obj.String(prefix),
		Street:        obj.String(prefix + "_street"),
		City:          obj.String(prefix + "_city"),
		State:         obj.String(prefix + "_state"),
		ZipCode:       obj.String(prefix + "_zip_code"),
		CountryCode:   obj.String(prefix + "_iso_country_code"),
		Undeliverable: obj.Bool("undeliverable_" + prefix),
	}
}

func decodeOperations(obj jsonflex.Object) Operations {
	return Operations{
		CargoCarried:            obj.Strings("cargo_carried"),
		OperationClassification: obj.Strings("operation_classification_desc"),
		HazardousMaterial:       obj.Bool("hazardous_material"),
		SmartWay:                obj.Bool("smartway_flag"),
		CARBTRU:                 obj.Bool("carbtru_flag"),
		PHMSA:                   obj.Bool("phmsa_flag"),
		MCS150Date:              obj.Time("mcs150_date"),
		MCS150Year:              obj.Int("mcs150_year"),
		MCS150Mileage:           obj.Int("mcs150_mileage"),
		BOC3CompanyName:         obj.String("boc3_company_name"),
	}
}

func decodeChangeHistory(obj jsonflex.Object) ChangeHistory {
	return ChangeHistory{
		NameChangeCount:    obj.Int("name_change_count"),
		NameLastChanged:    obj.Time("name_last_changed"),
		EmailChangeCount:   obj.Int("email_change_count"),
		EmailLastChanged:   obj.Time("email_last_changed"),
		PhoneChangeCount:   obj.Int("phone_change_count"),
		PhoneLastChanged:   obj.Time("phone_last_changed"),
		AddressChangeCount: obj.Int("address_change_count"),
		AddressLastChanged: obj.Time("address_last_changed"),
		ContactChangeCount: obj.Int("contact_change_count"),
		ContactLastChanged: obj.Time("contact_last_changed"),
	}
}

var networkLinkKinds = [...]NetworkLinkKind{
	NetworkLinkPhysicalAddress,
	NetworkLinkMailingAddress,
	NetworkLinkTelephone,
	NetworkLinkCellphone,
	NetworkLinkFax,
	NetworkLinkEmail,
	NetworkLinkEIN,
	NetworkLinkDUNS,
	NetworkLinkEquipment,
	NetworkLinkEquipmentExt,
}

func NetworkLinkVendorKey(kind NetworkLinkKind) string {
	return "network_graph_" + string(kind)
}

func decodeNetwork(obj jsonflex.Object) Network {
	return Network{
		IndicatorContact:     obj.Bool("indicator_network_graph_contact"),
		IndicatorEquipment:   obj.Bool("indicator_network_graph_equipment"),
		PhysicalAddressCount: obj.Int("network_graph_count_physical_address"),
		MailingAddressCount:  obj.Int("network_graph_count_mailing_address"),
		TelephoneNumberCount: obj.Int("network_graph_count_telephone_numbers"),
		CellphoneNumberCount: obj.Int("network_graph_count_cellphone_numbers"),
		FaxNumberCount:       obj.Int("network_graph_count_fax_numbers"),
		EmailAddressCount:    obj.Int("network_graph_count_email_address"),
		EINCount:             obj.Int("network_graph_count_ein"),
		DUNSCount:            obj.Int("network_graph_count_duns"),
		PowerUnitsCount:      obj.Int("network_graph_count_power_units"),
		TrailersCount:        obj.Int("network_graph_count_trailers"),
		Links:                decodeNetworkLinks(obj),
	}
}

func decodeNetworkLinks(obj jsonflex.Object) []NetworkLink {
	var links []NetworkLink
	for _, kind := range networkLinkKinds {
		for _, dot := range obj.Strings(NetworkLinkVendorKey(kind)) {
			link := NetworkLink{Kind: kind, DOTNumber: dot}
			if !slices.Contains(links, link) {
				links = append(links, link)
			}
		}
	}
	return links
}

func decodeLanes(obj jsonflex.Object) Lanes {
	var states []string
	for _, lane := range decodeObjects(obj, "preferred_lanes", decodeLaneState) {
		states = appendState(states, lane)
	}
	for _, state := range stringutils.SplitCSV(obj.Text("preferred_states")) {
		states = appendState(states, state)
	}
	return Lanes{PreferredStates: states}
}

func decodeLaneState(item jsonflex.Object, _ []byte) string {
	return item.Text("state")
}

func appendState(states []string, state string) []string {
	if state == "" || slices.Contains(states, state) {
		return states
	}
	return append(states, state)
}

func decodeBenchmarks(obj jsonflex.Object) Benchmarks {
	return Benchmarks{
		IndicatorIndustry:        obj.Bool("indicator_industry_benchmarks"),
		InspectionMileageRatio:   obj.Bool("indicator_benchmark_inspection_mileage_ratio"),
		InspectedPowerUnitsRatio: obj.Bool("indicator_benchmark_inspected_power_units_ratio"),
		PowerUnitMileageRatio:    obj.Bool("indicator_benchmark_power_unit_mileage_ratio"),
	}
}

func decodeRiskFactors(obj jsonflex.Object) map[string]bool {
	entries := decodeObjects(obj, "risk_factors", basicSnapshot)
	if len(entries) == 0 {
		return nil
	}
	first := entries[0]
	flags := make(map[string]bool, len(first))
	for _, key := range first.Keys() {
		if value := first.Bool(key); value != nil {
			flags[key] = value.Value()
		}
	}
	return flags
}

func decodeObjects[T any](
	obj jsonflex.Object,
	key string,
	decode func(item jsonflex.Object, raw []byte) T,
) []T {
	items, ok := obj.Array(key)
	if !ok || len(items) == 0 {
		return nil
	}
	out := make([]T, 0, len(items))
	for _, item := range items {
		if jsonflex.KindOf(item) != jsonflex.KindObject {
			continue
		}
		decoded, err := jsonflex.DecodeObject(item)
		if err != nil {
			continue
		}
		out = append(out, decode(decoded, cloneRaw(item)))
	}
	return out
}

func cloneRaw(raw sonic.NoCopyRawMessage) []byte {
	return slices.Clone([]byte(raw))
}
