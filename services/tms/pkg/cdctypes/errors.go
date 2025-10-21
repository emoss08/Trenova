package cdctypes

import (
	"errors"
)

var (
	ErrConsumerAlreadyRunning = errors.New("kafka consumer already running")
	ErrInvalidTimestampFormat = errors.New("invalid timestamp format in Avro data")

	ErrOrganizationIDMissing = errors.New("missing organization id in data")
	ErrBusinessUnitIDMissing = errors.New("missing business unit id in data")
)
