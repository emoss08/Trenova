package worker

import "errors"

var ErrInvalidLeaveType = errors.New("invalid leave type")

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

func (l LeaveType) IsValid() bool {
	switch l {
	case LeaveTypeFMLA, LeaveTypeMedical, LeaveTypeMilitary, LeaveTypeParental,
		LeaveTypePersonal, LeaveTypeOther:
		return true
	default:
		return false
	}
}

func LeaveTypeFromString(s string) (LeaveType, error) {
	leaveType := LeaveType(s)
	if !leaveType.IsValid() {
		return "", ErrInvalidLeaveType
	}
	return leaveType, nil
}
