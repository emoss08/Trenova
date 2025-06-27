package cdctypes

import (
	"github.com/bytedance/sonic"
)

// DebeziumAvroEnvelope represents the structure of Avro-encoded Debezium change events
// as they come from Kafka with Confluent Schema Registry.
// This structure matches the actual wire format we receive.
type DebeziumAvroEnvelope struct {
	// Before state - can be nil for INSERT operations
	// When present, contains a map with "Value" key containing the actual data
	Before map[string]any `json:"before"`

	// After state - can be nil for DELETE operations
	// When present, contains a map with "Value" key containing the actual data
	After map[string]any `json:"after"`

	// Source metadata about the database change
	Source DebeziumAvroSource `json:"source"`

	// Operation type: r=read, c=create, u=update, d=delete
	Op string `json:"op"`

	// Timestamp when the connector processed the event
	TsMs DebeziumTimestamp `json:"ts_ms"`

	// Transaction information (optional)
	Transaction *DebeziumTransaction `json:"transaction,omitempty"`
}

// DebeziumAvroSource contains source metadata in Avro format
type DebeziumAvroSource struct {
	Version   string           `json:"version"`   // Debezium version
	Connector string           `json:"connector"` // Source connector type
	Name      string           `json:"name"`      // Connector name
	TsMs      int64            `json:"ts_ms"`     // Source timestamp
	Snapshot  DebeziumOptional `json:"snapshot"`  // Snapshot info (optional)
	Db        string           `json:"db"`        // Database name
	Sequence  DebeziumOptional `json:"sequence"`  // Sequence info (optional)
	Schema    string           `json:"schema"`    // Schema name
	Table     string           `json:"table"`     // Table name
	TxId      *DebeziumLong    `json:"txId"`      // Transaction ID (optional)
	Lsn       *DebeziumLong    `json:"lsn"`       // Log sequence number (optional)
	Xmin      any              `json:"xmin"`      // PostgreSQL xmin (optional)
}

// DebeziumOptional represents Avro optional fields that come as {"string": value} or null
type DebeziumOptional struct {
	String string `json:"string,omitempty"`
}

// DebeziumLong represents Avro long fields that come as {"long": value}
type DebeziumLong struct {
	Long int64 `json:"long"`
}

// DebeziumTimestamp represents Avro timestamp fields that come as {"long": value} or direct int64
type DebeziumTimestamp struct {
	Long *int64 `json:"long,omitempty"`
}

// UnmarshalJSON handles both direct int64 and {"long": value} formats
func (t *DebeziumTimestamp) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as direct int64 first
	var directValue int64
	if err := sonic.Unmarshal(data, &directValue); err == nil {
		t.Long = &directValue
		return nil
	}

	// Try to unmarshal as object with "long" field
	type tempStruct struct {
		Long int64 `json:"long"`
	}
	var temp tempStruct
	if err := sonic.Unmarshal(data, &temp); err == nil {
		t.Long = &temp.Long
		return nil
	}

	return ErrInvalidTimestampFormat
}

// Value returns the timestamp value, handling both formats
func (t *DebeziumTimestamp) Value() int64 {
	if t.Long != nil {
		return *t.Long
	}
	return 0
}

// DebeziumTransaction represents transaction information
type DebeziumTransaction struct {
	ID                  string `json:"id"`
	TotalOrder          int64  `json:"total_order"`
	DataCollectionOrder int64  `json:"data_collection_order"`
}

// CDCEvent represents a normalized change data capture event
// This is the internal representation used by handlers after converting from Debezium format
type CDCEvent struct {
	// Operation type: create, update, delete, read
	Operation string `json:"operation"`

	// Table that was changed
	Table string `json:"table"`

	// Schema of the table
	Schema string `json:"schema"`

	// Before state (for updates and deletes)
	Before map[string]any `json:"before,omitempty"`

	// After state (for creates and updates)
	After map[string]any `json:"after,omitempty"`

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

	// Whether this is from a snapshot
	Snapshot bool `json:"snapshot"`
}

// ExtractValueField extracts the actual data from Debezium's nested structure
// Debezium wraps data in {"Value": {...actual data...}} format
func ExtractValueField(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	// Check if this is wrapped in a "Value" field (uppercase)
	if value, ok := data["Value"].(map[string]any); ok {
		return value
	}

	// Check if this is wrapped in a "value" field (lowercase)
	if value, ok := data["value"].(map[string]any); ok {
		return value
	}

	// If the data only has one key and it's a map, extract it
	// This handles cases where the wrapper key might be dynamic
	if len(data) == 1 {
		for _, v := range data {
			if valueMap, ok := v.(map[string]any); ok {
				return valueMap
			}
		}
	}

	// Otherwise return as-is
	return data
}

// ConvertAvroOptionalField converts Avro optional fields to their actual values
// Handles {"string": "value"}, {"int": 123}, etc. formats
func ConvertAvroOptionalField(field any) any {
	if field == nil {
		return nil
	}

	// Handle map format for optional fields
	if m, ok := field.(map[string]any); ok {
		// Return the first value found
		for _, v := range m {
			return v
		}
	}

	// Return as-is if not in optional format
	return field
}
