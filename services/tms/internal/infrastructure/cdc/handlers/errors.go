package handlers

import "errors"

var (
	ErrCreateEventMissingAfterData  = errors.New("create event missing 'after' data")
	ErrUpdateEventMissingAfterData  = errors.New("update event missing 'after' data")
	ErrDeleteEventMissingBeforeData = errors.New("delete event missing 'before' data")
	ErrNilDataProvided              = errors.New("nil data provided for shipment conversion")
	ErrDeleteEventMissingShipmentID = errors.New("delete event missing shipment ID")
	ErrShipmentIDMissing            = errors.New("shipment ID is missing or invalid")
)
