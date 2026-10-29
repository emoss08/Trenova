package capturepairing

// PairingStatus is where a device authorization grant is.
type PairingStatus string

const (
	PairingPending  = PairingStatus("Pending")
	PairingApproved = PairingStatus("Approved")
	PairingDenied   = PairingStatus("Denied")
	// PairingConsumed is an approved grant whose tokens were collected. A grant
	// is exchanged exactly once; a second poll with the same device code is
	// somebody replaying it.
	PairingConsumed = PairingStatus("Consumed")
	PairingExpired  = PairingStatus("Expired")
)

func (s PairingStatus) IsValid() bool {
	switch s {
	case PairingPending, PairingApproved, PairingDenied, PairingConsumed, PairingExpired:
		return true
	default:
		return false
	}
}

func (s PairingStatus) Terminal() bool {
	switch s {
	case PairingDenied, PairingConsumed, PairingExpired:
		return true
	case PairingPending, PairingApproved:
		return false
	default:
		return false
	}
}

func AllPairingStatuses() []PairingStatus {
	return []PairingStatus{
		PairingPending,
		PairingApproved,
		PairingDenied,
		PairingConsumed,
		PairingExpired,
	}
}
