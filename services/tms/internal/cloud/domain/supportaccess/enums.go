package supportaccess

type AccessMode string

const (
	AccessModeReadOnly  = AccessMode("read_only")
	AccessModeReadWrite = AccessMode("read_write")
)

func AllAccessModes() []AccessMode {
	return []AccessMode{AccessModeReadOnly, AccessModeReadWrite}
}

func (m AccessMode) String() string {
	return string(m)
}

func (m AccessMode) IsValid() bool {
	switch m {
	case AccessModeReadOnly, AccessModeReadWrite:
		return true
	default:
		return false
	}
}

func (m AccessMode) AllowsWrite() bool {
	return m == AccessModeReadWrite
}

type StaffRole string

const (
	StaffRoleSupport  = StaffRole("support")
	StaffRoleEngineer = StaffRole("engineer")
)

func AllStaffRoles() []StaffRole {
	return []StaffRole{StaffRoleSupport, StaffRoleEngineer}
}

func (r StaffRole) String() string {
	return string(r)
}

func (r StaffRole) IsValid() bool {
	switch r {
	case StaffRoleSupport, StaffRoleEngineer:
		return true
	default:
		return false
	}
}

type EndReason string

const (
	EndReasonExited        = EndReason("exited")
	EndReasonExpired       = EndReason("expired")
	EndReasonGrantRevoked  = EndReason("grant_revoked")
	EndReasonGrantExpired  = EndReason("grant_expired")
	EndReasonStaffRemoved  = EndReason("staff_removed")
	EndReasonSignedOut     = EndReason("signed_out")
	EndReasonReplaced      = EndReason("replaced")
	EndReasonAssuranceLost = EndReason("assurance_lost")
)

func AllEndReasons() []EndReason {
	return []EndReason{
		EndReasonExited,
		EndReasonExpired,
		EndReasonGrantRevoked,
		EndReasonGrantExpired,
		EndReasonStaffRemoved,
		EndReasonSignedOut,
		EndReasonReplaced,
		EndReasonAssuranceLost,
	}
}

func (r EndReason) String() string {
	return string(r)
}

func (r EndReason) IsValid() bool {
	switch r {
	case EndReasonExited,
		EndReasonExpired,
		EndReasonGrantRevoked,
		EndReasonGrantExpired,
		EndReasonStaffRemoved,
		EndReasonSignedOut,
		EndReasonReplaced,
		EndReasonAssuranceLost:
		return true
	default:
		return false
	}
}
