package carrierintelcontrol

type EnrollmentPolicy string

const (
	EnrollmentPolicyAllActive    = EnrollmentPolicy("AllActive")
	EnrollmentPolicyRecentlyUsed = EnrollmentPolicy("RecentlyUsed")
	EnrollmentPolicyManual       = EnrollmentPolicy("Manual")
)

func (p EnrollmentPolicy) String() string { return string(p) }

func (p EnrollmentPolicy) IsValid() bool {
	switch p {
	case EnrollmentPolicyAllActive, EnrollmentPolicyRecentlyUsed, EnrollmentPolicyManual:
		return true
	default:
		return false
	}
}

type OutagePolicy string

const (
	OutagePolicyFailOpen   = OutagePolicy("FailOpen")
	OutagePolicyFailClosed = OutagePolicy("FailClosed")
)

func (p OutagePolicy) String() string { return string(p) }

func (p OutagePolicy) IsValid() bool {
	return p == OutagePolicyFailOpen || p == OutagePolicyFailClosed
}
