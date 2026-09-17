package fmcsa

import (
	"fmt"
	"slices"

	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/shopspring/decimal"
)

var insuranceUnit = decimal.NewFromInt(1000)

type Carrier struct {
	AllowedToOperate              *jsonflex.Bool
	BIPDInsuranceOnFile           *jsonflex.Float
	BIPDInsuranceRequired         *jsonflex.Bool
	BIPDRequiredAmount            *jsonflex.Float
	BondInsuranceOnFile           *jsonflex.Float
	BondInsuranceRequired         *jsonflex.Bool
	BrokerAuthorityStatus         *jsonflex.String
	CargoInsuranceOnFile          *jsonflex.Float
	CargoInsuranceRequired        *jsonflex.Bool
	CarrierOperationCode          *jsonflex.String
	CarrierOperationDesc          *jsonflex.String
	CommonAuthorityStatus         *jsonflex.String
	ContractAuthorityStatus       *jsonflex.String
	CrashTotal                    *jsonflex.Int
	DBAName                       *jsonflex.String
	DOTNumber                     *jsonflex.String
	DriverInspections             *jsonflex.Int
	DriverOOSInspections          *jsonflex.Int
	DriverOOSRate                 *jsonflex.Float
	DriverOOSRateNationalAverage  *jsonflex.Float
	EIN                           *jsonflex.String
	FatalCrash                    *jsonflex.Int
	HazmatInspections             *jsonflex.Int
	HazmatOOSInspections          *jsonflex.Int
	HazmatOOSRate                 *jsonflex.Float
	HazmatOOSRateNationalAverage  *jsonflex.Float
	InjuryCrash                   *jsonflex.Int
	IsPassengerCarrier            *jsonflex.Bool
	ISSScore                      *jsonflex.Float
	LegalName                     *jsonflex.String
	MCS150Outdated                *jsonflex.Bool
	OOSDate                       *jsonflex.Time
	OOSRateNationalAverageYear    *jsonflex.String
	PhysicalCity                  *jsonflex.String
	PhysicalCountry               *jsonflex.String
	PhysicalState                 *jsonflex.String
	PhysicalStreet                *jsonflex.String
	PhysicalZipcode               *jsonflex.String
	ReviewDate                    *jsonflex.Time
	ReviewType                    *jsonflex.String
	SafetyRating                  *jsonflex.String
	SafetyRatingDate              *jsonflex.Time
	SafetyReviewDate              *jsonflex.Time
	SafetyReviewType              *jsonflex.String
	SnapshotDate                  *jsonflex.Time
	StatusCode                    *jsonflex.String
	TotalDrivers                  *jsonflex.Int
	TotalPowerUnits               *jsonflex.Int
	TowawayCrash                  *jsonflex.Int
	VehicleInspections            *jsonflex.Int
	VehicleOOSInspections         *jsonflex.Int
	VehicleOOSRate                *jsonflex.Float
	VehicleOOSRateNationalAverage *jsonflex.Float
	Raw                           []byte
}

type Basic struct {
	ID                                   *jsonflex.String
	Percentile                           *jsonflex.Float
	RunDate                              *jsonflex.Time
	Code                                 *jsonflex.String
	CodeMCMIS                            *jsonflex.String
	ShortDescription                     *jsonflex.String
	ViolationThreshold                   *jsonflex.Float
	ExceededInterventionThreshold        *jsonflex.Bool
	MeasureValue                         *jsonflex.Float
	OnRoadPerformanceThresholdViolation  *jsonflex.Bool
	SeriousViolationFromInvestigation12M *jsonflex.Bool
	TotalInspectionsWithViolation        *jsonflex.Int
	TotalViolations                      *jsonflex.Int
	Raw                                  []byte
}

type OOSEntry struct {
	Date   *jsonflex.Time
	Reason *jsonflex.String
	Status *jsonflex.String
	Raw    []byte
}

type Docket struct {
	DocketNumber   *jsonflex.String
	DocketNumberID *jsonflex.String
	DOTNumber      *jsonflex.String
	Prefix         *jsonflex.String
}

