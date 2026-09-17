package carrierokconnector

import (
	"cmp"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/carrierok"
	"github.com/emoss08/trenova/shared/jsonflex"
)

const riskFactorBOC3OnFile = "boc3_on_file"

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
	carrierok.NetworkLinkMailingAddress:  carrierintel.NetworkKindAddress,
	carrierok.NetworkLinkTelephone:       carrierintel.NetworkKindPhone,
	carrierok.NetworkLinkCellphone:       carrierintel.NetworkKindPhone,
	carrierok.NetworkLinkFax:             carrierintel.NetworkKindPhone,
	carrierok.NetworkLinkEmail:           carrierintel.NetworkKindEmail,
	carrierok.NetworkLinkEIN:             carrierintel.NetworkKindEIN,
	carrierok.NetworkLinkEquipment:       carrierintel.NetworkKindEquipment,
	carrierok.NetworkLinkEquipmentExt:    carrierintel.NetworkKindEquipment,
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
		Operations:    normalizeOperations(p),
		ChangeHistory: normalizeChangeHistory(&p.ChangeHistory),
		Network:       normalizeNetwork(&p.Network, intelkit.Text(p.Identity.DOTNumber)),
		Lanes:         normalizeLanes(&p.Lanes),
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
		DOTAgeDays:       intelkit.Int(src.DOTAgeDays),
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
	pending    *jsonflex.Bool
	review     *jsonflex.Bool
	revocation *jsonflex.Bool
	ageDays    *jsonflex.Int
	start      *jsonflex.Time
}

