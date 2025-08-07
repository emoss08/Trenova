/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package services

import "github.com/emoss08/trenova/shared/cdctypes"

// CDCService defines the interface for Change Data Capture services
type CDCService interface {
	// Start begins consuming CDC events
	Start() error

	// Stop gracefully stops consuming CDC events
	Stop() error

	// IsRunning returns whether the CDC service is currently running
	IsRunning() bool

	// RegisterHandler registers a handler for a specific table
	RegisterHandler(table string, handler CDCEventHandler)
}

// CDCEventHandler defines the interface for handling CDC events
type CDCEventHandler interface {
	// HandleEvent processes a CDC event for a specific table
	HandleEvent(event *cdctypes.CDCEvent) error

	// GetTableName returns the table name this handler is responsible for
	GetTableName() string
}
