package cdctypes

import "github.com/bytedance/sonic"

// DebeziumAvroEnvelope represents the structure of Avro-encoded Debezium change events
// as they come from Kafka with Confluent Schema Registry.
// This structure matches the actual wire format we receive.
type DebeziumAvroEnvelope struct {
	Before      map[string]any       `json:"before"`
	After       map[string]any       `json:"after"`
	Source      DebeziumAvroSource   `json:"source"`
	Op          string               `json:"op"`
	TsMs        DebeziumTimestamp    `json:"ts_ms"`
	Transaction *DebeziumTransaction `json:"transaction,omitempty"`
}

// DebeziumAvroSource contains source metadata in Avro format
type DebeziumAvroSource struct {
	Version   string           `json:"version"`
	Connector string           `json:"connector"`
	Name      string           `json:"name"`
	TsMs      int64            `json:"ts_ms"`
	Snapshot  DebeziumOptional `json:"snapshot,omitempty"`
	Db        string           `json:"db"`
	Sequence  DebeziumOptional `json:"sequence,omitempty"`
	Schema    string           `json:"schema"`
	Table     string           `json:"table"`
	TxId      *DebeziumLong    `json:"txId,omitempty"`
	Lsn       *DebeziumLong    `json:"lsn,omitempty"`
	Xmin      any              `json:"xmin,omitempty"`
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
	var directValue int64
	if err := sonic.Unmarshal(data, &directValue); err == nil {
		t.Long = &directValue
		return nil
	}

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
	Operation string         `json:"operation"`
	Table     string         `json:"table"`
	Schema    string         `json:"schema"`
	Before    map[string]any `json:"before,omitempty"`
	After     map[string]any `json:"after,omitempty"`
	Metadata  CDCMetadata    `json:"metadata"`
}

// CDCMetadata contains metadata about the CDC event
type CDCMetadata struct {
	Timestamp     int64     `json:"timestamp"`
	Source        CDCSource `json:"source"`
	TransactionID string    `json:"transactionId,omitempty"`
	LSN           int64     `json:"lsn,omitempty"`
}

// CDCSource contains information about the source database
type CDCSource struct {
	Database  string `json:"database"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Connector string `json:"connector"`
	Version   string `json:"version"`
	Snapshot  bool   `json:"snapshot"`
}

// ExtractValueField extracts the actual data from Debezium's nested structure
// Debezium wraps data in {"Value": {...actual data...}} format
func ExtractValueField(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	if value, ok := data["Value"].(map[string]any); ok {
		return value
	}

	if value, ok := data["value"].(map[string]any); ok {
		return value
	}

	if len(data) == 1 {
		for _, v := range data {
			if valueMap, ok := v.(map[string]any); ok {
				return valueMap
			}
		}
	}

	return data
}

// ConvertAvroOptionalField converts Avro optional fields to their actual values
// Handles {"string": "value"}, {"int": 123}, etc. formats
func ConvertAvroOptionalField(field any) any {
	if field == nil {
		return nil
	}

	if m, ok := field.(map[string]any); ok {
		for _, v := range m {
			return v
		}
	}

	return field
}
