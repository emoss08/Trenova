package carrierintel

type ProfileMergeInput struct {
	Current       *CarrierIntelSnapshot
	Incoming      *Profile
	IncomingDepth LookupDepth
	Control       *CarrierIntelControl
	Now           int64
}

type ProfileMergeResult struct {
	Profile        *Profile
	Depth          LookupDepth
	DepthFetchedAt int64
	Merged         bool
}

func ResolveIncomingProfile(in *ProfileMergeInput) ProfileMergeResult {
	in.Incoming.NormalizeCoverage()
	replaced := ProfileMergeResult{
		Profile:        in.Incoming,
		Depth:          in.IncomingDepth,
		DepthFetchedAt: in.Now,
	}

	current := in.Current
	if current == nil || current.NotFound || current.Profile == nil ||
		in.IncomingDepth.Satisfies(current.Depth) ||
		!current.DepthIsFresh(in.Now, in.Control.TTLSecondsForDepth(current.Depth)) {
		return replaced
	}

	return ProfileMergeResult{
		Profile:        MergeProfiles(current.Profile, in.Incoming),
		Depth:          current.Depth,
		DepthFetchedAt: current.DepthAsOf(),
		Merged:         true,
	}
}

func MergeProfiles(base, overlay *Profile) *Profile {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}

	merged := &Profile{
		Identity:      mergeStruct(base.Identity, overlay.Identity, mergeIdentity),
		Authority:     mergeStruct(base.Authority, overlay.Authority, mergeAuthority),
		Insurance:     mergeStruct(base.Insurance, overlay.Insurance, mergeInsurance),
		Safety:        mergeStruct(base.Safety, overlay.Safety, mergeSafety),
		Basics:        mergeSlice(base.Basics, overlay.Basics),
		Inspections:   mergeStruct(base.Inspections, overlay.Inspections, mergeInspections),
		Crashes:       mergeStruct(base.Crashes, overlay.Crashes, mergeCrashes),
		Fleet:         mergeStruct(base.Fleet, overlay.Fleet, mergeFleet),
		Equipment:     mergeSlice(base.Equipment, overlay.Equipment),
		Contacts:      mergeStruct(base.Contacts, overlay.Contacts, mergeContacts),
		Operations:    mergeStruct(base.Operations, overlay.Operations, mergeOperations),
		ChangeHistory: mergeStruct(base.ChangeHistory, overlay.ChangeHistory, mergeChangeHistory),
		Network:       mergeStruct(base.Network, overlay.Network, mergeNetwork),
		Lanes:         mergeStruct(base.Lanes, overlay.Lanes, mergeLanes),
		Benchmarks:    mergeStruct(base.Benchmarks, overlay.Benchmarks, mergeBenchmarks),
	}

	sections := AllSections()
	merged.Coverage = make([]Section, 0, len(sections))
	for _, section := range sections {
		if base.Covers(section) || overlay.Covers(section) {
			merged.Coverage = append(merged.Coverage, section)
		}
	}
	merged.NormalizeCoverage()
	return merged
}

func mergeStruct[T any](base, overlay *T, merge func(dst, overlay *T)) *T {
	if overlay == nil {
		return base
	}
	if base == nil {
		return overlay
	}
	out := *base
	merge(&out, overlay)
	return &out
}

func mergePtr[T any](dst **T, overlay *T) {
	if overlay != nil {
		*dst = overlay
	}
}

func mergeText[S ~string](dst *S, overlay S) {
	if overlay != "" {
		*dst = overlay
	}
}

func mergeSlice[S ~[]E, E any](base, overlay S) S {
	if len(overlay) > 0 {
		return overlay
	}
	return base
}

func mergeIdentity(dst, overlay *Identity) {
	mergeText(&dst.DOTNumber, overlay.DOTNumber)
	if overlay.DocketNumber != "" {
		dst.DocketNumber = overlay.DocketNumber
		dst.DocketPrefix = overlay.DocketPrefix
	}
	mergeText(&dst.LegalName, overlay.LegalName)
	mergeText(&dst.DBAName, overlay.DBAName)
	mergeText(&dst.EIN, overlay.EIN)
	mergeText(&dst.USDOTStatus, overlay.USDOTStatus)
	mergeText(&dst.EntityType, overlay.EntityType)
	mergeText(&dst.CarrierOperation, overlay.CarrierOperation)
	mergePtr(&dst.DOTAddedAt, overlay.DOTAddedAt)
	mergePtr(&dst.DOTAgeDays, overlay.DOTAgeDays)
	if !overlay.PhysicalAddress.IsZero() {
		dst.PhysicalAddress = overlay.PhysicalAddress
	}
	if !overlay.MailingAddress.IsZero() {
		dst.MailingAddress = overlay.MailingAddress
	}
}

