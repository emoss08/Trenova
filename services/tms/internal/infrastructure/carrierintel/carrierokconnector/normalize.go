package carrierokconnector

import (
	"bytes"
	"math"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/jsonflex"
)

const (
	daysPerYear = 365.25
	riskKey     = "risk_score"
)

var riskKeyToken = []byte(`"` + riskKey + `"`)

var basicCategories = map[carrierok.BasicCategory]worker.CSABasic{
	carrierok.BasicUnsafeDriving:       worker.BasicUnsafeDriving,
	carrierok.BasicHoursOfService:      worker.BasicHOSCompliance,
	carrierok.BasicVehicleMaintenance:  worker.BasicVehicleMaintenance,
	carrierok.BasicControlledSubstance: worker.BasicControlledSubstances,
	carrierok.BasicDriverFitness:       worker.BasicDriverFitness,
	carrierok.BasicHazardousMaterials:  worker.BasicHazmatCompliance,
	carrierok.BasicCrashIndicator:      worker.BasicCrashIndicator,
}

var networkKinds = map[carrierok.NetworkLinkKind]carrierintel.NetworkKind{
	carrierok.NetworkLinkPhysicalAddress: carrierintel.NetworkKindAddress,
	carrierok.NetworkLinkTelephone:       carrierintel.NetworkKindPhone,
	carrierok.NetworkLinkEmail:           carrierintel.NetworkKindEmail,
	carrierok.NetworkLinkEIN:             carrierintel.NetworkKindEIN,
	carrierok.NetworkLinkEquipment:       carrierintel.NetworkKindEquipment,
}

func normalizeProfile(p *carrierok.Profile) *carrierintel.Profile {
	if p == nil {
		return &carrierintel.Profile{Coverage: []carrierintel.Section{}}
	}

	profile := &carrierintel.Profile{
		Identity:      normalizeIdentity(p),
		Authority:     normalizeAuthority(&p.Authority),
		Insurance:     normalizeInsurance(&p.Insurance),
		Safety:        normalizeSafety(p),
		Basics:        normalizeBasics(p.Basics),
		Inspections:   normalizeInspections(&p.Inspections),
		Crashes:       normalizeCrashes(&p.Crashes),
		Fleet:         normalizeFleet(&p.Fleet),
		Equipment:     normalizeEquipment(p.Fleet.Equipment),
		Contacts:      normalizeContacts(&p.Contacts),
		Operations:    normalizeOperations(&p.Operations),
		ChangeHistory: normalizeChangeHistory(&p.ChangeHistory),
		Network:       normalizeNetwork(&p.Network),
		Lanes:         normalizeLanes(&p.Loads),
		Benchmarks:    normalizeBenchmarks(&p.Benchmarks),
	}
	profile.NormalizeCoverage()
	return profile
}

func normalizeIdentity(p *carrierok.Profile) *carrierintel.Identity {
	src := &p.Identity
	prefix := strings.ToUpper(intelkit.Text(src.DocketPrefix))
	number := intelkit.Text(src.DocketNumber)
	if splitPrefix, splitNumber := intelkit.SplitDocket(number); splitNumber != "" {
		number = splitNumber
		if prefix == "" {
			prefix = splitPrefix
		}
	}
	if number == "" {
		splitPrefix, splitNumber := intelkit.SplitDocket(intelkit.Text(src.Docket))
		number = splitNumber
		if prefix == "" {
			prefix = splitPrefix
		}
	}

	identity := &carrierintel.Identity{
		DOTNumber:        intelkit.Text(src.DOTNumber),
		DocketPrefix:     prefix,
		DocketNumber:     number,
		LegalName:        intelkit.Text(src.LegalName),
		DBAName:          intelkit.Text(src.DBAName),
		EIN:              intelkit.Text(src.EIN),
		USDOTStatus:      strings.ToUpper(intelkit.Text(src.USDOTStatus)),
		EntityType:       intelkit.Text(src.EntityType),
		CarrierOperation: intelkit.Text(src.CarrierOperation),
		DOTAddedAt:       intelkit.Unix(src.AddedDate),
		DOTAgeDays:       yearsToDays(src.DOTAge),
		PhysicalAddress:  normalizeAddress(&p.Addresses.Physical),
		MailingAddress:   normalizeAddress(&p.Addresses.Mailing),
	}
	if identity.DocketNumber == "" {
		identity.DocketPrefix = ""
	}
	if *identity == (carrierintel.Identity{}) {
		return nil
	}
	return identity
}