type Authority struct {
	AuthorizedForBroker         *jsonflex.Bool
	AuthorizedForHouseholdGoods *jsonflex.Bool
	AuthorizedForPassenger      *jsonflex.Bool
	AuthorizedForProperty       *jsonflex.Bool
	BrokerAuthorityStatus       *jsonflex.String
	CommonAuthorityStatus       *jsonflex.String
	ContractAuthorityStatus     *jsonflex.String
	DocketNumber                *jsonflex.String
	DOTNumber                   *jsonflex.String
	Prefix                      *jsonflex.String
	Raw                         []byte
}

type CompositeCarrier struct {
	Carrier     Carrier
	Basics      []Basic
	Cargo       []string
	Operations  []string
	OOS         []OOSEntry
	Dockets     []Docket
	Authorities []Authority
	Raw         []byte
}

func InsuranceDollars(value *jsonflex.Float) *decimal.Decimal {
	if value == nil {
		return nil
	}
	dollars := decimal.NewFromFloat(value.Value()).Mul(insuranceUnit)
	return &dollars
}

func DecodeCarrier(data []byte) (*Carrier, error) {
	carrier := new(Carrier)
	if err := carrier.UnmarshalJSON(data); err != nil {
		return nil, err
	}
	return carrier, nil
}

func (c *Carrier) UnmarshalJSON(data []byte) error {
	raw := slices.Clone(jsonflex.Unwrap(data, "carrier"))
	obj, err := jsonflex.DecodeObject(raw)
	if err != nil {
		return fmt.Errorf("%w: carrier: %w", ErrUnexpectedPayload, err)
	}

	carrier := Carrier{
		AllowedToOperate:              obj.Bool("allowedToOperate"),
		BIPDInsuranceOnFile:           obj.Float("bipdInsuranceOnFile"),
		BIPDInsuranceRequired:         obj.Bool("bipdInsuranceRequired"),
		BIPDRequiredAmount:            obj.Float("bipdRequiredAmount"),
		BondInsuranceOnFile:           obj.Float("bondInsuranceOnFile"),
		BondInsuranceRequired:         obj.Bool("bondInsuranceRequired"),
		BrokerAuthorityStatus:         obj.String("brokerAuthorityStatus"),
		CargoInsuranceOnFile:          obj.Float("cargoInsuranceOnFile"),
		CargoInsuranceRequired:        obj.Bool("cargoInsuranceRequired"),
		CommonAuthorityStatus:         obj.String("commonAuthorityStatus"),
		ContractAuthorityStatus:       obj.String("contractAuthorityStatus"),
		CrashTotal:                    obj.Int("crashTotal"),
		DBAName:                       obj.String("dbaName"),
		DOTNumber:                     obj.String("dotNumber"),
		DriverInspections:             obj.Int("driverInsp"),
		DriverOOSInspections:          obj.Int("driverOosInsp"),
		DriverOOSRate:                 obj.Float("driverOosRate"),
		DriverOOSRateNationalAverage:  obj.Float("driverOosRateNationalAverage"),
		EIN:                           obj.String("ein"),
		FatalCrash:                    obj.Int("fatalCrash"),
		HazmatInspections:             obj.Int("hazmatInsp"),
		HazmatOOSInspections:          obj.Int("hazmatOosInsp"),
		HazmatOOSRate:                 obj.Float("hazmatOosRate"),
		HazmatOOSRateNationalAverage:  obj.Float("hazmatOosRateNationalAverage"),
		InjuryCrash:                   obj.Int("injCrash"),
		IsPassengerCarrier:            obj.Bool("isPassengerCarrier"),
		ISSScore:                      obj.Float("issScore"),
		LegalName:                     obj.String("legalName"),
		MCS150Outdated:                obj.Bool("mcs150Outdated"),
		OOSDate:                       obj.Time("oosDate"),
		OOSRateNationalAverageYear:    obj.String("oosRateNationalAverageYear"),
		PhysicalCity:                  obj.String("phyCity"),
		PhysicalCountry:               obj.String("phyCountry"),
		PhysicalState:                 obj.String("phyState"),
		PhysicalStreet:                obj.String("phyStreet"),
		PhysicalZipcode:               obj.String("phyZipcode"),
		ReviewDate:                    obj.Time("reviewDate"),
		ReviewType:                    obj.String("reviewType"),
		SafetyRating:                  obj.String("safetyRating"),
		SafetyRatingDate:              obj.Time("safetyRatingDate"),
		SafetyReviewDate:              obj.Time("safetyReviewDate"),
		SafetyReviewType:              obj.String("safetyReviewType"),
		SnapshotDate:                  obj.Time("snapshotDate"),
		StatusCode:                    obj.String("statusCode"),
		TotalDrivers:                  obj.Int("totalDrivers"),
		TotalPowerUnits:               obj.Int("totalPowerUnits"),
		TowawayCrash:                  obj.Int("towawayCrash"),
		VehicleInspections:            obj.Int("vehicleInsp"),
		VehicleOOSInspections:         obj.Int("vehicleOosInsp"),
		VehicleOOSRate:                obj.Float("vehicleOosRate"),
		VehicleOOSRateNationalAverage: obj.Float("vehicleOosRateNationalAverage"),
		Raw:                           raw,
	}
	if operation, ok := obj.Object("carrierOperation"); ok {
		carrier.CarrierOperationCode = operation.String("carrierOperationCode")
		carrier.CarrierOperationDesc = operation.String("carrierOperationDesc")
	}
	*c = carrier
	return nil
}