func mergeAuthority(dst, overlay *Authority) {
	dst.Common = mergeStruct(dst.Common, overlay.Common, mergeGrant)
	dst.Contract = mergeStruct(dst.Contract, overlay.Contract, mergeGrant)
	dst.Broker = mergeStruct(dst.Broker, overlay.Broker, mergeGrant)
	mergePtr(&dst.TotalRevocations, overlay.TotalRevocations)
	mergePtr(&dst.LastRevocationAt, overlay.LastRevocationAt)
	dst.History = mergeSlice(dst.History, overlay.History)
}

func mergeGrant(dst, overlay *AuthorityGrant) {
	if overlay.IsKnown() {
		dst.Status = overlay.Status
		dst.RevocationPending = overlay.RevocationPending
	}
	dst.Pending = overlay.Pending
	dst.UnderReview = overlay.UnderReview
	mergePtr(&dst.GrantedAt, overlay.GrantedAt)
	mergePtr(&dst.AgeDays, overlay.AgeDays)
}

func mergeInsurance(dst, overlay *Insurance) {
	mergePtr(&dst.BIPDOnFile, overlay.BIPDOnFile)
	mergePtr(&dst.BIPDRequired, overlay.BIPDRequired)
	mergePtr(&dst.CargoOnFile, overlay.CargoOnFile)
	mergePtr(&dst.CargoRequired, overlay.CargoRequired)
	mergePtr(&dst.BondOnFile, overlay.BondOnFile)
	mergePtr(&dst.BondRequired, overlay.BondRequired)
	mergePtr(&dst.PendingCancelAt, overlay.PendingCancelAt)
	mergePtr(&dst.LastCanceledAt, overlay.LastCanceledAt)
	mergePtr(&dst.CancelCount, overlay.CancelCount)
	dst.Filings = mergeSlice(dst.Filings, overlay.Filings)
}

func mergeSafety(dst, overlay *Safety) {
	mergeText(&dst.Rating, overlay.Rating)
	mergePtr(&dst.RatingDate, overlay.RatingDate)
	mergePtr(&dst.ISSValue, overlay.ISSValue)
	mergeText(&dst.ISSRecommendation, overlay.ISSRecommendation)
	if overlay.RiskScore != RiskLevelUnknown {
		mergeText(&dst.RiskScore, overlay.RiskScore)
	}
	mergePtr(&dst.RiskProbability, overlay.RiskProbability)
	mergePtr(&dst.SafetyScore, overlay.SafetyScore)
	mergePtr(&dst.OutOfServiceOrder, overlay.OutOfServiceOrder)
	mergePtr(&dst.OutOfServiceAt, overlay.OutOfServiceAt)
	mergeText(&dst.LatestReviewType, overlay.LatestReviewType)
	mergePtr(&dst.LatestReviewAt, overlay.LatestReviewAt)
}

func mergeInspections(dst, overlay *Inspections) {
	mergePtr(&dst.Total, overlay.Total)
	mergePtr(&dst.Driver, overlay.Driver)
	mergePtr(&dst.Vehicle, overlay.Vehicle)
	mergePtr(&dst.Hazmat, overlay.Hazmat)
	mergePtr(&dst.DriverOOS, overlay.DriverOOS)
	mergePtr(&dst.VehicleOOS, overlay.VehicleOOS)
	mergePtr(&dst.HazmatOOS, overlay.HazmatOOS)
	mergePtr(&dst.DriverOOSRate, overlay.DriverOOSRate)
	mergePtr(&dst.VehicleOOSRate, overlay.VehicleOOSRate)
	mergePtr(&dst.HazmatOOSRate, overlay.HazmatOOSRate)
	mergePtr(&dst.NationalDriverOOSRate, overlay.NationalDriverOOSRate)
	mergePtr(&dst.NationalVehicleOOS, overlay.NationalVehicleOOS)
	mergePtr(&dst.NationalHazmatOOSRate, overlay.NationalHazmatOOSRate)
	mergePtr(&dst.LastInspectionAt, overlay.LastInspectionAt)
}

func mergeCrashes(dst, overlay *Crashes) {
	mergePtr(&dst.Total, overlay.Total)
	mergePtr(&dst.Fatal, overlay.Fatal)
	mergePtr(&dst.Injury, overlay.Injury)
	mergePtr(&dst.Tow, overlay.Tow)
	mergePtr(&dst.LastCrashAt, overlay.LastCrashAt)
}

