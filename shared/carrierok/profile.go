package carrierok

import (
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/jsonflex"
)

type BasicCategory string

const (
	BasicUnsafeDriving         BasicCategory = "unsafe_driving"
	BasicHoursOfService        BasicCategory = "hours_of_service"
	BasicVehicleMaintenance    BasicCategory = "vehicle_maintenance"
	BasicControlledSubstance   BasicCategory = "controlled_substance"
	BasicDriverFitness         BasicCategory = "driver_fitness"
	BasicHazardousMaterials    BasicCategory = "hazardous_materials"
	BasicCrashIndicator        BasicCategory = "crash_indicator"
	vehicleMaintenanceMisspelt               = "vehicle_maintence"
)

type NetworkLinkKind string

const (
	NetworkLinkPhysicalAddress NetworkLinkKind = "physical_address"
	NetworkLinkTelephone       NetworkLinkKind = "telephone_number"
	NetworkLinkEmail           NetworkLinkKind = "email"
	NetworkLinkEIN             NetworkLinkKind = "ein"
	NetworkLinkEquipment       NetworkLinkKind = "equipment"
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
	Loads         Loads
	Benchmarks    Benchmarks
	Raw           []byte
}

type Identity struct {
	DOTNumber        *jsonflex.String
	DocketPrefix     *jsonflex.String
	DocketNumber     *jsonflex.String
	Docket           *jsonflex.String
	DocID            *jsonflex.String
	EIN              *jsonflex.String
	LegalName        *jsonflex.String
	DBAName          *jsonflex.String
	DBAFlag          *jsonflex.Bool
	EntityType       *jsonflex.String
	CarrierOperation *jsonflex.String
	OrganizationType *jsonflex.String
	USDOTStatus      *jsonflex.String
	AddedDate        *jsonflex.Time
	DOTAge           *jsonflex.Float
	SnapshotDate     *jsonflex.Time
}

type Authority struct {
	Common                  *jsonflex.String
	Contract                *jsonflex.String
	Broker                  *jsonflex.String
	CommonPending           *jsonflex.String
	ContractPending         *jsonflex.String
	BrokerPending           *jsonflex.String
	CommonReview            *jsonflex.String
	ContractReview          *jsonflex.String
	BrokerReview            *jsonflex.String
	CommonRevocation        *jsonflex.String
	ContractRevocation      *jsonflex.String
	BrokerRevocation        *jsonflex.String
	AgeCommon               *jsonflex.Float
	AgeContract             *jsonflex.Float
	AgeBroker               *jsonflex.Float
	StartCommon             *jsonflex.Time
	StartContract           *jsonflex.Time
	TotalRevocations        *jsonflex.Int
	DaysSinceLastRevocation *jsonflex.Int
	LastRevocationDate      *jsonflex.Time
	History                 []AuthorityEvent
}

type AuthorityEvent struct {
	AuthorityType *jsonflex.String
	Action        *jsonflex.String
	ServedDate    *jsonflex.Time
	EffectiveDate *jsonflex.Time
	Raw           []byte
}

type Insurance struct {
	BIPDOnFile        *jsonflex.Float
	BIPDRequired      *jsonflex.Float
	CargoOnFile       *jsonflex.Float
	CargoRequired     *jsonflex.Float
	BondOnFile        *jsonflex.Float
	BondRequired      *jsonflex.Float
	Indicator         *jsonflex.String
	CancelCount       *jsonflex.Int
	LastCanceled      *jsonflex.Time
	PendingCancelDate *jsonflex.Time
	History           []InsurancePolicy
}

type InsurancePolicy struct {
	Type                *jsonflex.String
	Insurer             *jsonflex.String
	PolicyNumber        *jsonflex.String
	Coverage            *jsonflex.Float
	EffectiveDate       *jsonflex.Time
	CancelEffectiveDate *jsonflex.Time
	CancelMethod        *jsonflex.String
	Raw                 []byte
}

type Safety struct {
	ISSValue                *jsonflex.Float
	ISSRecommendation       *jsonflex.String
	ISSRecommendationReason *jsonflex.String
	RiskScore               *jsonflex.Float
	RiskScoreProbability    *jsonflex.Float
	SafetyScore             *jsonflex.Float
	SafetyRating            *jsonflex.String
	SafetyRatingDate        *jsonflex.Time
	LatestReviewType        *jsonflex.String
	LatestReviewDate        *jsonflex.Time
	IndicatorCarrierSafety  *jsonflex.String
	OutOfServiceFlag        *jsonflex.Bool
	OutOfServiceDate        *jsonflex.Time
}

