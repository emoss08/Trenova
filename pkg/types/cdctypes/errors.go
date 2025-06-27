package cdctypes

import "github.com/rotisserie/eris"

var (
	// * CDC consumer errors
	ErrConsumerAlreadyRunning = eris.New("kafka consumer already running")
	ErrInvalidTimestampFormat = eris.New("invalid timestamp format in Avro data")

	// * Tenant isolation errors
	ErrOrganizationIDMissing = eris.New("missing organization id in data")
	ErrBusinessUnitIDMissing = eris.New("missing business unit id in data")
)