func yearsToDays(years *jsonflex.Float) *int {
	if years == nil || years.Value() < 0 {
		return nil
	}
	days := int(math.Round(years.Value() * daysPerYear))
	return &days
}

func normalizeAddress(src *carrierok.Address) *carrierintel.Address {
	return intelkit.Address(intelkit.AddressParts{
		Full:        intelkit.Text(src.Full),
		Street:      intelkit.Text(src.Street),
		City:        intelkit.Text(src.City),
		State:       intelkit.Text(src.State),
		PostalCode:  intelkit.Text(src.ZipCode),
		Country:     intelkit.Text(src.CountryCode),
		Undelivered: intelkit.Bool(src.Undeliverable),
	})
}

type grantSource struct {
	status     *jsonflex.String
	pending    *jsonflex.String
	review     *jsonflex.String
	revocation *jsonflex.String
	age        *jsonflex.Float
	start      *jsonflex.Time
}

func normalizeAuthority(src *carrierok.Authority) *carrierintel.Authority {
	authority := &carrierintel.Authority{
		Common: normalizeGrant(grantSource{
			status:     src.Common,
			pending:    src.CommonPending,
			review:     src.CommonReview,
			revocation: src.CommonRevocation,
			age:        src.AgeCommon,
			start:      src.StartCommon,
		}),
		Contract: normalizeGrant(grantSource{
			status:     src.Contract,
			pending:    src.ContractPending,
			review:     src.ContractReview,
			revocation: src.ContractRevocation,
			age:        src.AgeContract,
			start:      src.StartContract,
		}),
		Broker: normalizeGrant(grantSource{
			status:     src.Broker,
			pending:    src.BrokerPending,
			review:     src.BrokerReview,
			revocation: src.BrokerRevocation,
			age:        src.AgeBroker,
		}),
		TotalRevocations: intelkit.Int(src.TotalRevocations),
		LastRevocationAt: intelkit.Unix(src.LastRevocationDate),
		History:          normalizeAuthorityHistory(src.History),
	}
	if authority.Common == nil && authority.Contract == nil && authority.Broker == nil &&
		authority.TotalRevocations == nil && authority.LastRevocationAt == nil &&
		len(authority.History) == 0 {
		return nil
	}
	return authority
}

func normalizeGrant(src grantSource) *carrierintel.AuthorityGrant {
	if src.status == nil && src.pending == nil && src.review == nil && src.revocation == nil &&
		src.age == nil && src.start == nil {
		return nil
	}

	status := intelkit.AuthorityStatus(intelkit.Text(src.status))
	if src.status == nil {
		status = carrierintel.AuthorityStatusUnknown
	}
	revocationFlagged := intelkit.Truthy(src.revocation)
	if revocationFlagged && status != carrierintel.AuthorityStatusActive {
		status = carrierintel.AuthorityStatusRevoked
	}

	grant := &carrierintel.AuthorityGrant{
		Status:            status,
		Pending:           intelkit.Truthy(src.pending),
		UnderReview:       intelkit.Truthy(src.review),
		RevocationPending: revocationFlagged && status == carrierintel.AuthorityStatusActive,
		GrantedAt:         intelkit.Unix(src.start),
		AgeDays:           intelkit.IntFromFloat(src.age),
	}
	return grant
}

func normalizeAuthorityHistory(
	events []carrierok.AuthorityEvent,
) []carrierintel.AuthorityHistoryEntry {
	if len(events) == 0 {
		return nil
	}
	history := make([]carrierintel.AuthorityHistoryEntry, 0, len(events))
	for idx := range events {
		event := &events[idx]
		entry := carrierintel.AuthorityHistoryEntry{
			AuthorityType: strings.ToUpper(intelkit.Text(event.AuthorityType)),
			Action:        strings.ToUpper(intelkit.Text(event.Action)),
			ServedAt:      intelkit.Unix(event.ServedDate),
			EffectiveAt:   intelkit.Unix(event.EffectiveDate),
		}
		if entry == (carrierintel.AuthorityHistoryEntry{}) {
			continue
		}
		history = append(history, entry)
	}
	if len(history) == 0 {
		return nil
	}
	return history
}