type BasicScore struct {
	Measure               *jsonflex.Float
	Percentile            *jsonflex.Float
	Alert                 *jsonflex.Bool
	RoadsideAlert         *jsonflex.Bool
	ACIndicator           *jsonflex.Bool
	InterventionThreshold *jsonflex.Float
}

type Inspections struct {
	ViolationsTotal        *jsonflex.Int
	LastViolationDate      *jsonflex.Time
	Total                  *jsonflex.Int
	Driver                 *jsonflex.Int
	Vehicle                *jsonflex.Int
	Hazmat                 *jsonflex.Int
	DriverOutOfService     *jsonflex.Int
	VehicleOutOfService    *jsonflex.Int
	HazmatOutOfService     *jsonflex.Int
	DriverOutOfServicePct  *jsonflex.Float
	VehicleOutOfServicePct *jsonflex.Float
	HazmatOutOfServicePct  *jsonflex.Float
	NationalAvgOOSDriver   *jsonflex.Float
	NationalAvgOOSVehicle  *jsonflex.Float
	NationalAvgOOSHazmat   *jsonflex.Float
	OOSAlertDriver         *jsonflex.Bool
	OOSAlertVehicle        *jsonflex.Bool
	OOSAlertHazmat         *jsonflex.Bool
	LastInspectionDate     *jsonflex.Time
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
	EquipmentSummary   []byte
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
	IndicatorContact     *jsonflex.String
	IndicatorEquipment   *jsonflex.String
	PhysicalAddressCount *jsonflex.Int
	MailingAddressCount  *jsonflex.Int
	TelephoneNumberCount *jsonflex.Int
	CellphoneNumberCount *jsonflex.Int
	EmailAddressCount    *jsonflex.Int
	EINCount             *jsonflex.Int
	DUNSCount            *jsonflex.Int
	EquipmentCount       *jsonflex.Int
	Links                []NetworkLink
}

type NetworkLink struct {
	Kind      NetworkLinkKind
	DOTNumber string
	LegalName string
	Value     string
	Status    string
}

type Loads struct {
	Total              *jsonflex.Int
	LTL                *jsonflex.Int
	FTL                *jsonflex.Int
	LTLPercentage      *jsonflex.Float
	FTLPercentage      *jsonflex.Float
	Deadheads          *jsonflex.Int
	DeadheadPercentage *jsonflex.Float
	FirstLoadDate      *jsonflex.Time
	LastLoadDate       *jsonflex.Time
	PreferredLanes     []Lane
}

type Lane struct {
	OriginCity       *jsonflex.String
	OriginState      *jsonflex.String
	DestinationCity  *jsonflex.String
	DestinationState *jsonflex.String
	Loads            *jsonflex.Int
}

