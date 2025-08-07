/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package cdctypes

import (
	"errors"
)

var (
	// * CDC consumer errors
	ErrConsumerAlreadyRunning = errors.New("kafka consumer already running")
	ErrInvalidTimestampFormat = errors.New("invalid timestamp format in Avro data")

	// * Tenant isolation errors
	ErrOrganizationIDMissing = errors.New("missing organization id in data")
	ErrBusinessUnitIDMissing = errors.New("missing business unit id in data")
)