func normalizeInsurance(src *carrierok.Insurance) *carrierintel.Insurance {
	insurance := &carrierintel.Insurance{
		BIPDOnFile:      intelkit.Decimal(src.BIPDOnFile),
		BIPDRequired:    intelkit.Decimal(src.BIPDRequired),
		CargoOnFile:     intelkit.Decimal(src.CargoOnFile),
		CargoRequired:   intelkit.Decimal(src.CargoRequired),
		BondOnFile:      intelkit.Decimal(src.BondOnFile),
		BondRequired:    intelkit.Decimal(src.BondRequired),
		PendingCancelAt: intelkit.Unix(src.PendingCancelDate),
		LastCanceledAt:  intelkit.Unix(src.LastCanceled),
		CancelCount:     intelkit.Int(src.CancelCount),
		Filings:         normalizeFilings(src.History),
	}
	if insurance.BIPDOnFile == nil && insurance.BIPDRequired == nil &&
		insurance.CargoOnFile == nil && insurance.CargoRequired == nil &&
		insurance.BondOnFile == nil && insurance.BondRequired == nil &&
		insurance.PendingCancelAt == nil && insurance.LastCanceledAt == nil &&
		insurance.CancelCount == nil && len(insurance.Filings) == 0 {
		return nil
	}
	return insurance
}

func normalizeFilings(policies []carrierok.InsurancePolicy) []carrierintel.InsuranceFiling {
	if len(policies) == 0 {
		return nil
	}
	filings := make([]carrierintel.InsuranceFiling, 0, len(policies))
	for idx := range policies {
		policy := &policies[idx]
		filings = append(filings, carrierintel.InsuranceFiling{
			Type:              filingType(intelkit.Text(policy.Type)),
			InsurerName:       intelkit.Text(policy.Insurer),
			PolicyNumber:      intelkit.Text(policy.PolicyNumber),
			Coverage:          intelkit.Decimal(policy.Coverage),
			EffectiveAt:       intelkit.Unix(policy.EffectiveDate),
			CancelEffectiveAt: intelkit.Unix(policy.CancelEffectiveDate),
			CancelMethod:      intelkit.Text(policy.CancelMethod),
		})
	}
	return filings
}

func filingType(value string) carrierintel.InsuranceFilingType {
	upper := strings.ToUpper(value)
	switch {
	case strings.Contains(upper, "BIPD"), strings.Contains(upper, "BI&PD"),
		strings.Contains(upper, "BI & PD"), strings.Contains(upper, "LIABILITY"):
		return carrierintel.InsuranceFilingTypeBIPD
	case strings.Contains(upper, "CARGO"):
		return carrierintel.InsuranceFilingTypeCargo
	case strings.Contains(upper, "BOND"), strings.Contains(upper, "SURETY"),
		strings.Contains(upper, "TRUST"):
		return carrierintel.InsuranceFilingTypeBond
	default:
		return carrierintel.InsuranceFilingTypeOther
	}
}

func normalizeSafety(p *carrierok.Profile) *carrierintel.Safety {
	src := &p.Safety
	safety := &carrierintel.Safety{
		RatingDate:        intelkit.Unix(src.SafetyRatingDate),
		ISSValue:          intelkit.IntFromFloat(src.ISSValue),
		ISSRecommendation: intelkit.Text(src.ISSRecommendation),
		RiskScore:         riskLevel(p),
		RiskProbability:   intelkit.Float(src.RiskScoreProbability),
		SafetyScore:       intelkit.Float(src.SafetyScore),
		OutOfServiceOrder: intelkit.Bool(src.OutOfServiceFlag),
		OutOfServiceAt:    intelkit.Unix(src.OutOfServiceDate),
		LatestReviewType:  intelkit.Text(src.LatestReviewType),
		LatestReviewAt:    intelkit.Unix(src.LatestReviewDate),
	}
	if src.SafetyRating != nil {
		safety.Rating = intelkit.SafetyRating(intelkit.Text(src.SafetyRating))
	}
	if *safety == (carrierintel.Safety{}) {
		return nil
	}
	return safety
}

func riskLevel(p *carrierok.Profile) carrierintel.RiskLevel {
	if p.Safety.RiskScore != nil || !bytes.Contains(p.Raw, riskKeyToken) {
		return ""
	}
	obj, err := jsonflex.DecodeObject(p.Raw)
	if err != nil {
		return ""
	}
	label := strings.ToUpper(strings.Join(strings.Fields(obj.Text(riskKey)), ""))
	switch label {
	case "LOW":
		return carrierintel.RiskLevelLow
	case "MEDIUM", "MODERATE":
		return carrierintel.RiskLevelModerate
	case "ELEVATED":
		return carrierintel.RiskLevelElevated
	case "HIGH":
		return carrierintel.RiskLevelHigh
	case "VERYHIGH":
		return carrierintel.RiskLevelVeryHigh
	default:
		return ""
	}
}

