package ladingdoc

import "strings"

const EventKindException = "Exception"

var statusKinds = map[string]string{
	"AF": "DepartedPickup",
	"X3": "ArrivedPickup",
	"X1": "ArrivedDelivery",
	"D1": "Delivered",
	"CD": "DepartedDelivery",
	"X6": "EnRoute",
	"AG": "EtaChanged",
	"SD": "Delayed",
	"AA": "AppointmentSet",
	"A9": EventKindException,
	"CP": "Loaded",
	"P1": "Dispatched",
}

var kindStatuses = func() map[string]string {
	out := make(map[string]string, len(statusKinds))
	for code, kind := range statusKinds {
		out[kind] = code
	}
	return out
}()

// EventKindOf is the event kind of an X12 shipment status code (AT701).
func EventKindOf(code string) string {
	if kind, ok := statusKinds[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return kind
	}
	return EventKindException
}

// StatusCodeOf is the X12 shipment status code of an event kind.
func StatusCodeOf(kind string) string {
	return kindStatuses[kind]
}
