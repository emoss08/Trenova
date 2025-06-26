package services

import (
	"context"
)

// CDCService defines the interface for Change Data Capture services
type CDCService interface {
	// Start begins consuming CDC events
	Start(ctx context.Context) error

	// Stop gracefully stops consuming CDC events
	Stop() error

	// IsRunning returns whether the CDC service is currently running
	IsRunning() bool

	// GetMetrics returns CDC service metrics
	GetMetrics() map[string]interface{}

	// RegisterHandler registers a handler for a specific table
	RegisterHandler(table string, handler CDCEventHandler)
}

// CDCEventHandler defines the interface for handling CDC events
type CDCEventHandler interface {
	// HandleEvent processes a CDC event for a specific table
	HandleEvent(event CDCEvent) error

	// GetTableName returns the table name this handler is responsible for
	GetTableName() string
}

// CDCEvent represents a change data capture event
type CDCEvent struct {
	// Operation type: create, update, delete, read
	Operation string `json:"operation"`

	// Table that was changed
	Table string `json:"table"`

	// Schema of the table
	Schema string `json:"schema"`

	// Before state (for updates and deletes)
	Before map[string]interface{} `json:"before,omitempty"`

	// After state (for creates and updates)
	After map[string]interface{} `json:"after,omitempty"`

	// Metadata about the change
	Metadata CDCMetadata `json:"metadata"`
}

// CDCMetadata contains metadata about the CDC event
type CDCMetadata struct {
	// Timestamp when the change occurred
	Timestamp int64 `json:"timestamp"`

	// Source database information
	Source CDCSource `json:"source"`

	// Transaction ID
	TransactionID string `json:"transactionId,omitempty"`

	// LSN (Log Sequence Number) for PostgreSQL
	LSN int64 `json:"lsn,omitempty"`
}

// CDCSource contains information about the source database
type CDCSource struct {
	// Database name
	Database string `json:"database"`

	// Schema name
	Schema string `json:"schema"`

	// Table name
	Table string `json:"table"`

	// Connector name
	Connector string `json:"connector"`

	// Version of the connector
	Version string `json:"version"`
}