func normalizeBasics(
	src map[carrierok.BasicCategory]carrierok.BasicScore,
) []carrierintel.BasicMeasure {
	if len(src) == 0 {
		return nil
	}
	categories := carrierok.BasicCategories()
	basics := make([]carrierintel.BasicMeasure, 0, len(categories))
	for _, category := range categories {
		score, ok := src[category]
		if !ok {
			continue
		}
		basic, known := basicCategories[category]
		if !known {
			continue
		}
		basics = append(basics, carrierintel.BasicMeasure{
			Basic:         basic,
			Measure:       intelkit.NonNegativeFloat(score.Measure),
			Percentile:    intelkit.Percent(score.Percentile),
			Threshold:     intelkit.Percent(score.InterventionThreshold),
			Alert:         score.Alert.Value(),
			RoadsideAlert: score.RoadsideAlert.Value(),
			ACIndicator:   score.ACIndicator.Value(),
		})
	}
	if len(basics) == 0 {
		return nil
	}
	return basics
}

func normalizeInspections(src *carrierok.Inspections) *carrierintel.Inspections {
	inspections := &carrierintel.Inspections{
		Total:                 intelkit.Int(src.Total),
		Driver:                intelkit.Int(src.Driver),
		Vehicle:               intelkit.Int(src.Vehicle),
		Hazmat:                intelkit.Int(src.Hazmat),
		DriverOOS:             intelkit.Int(src.DriverOutOfService),
		VehicleOOS:            intelkit.Int(src.VehicleOutOfService),
		HazmatOOS:             intelkit.Int(src.HazmatOutOfService),
		DriverOOSRate:         intelkit.Percent(src.DriverOutOfServicePct),
		VehicleOOSRate:        intelkit.Percent(src.VehicleOutOfServicePct),
		HazmatOOSRate:         intelkit.Percent(src.HazmatOutOfServicePct),
		NationalDriverOOSRate: intelkit.Percent(src.NationalAvgOOSDriver),
		NationalVehicleOOS:    intelkit.Percent(src.NationalAvgOOSVehicle),
		NationalHazmatOOSRate: intelkit.Percent(src.NationalAvgOOSHazmat),
		LastInspectionAt:      intelkit.Unix(src.LastInspectionDate),
	}
	if *inspections == (carrierintel.Inspections{}) {
		return nil
	}
	return inspections
}

func normalizeCrashes(src *carrierok.Crashes) *carrierintel.Crashes {
	crashes := &carrierintel.Crashes{
		Total:       intelkit.Int(src.Total),
		Fatal:       intelkit.Int(src.Fatalities),
		Injury:      intelkit.Int(src.Injuries),
		Tow:         intelkit.Int(src.TowAway),
		LastCrashAt: intelkit.Unix(src.LastCrashDate),
	}
	if *crashes == (carrierintel.Crashes{}) {
		return nil
	}
	return crashes
}

func normalizeFleet(src *carrierok.Fleet) *carrierintel.Fleet {
	fleet := &carrierintel.Fleet{
		PowerUnits:         intelkit.Int(src.TotalPowerUnits),
		Drivers:            intelkit.Int(src.TotalDrivers),
		CDLDrivers:         intelkit.Int(src.TotalDriversCDL),
		OwnedTractors:      intelkit.Int(src.OwnedTractors),
		TermLeasedTractors: intelkit.Int(src.TermLeasedTractors),
		OwnedTrailers:      intelkit.Int(src.OwnedTrailers),
		TermLeasedTrailers: intelkit.Int(src.TermLeasedTrailers),
		Trailers:           intelkit.Int(src.TotalTrailers),
		Trucks:             intelkit.Int(src.TotalTrucks),
	}
	if *fleet == (carrierintel.Fleet{}) {
		return nil
	}
	return fleet
}

