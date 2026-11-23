package workercompliance

import "github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"

type DispatchRequirement string

const (
	DispatchRequirementLicense            = DispatchRequirement("License")
	DispatchRequirementMedicalCard        = DispatchRequirement("MedicalCard")
	DispatchRequirementPhysical           = DispatchRequirement("Physical")
	DispatchRequirementMVR                = DispatchRequirement("MVR")
	DispatchRequirementHazmat             = DispatchRequirement("Hazmat")
	DispatchRequirementClearinghouseQuery = DispatchRequirement("ClearinghouseQuery")
	DispatchRequirementDrugAlcohol        = DispatchRequirement("DrugAlcohol")
	DispatchRequirementOther              = DispatchRequirement("Other")
)

func (r DispatchRequirement) String() string { return string(r) }

func DispatchRequirementForProfileField(
	field workercredential.CredentialProfileField,
) (DispatchRequirement, bool) {
	switch field {
	case workercredential.CredentialProfileFieldLicenseExpiry:
		return DispatchRequirementLicense, true
	case workercredential.CredentialProfileFieldMedicalCardExpiry:
		return DispatchRequirementMedicalCard, true
	case workercredential.CredentialProfileFieldPhysicalDueDate:
		return DispatchRequirementPhysical, true
	case workercredential.CredentialProfileFieldMVRDueDate:
		return DispatchRequirementMVR, true
	case workercredential.CredentialProfileFieldHazmatExpiry:
		return DispatchRequirementHazmat, true
	case workercredential.CredentialProfileFieldNone,
		workercredential.CredentialProfileFieldTWICExpiry:
		return "", false
	default:
		return "", false
	}
}

type DispatchProbe struct {
	Requirement DispatchRequirement
	At          int64
}

type DispatchStop struct {
	Requirement DispatchRequirement
	At          int64
	HazmatOnly  bool
	Regulation  string
}

type DispatchVerdicts struct {
	RefusedToday []DispatchRequirement
	Stops        []DispatchStop
	Withheld     bool
}

func (v *DispatchVerdicts) Without(requirements ...DispatchRequirement) *DispatchVerdicts {
	if v == nil {
		return nil
	}
	hidden := func(requirement DispatchRequirement) bool {
		for _, candidate := range requirements {
			if candidate == requirement {
				return true
			}
		}
		return false
	}
	out := &DispatchVerdicts{
		RefusedToday: make([]DispatchRequirement, 0, len(v.RefusedToday)),
		Stops:        make([]DispatchStop, 0, len(v.Stops)),
		Withheld:     v.Withheld,
	}
	for _, refused := range v.RefusedToday {
		if hidden(refused) {
			out.Withheld = true
			continue
		}
		out.RefusedToday = append(out.RefusedToday, refused)
	}
	for _, stop := range v.Stops {
		if !hidden(stop.Requirement) {
			out.Stops = append(out.Stops, stop)
		}
	}
	return out
}

func (v *DispatchVerdicts) RefusedOutOfSight() bool {
	return v != nil && v.Withheld && len(v.RefusedToday) == 0
}

func (v *DispatchVerdicts) Refused() bool {
	return v != nil && len(v.RefusedToday) > 0
}

func (v *DispatchVerdicts) RefusesTodayOn(requirement DispatchRequirement) bool {
	if v == nil {
		return false
	}
	for _, refused := range v.RefusedToday {
		if refused == requirement {
			return true
		}
	}
	return false
}

func (v *DispatchVerdicts) StopAt(probe DispatchProbe) (DispatchStop, bool) {
	if v == nil {
		return DispatchStop{}, false
	}
	for _, stop := range v.Stops {
		if stop.Requirement == probe.Requirement && stop.At == probe.At {
			return stop, true
		}
	}
	return DispatchStop{}, false
}

func DispatchProbes(in ClearanceInput) []DispatchProbe {
	probes := make([]DispatchProbe, 0, 8)
	seen := make(map[DispatchProbe]struct{}, 8)
	add := func(probe DispatchProbe) {
		if _, ok := seen[probe]; ok {
			return
		}
		seen[probe] = struct{}{}
		probes = append(probes, probe)
	}
	if in.Credentials != nil {
		for _, item := range in.Credentials.Items {
			if probe, ok := credentialProbe(item); ok {
				add(probe)
			}
		}
	}
	if due := clearinghouseDue(in.DrugAlcohol); due != nil {
		add(DispatchProbe{Requirement: DispatchRequirementClearinghouseQuery, At: *due})
	}
	return probes
}

func credentialProbe(item *workercredential.CredentialSummaryItem) (DispatchProbe, bool) {
	if item == nil || item.CredentialType == nil || item.Credential == nil ||
		item.Credential.ExpiresAt == nil || *item.Credential.ExpiresAt <= 0 {
		return DispatchProbe{}, false
	}
	requirement, ok := DispatchRequirementForProfileField(item.CredentialType.ProfileField)
	if !ok {
		return DispatchProbe{}, false
	}
	return DispatchProbe{Requirement: requirement, At: *item.Credential.ExpiresAt}, true
}