func mergeFleet(dst, overlay *Fleet) {
	mergePtr(&dst.PowerUnits, overlay.PowerUnits)
	mergePtr(&dst.Drivers, overlay.Drivers)
	mergePtr(&dst.CDLDrivers, overlay.CDLDrivers)
	mergePtr(&dst.OwnedTractors, overlay.OwnedTractors)
	mergePtr(&dst.TermLeasedTractors, overlay.TermLeasedTractors)
	mergePtr(&dst.OwnedTrailers, overlay.OwnedTrailers)
	mergePtr(&dst.TermLeasedTrailers, overlay.TermLeasedTrailers)
	mergePtr(&dst.Trailers, overlay.Trailers)
	mergePtr(&dst.Trucks, overlay.Trucks)
}

func mergeContacts(dst, overlay *Contacts) {
	mergeText(&dst.Phone, overlay.Phone)
	mergeText(&dst.Cellphone, overlay.Cellphone)
	mergeText(&dst.Fax, overlay.Fax)
	mergeText(&dst.Email, overlay.Email)
	mergeText(&dst.PrimaryContact, overlay.PrimaryContact)
	mergeText(&dst.SecondaryContact, overlay.SecondaryContact)
}

func mergeOperations(dst, overlay *Operations) {
	dst.Classification = mergeSlice(dst.Classification, overlay.Classification)
	dst.CargoCarried = mergeSlice(dst.CargoCarried, overlay.CargoCarried)
	mergePtr(&dst.HazmatCarrier, overlay.HazmatCarrier)
	mergePtr(&dst.MCS150At, overlay.MCS150At)
	mergePtr(&dst.MCS150Mileage, overlay.MCS150Mileage)
	mergePtr(&dst.BOC3OnFile, overlay.BOC3OnFile)
	mergeText(&dst.BOC3Agent, overlay.BOC3Agent)
	mergePtr(&dst.SmartWay, overlay.SmartWay)
	mergePtr(&dst.CARBCompliant, overlay.CARBCompliant)
	mergePtr(&dst.PHMSA, overlay.PHMSA)
}

func mergeChangeHistory(dst, overlay *ChangeHistory) {
	mergePtr(&dst.NameChanges, overlay.NameChanges)
	mergePtr(&dst.NameLastChangedAt, overlay.NameLastChangedAt)
	mergePtr(&dst.EmailChanges, overlay.EmailChanges)
	mergePtr(&dst.EmailLastChangedAt, overlay.EmailLastChangedAt)
	mergePtr(&dst.PhoneChanges, overlay.PhoneChanges)
	mergePtr(&dst.PhoneLastChangedAt, overlay.PhoneLastChangedAt)
	mergePtr(&dst.AddressChanges, overlay.AddressChanges)
	mergePtr(&dst.AddressLastChangeAt, overlay.AddressLastChangeAt)
	mergePtr(&dst.ContactChanges, overlay.ContactChanges)
	mergePtr(&dst.ContactLastChangeAt, overlay.ContactLastChangeAt)
}

func mergeNetwork(dst, overlay *Network) {
	mergePtr(&dst.SharedAddresses, overlay.SharedAddresses)
	mergePtr(&dst.SharedPhones, overlay.SharedPhones)
	mergePtr(&dst.SharedEmails, overlay.SharedEmails)
	mergePtr(&dst.SharedEINs, overlay.SharedEINs)
	mergePtr(&dst.SharedEquipment, overlay.SharedEquipment)
	dst.Links = mergeSlice(dst.Links, overlay.Links)
}

func mergeLanes(dst, overlay *Lanes) {
	mergePtr(&dst.TotalLoads, overlay.TotalLoads)
	mergePtr(&dst.FTLPercent, overlay.FTLPercent)
	mergePtr(&dst.LTLPercent, overlay.LTLPercent)
	mergePtr(&dst.DeadheadPercent, overlay.DeadheadPercent)
	mergePtr(&dst.FirstLoadAt, overlay.FirstLoadAt)
	mergePtr(&dst.LastLoadAt, overlay.LastLoadAt)
	dst.Preferred = mergeSlice(dst.Preferred, overlay.Preferred)
	dst.PreferredStates = mergeSlice(dst.PreferredStates, overlay.PreferredStates)
}

func mergeBenchmarks(dst, overlay *Benchmarks) {
	mergePtr(&dst.AnyAnomaly, overlay.AnyAnomaly)
	mergePtr(&dst.InspectionMileageAnomaly, overlay.InspectionMileageAnomaly)
	mergePtr(&dst.InspectedUnitsAnomaly, overlay.InspectedUnitsAnomaly)
	mergePtr(&dst.PowerUnitMileageAnomaly, overlay.PowerUnitMileageAnomaly)
}
