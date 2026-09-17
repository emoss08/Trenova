package carrierintel

import (
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/shopspring/decimal"
)

type Profile struct {
	Identity      *Identity      `json:"identity,omitempty"`
	Authority     *Authority     `json:"authority,omitempty"`
	Insurance     *Insurance     `json:"insurance,omitempty"`
	Safety        *Safety        `json:"safety,omitempty"`
	Basics        []BasicMeasure `json:"basics,omitempty"`
	Inspections   *Inspections   `json:"inspections,omitempty"`
	Crashes       *Crashes       `json:"crashes,omitempty"`
	Fleet         *Fleet         `json:"fleet,omitempty"`
	Equipment     []Equipment    `json:"equipment,omitempty"`
	Contacts      *Contacts      `json:"contacts,omitempty"`
	Operations    *Operations    `json:"operations,omitempty"`
	ChangeHistory *ChangeHistory `json:"changeHistory,omitempty"`
	Network       *Network       `json:"network,omitempty"`
	Lanes         *Lanes         `json:"lanes,omitempty"`
	Benchmarks    *Benchmarks    `json:"benchmarks,omitempty"`
	Coverage      []Section      `json:"coverage"`
}

type Address struct {
	Line1       string `json:"line1,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	PostalCode  string `json:"postalCode,omitempty"`
	Country     string `json:"country,omitempty"`
	Undelivered *bool  `json:"undelivered,omitempty"`
}

func (a *Address) IsZero() bool {
	return a == nil || (a.Line1 == "" && a.City == "" && a.State == "" && a.PostalCode == "")
}

type Identity struct {
	DOTNumber        string   `json:"dotNumber"`
	DocketPrefix     string   `json:"docketPrefix,omitempty"`
	DocketNumber     string   `json:"docketNumber,omitempty"`
	LegalName        string   `json:"legalName,omitempty"`
	DBAName          string   `json:"dbaName,omitempty"`
	EIN              string   `json:"ein,omitempty"`
	USDOTStatus      string   `json:"usdotStatus,omitempty"`
	EntityType       string   `json:"entityType,omitempty"`
	CarrierOperation string   `json:"carrierOperation,omitempty"`
	DOTAddedAt       *int64   `json:"dotAddedAt,omitempty"`
	DOTAgeDays       *int     `json:"dotAgeDays,omitempty"`
	PhysicalAddress  *Address `json:"physicalAddress,omitempty"`
	MailingAddress   *Address `json:"mailingAddress,omitempty"`
}

func (i *Identity) USDOTActive() (active, known bool) {
	if i == nil || strings.TrimSpace(i.USDOTStatus) == "" {
		return false, false
	}
	status := strings.ToUpper(strings.TrimSpace(i.USDOTStatus))
	return status == "ACTIVE" || status == "A" || status == "AUTHORIZED", true
}

func (i *Identity) ProviderRef() string {
	if i == nil || i.DOTNumber == "" {
		return ""
	}
	if i.DocketNumber == "" {
		return i.DOTNumber
	}
	prefix := i.DocketPrefix
	if prefix == "" {
		prefix = "MC"
	}
	return i.DOTNumber + "-" + prefix + i.DocketNumber
}

type AuthorityGrant struct {
	Status            AuthorityStatus `json:"status"`
	Pending           bool            `json:"pending"`
	UnderReview       bool            `json:"underReview"`
	RevocationPending bool            `json:"revocationPending"`
	GrantedAt         *int64          `json:"grantedAt,omitempty"`
	AgeDays           *int            `json:"ageDays,omitempty"`
}

func (g *AuthorityGrant) IsActive() bool {
	return g != nil && g.Status == AuthorityStatusActive
}

func (g *AuthorityGrant) IsKnown() bool {
	return g != nil && g.Status != "" && g.Status != AuthorityStatusUnknown
}

type AuthorityHistoryEntry struct {
	AuthorityType string `json:"authorityType"`
	Action        string `json:"action"`
	ServedAt      *int64 `json:"servedAt,omitempty"`
	EffectiveAt   *int64 `json:"effectiveAt,omitempty"`
}

type Authority struct {
	Common           *AuthorityGrant         `json:"common,omitempty"`
	Contract         *AuthorityGrant         `json:"contract,omitempty"`
	Broker           *AuthorityGrant         `json:"broker,omitempty"`
	TotalRevocations *int                    `json:"totalRevocations,omitempty"`
	LastRevocationAt *int64                  `json:"lastRevocationAt,omitempty"`
	History          []AuthorityHistoryEntry `json:"history,omitempty"`
}

func (a *Authority) CarrierAuthorityActive() (active, known bool) {
	if a == nil {
		return false, false
	}
	known = a.Common.IsKnown() || a.Contract.IsKnown()
	return a.Common.IsActive() || a.Contract.IsActive(), known
}

func (a *Authority) OldestActiveAgeDays() *int {
	if a == nil {
		return nil
	}
	var best *int
	for _, grant := range []*AuthorityGrant{a.Common, a.Contract, a.Broker} {
		if !grant.IsActive() || grant.AgeDays == nil {
			continue
		}
		if best == nil || *grant.AgeDays > *best {
			v := *grant.AgeDays
			best = &v
		}
	}
	return best
}

type InsuranceFiling struct {
	Type              InsuranceFilingType `json:"type"`
	Status            string              `json:"status,omitempty"`
	InsurerName       string              `json:"insurerName,omitempty"`
	PolicyNumber      string              `json:"policyNumber,omitempty"`
	Coverage          *decimal.Decimal    `json:"coverage,omitempty"`
	EffectiveAt       *int64              `json:"effectiveAt,omitempty"`
	CancelEffectiveAt *int64              `json:"cancelEffectiveAt,omitempty"`
	CancelMethod      string              `json:"cancelMethod,omitempty"`
}

type Insurance struct {
	BIPDOnFile      *decimal.Decimal  `json:"bipdOnFile,omitempty"`
	BIPDRequired    *decimal.Decimal  `json:"bipdRequired,omitempty"`
	CargoOnFile     *decimal.Decimal  `json:"cargoOnFile,omitempty"`
	CargoRequired   *decimal.Decimal  `json:"cargoRequired,omitempty"`
	BondOnFile      *decimal.Decimal  `json:"bondOnFile,omitempty"`
	BondRequired    *decimal.Decimal  `json:"bondRequired,omitempty"`
	PendingCancelAt *int64            `json:"pendingCancelAt,omitempty"`
	LastCanceledAt  *int64            `json:"lastCanceledAt,omitempty"`
	CancelCount     *int              `json:"cancelCount,omitempty"`
	Filings         []InsuranceFiling `json:"filings,omitempty"`
}

type Safety struct {
	Rating            SafetyRating `json:"rating,omitempty"`
	RatingDate        *int64       `json:"ratingDate,omitempty"`
	ISSValue          *int         `json:"issValue,omitempty"`
	ISSRecommendation string       `json:"issRecommendation,omitempty"`
	RiskScore         RiskLevel    `json:"riskScore,omitempty"`
	RiskProbability   *float64     `json:"riskProbability,omitempty"`
	SafetyScore       *float64     `json:"safetyScore,omitempty"`
	OutOfServiceOrder *bool        `json:"outOfServiceOrder,omitempty"`
	OutOfServiceAt    *int64       `json:"outOfServiceAt,omitempty"`
	LatestReviewType  string       `json:"latestReviewType,omitempty"`
	LatestReviewAt    *int64       `json:"latestReviewAt,omitempty"`
}

type BasicMeasure struct {
	Basic         worker.CSABasic `json:"basic"`
	Measure       *float64        `json:"measure,omitempty"`
	Percentile    *float64        `json:"percentile,omitempty"`
	Threshold     *float64        `json:"threshold,omitempty"`
	Alert         bool            `json:"alert"`
	RoadsideAlert bool            `json:"roadsideAlert"`
	ACIndicator   bool            `json:"acIndicator"`
	Violations    *int            `json:"violations,omitempty"`
	OOSViolations *int            `json:"oosViolations,omitempty"`
	MeasuredAt    *int64          `json:"measuredAt,omitempty"`
}

type Inspections struct {
	Total                 *int     `json:"total,omitempty"`
	Driver                *int     `json:"driver,omitempty"`
	Vehicle               *int     `json:"vehicle,omitempty"`
	Hazmat                *int     `json:"hazmat,omitempty"`
	DriverOOS             *int     `json:"driverOos,omitempty"`
	VehicleOOS            *int     `json:"vehicleOos,omitempty"`
	HazmatOOS             *int     `json:"hazmatOos,omitempty"`
	DriverOOSRate         *float64 `json:"driverOosRate,omitempty"`
	VehicleOOSRate        *float64 `json:"vehicleOosRate,omitempty"`
	HazmatOOSRate         *float64 `json:"hazmatOosRate,omitempty"`
	NationalDriverOOSRate *float64 `json:"nationalDriverOosRate,omitempty"`
	NationalVehicleOOS    *float64 `json:"nationalVehicleOosRate,omitempty"`
	NationalHazmatOOSRate *float64 `json:"nationalHazmatOosRate,omitempty"`
	LastInspectionAt      *int64   `json:"lastInspectionAt,omitempty"`
}

type Crashes struct {
	Total       *int   `json:"total,omitempty"`
	Fatal       *int   `json:"fatal,omitempty"`
	Injury      *int   `json:"injury,omitempty"`
	Tow         *int   `json:"tow,omitempty"`
	LastCrashAt *int64 `json:"lastCrashAt,omitempty"`
}

type Fleet struct {
	PowerUnits         *int `json:"powerUnits,omitempty"`
	Drivers            *int `json:"drivers,omitempty"`
	CDLDrivers         *int `json:"cdlDrivers,omitempty"`
	OwnedTractors      *int `json:"ownedTractors,omitempty"`
	TermLeasedTractors *int `json:"termLeasedTractors,omitempty"`
	OwnedTrailers      *int `json:"ownedTrailers,omitempty"`
	TermLeasedTrailers *int `json:"termLeasedTrailers,omitempty"`
	Trailers           *int `json:"trailers,omitempty"`
	Trucks             *int `json:"trucks,omitempty"`
}

type Equipment struct {
	VIN         string   `json:"vin,omitempty"`
	UnitType    UnitType `json:"unitType,omitempty"`
	Category    string   `json:"category,omitempty"`
	Make        string   `json:"make,omitempty"`
	Model       string   `json:"model,omitempty"`
	Year        *int     `json:"year,omitempty"`
	PlateNumber string   `json:"plateNumber,omitempty"`
	PlateState  string   `json:"plateState,omitempty"`
	UnitNumber  string   `json:"unitNumber,omitempty"`
}

type Contacts struct {
	Phone            string `json:"phone,omitempty"`
	Cellphone        string `json:"cellphone,omitempty"`
	Fax              string `json:"fax,omitempty"`
	Email            string `json:"email,omitempty"`
	PrimaryContact   string `json:"primaryContact,omitempty"`
	SecondaryContact string `json:"secondaryContact,omitempty"`
}

type Operations struct {
	Classification []string `json:"classification,omitempty"`
	CargoCarried   []string `json:"cargoCarried,omitempty"`
	HazmatCarrier  *bool    `json:"hazmatCarrier,omitempty"`
	MCS150At       *int64   `json:"mcs150At,omitempty"`
	MCS150Mileage  *int64   `json:"mcs150Mileage,omitempty"`
	BOC3OnFile     *bool    `json:"boc3OnFile,omitempty"`
	BOC3Agent      string   `json:"boc3Agent,omitempty"`
	SmartWay       *bool    `json:"smartWay,omitempty"`
	CARBCompliant  *bool    `json:"carbCompliant,omitempty"`
	PHMSA          *bool    `json:"phmsa,omitempty"`
}

type ChangeHistory struct {
	NameChanges         *int   `json:"nameChanges,omitempty"`
	NameLastChangedAt   *int64 `json:"nameLastChangedAt,omitempty"`
	EmailChanges        *int   `json:"emailChanges,omitempty"`
	EmailLastChangedAt  *int64 `json:"emailLastChangedAt,omitempty"`
	PhoneChanges        *int   `json:"phoneChanges,omitempty"`
	PhoneLastChangedAt  *int64 `json:"phoneLastChangedAt,omitempty"`
	AddressChanges      *int   `json:"addressChanges,omitempty"`
	AddressLastChangeAt *int64 `json:"addressLastChangedAt,omitempty"`
	ContactChanges      *int   `json:"contactChanges,omitempty"`
	ContactLastChangeAt *int64 `json:"contactLastChangedAt,omitempty"`
}

func (h *ChangeHistory) TotalChanges() int {
	if h == nil {
		return 0
	}
	total := 0
	for _, v := range []*int{h.NameChanges, h.EmailChanges, h.PhoneChanges, h.AddressChanges} {
		if v != nil {
			total += *v
		}
	}
	return total
}

func (h *ChangeHistory) LatestChangeAt() *int64 {
	if h == nil {
		return nil
	}
	var latest *int64
	for _, v := range []*int64{
		h.NameLastChangedAt, h.EmailLastChangedAt, h.PhoneLastChangedAt,
		h.AddressLastChangeAt, h.ContactLastChangeAt,
	} {
		if v != nil && (latest == nil || *v > *latest) {
			latest = v
		}
	}
	return latest
}

type NetworkLink struct {
	Kind      NetworkKind `json:"kind"`
	DOTNumber string      `json:"dotNumber,omitempty"`
	LegalName string      `json:"legalName,omitempty"`
	Value     string      `json:"value,omitempty"`
	Status    string      `json:"status,omitempty"`
}

type Network struct {
	SharedAddresses *int          `json:"sharedAddresses,omitempty"`
	SharedPhones    *int          `json:"sharedPhones,omitempty"`
	SharedEmails    *int          `json:"sharedEmails,omitempty"`
	SharedEINs      *int          `json:"sharedEins,omitempty"`
	SharedEquipment *int          `json:"sharedEquipment,omitempty"`
	Links           []NetworkLink `json:"links,omitempty"`
}

func (n *Network) SharedCount(kind NetworkKind) int {
	if n == nil {
		return 0
	}
	var v *int
	switch kind {
	case NetworkKindAddress:
		v = n.SharedAddresses
	case NetworkKindPhone:
		v = n.SharedPhones
	case NetworkKindEmail:
		v = n.SharedEmails
	case NetworkKindEIN:
		v = n.SharedEINs
	case NetworkKindEquipment:
		v = n.SharedEquipment
	}
	if v == nil {
		return 0
	}
	return *v
}

type Lane struct {
	OriginCity       string `json:"originCity,omitempty"`
	OriginState      string `json:"originState,omitempty"`
	DestinationCity  string `json:"destinationCity,omitempty"`
	DestinationState string `json:"destinationState,omitempty"`
	Loads            *int   `json:"loads,omitempty"`
}

type Lanes struct {
	TotalLoads      *int     `json:"totalLoads,omitempty"`
	FTLPercent      *float64 `json:"ftlPercent,omitempty"`
	LTLPercent      *float64 `json:"ltlPercent,omitempty"`
	DeadheadPercent *float64 `json:"deadheadPercent,omitempty"`
	FirstLoadAt     *int64   `json:"firstLoadAt,omitempty"`
	LastLoadAt      *int64   `json:"lastLoadAt,omitempty"`
	Preferred       []Lane   `json:"preferred,omitempty"`
	PreferredStates []string `json:"preferredStates,omitempty"`
}

type Benchmarks struct {
	AnyAnomaly               *bool `json:"anyAnomaly,omitempty"`
	InspectionMileageAnomaly *bool `json:"inspectionMileageAnomaly,omitempty"`
	InspectedUnitsAnomaly    *bool `json:"inspectedUnitsAnomaly,omitempty"`
	PowerUnitMileageAnomaly  *bool `json:"powerUnitMileageAnomaly,omitempty"`
}

func (p *Profile) Covers(section Section) bool {
	return p != nil && slices.Contains(p.Coverage, section)
}

func (p *Profile) CoversAll(sections []Section) bool {
	for _, s := range sections {
		if !p.Covers(s) {
			return false
		}
	}
	return true
}

func (p *Profile) DOTNumber() string {
	if p == nil || p.Identity == nil {
		return ""
	}
	return p.Identity.DOTNumber
}

func (p *Profile) Basic(basic worker.CSABasic) *BasicMeasure {
	if p == nil {
		return nil
	}
	for idx := range p.Basics {
		if p.Basics[idx].Basic == basic {
			return &p.Basics[idx]
		}
	}
	return nil
}

func (p *Profile) NormalizeCoverage() {
	if p == nil {
		return
	}
	present := map[Section]bool{
		SectionIdentity:      p.Identity != nil,
		SectionAuthority:     p.Authority != nil,
		SectionInsurance:     p.Insurance != nil,
		SectionSafety:        p.Safety != nil,
		SectionBasics:        len(p.Basics) > 0,
		SectionInspections:   p.Inspections != nil,
		SectionCrashes:       p.Crashes != nil,
		SectionFleet:         p.Fleet != nil,
		SectionEquipment:     len(p.Equipment) > 0,
		SectionContacts:      p.Contacts != nil,
		SectionOperations:    p.Operations != nil,
		SectionChangeHistory: p.ChangeHistory != nil,
		SectionNetwork:       p.Network != nil,
		SectionLanes:         p.Lanes != nil,
		SectionBenchmarks:    p.Benchmarks != nil,
	}
	coverage := make([]Section, 0, len(present))
	for _, section := range AllSections() {
		if present[section] || slices.Contains(p.Coverage, section) {
			coverage = append(coverage, section)
		}
	}
	p.Coverage = coverage
}

func (p *Profile) ContentHash() (string, error) {
	raw, err := sonic.Marshal(p)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

func (p *Profile) DeriveRiskLevel(findings []Finding) RiskLevel {
	if p == nil {
		return RiskLevelUnknown
	}
	level := RiskLevelUnknown
	if p.Safety != nil && p.Safety.RiskScore.IsValid() && p.Safety.RiskScore != RiskLevelUnknown {
		level = p.Safety.RiskScore
	}

	blocking, warnings := 0, 0
	for idx := range findings {
		finding := &findings[idx]
		if finding.Unverifiable || finding.Overridden {
			continue
		}
		switch finding.Action {
		case RuleActionBlock:
			blocking++
		case RuleActionWarn:
			warnings++
		}
	}

	derived := RiskLevelLow
	switch {
	case blocking > 0:
		derived = RiskLevelVeryHigh
	case warnings >= 3:
		derived = RiskLevelHigh
	case warnings == 2:
		derived = RiskLevelElevated
	case warnings == 1:
		derived = RiskLevelModerate
	}

	if derived.Rank() > level.Rank() {
		return derived
	}
	return level
}
