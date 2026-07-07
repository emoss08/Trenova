package editransport

import "errors"

var (
	ErrVanMailboxIDRequired            = errors.New("VAN mailbox ID is required for EDI delivery")
	ErrEDICommunicationProfileRequired = errors.New(
		"EDI communication profile is required for delivery",
	)
)
