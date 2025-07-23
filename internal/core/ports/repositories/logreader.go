// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package repositories

import (
	"encoding/json" //nolint:depguard // This is a dependency for the log reader
	"time"

	"github.com/emoss08/trenova/internal/core/ports"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Level      string          `json:"level"`
	App        string          `json:"app"`
	Version    any             `json:"version"`
	Hostname   string          `json:"hostname"`
	Message    json.RawMessage `json:"message"`
	Caller     string          `json:"caller"`
	Time       time.Time       `json:"time"`
	GoVersion  string          `json:"goVersion,omitempty"`
	GoRoutines int             `json:"goRoutines,omitempty"`
	CPU        int             `json:"cpu,omitempty"`
	Method     string          `json:"method,omitempty"`
	Path       string          `json:"path,omitempty"`

	// Common service fields
	Service   string `json:"service,omitempty"`
	Component string `json:"component,omitempty"`
	Client    string `json:"client,omitempty"`

	// Search service specific fields
	BatchSize     int    `json:"batchSize,omitempty"`
	BatchInterval int    `json:"batchInterval,omitempty"`
	IndexPrefix   string `json:"indexPrefix,omitempty"`
	IndexName     string `json:"idxName,omitempty"`

	// Server specific fields
	ListenAddress string `json:"listenAddress,omitempty"`

	// Additional fields for flexibility
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ListLogOptions contains options for listing logs
type ListLogOptions struct {
	*ports.LimitOffsetQueryOptions
	StartDate  time.Time `json:"startDate"`
	EndDate    time.Time `json:"endDate"`
	LogLevel   []string  `json:"logLevel"`   // Filter by log levels (debug, info, etc.)
	Service    []string  `json:"service"`    // Filter by service names
	Component  []string  `json:"component"`  // Filter by component names
	SearchText string    `json:"searchText"` // Full-text search in message and metadata
}

// LogFileInfo represents information about a log file
type LogFileInfo struct {
	Name         string    `json:"name"`
	Size         string    `json:"size"`
	ModTime      time.Time `json:"modTime"`
	IsCompressed bool      `json:"isCompressed"`
}