func (c *Carrier) MarshalJSON() ([]byte, error) {
	if c == nil || len(c.Raw) == 0 {
		return []byte("null"), nil
	}
	return slices.Clone(c.Raw), nil
}

func decodeBasic(item jsonflex.Object, raw []byte) Basic {
	basic := Basic{
		ID:                            item.String("basicsId"),
		Percentile:                    item.Float("basicsPercentile"),
		RunDate:                       item.Time("basicsRunDate"),
		ViolationThreshold:            item.Float("basicsViolationThreshold"),
		ExceededInterventionThreshold: item.Bool("exceededFMCSAInterventionThreshold"),
		MeasureValue:                  item.Float("measureValue"),
		OnRoadPerformanceThresholdViolation: item.Bool(
			"onRoadPerformanceThresholdViolationIndicator",
		),
		SeriousViolationFromInvestigation12M: item.Bool(
			"seriousViolationFromInvestigationPast12MonthIndicator",
		),
		TotalInspectionsWithViolation: item.Int("totalInspectionWithViolation"),
		TotalViolations:               item.Int("totalViolation"),
		Raw:                           raw,
	}
	if basicType, ok := item.Object("basicsType"); ok {
		basic.Code = basicType.String("basicsCode")
		basic.CodeMCMIS = basicType.String("basicsCodeMcmis")
		basic.ShortDescription = basicType.String("basicsShortDesc")
	}
	return basic
}

func decodeOOSEntry(item jsonflex.Object, raw []byte) OOSEntry {
	return OOSEntry{
		Date:   item.Time("oosDate", "date", "outOfServiceDate"),
		Reason: item.String("oosReason", "reason", "oosReasonDesc", "outOfServiceReason"),
		Status: item.String("oosStatus", "status"),
		Raw:    raw,
	}
}

func decodeDocket(item jsonflex.Object, _ []byte) Docket {
	return Docket{
		DocketNumber:   item.String("docketNumber"),
		DocketNumberID: item.String("docketNumberId"),
		DOTNumber:      item.String("dotNumber"),
		Prefix:         item.String("prefix"),
	}
}

func decodeAuthority(item jsonflex.Object, raw []byte) Authority {
	return Authority{
		AuthorizedForBroker:         item.Bool("authorizedForBroker"),
		AuthorizedForHouseholdGoods: item.Bool("authorizedForHouseholdGoods"),
		AuthorizedForPassenger:      item.Bool("authorizedForPassenger"),
		AuthorizedForProperty:       item.Bool("authorizedForProperty"),
		BrokerAuthorityStatus:       item.String("brokerAuthorityStatus"),
		CommonAuthorityStatus:       item.String("commonAuthorityStatus"),
		ContractAuthorityStatus:     item.String("contractAuthorityStatus"),
		DocketNumber:                item.String("docketNumber"),
		DOTNumber:                   item.String("dotNumber"),
		Prefix:                      item.String("prefix"),
		Raw:                         raw,
	}
}
