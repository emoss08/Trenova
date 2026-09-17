package fmcsaconnector

import (
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/infrastructure/carrierintel/intelkit"
	"github.com/emoss08/trenova/shared/fmcsa"
	"github.com/emoss08/trenova/shared/jsonflex"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/shopspring/decimal"
)

const (
	preferredDocketPrefix = "MC"
	rescindedMarker       = "RESCIND"
)

type carrierExtras struct {
	basics      []fmcsa.Basic
	cargo       []string
	operations  []string
	oos         []fmcsa.OOSEntry
	dockets     []fmcsa.Docket
	authorities []fmcsa.Authority
	complete    bool
}

func normalizeComposite(composite *fmcsa.CompositeCarrier) *carrierintel.Profile {
	if composite == nil {
		return &carrierintel.Profile{Coverage: []carrierintel.Section{}}
	}
	return normalize(&composite.Carrier, &carrierExtras{
		basics:      composite.Basics,
		cargo:       composite.Cargo,
		operations:  composite.Operations,
		oos:         composite.OOS,
		dockets:     composite.Dockets,
		authorities: composite.Authorities,
		complete:    true,
	})
}

func normalizeCarrier(carrier *fmcsa.Carrier) *carrierintel.Profile {
	if carrier == nil {
		return &carrierintel.Profile{Coverage: []carrierintel.Section{}}
	}
	return normalize(carrier, &carrierExtras{})
}

func normalize(carrier *fmcsa.Carrier, extras *carrierExtras) *carrierintel.Profile {
	authority := preferredAuthority(extras.authorities)
	profile := &carrierintel.Profile{
		Identity:    normalizeIdentity(carrier, extras, authority),
		Authority:   normalizeAuthority(carrier, authority),
		Insurance:   normalizeInsurance(carrier),
		Safety:      normalizeSafety(carrier, extras),
		Basics:      normalizeBasics(extras.basics),
		Inspections: normalizeInspections(carrier),
		Crashes:     normalizeCrashes(carrier),
		Fleet:       normalizeFleet(carrier),
		Operations:  normalizeOperations(extras),
	}
	profile.NormalizeCoverage()
	return profile
}

func normalizeIdentity(
	carrier *fmcsa.Carrier,
	extras *carrierExtras,
	authority *fmcsa.Authority,
) *carrierintel.Identity {
	prefix, number := docketFor(extras.dockets, authority)
	identity := &carrierintel.Identity{
		DOTNumber:        intelkit.Text(carrier.DOTNumber),
		DocketPrefix:     prefix,
		DocketNumber:     number,
		LegalName:        intelkit.Text(carrier.LegalName),
		DBAName:          intelkit.Text(carrier.DBAName),
		EIN:              intelkit.Text(carrier.EIN),
		USDOTStatus:      usdotStatus(carrier),
		CarrierOperation: intelkit.Text(carrier.CarrierOperationDesc),
		PhysicalAddress: intelkit.Address(intelkit.AddressParts{
			Street:     intelkit.Text(carrier.PhysicalStreet),
			City:       intelkit.Text(carrier.PhysicalCity),
			State:      intelkit.Text(carrier.PhysicalState),
			PostalCode: intelkit.Text(carrier.PhysicalZipcode),
			Country:    intelkit.Text(carrier.PhysicalCountry),
		}),
	}
	if *identity == (carrierintel.Identity{}) {
		return nil
	}
	return identity
}

func usdotStatus(carrier *fmcsa.Carrier) string {
	if status := intelkit.USDOTStatus(intelkit.Text(carrier.StatusCode)); status != "" {
		return status
	}
	if carrier.AllowedToOperate == nil {
		return ""
	}
	if carrier.AllowedToOperate.Value() {
		return "ACTIVE"
	}
	return "INACTIVE"
}

func docketFor(dockets []fmcsa.Docket, authority *fmcsa.Authority) (prefix, number string) {
	var fallbackPrefix, fallbackNumber string
	for idx := range dockets {
		docket := &dockets[idx]
		candidatePrefix, candidateNumber := docketParts(docket.Prefix, docket.DocketNumber)
		if candidateNumber == "" {
			continue
		}
		if candidatePrefix == preferredDocketPrefix {
			return candidatePrefix, candidateNumber
		}
		if fallbackNumber == "" {
			fallbackPrefix, fallbackNumber = candidatePrefix, candidateNumber
		}
	}
	if fallbackNumber != "" {
		return fallbackPrefix, fallbackNumber
	}
	if authority != nil {
		return docketParts(authority.Prefix, authority.DocketNumber)
	}
	return "", ""
}

func docketParts(prefix, number *jsonflex.String) (normalizedPrefix, normalizedNumber string) {
	splitPrefix, splitNumber := intelkit.SplitDocket(intelkit.Text(number))
	normalizedPrefix = strings.ToUpper(intelkit.Text(prefix))
	if normalizedPrefix == "" {
		normalizedPrefix = splitPrefix
	}
	if splitNumber == "" {
		return "", ""
	}
	return normalizedPrefix, splitNumber
}

