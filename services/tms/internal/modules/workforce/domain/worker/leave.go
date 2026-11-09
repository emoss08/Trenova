package worker

// LeaveType classifies a leave of absence. It is stamped on the worker while
// a LeaveStarted event is open so the board can tell FMLA from a suspension.
type LeaveType string

const (
	LeaveTypeFMLA     = LeaveType("FMLA")
	LeaveTypeMedical  = LeaveType("Medical")
	LeaveTypeMilitary = LeaveType("Military")
	LeaveTypeParental = LeaveType("Parental")
	LeaveTypePersonal = LeaveType("Personal")
	LeaveTypeOther    = LeaveType("Other")
)

func (l LeaveType) String() string { return string(l) }

func LeaveTypeValues() []LeaveType {
	return []LeaveType{
		LeaveTypeFMLA,
		LeaveTypeMedical,
		LeaveTypeMilitary,
		LeaveTypeParental,
		LeaveTypePersonal,
		LeaveTypeOther,
	}
}

func (l LeaveType) IsValid() bool {
	switch l {
	case LeaveTypeFMLA, LeaveTypeMedical, LeaveTypeMilitary, LeaveTypeParental,
		LeaveTypePersonal, LeaveTypeOther:
		return true
	default:
		return false
	}
}