func normalizeEquipment(src []carrierok.Equipment) []carrierintel.Equipment {
	if len(src) == 0 {
		return nil
	}
	equipment := make([]carrierintel.Equipment, 0, len(src))
	for idx := range src {
		item := &src[idx]
		category := strings.ToUpper(intelkit.Text(item.UnitType))
		unit := carrierintel.Equipment{
			VIN:         strings.ToUpper(intelkit.Text(item.VIN)),
			UnitType:    unitType(category),
			Category:    category,
			Make:        intelkit.Text(item.Make),
			Model:       intelkit.Text(item.Model),
			Year:        intelkit.Int(item.Year),
			PlateNumber: strings.ToUpper(intelkit.Text(item.PlateNumber)),
			PlateState:  strings.ToUpper(intelkit.Text(item.PlateState)),
			UnitNumber:  intelkit.Text(item.UnitNumber),
		}
		if unit.VIN == "" && unit.PlateNumber == "" && unit.UnitNumber == "" {
			continue
		}
		equipment = append(equipment, unit)
	}
	if len(equipment) == 0 {
		return nil
	}
	return equipment
}

func unitType(category string) carrierintel.UnitType {
	switch {
	case strings.Contains(category, "TRAILER"):
		return carrierintel.UnitTypeTrailer
	case strings.Contains(category, "STRAIGHT"):
		return carrierintel.UnitTypeStraight
	case strings.Contains(category, "TRUCK"), strings.Contains(category, "TRACTOR"):
		return carrierintel.UnitTypeTractor
	default:
		return ""
	}
}

func normalizeContacts(src *carrierok.Contacts) *carrierintel.Contacts {
	contacts := &carrierintel.Contacts{
		Phone:            intelkit.Text(src.Telephone),
		Cellphone:        intelkit.Text(src.Cellphone),
		Fax:              intelkit.Text(src.Fax),
		Email:            strings.ToLower(intelkit.Text(src.Email)),
		PrimaryContact:   intelkit.Text(src.PrimaryContact),
		SecondaryContact: intelkit.Text(src.SecondaryContact),
	}
	if *contacts == (carrierintel.Contacts{}) {
		return nil
	}
	return contacts
}

func normalizeOperations(src *carrierok.Operations) *carrierintel.Operations {
	operations := &carrierintel.Operations{
		Classification: intelkit.SplitList(src.OperationClassification),
		CargoCarried:   intelkit.SplitList(src.CargoCarried),
		HazmatCarrier:  intelkit.Bool(src.HazardousMaterial),
		MCS150At:       intelkit.Unix(src.MCS150Date),
		MCS150Mileage:  intelkit.Int64(src.MCS150Mileage),
		SmartWay:       intelkit.Bool(src.SmartWay),
		CARBCompliant:  intelkit.Bool(src.CARBTRU),
		PHMSA:          intelkit.Bool(src.PHMSA),
	}
	if agent := intelkit.Text(src.BOC3CompanyName); agent != "" {
		onFile := true
		operations.BOC3Agent = agent
		operations.BOC3OnFile = &onFile
	}
	if len(operations.Classification) == 0 && len(operations.CargoCarried) == 0 &&
		operations.HazmatCarrier == nil && operations.MCS150At == nil &&
		operations.MCS150Mileage == nil && operations.BOC3OnFile == nil &&
		operations.SmartWay == nil && operations.CARBCompliant == nil && operations.PHMSA == nil {
		return nil
	}
	return operations
}

func normalizeChangeHistory(src *carrierok.ChangeHistory) *carrierintel.ChangeHistory {
	history := &carrierintel.ChangeHistory{
		NameChanges:         intelkit.Int(src.NameChangeCount),
		NameLastChangedAt:   intelkit.Unix(src.NameLastChanged),
		EmailChanges:        intelkit.Int(src.EmailChangeCount),
		EmailLastChangedAt:  intelkit.Unix(src.EmailLastChanged),
		PhoneChanges:        intelkit.Int(src.PhoneChangeCount),
		PhoneLastChangedAt:  intelkit.Unix(src.PhoneLastChanged),
		AddressChanges:      intelkit.Int(src.AddressChangeCount),
		AddressLastChangeAt: intelkit.Unix(src.AddressLastChanged),
		ContactChanges:      intelkit.Int(src.ContactChangeCount),
		ContactLastChangeAt: intelkit.Unix(src.ContactLastChanged),
	}
	if *history == (carrierintel.ChangeHistory{}) {
		return nil
	}
	return history
}