func preferredAuthority(authorities []fmcsa.Authority) *fmcsa.Authority {
	if len(authorities) == 0 {
		return nil
	}
	for idx := range authorities {
		if strings.EqualFold(intelkit.Text(authorities[idx].Prefix), preferredDocketPrefix) {
			return &authorities[idx]
		}
	}
	return &authorities[0]
}

func normalizeAuthority(
	carrier *fmcsa.Carrier,
	authority *fmcsa.Authority,
) *carrierintel.Authority {
	common := carrier.CommonAuthorityStatus
	contract := carrier.ContractAuthorityStatus
	broker := carrier.BrokerAuthorityStatus
	var brokerAuthorized *jsonflex.Bool
	if authority != nil {
		common = firstString(common, authority.CommonAuthorityStatus)
		contract = firstString(contract, authority.ContractAuthorityStatus)
		broker = firstString(broker, authority.BrokerAuthorityStatus)
		brokerAuthorized = authority.AuthorizedForBroker
	}

	result := &carrierintel.Authority{
		Common:   grant(common),
		Contract: grant(contract),
		Broker:   grant(broker),
	}
	if result.Broker == nil && brokerAuthorized != nil {
		status := carrierintel.AuthorityStatusNone
		if brokerAuthorized.Value() {
			status = carrierintel.AuthorityStatusActive
		}
		result.Broker = &carrierintel.AuthorityGrant{Status: status}
	}
	if result.Common == nil && result.Contract == nil && result.Broker == nil {
		return nil
	}
	return result
}

