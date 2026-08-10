package ediinboundservice

// X12 element 558 (Reservation Action Code) values carried in B1-04 of a 990.
const (
	reservationCodeAccepted           = "A"
	reservationCodeDeclined           = "D"
	reservationCodeNotAccepted        = "N"
	reservationCodeDeclinedRateChange = "R"
)

func isReservationDecline(code string) bool {
	switch code {
	case reservationCodeDeclined, reservationCodeNotAccepted, reservationCodeDeclinedRateChange:
		return true
	default:
		return false
	}
}