func normalizeAuthority(src *carrierok.Authority) *carrierintel.Authority {
	authority := &carrierintel.Authority{
		Common: normalizeGrant(grantSource{
			status:     src.Common,
			pending:    src.CommonPending,
			review:     src.CommonReview,
			revocation: src.CommonRevocation,
			ageDays:    src.AgeCommonDays,
			start:      src.StartCommon,
		}),
		Contract: normalizeGrant(grantSource{
			status:     src.Contract,
			pending:    src.ContractPending,
			review:     src.ContractReview,
			revocation: src.ContractRevocation,
			ageDays:    src.AgeContractDays,
			start:      src.StartContract,
		}),
		Broker: normalizeGrant(grantSource{
			status:     src.Broker,
			pending:    src.BrokerPending,
			review:     src.BrokerReview,
			revocation: src.BrokerRevocation,
			ageDays:    src.AgeBrokerDays,
			start:      src.StartBroker,
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
		src.ageDays == nil && src.start == nil {
		return nil
	}

	status := intelkit.AuthorityStatus(intelkit.Text(src.status))
	if src.status == nil {
		status = carrierintel.AuthorityStatusUnknown
	}
	revocationFlagged := src.revocation.Value()
	if revocationFlagged && status != carrierintel.AuthorityStatusActive {
		status = carrierintel.AuthorityStatusRevoked
	}

	grant := &carrierintel.AuthorityGrant{
		Status:            status,
		Pending:           src.pending.Value(),
		UnderReview:       src.review.Value(),
		RevocationPending: revocationFlagged && status == carrierintel.AuthorityStatusActive,
		GrantedAt:         intelkit.Unix(src.start),
		AgeDays:           intelkit.Int(src.ageDays),
	}
	return grant
}

func normalizeAuthorityHistory(
	events []carrierok.AuthorityEvent,
) []carrierintel.AuthorityHistoryEntry {
	if len(events) == 0 {
		return nil
	}
	history := make([]carrierintel.AuthorityHistoryEntry, 0, len(events)*2)
	for idx := range events {
		event := &events[idx]
		authorityType := strings.ToUpper(intelkit.Text(event.AuthorityType))
		history = appendAuthorityEntry(
			history,
			authorityType,
			event.OriginalAction,
			event.OriginalServedDate,
		)
		history = appendAuthorityEntry(
			history,
			authorityType,
			event.DispositionAction,
			event.DispositionServedDate,
		)
	}
	if len(history) == 0 {
		return nil
	}
	slices.SortStableFunc(history, func(a, b carrierintel.AuthorityHistoryEntry) int {
		return compareServedDesc(a.ServedAt, b.ServedAt)
	})
	return history
}

func appendAuthorityEntry(
	history []carrierintel.AuthorityHistoryEntry,
	authorityType string,
	action *jsonflex.String,
	served *jsonflex.Time,
) []carrierintel.AuthorityHistoryEntry {
	entry := carrierintel.AuthorityHistoryEntry{
		AuthorityType: authorityType,
		Action:        strings.ToUpper(intelkit.Text(action)),
		ServedAt:      intelkit.Unix(served),
	}
	if entry.Action == "" && entry.ServedAt == nil {
		return history
	}
	return append(history, entry)
}

func compareServedDesc(a, b *int64) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return 1
	case b == nil:
		return -1
	default:
		return cmp.Compare(*b, *a)
	}
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
			Status:            strings.ToUpper(intelkit.Text(policy.Status)),
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
		ISSValue:          intelkit.Int(src.ISSValue),
		ISSRecommendation: intelkit.Text(src.ISSRecommendation),
		RiskScore:         riskLevel(intelkit.Text(src.RiskScore)),
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

func riskLevel(value string) carrierintel.RiskLevel {
	label := strings.ToUpper(strings.Join(strings.Fields(value), ""))
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
			Alert:         score.Alert.Value(),
			ACIndicator:   score.ACIndicator.Value(),
			Violations:    intelkit.Int(score.Violations),
			OOSViolations: intelkit.Int(score.OOSViolations),
			MeasuredAt:    intelkit.Unix(score.MeasuredAt),
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
		DriverOOSRate:         intelkit.FractionPercent(src.DriverOutOfServiceRate),
		VehicleOOSRate:        intelkit.FractionPercent(src.VehicleOutOfServiceRate),
		HazmatOOSRate:         intelkit.FractionPercent(src.HazmatOutOfServiceRate),
		NationalDriverOOSRate: intelkit.FractionPercent(src.NationalAvgOOSDriver),
		NationalVehicleOOS:    intelkit.FractionPercent(src.NationalAvgOOSVehicle),
		NationalHazmatOOSRate: intelkit.FractionPercent(src.NationalAvgOOSHazmat),
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

func normalizeOperations(p *carrierok.Profile) *carrierintel.Operations {
	src := &p.Operations
	operations := &carrierintel.Operations{
		Classification: intelkit.SplitList(src.OperationClassification),
		CargoCarried:   intelkit.SplitList(src.CargoCarried),
		HazmatCarrier:  intelkit.Bool(src.HazardousMaterial),
		MCS150At:       intelkit.Unix(src.MCS150Date),
		MCS150Mileage:  intelkit.Int64(src.MCS150Mileage),
		BOC3Agent:      intelkit.Text(src.BOC3CompanyName),
		BOC3OnFile:     p.RiskFactor(riskFactorBOC3OnFile),
		SmartWay:       intelkit.Bool(src.SmartWay),
		CARBCompliant:  intelkit.Bool(src.CARBTRU),
		PHMSA:          intelkit.Bool(src.PHMSA),
	}
	if operations.BOC3OnFile == nil && operations.BOC3Agent != "" {
		onFile := true
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

func normalizeNetwork(src *carrierok.Network, ownDOT string) *carrierintel.Network {
	links := normalizeNetworkLinks(src.Links, ownDOT)
	network := &carrierintel.Network{
		SharedAddresses: sharedCount(
			links,
			carrierintel.NetworkKindAddress,
			src.PhysicalAddressCount,
			src.MailingAddressCount,
		),
		SharedPhones: sharedCount(
			links,
			carrierintel.NetworkKindPhone,
			src.TelephoneNumberCount,
			src.CellphoneNumberCount,
			src.FaxNumberCount,
		),
		SharedEmails: sharedCount(links, carrierintel.NetworkKindEmail, src.EmailAddressCount),
		SharedEINs:   sharedCount(links, carrierintel.NetworkKindEIN, src.EINCount),
		SharedEquipment: sharedCount(
			links,
			carrierintel.NetworkKindEquipment,
			src.PowerUnitsCount,
			src.TrailersCount,
		),
		Links: links,
	}
	if network.SharedAddresses == nil && network.SharedPhones == nil &&
		network.SharedEmails == nil && network.SharedEINs == nil &&
		network.SharedEquipment == nil && len(network.Links) == 0 {
		return nil
	}
	return network
}

func normalizeNetworkLinks(src []carrierok.NetworkLink, ownDOT string) []carrierintel.NetworkLink {
	if len(src) == 0 {
		return nil
	}
	links := make([]carrierintel.NetworkLink, 0, len(src))
	for idx := range src {
		kind, ok := networkKinds[src[idx].Kind]
		dot := strings.TrimSpace(src[idx].DOTNumber)
		if !ok || dot == "" || dot == ownDOT {
			continue
		}
		link := carrierintel.NetworkLink{Kind: kind, DOTNumber: dot}
		if slices.Contains(links, link) {
			continue
		}
		links = append(links, link)
	}
	if len(links) == 0 {
		return nil
	}
	return links
}

func sharedCount(
	links []carrierintel.NetworkLink,
	kind carrierintel.NetworkKind,
	vendorCounts ...*jsonflex.Int,
) *int {
	linked := 0
	for idx := range links {
		if links[idx].Kind == kind {
			linked++
		}
	}
	if linked > 0 {
		return &linked
	}
	return intelkit.SumInts(vendorCounts...)
}

func normalizeLanes(src *carrierok.Lanes) *carrierintel.Lanes {
	if len(src.PreferredStates) == 0 {
		return nil
	}
	states := make([]string, 0, len(src.PreferredStates))
	for _, state := range src.PreferredStates {
		normalized := strings.ToUpper(strings.TrimSpace(state))
		if normalized != "" && !slices.Contains(states, normalized) {
			states = append(states, normalized)
		}
	}
	if len(states) == 0 {
		return nil
	}
	return &carrierintel.Lanes{PreferredStates: states}
}

func normalizeBenchmarks(src *carrierok.Benchmarks) *carrierintel.Benchmarks {
	benchmarks := &carrierintel.Benchmarks{
		AnyAnomaly:               intelkit.Bool(src.IndicatorIndustry),
		InspectionMileageAnomaly: intelkit.Bool(src.InspectionMileageRatio),
		InspectedUnitsAnomaly:    intelkit.Bool(src.InspectedPowerUnitsRatio),
		PowerUnitMileageAnomaly:  intelkit.Bool(src.PowerUnitMileageRatio),
	}
	if *benchmarks == (carrierintel.Benchmarks{}) {
		return nil
	}
	return benchmarks
}
