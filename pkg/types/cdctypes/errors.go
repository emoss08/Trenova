// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

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