func normalizeNetwork(src *carrierok.Network) *carrierintel.Network {
	network := &carrierintel.Network{
		SharedAddresses: intelkit.SumInts(src.PhysicalAddressCount, src.MailingAddressCount),
		SharedPhones:    intelkit.SumInts(src.TelephoneNumberCount, src.CellphoneNumberCount),
		SharedEmails:    intelkit.Int(src.EmailAddressCount),
		SharedEINs:      intelkit.Int(src.EINCount),
		SharedEquipment: intelkit.Int(src.EquipmentCount),
		Links:           normalizeNetworkLinks(src.Links),
	}
	if network.SharedAddresses == nil && network.SharedPhones == nil &&
		network.SharedEmails == nil && network.SharedEINs == nil &&
		network.SharedEquipment == nil && len(network.Links) == 0 {
		return nil
	}
	return network
}

func normalizeNetworkLinks(src []carrierok.NetworkLink) []carrierintel.NetworkLink {
	if len(src) == 0 {
		return nil
	}
	links := make([]carrierintel.NetworkLink, 0, len(src))
	for idx := range src {
		link := &src[idx]
		kind, ok := networkKinds[link.Kind]
		if !ok {
			continue
		}
		links = append(links, carrierintel.NetworkLink{
			Kind:      kind,
			DOTNumber: link.DOTNumber,
			LegalName: link.LegalName,
			Value:     link.Value,
			Status:    strings.ToUpper(link.Status),
		})
	}
	if len(links) == 0 {
		return nil
	}
	return links
}

func normalizeLanes(src *carrierok.Loads) *carrierintel.Lanes {
	lanes := &carrierintel.Lanes{
		TotalLoads:      intelkit.Int(src.Total),
		FTLPercent:      intelkit.NonNegativeFloat(src.FTLPercentage),
		LTLPercent:      intelkit.NonNegativeFloat(src.LTLPercentage),
		DeadheadPercent: intelkit.NonNegativeFloat(src.DeadheadPercentage),
		FirstLoadAt:     intelkit.Unix(src.FirstLoadDate),
		LastLoadAt:      intelkit.Unix(src.LastLoadDate),
		Preferred:       normalizePreferredLanes(src.PreferredLanes),
	}
	if lanes.TotalLoads == nil && lanes.FTLPercent == nil && lanes.LTLPercent == nil &&
		lanes.DeadheadPercent == nil && lanes.FirstLoadAt == nil && lanes.LastLoadAt == nil &&
		len(lanes.Preferred) == 0 {
		return nil
	}
	return lanes
}

func normalizePreferredLanes(src []carrierok.Lane) []carrierintel.Lane {
	if len(src) == 0 {
		return nil
	}
	lanes := make([]carrierintel.Lane, 0, len(src))
	for idx := range src {
		lane := &src[idx]
		entry := carrierintel.Lane{
			OriginCity:       intelkit.Text(lane.OriginCity),
			OriginState:      strings.ToUpper(intelkit.Text(lane.OriginState)),
			DestinationCity:  intelkit.Text(lane.DestinationCity),
			DestinationState: strings.ToUpper(intelkit.Text(lane.DestinationState)),
			Loads:            intelkit.Int(lane.Loads),
		}
		if entry == (carrierintel.Lane{}) {
			continue
		}
		lanes = append(lanes, entry)
	}
	if len(lanes) == 0 {
		return nil
	}
	return lanes
}

func normalizeBenchmarks(src *carrierok.Benchmarks) *carrierintel.Benchmarks {
	benchmarks := &carrierintel.Benchmarks{
		AnyAnomaly:               anomaly(src.IndicatorIndustry),
		InspectionMileageAnomaly: anomaly(src.InspectionMileageRatio),
		InspectedUnitsAnomaly:    anomaly(src.InspectedPowerUnitsRatio),
		PowerUnitMileageAnomaly:  anomaly(src.PowerUnitMileageRatio),
	}
	if *benchmarks == (carrierintel.Benchmarks{}) {
		return nil
	}
	return benchmarks
}

func anomaly(indicator *jsonflex.String) *bool {
	if indicator == nil {
		return nil
	}
	var flagged bool
	switch strings.ToUpper(intelkit.Text(indicator)) {
	case "GREEN", "N", "NO", "FALSE", "NORMAL", "OK":
		flagged = false
	case "YELLOW", "RED", "ORANGE", "Y", "YES", "TRUE", "ANOMALY", "ANOMALOUS":
		flagged = true
	default:
		return nil
	}
	return &flagged
}