func firstString(values ...*jsonflex.String) *jsonflex.String {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func grant(status *jsonflex.String) *carrierintel.AuthorityGrant {
	if status == nil {
		return nil
	}
	return &carrierintel.AuthorityGrant{Status: intelkit.AuthorityStatus(intelkit.Text(status))}
}

func normalizeInsurance(carrier *fmcsa.Carrier) *carrierintel.Insurance {
	insurance := &carrierintel.Insurance{
		BIPDOnFile: fmcsa.InsuranceDollars(carrier.BIPDInsuranceOnFile),
		BIPDRequired: requiredAmount(
			fmcsa.InsuranceDollars(carrier.BIPDRequiredAmount),
			carrier.BIPDInsuranceRequired,
		),
		CargoOnFile:   fmcsa.InsuranceDollars(carrier.CargoInsuranceOnFile),
		CargoRequired: requiredAmount(nil, carrier.CargoInsuranceRequired),
		BondOnFile:    fmcsa.InsuranceDollars(carrier.BondInsuranceOnFile),
		BondRequired:  requiredAmount(nil, carrier.BondInsuranceRequired),
	}
	if insurance.BIPDOnFile == nil && insurance.BIPDRequired == nil &&
		insurance.CargoOnFile == nil && insurance.CargoRequired == nil &&
		insurance.BondOnFile == nil && insurance.BondRequired == nil {
		return nil
	}
	return insurance
}

func requiredAmount(amount *decimal.Decimal, required *jsonflex.Bool) *decimal.Decimal {
	if amount != nil {
		return amount
	}
	if required != nil && !required.Value() {
		zero := decimal.Zero
		return &zero
	}
	return nil
}

func normalizeSafety(carrier *fmcsa.Carrier, extras *carrierExtras) *carrierintel.Safety {
	safety := &carrierintel.Safety{
		RatingDate:        intelkit.Unix(carrier.SafetyRatingDate),
		ISSValue:          intelkit.IntFromFloat(carrier.ISSScore),
		OutOfServiceAt:    intelkit.Unix(carrier.OOSDate),
		OutOfServiceOrder: outOfServiceOrder(carrier, extras),
		LatestReviewType: stringutils.FirstNonEmpty(
			intelkit.Text(carrier.ReviewType),
			intelkit.Text(carrier.SafetyReviewType),
		),
		LatestReviewAt: firstUnix(carrier.ReviewDate, carrier.SafetyReviewDate),
	}
	if carrier.SafetyRating != nil {
		safety.Rating = intelkit.SafetyRating(intelkit.Text(carrier.SafetyRating))
	}
	if *safety == (carrierintel.Safety{}) {
		return nil
	}
	return safety
}

func firstUnix(values ...*jsonflex.Time) *int64 {
	for _, value := range values {
		if unix := intelkit.Unix(value); unix != nil {
			return unix
		}
	}
	return nil
}

func outOfServiceOrder(carrier *fmcsa.Carrier, extras *carrierExtras) *bool {
	if !extras.complete {
		return nil
	}
	ordered := hasActiveOOSEntry(extras.oos) && carrier.AllowedToOperate != nil &&
		!carrier.AllowedToOperate.Value()
	return &ordered
}

func hasActiveOOSEntry(entries []fmcsa.OOSEntry) bool {
	for idx := range entries {
		status := strings.ToUpper(intelkit.Text(entries[idx].Status))
		if !strings.Contains(status, rescindedMarker) {
			return true
		}
	}
	return false
}

func normalizeBasics(src []fmcsa.Basic) []carrierintel.BasicMeasure {
	if len(src) == 0 {
		return nil
	}
	basics := make([]carrierintel.BasicMeasure, 0, len(src))
	seen := make(map[worker.CSABasic]struct{}, len(src))
	for idx := range src {
		item := &src[idx]
		basic, ok := csaBasic(item)
		if !ok {
			continue
		}
		if _, duplicate := seen[basic]; duplicate {
			continue
		}
		seen[basic] = struct{}{}
		basics = append(basics, carrierintel.BasicMeasure{
			Basic:         basic,
			Measure:       intelkit.NonNegativeFloat(item.MeasureValue),
			Percentile:    intelkit.NonNegativeFloat(item.Percentile),
			Threshold:     intelkit.NonNegativeFloat(item.ViolationThreshold),
			Alert:         item.ExceededInterventionThreshold.Value(),
			RoadsideAlert: item.OnRoadPerformanceThresholdViolation.Value(),
			ACIndicator:   item.SeriousViolationFromInvestigation12M.Value(),
		})
	}
	if len(basics) == 0 {
		return nil
	}
	return basics
}

func csaBasic(item *fmcsa.Basic) (worker.CSABasic, bool) {
	for _, label := range []string{
		intelkit.Text(item.ShortDescription),
		intelkit.Text(item.CodeMCMIS),
		intelkit.Text(item.Code),
	} {
		if basic, ok := csaBasicFromLabel(label); ok {
			return basic, true
		}
	}
	return "", false
}

func csaBasicFromLabel(label string) (worker.CSABasic, bool) {
	key := stringutils.NormalizeIdentifier(label)
	switch {
	case key == "":
		return "", false
	case strings.Contains(key, "UNSAFE"):
		return worker.BasicUnsafeDriving, true
	case strings.HasPrefix(key, "HOS"), strings.Contains(key, "HOURSOFSERVICE"):
		return worker.BasicHOSCompliance, true
	case strings.Contains(key, "FITNESS"):
		return worker.BasicDriverFitness, true
	case strings.Contains(key, "DRUG"), strings.Contains(key, "SUBST"),
		strings.Contains(key, "ALCOHOL"):
		return worker.BasicControlledSubstances, true
	case strings.Contains(key, "MAINT"):
		return worker.BasicVehicleMaintenance, true
	case strings.HasPrefix(key, "HM"), strings.Contains(key, "HAZ"):
		return worker.BasicHazmatCompliance, true
	case strings.Contains(key, "CRASH"):
		return worker.BasicCrashIndicator, true
	default:
		return "", false
	}
}

func normalizeInspections(carrier *fmcsa.Carrier) *carrierintel.Inspections {
	inspections := &carrierintel.Inspections{
		Driver:                intelkit.Int(carrier.DriverInspections),
		Vehicle:               intelkit.Int(carrier.VehicleInspections),
		Hazmat:                intelkit.Int(carrier.HazmatInspections),
		DriverOOS:             intelkit.Int(carrier.DriverOOSInspections),
		VehicleOOS:            intelkit.Int(carrier.VehicleOOSInspections),
		HazmatOOS:             intelkit.Int(carrier.HazmatOOSInspections),
		DriverOOSRate:         intelkit.NonNegativeFloat(carrier.DriverOOSRate),
		VehicleOOSRate:        intelkit.NonNegativeFloat(carrier.VehicleOOSRate),
		HazmatOOSRate:         intelkit.NonNegativeFloat(carrier.HazmatOOSRate),
		NationalDriverOOSRate: intelkit.NonNegativeFloat(carrier.DriverOOSRateNationalAverage),
		NationalVehicleOOS:    intelkit.NonNegativeFloat(carrier.VehicleOOSRateNationalAverage),
		NationalHazmatOOSRate: intelkit.NonNegativeFloat(carrier.HazmatOOSRateNationalAverage),
	}
	if *inspections == (carrierintel.Inspections{}) {
		return nil
	}
	return inspections
}

func normalizeCrashes(carrier *fmcsa.Carrier) *carrierintel.Crashes {
	crashes := &carrierintel.Crashes{
		Total:  intelkit.Int(carrier.CrashTotal),
		Fatal:  intelkit.Int(carrier.FatalCrash),
		Injury: intelkit.Int(carrier.InjuryCrash),
		Tow:    intelkit.Int(carrier.TowawayCrash),
	}
	if *crashes == (carrierintel.Crashes{}) {
		return nil
	}
	return crashes
}

func normalizeFleet(carrier *fmcsa.Carrier) *carrierintel.Fleet {
	fleet := &carrierintel.Fleet{
		PowerUnits: intelkit.Int(carrier.TotalPowerUnits),
		Drivers:    intelkit.Int(carrier.TotalDrivers),
	}
	if *fleet == (carrierintel.Fleet{}) {
		return nil
	}
	return fleet
}

func normalizeOperations(extras *carrierExtras) *carrierintel.Operations {
	operations := &carrierintel.Operations{
		Classification: sliceutils.DedupeStrings(extras.operations),
		CargoCarried:   sliceutils.DedupeStrings(extras.cargo),
	}
	if len(operations.Classification) == 0 && len(operations.CargoCarried) == 0 {
		return nil
	}
	return operations
}