type Benchmarks struct {
	IndicatorIndustry        *jsonflex.String
	InspectionMileageRatio   *jsonflex.String
	InspectedPowerUnitsRatio *jsonflex.String
	PowerUnitMileageRatio    *jsonflex.String
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
		Loads:         decodeLoads(obj),
		Benchmarks:    decodeBenchmarks(obj),
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

func decodeIdentity(obj jsonflex.Object) Identity {
	return Identity{
		DOTNumber:        obj.String("dot_number"),
		DocketPrefix:     obj.String("docket_prefix"),
		DocketNumber:     obj.String("docket_number"),
		Docket:           obj.String("docket"),
		DocID:            obj.String("doc_id"),
		EIN:              obj.String("ein"),
		LegalName:        obj.String("legal_name"),
		DBAName:          obj.String("dba_name"),
		DBAFlag:          obj.Bool("dba_flag"),
		EntityType:       obj.String("entity_type_desc"),
		CarrierOperation: obj.String("carrier_operation_desc"),
		OrganizationType: obj.String("organization_type_desc"),
		USDOTStatus:      obj.String("usdot_status"),
		AddedDate:        obj.Time("added_date"),
		DOTAge:           obj.Float("dot_age"),
		SnapshotDate:     obj.Time("snapshot_date"),
	}
}

func decodeAuthority(obj jsonflex.Object) Authority {
	return Authority{
		Common:                  obj.String("authority_common"),
		Contract:                obj.String("authority_contract"),
		Broker:                  obj.String("authority_broker"),
		CommonPending:           obj.String("authority_common_pending"),
		ContractPending:         obj.String("authority_contract_pending"),
		BrokerPending:           obj.String("authority_broker_pending"),
		CommonReview:            obj.String("authority_common_review"),
		ContractReview:          obj.String("authority_contract_review"),
		BrokerReview:            obj.String("authority_broker_review"),
		CommonRevocation:        obj.String("authority_common_revocation"),
		ContractRevocation:      obj.String("authority_contract_revocation"),
		BrokerRevocation:        obj.String("authority_broker_revocation"),
		AgeCommon:               obj.Float("authority_age_common"),
		AgeContract:             obj.Float("authority_age_contract"),
		AgeBroker:               obj.Float("authority_age_broker"),
		StartCommon:             obj.Time("authority_start_common"),
		StartContract:           obj.Time("authority_start_contract"),
		TotalRevocations:        obj.Int("total_revocations"),
		DaysSinceLastRevocation: obj.Int("days_since_last_revocation"),
		LastRevocationDate:      obj.Time("last_revocation_date"),
		History:                 decodeObjects(obj, decodeAuthorityEvent, "authority_history"),
	}
}

func decodeAuthorityEvent(item jsonflex.Object, raw []byte) AuthorityEvent {
	return AuthorityEvent{
		AuthorityType: item.String("authority_type", "type", "docket_type"),
		Action:        item.String("action", "status", "authority_action"),
		ServedDate:    item.Time("served_date", "serve_date", "date"),
		EffectiveDate: item.Time("effective_date", "effective"),
		Raw:           raw,
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
		Indicator:         obj.String("indicator_insurance"),
		CancelCount:       obj.Int("insurance_cancel_count"),
		LastCanceled:      obj.Time("insurance_last_canceled"),
		PendingCancelDate: obj.Time("insurance_pending_cancel_date"),
		History:           decodeObjects(obj, decodeInsurancePolicy, "insurance_history"),
	}
}

func decodeInsurancePolicy(item jsonflex.Object, raw []byte) InsurancePolicy {
	return InsurancePolicy{
		Type:         item.String("insurance_type", "type", "insurance_type_desc"),
		Insurer:      item.String("insurer", "company_name", "insurance_company", "carrier_name"),
		PolicyNumber: item.String("policy_number", "policy_no", "policy"),
		Coverage: item.Float(
			"coverage",
			"amount",
			"coverage_amount",
			"max_coverage_amount",
			"coverage_to",
		),
		EffectiveDate: item.Time("effective_date", "effective", "start_date"),
		CancelEffectiveDate: item.Time(
			"cancel_effective_date",
			"cancellation_date",
			"cancel_date",
			"canceled_date",
		),
		CancelMethod: item.String("cancel_method", "cancellation_method"),
		Raw:          raw,
	}
}

func decodeSafety(obj jsonflex.Object) Safety {
	return Safety{
		ISSValue:                obj.Float("iss_value"),
		ISSRecommendation:       obj.String("iss_recommendation"),
		ISSRecommendationReason: obj.String("iss_recommendation_reason"),
		RiskScore:               obj.Float("risk_score"),
		RiskScoreProbability:    obj.Float("risk_score_probability"),
		SafetyScore:             obj.Float("safety_score"),
		SafetyRating:            obj.String("safety_rating_desc"),
		SafetyRatingDate:        obj.Time("safety_rating_date"),
		LatestReviewType:        obj.String("latest_review_type_desc"),
		LatestReviewDate:        obj.Time("latest_review_date"),
		IndicatorCarrierSafety:  obj.String("indicator_carrier_safety"),
		OutOfServiceFlag:        obj.Bool("out_of_service_flag"),
		OutOfServiceDate:        obj.Time("out_of_service_date"),
	}
}

func basicKeys(category BasicCategory) []string {
	if category == BasicVehicleMaintenance {
		return []string{string(category), vehicleMaintenanceMisspelt}
	}
	return []string{string(category)}
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

func decodeBasics(obj jsonflex.Object) map[BasicCategory]BasicScore {
	categories := BasicCategories()
	basics := make(map[BasicCategory]BasicScore, len(categories))
	for _, category := range categories {
		suffixes := basicKeys(category)
		score := BasicScore{
			Measure:               obj.Float(prefixed("basic_measure_", suffixes)...),
			Percentile:            obj.Float(prefixed("basic_percentile_", suffixes)...),
			Alert:                 obj.Bool(prefixed("basic_alert_", suffixes)...),
			RoadsideAlert:         obj.Bool(prefixed("basic_roadside_alert_", suffixes)...),
			ACIndicator:           obj.Bool(prefixed("basic_ac_indicator_", suffixes)...),
			InterventionThreshold: obj.Float(prefixed("intervention_threshold_", suffixes)...),
		}
		if score.Measure != nil || score.Percentile != nil || score.Alert != nil ||
			score.RoadsideAlert != nil || score.ACIndicator != nil ||
			score.InterventionThreshold != nil {
			basics[category] = score
		}
	}
	return basics
}

func prefixed(prefix string, suffixes []string) []string {
	keys := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		keys = append(keys, prefix+suffix)
	}
	return keys
}

func decodeInspections(obj jsonflex.Object) Inspections {
	return Inspections{
		ViolationsTotal:        obj.Int("violations_total"),
		LastViolationDate:      obj.Time("last_violation_date"),
		Total:                  obj.Int("inspections_total"),
		Driver:                 obj.Int("inspections_driver"),
		Vehicle:                obj.Int("inspections_vehicle"),
		Hazmat:                 obj.Int("inspections_hazmat"),
		DriverOutOfService:     obj.Int("inspections_driver_out_of_service"),
		VehicleOutOfService:    obj.Int("inspections_vehicle_out_of_service"),
		HazmatOutOfService:     obj.Int("inspections_hazmat_out_of_service"),
		DriverOutOfServicePct:  obj.Float("inspections_driver_out_of_service_pct"),
		VehicleOutOfServicePct: obj.Float("inspections_vehicle_out_of_service_pct"),
		HazmatOutOfServicePct:  obj.Float("inspections_hazmat_out_of_service_pct"),
		NationalAvgOOSDriver:   obj.Float("natl_avg_oos_driver"),
		NationalAvgOOSVehicle:  obj.Float("natl_avg_oos_vehicle"),
		NationalAvgOOSHazmat:   obj.Float("natl_avg_oos_hazmat"),
		OOSAlertDriver:         obj.Bool("oos_alert_driver"),
		OOSAlertVehicle:        obj.Bool("oos_alert_vehicle"),
		OOSAlertHazmat:         obj.Bool("oos_alert_hazmat"),
		LastInspectionDate:     obj.Time("last_inspection_date"),
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
		EquipmentSummary:   obj.Raw("equipment_summary"),
		Equipment: decodeObjects(
			obj,
			decodeEquipment,
			"equipment",
			"equipment_history",
			"vehicles",
		),
	}
}

func decodeEquipment(item jsonflex.Object, raw []byte) Equipment {
	return Equipment{
		VIN:         item.String("vin", "vin_number"),
		UnitType:    item.String("unit_type", "type", "equipment_type"),
		Make:        item.String("make"),
		Model:       item.String("model"),
		Year:        item.Int("year", "model_year"),
		PlateNumber: item.String("plate_number", "license_plate", "plate"),
		PlateState:  item.String("plate_state", "license_state"),
		UnitNumber:  item.String("unit_number", "unit"),
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
		CountryCode:   obj.String(prefix + "_country_code"),
		Undeliverable: obj.Bool(prefix + "_undeliverable"),
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

var networkLinkSources = [...]struct {
	key  string
	kind NetworkLinkKind
}{
	{key: "network_graph_physical_address", kind: NetworkLinkPhysicalAddress},
	{key: "network_graph_telephone_number", kind: NetworkLinkTelephone},
	{key: "network_graph_email", kind: NetworkLinkEmail},
	{key: "network_graph_ein", kind: NetworkLinkEIN},
	{key: "network_graph_equipment", kind: NetworkLinkEquipment},
}

func decodeNetwork(obj jsonflex.Object) Network {
	return Network{
		IndicatorContact:     obj.String("indicator_network_graph_contact"),
		IndicatorEquipment:   obj.String("indicator_network_graph_equipment"),
		PhysicalAddressCount: obj.Int("network_graph_count_physical_address"),
		MailingAddressCount:  obj.Int("network_graph_count_mailing_address"),
		TelephoneNumberCount: obj.Int("network_graph_count_telephone_numbers"),
		CellphoneNumberCount: obj.Int("network_graph_count_cellphone_numbers"),
		EmailAddressCount:    obj.Int("network_graph_count_email_address"),
		EINCount:             obj.Int("network_graph_count_ein"),
		DUNSCount:            obj.Int("network_graph_count_duns"),
		EquipmentCount:       obj.Int("network_graph_count_equipment"),
		Links:                decodeNetworkLinks(obj),
	}
}

func decodeNetworkLinks(obj jsonflex.Object) []NetworkLink {
	var links []NetworkLink
	for _, source := range networkLinkSources {
		items, ok := obj.Array(source.key)
		if !ok || len(items) == 0 {
			continue
		}
		if links == nil {
			links = make([]NetworkLink, 0, len(items))
		}
		for _, item := range items {
			if link, valid := decodeNetworkLink(source.kind, item); valid {
				links = append(links, link)
			}
		}
	}
	return links
}

func decodeNetworkLink(kind NetworkLinkKind, raw []byte) (NetworkLink, bool) {
	switch jsonflex.KindOf(raw) {
	case jsonflex.KindObject:
		item, err := jsonflex.DecodeObject(raw)
		if err != nil {
			return NetworkLink{}, false
		}
		link := NetworkLink{
			Kind:      kind,
			DOTNumber: item.Text("dot_number", "usdot", "dot"),
			LegalName: item.Text("legal_name", "name", "company_name"),
			Value: item.Text(
				"value",
				"address",
				"physical_address",
				"phone",
				"telephone_number",
				"email",
				"email_address",
				"ein",
				"vin",
			),
			Status: item.Text("status", "usdot_status", "authority_status"),
		}
		if link.DOTNumber == "" && link.LegalName == "" && link.Value == "" {
			return NetworkLink{}, false
		}
		return link, true
	case jsonflex.KindString, jsonflex.KindNumber:
		value, ok := jsonflex.ParseString(raw)
		if !ok {
			return NetworkLink{}, false
		}
		return NetworkLink{Kind: kind, Value: value.Value()}, true
	default:
		return NetworkLink{}, false
	}
}

func decodeLoads(obj jsonflex.Object) Loads {
	return Loads{
		Total:              obj.Int("total_loads"),
		LTL:                obj.Int("ltl_loads"),
		FTL:                obj.Int("ftl_loads"),
		LTLPercentage:      obj.Float("ltl_percentage"),
		FTLPercentage:      obj.Float("ftl_percentage"),
		Deadheads:          obj.Int("deadheads"),
		DeadheadPercentage: obj.Float("deadhead_percentage"),
		FirstLoadDate:      obj.Time("first_load_date"),
		LastLoadDate:       obj.Time("last_load_date"),
		PreferredLanes:     decodeObjects(obj, decodeLane, "preferred_lanes"),
	}
}

func decodeLane(item jsonflex.Object, _ []byte) Lane {
	return Lane{
		OriginCity:       item.String("origin_city"),
		OriginState:      item.String("origin_state"),
		DestinationCity:  item.String("destination_city"),
		DestinationState: item.String("destination_state"),
		Loads:            item.Int("loads", "count", "load_count"),
	}
}

func decodeBenchmarks(obj jsonflex.Object) Benchmarks {
	return Benchmarks{
		IndicatorIndustry:        obj.String("indicator_industry_benchmarks"),
		InspectionMileageRatio:   obj.String("indicator_benchmark_inspection_mileage_ratio"),
		InspectedPowerUnitsRatio: obj.String("indicator_benchmark_inspected_power_units_ratio"),
		PowerUnitMileageRatio:    obj.String("indicator_benchmark_power_unit_mileage_ratio"),
	}
}

func decodeObjects[T any](
	obj jsonflex.Object,
	decode func(item jsonflex.Object, raw []byte) T,
	keys ...string,
) []T {
	items, ok := obj.Array(keys...)
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
