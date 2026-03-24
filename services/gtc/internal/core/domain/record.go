package domain

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
)

type Operation string

const (
	OperationInsert   Operation = "INSERT"
	OperationUpdate   Operation = "UPDATE"
	OperationDelete   Operation = "DELETE"
	OperationTruncate Operation = "TRUNCATE"
	OperationSnapshot Operation = "SNAPSHOT"
)

func (o Operation) String() string {
	return string(o)
}

type DestinationKind string

const (
	DestinationMeilisearch DestinationKind = "meilisearch"
	DestinationRedisJSON   DestinationKind = "redis_json"
	DestinationRedisStream DestinationKind = "redis_stream"
)

type RecordMetadata struct {
	LSN           string
	CommitLSN     string
	TransactionID uint32
	Timestamp     time.Time
	Snapshot      bool
}

type EventMetadata = RecordMetadata

type SourceRecord struct {
	Operation Operation
	Schema    string
	Table     string
	OldData   map[string]any
	NewData   map[string]any
	Metadata  RecordMetadata
}

type CDCEvent = SourceRecord

type TransactionRecords struct {
	LSN           string
	CommitLSN     string
	TransactionID uint32
	Timestamp     time.Time
	Records       []SourceRecord
}

type DeadLetterRecord struct {
	TransactionID uint32
	CommitLSN     string
	Projection    string
	Error         string
	Attempts      int
	Record        SourceRecord
	CreatedAt     time.Time
}

func (r SourceRecord) FullTableName() string {
	return fmt.Sprintf("%s.%s", r.Schema, r.Table)
}

func (r SourceRecord) PrimaryData() map[string]any {
	if r.NewData != nil {
		return r.NewData
	}
	return r.OldData
}

type Destination struct {
	Kind        DestinationKind
	Index       string
	KeyTemplate string
	Stream      string
}

type Projection struct {
	Name             string
	SourceSchema     string
	SourceTable      string
	PrimaryKeys      []string
	Fields           []string
	SearchableFields []string
	FilterableFields []string
	IgnoredUpdates   []string
	Destination      Destination
}

func (p Projection) FullTableName() string {
	return fmt.Sprintf("%s.%s", p.SourceSchema, p.SourceTable)
}

type SnapshotBinding struct {
	Schema      string
	Table       string
	PrimaryKeys []string
}

func (b SnapshotBinding) FullTableName() string {
	return fmt.Sprintf("%s.%s", b.Schema, b.Table)
}

type TableMetadata struct {
	Schema      string
	Table       string
	PrimaryKeys []string
}

func (m TableMetadata) FullTableName() string {
	return fmt.Sprintf("%s.%s", m.Schema, m.Table)
}

type Cursor struct {
	Values []any `json:"values"`
}

func (c Cursor) IsZero() bool {
	return len(c.Values) == 0
}

func (c Cursor) Marshal() (string, error) {
	if len(c.Values) == 0 {
		return "", nil
	}

	payload, err := json.Marshal(c)
	if err != nil {
		return "", fmt.Errorf("marshal cursor: %w", err)
	}

	return string(payload), nil
}

func ParseCursor(raw string) (Cursor, error) {
	if strings.TrimSpace(raw) == "" {
		return Cursor{}, nil
	}

	var cursor Cursor
	if err := json.Unmarshal([]byte(raw), &cursor); err != nil {
		return Cursor{}, fmt.Errorf("parse cursor: %w", err)
	}

	return cursor, nil
}

func RecordKey(data map[string]any, keyFields []string) ([]any, error) {
	if len(keyFields) == 0 {
		return nil, fmt.Errorf("record key fields are empty")
	}
	if data == nil {
		return nil, fmt.Errorf("record data is empty")
	}

	values := make([]any, 0, len(keyFields))
	for _, field := range keyFields {
		value, ok := data[field]
		if !ok {
			return nil, fmt.Errorf("record missing key field %q", field)
		}
		values = append(values, value)
	}

	return values, nil
}

func PrimaryKey(record SourceRecord, keyFields []string) ([]any, error) {
	return RecordKey(record.PrimaryData(), keyFields)
}

func KeyString(values []any) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%v", value))
	}
	return strings.Join(parts, "|")
}

func EqualStringSlices(left []string, right []string) bool {
	return slices.Equal(left, right)
}

func ParseFullTableName(value string) (string, string, error) {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid table name %q", value)
	}

	schema := strings.TrimSpace(parts[0])
	table := strings.TrimSpace(parts[1])
	if schema == "" || table == "" {
		return "", "", fmt.Errorf("invalid table name %q", value)
	}

	return schema, table, nil
}

func ChangedFields(oldData map[string]any, newData map[string]any) []string {
	if oldData == nil || newData == nil {
		return nil
	}

	seen := make(map[string]struct{}, len(oldData)+len(newData))
	changed := make([]string, 0)

	for key := range oldData {
		seen[key] = struct{}{}
	}
	for key := range newData {
		seen[key] = struct{}{}
	}

	for key := range seen {
		if !reflect.DeepEqual(oldData[key], newData[key]) {
			changed = append(changed, key)
		}
	}

	return changed
}
