package cdctypes

import "github.com/bytedance/sonic"

type DebeziumAvroEnvelope struct {
	Before      map[string]any       `json:"before"`
	After       map[string]any       `json:"after"`
	Source      DebeziumAvroSource   `json:"source"`
	Op          string               `json:"op"`
	TsMs        DebeziumTimestamp    `json:"ts_ms"`
	Transaction *DebeziumTransaction `json:"transaction,omitempty"`
}

type DebeziumAvroSource struct {
	Version   string           `json:"version"`
	Connector string           `json:"connector"`
	Name      string           `json:"name"`
	TsMs      int64            `json:"ts_ms"`
	Snapshot  DebeziumOptional `json:"snapshot,omitzero"`
	Db        string           `json:"db"`
	Sequence  DebeziumOptional `json:"sequence,omitzero"`
	Schema    string           `json:"schema"`
	Table     string           `json:"table"`
	TxId      *DebeziumLong    `json:"txId,omitempty"`
	Lsn       *DebeziumLong    `json:"lsn,omitempty"`
	Xmin      any              `json:"xmin,omitempty"`
}

type DebeziumOptional struct {
	String string `json:"string,omitempty"`
}

type DebeziumLong struct {
	Long int64 `json:"long"`
}

type DebeziumTimestamp struct {
	Long *int64 `json:"long,omitempty"`
}

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

func (t *DebeziumTimestamp) Value() int64 {
	if t.Long != nil {
		return *t.Long
	}
	return 0
}

type DebeziumTransaction struct {
	ID                  string `json:"id"`
	TotalOrder          int64  `json:"total_order"`
	DataCollectionOrder int64  `json:"data_collection_order"`
}

type CDCEvent struct {
	Operation string         `json:"operation"`
	Table     string         `json:"table"`
	Schema    string         `json:"schema"`
	Before    map[string]any `json:"before,omitempty"`
	After     map[string]any `json:"after,omitempty"`
	Metadata  CDCMetadata    `json:"metadata"`
}

type CDCMetadata struct {
	Timestamp     int64     `json:"timestamp"`
	Source        CDCSource `json:"source"`
	TransactionID string    `json:"transactionId,omitempty"`
	LSN           int64     `json:"lsn,omitempty"`
}

type CDCSource struct {
	Database  string `json:"database"`
	Schema    string `json:"schema"`
	Table     string `json:"table"`
	Connector string `json:"connector"`
	Version   string `json:"version"`
	Snapshot  bool   `json:"snapshot"`
}

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
