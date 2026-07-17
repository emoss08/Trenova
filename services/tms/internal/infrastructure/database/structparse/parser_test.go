package structparse

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustParsePackage(t *testing.T, dir string) []Model {
	t.Helper()
	result, err := ParsePackage(dir)
	if err != nil {
		t.Fatalf("ParsePackage() error = %v", err)
	}
	return result.Models
}

func mustParsePackageWithWarnings(t *testing.T, dir string) ([]Model, []string) {
	t.Helper()
	result, err := ParsePackage(dir)
	if err != nil {
		t.Fatalf("ParsePackage() error = %v", err)
	}
	return result.Models, result.Warnings
}

func TestParseBunFieldTag(t *testing.T) {
	tests := []struct {
		name         string
		tag          string
		wantCol      string
		wantRelKind  RelationKind
		wantJoins    []JoinPair
		wantM2MTable string
		wantSQLType  string
		wantScan     bool
		wantPK       bool
		wantNotNull  bool
		wantNullZero bool
		wantArray    bool
	}{
		{
			name:        "simple column with pk",
			tag:         "id,pk,type:VARCHAR(100)",
			wantCol:     "id",
			wantPK:      true,
			wantSQLType: "VARCHAR(100)",
		},
		{
			name:        "column with type and notnull",
			tag:         "business_unit_id,type:VARCHAR(100),notnull,pk",
			wantCol:     "business_unit_id",
			wantPK:      true,
			wantNotNull: true,
			wantSQLType: "VARCHAR(100)",
		},
		{
			name:        "column with default",
			tag:         "status,type:status_enum,notnull,default:'Active'",
			wantCol:     "status",
			wantNotNull: true,
			wantSQLType: "status_enum",
		},
		{
			name:        "scanonly column",
			tag:         "search_vector,type:TSVECTOR,scanonly",
			wantCol:     "search_vector",
			wantScan:    true,
			wantSQLType: "TSVECTOR",
		},
		{
			name:        "relationship belongs-to",
			tag:         "rel:belongs-to,join:state_id=id",
			wantRelKind: RelationBelongsTo,
			wantJoins:   []JoinPair{{Local: "state_id", Remote: "id"}},
		},
		{
			name:        "relationship has-many",
			tag:         "rel:has-many,join:id=worker_id,join:organization_id=organization_id",
			wantRelKind: RelationHasMany,
			wantJoins: []JoinPair{
				{Local: "id", Remote: "worker_id"},
				{Local: "organization_id", Remote: "organization_id"},
			},
		},
		{
			name:        "relationship has-one with composite join",
			tag:         "rel:has-one,join:id=worker_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id",
			wantRelKind: RelationHasOne,
			wantJoins: []JoinPair{
				{Local: "id", Remote: "worker_id"},
				{Local: "organization_id", Remote: "organization_id"},
				{Local: "business_unit_id", Remote: "business_unit_id"},
			},
		},
		{
			name:         "many-to-many",
			tag:          "m2m:customer_billing_profile_document_types,join:BillingProfile=DocumentType",
			wantM2MTable: "customer_billing_profile_document_types",
		},
		{
			name:    "excluded field",
			tag:     "-",
			wantCol: "",
		},
		{
			name:    "empty tag",
			tag:     "",
			wantCol: "",
		},
		{
			name:        "JSONB column",
			tag:         "options,type:JSONB",
			wantCol:     "options",
			wantSQLType: "JSONB",
		},
		{
			name:         "column with nullzero",
			tag:          "fleet_code_id,type:VARCHAR(100),nullzero",
			wantCol:      "fleet_code_id",
			wantNullZero: true,
			wantSQLType:  "VARCHAR(100)",
		},
		{
			name:        "rank scanonly",
			tag:         "rank,type:VARCHAR(100),scanonly",
			wantCol:     "rank",
			wantScan:    true,
			wantSQLType: "VARCHAR(100)",
		},
		{
			name:        "keyword-only tag returns empty column",
			tag:         "pk,notnull",
			wantCol:     "",
			wantPK:      true,
			wantNotNull: true,
		},
		{
			name:     "scanonly-only tag returns empty column",
			tag:      "scanonly",
			wantCol:  "",
			wantScan: true,
		},
		{
			name:         "nullzero-only tag returns empty column",
			tag:          "nullzero",
			wantCol:      "",
			wantNullZero: true,
		},
		{
			name:    "column with default containing quotes",
			tag:     "created_at,notnull,default:extract(epoch from current_timestamp)::bigint",
			wantCol: "created_at",

			wantNotNull: true,
		},
		{
			name:    "column with soft_delete keyword",
			tag:     "deleted_at,soft_delete",
			wantCol: "deleted_at",
		},
		{
			name:      "array column",
			tag:       "tags,type:TEXT[],array",
			wantCol:   "tags",
			wantArray: true,

			wantSQLType: "TEXT[]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed := parseBunFieldTag(tt.tag)
			if parsed.ColumnName != tt.wantCol {
				t.Errorf("column = %q, want %q", parsed.ColumnName, tt.wantCol)
			}
			if parsed.RelationKind != tt.wantRelKind {
				t.Errorf("relationKind = %q, want %q", parsed.RelationKind, tt.wantRelKind)
			}
			if len(parsed.JoinPairs) != len(tt.wantJoins) {
				t.Fatalf("joinPairs = %v, want %v", parsed.JoinPairs, tt.wantJoins)
			}
			for i, jp := range tt.wantJoins {
				if parsed.JoinPairs[i] != jp {
					t.Errorf("joinPairs[%d] = %v, want %v", i, parsed.JoinPairs[i], jp)
				}
			}
			if parsed.M2MTable != tt.wantM2MTable {
				t.Errorf("m2mTable = %q, want %q", parsed.M2MTable, tt.wantM2MTable)
			}
			if parsed.SQLType != tt.wantSQLType {
				t.Errorf("sqlType = %q, want %q", parsed.SQLType, tt.wantSQLType)
			}
			if parsed.IsScanOnly != tt.wantScan {
				t.Errorf("isScanOnly = %v, want %v", parsed.IsScanOnly, tt.wantScan)
			}
			if parsed.IsPK != tt.wantPK {
				t.Errorf("isPK = %v, want %v", parsed.IsPK, tt.wantPK)
			}
			if parsed.IsNotNull != tt.wantNotNull {
				t.Errorf("isNotNull = %v, want %v", parsed.IsNotNull, tt.wantNotNull)
			}
			if parsed.IsNullZero != tt.wantNullZero {
				t.Errorf("isNullZero = %v, want %v", parsed.IsNullZero, tt.wantNullZero)
			}
			if parsed.IsArray != tt.wantArray {
				t.Errorf("isArray = %v, want %v", parsed.IsArray, tt.wantArray)
			}
		})
	}
}

func TestParseBaseModelTag(t *testing.T) {
	tests := []struct {
		name      string
		tag       string
		wantTable string
		wantAlias string
	}{
		{
			name:      "standard table and alias",
			tag:       "table:workers,alias:wrk",
			wantTable: "workers",
			wantAlias: "wrk",
		},
		{
			name:      "table only",
			tag:       "table:shipments",
			wantTable: "shipments",
		},
		{
			name: "empty tag",
			tag:  "",
		},
		{
			name:      "alias before table",
			tag:       "alias:sp,table:shipments",
			wantTable: "shipments",
			wantAlias: "sp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			table, alias := parseBaseModelTag(tt.tag)
			if table != tt.wantTable {
				t.Errorf("table = %q, want %q", table, tt.wantTable)
			}
			if alias != tt.wantAlias {
				t.Errorf("alias = %q, want %q", alias, tt.wantAlias)
			}
		})
	}
}

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		name string
		tag  string
		want string
	}{
		{"simple name", "firstName", "firstName"},
		{"with omitempty", "firstName,omitempty", "firstName"},
		{"dash excluded", "-", ""},
		{"empty", "", ""},
		{"dash with name", "-,", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseJSONTag(tt.tag)
			if got != tt.want {
				t.Errorf("parseJSONTag(%q) = %q, want %q", tt.tag, got, tt.want)
			}
		})
	}
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestParsePackage_SkipsNamedEmbedField(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type Cursor struct {
	Seq int64 `+"`"+`json:"-" bun:"seq"`+"`"+`
}

type Entity struct {
	bun.BaseModel `+"`"+`bun:"table:entities,alias:e" json:"-"`+"`"+`

	Cursor Cursor `+"`"+`json:"-" bun:",embed"`+"`"+`

	ID   string `+"`"+`json:"id"   bun:"id,pk,type:VARCHAR(100)"`+"`"+`
	Name string `+"`"+`json:"name" bun:"name,type:VARCHAR(100),notnull"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	for _, field := range models[0].Fields {
		if field.ColumnName == "embed" || field.GoName == "Cursor" {
			t.Fatalf("named embed field was parsed as a column: %+v", field)
		}
	}
	if len(models[0].Fields) != 2 {
		t.Fatalf(
			"got %d fields, want 2 (ID, Name). Fields: %+v",
			len(models[0].Fields),
			models[0].Fields,
		)
	}
}

func TestParsePackage(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type TestEntity struct {
	bun.BaseModel `+"`"+`bun:"table:test_entities,alias:te" json:"-"`+"`"+`

	ID        string `+"`"+`json:"id"        bun:"id,pk,type:VARCHAR(100)"`+"`"+`
	Name      string `+"`"+`json:"name"      bun:"name,type:VARCHAR(100),notnull"`+"`"+`
	Status    string `+"`"+`json:"status"    bun:"status,type:VARCHAR(50),notnull"`+"`"+`
	Hidden    string `+"`"+`json:"-"         bun:"hidden_col,type:TEXT"`+"`"+`
	Computed  string `+"`"+`json:"-"         bun:"computed,type:TSVECTOR,scanonly"`+"`"+`
	Excluded  map[string]any `+"`"+`json:"excluded" bun:"-"`+"`"+`

	Related *TestEntity `+"`"+`json:"related,omitempty" bun:"rel:belongs-to,join:related_id=id"`+"`"+`
}

type SecondEntity struct {
	bun.BaseModel `+"`"+`bun:"table:second_entities,alias:se" json:"-"`+"`"+`

	ID   string `+"`"+`json:"id"   bun:"id,pk,type:VARCHAR(100)"`+"`"+`
	Code string `+"`"+`json:"code" bun:"code,type:VARCHAR(50),notnull"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)

	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}

	var te *Model
	for i := range models {
		if models[i].StructName == "TestEntity" {
			te = &models[i]
			break
		}
	}
	if te == nil {
		t.Fatal("TestEntity not found")
	}

	if te.TableName != "test_entities" {
		t.Errorf("TableName = %q, want %q", te.TableName, "test_entities")
	}
	if te.Alias != "te" {
		t.Errorf("Alias = %q, want %q", te.Alias, "te")
	}

	if len(te.Fields) != 5 {
		t.Fatalf("got %d fields, want 5. Fields: %+v", len(te.Fields), te.Fields)
	}

	if !te.Fields[0].IsPK {
		t.Error("ID should be PK")
	}

	hidden, ok := te.Field("Hidden")
	if !ok {
		t.Fatal("Hidden field not found")
	}
	if hidden.JSONName != "" {
		t.Errorf("Hidden.JSONName = %q, want empty", hidden.JSONName)
	}

	computed, ok := te.Field("Computed")
	if !ok {
		t.Fatal("Computed field not found")
	}
	if !computed.IsScanOnly {
		t.Error("Computed should be scanonly")
	}

	fieldMapEntries := te.FieldMapEntries()
	if len(fieldMapEntries) != 3 {
		t.Errorf("got %d FieldMap entries, want 3 (ID, Name, Status)", len(fieldMapEntries))
	}

	if len(te.Relations) != 1 {
		t.Fatalf("got %d relations, want 1 (Related)", len(te.Relations))
	}
	rel := te.Relations[0]
	if rel.GoName != "Related" {
		t.Errorf("relation GoName = %q, want %q", rel.GoName, "Related")
	}
	if rel.Kind != RelationBelongsTo {
		t.Errorf("relation Kind = %q, want %q", rel.Kind, RelationBelongsTo)
	}
	if rel.JSONName != "related" {
		t.Errorf("relation JSONName = %q, want %q", rel.JSONName, "related")
	}
	if rel.GoType != "*TestEntity" {
		t.Errorf("relation GoType = %q, want %q", rel.GoType, "*TestEntity")
	}
	if len(rel.JoinPairs) != 1 ||
		rel.JoinPairs[0] != (JoinPair{Local: "related_id", Remote: "id"}) {
		t.Errorf("relation JoinPairs = %v, want [{related_id id}]", rel.JoinPairs)
	}

	if te.HasTenantFields() {
		t.Error("TestEntity should not have tenant fields")
	}

	filterable := te.FilterableFields()
	if len(filterable) != 3 {
		t.Errorf("got %d filterable fields, want 3", len(filterable))
	}

	insertable := te.InsertableColumns()
	if len(insertable) != 4 {
		t.Errorf("got %d insertable columns, want 4 (ID, Name, Status, Hidden)", len(insertable))
	}

	pks := te.PKColumns()
	if len(pks) != 1 || pks[0] != "id" {
		t.Errorf("PKColumns = %v, want [id]", pks)
	}

	var se *Model
	for i := range models {
		if models[i].StructName == "SecondEntity" {
			se = &models[i]
		}
	}
	if se == nil {
		t.Fatal("SecondEntity not found")
	}
	if len(se.Fields) != 2 {
		t.Errorf("SecondEntity has %d fields, want 2", len(se.Fields))
	}
}

func TestParsePackage_FieldTypeMetadata(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import (
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

type TypedEntity struct {
	bun.BaseModel `+"`"+`bun:"table:typed_entities,alias:ty" json:"-"`+"`"+`

	ID       string              `+"`"+`json:"id"       bun:"id,pk,type:VARCHAR(100)"`+"`"+`
	Amount   decimal.NullDecimal `+"`"+`json:"amount"   bun:"amount,type:NUMERIC(19,4),nullzero"`+"`"+`
	Note     *string             `+"`"+`json:"note"     bun:"note,type:TEXT,nullzero"`+"`"+`
	Created  int64               `+"`"+`json:"created"  bun:"created_at,type:BIGINT,notnull"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	amount, ok := models[0].Field("Amount")
	if !ok {
		t.Fatal("Amount not found")
	}
	if amount.GoType != "decimal.NullDecimal" {
		t.Errorf("Amount.GoType = %q, want decimal.NullDecimal", amount.GoType)
	}
	if amount.SQLType != "NUMERIC(19,4)" {
		t.Errorf("Amount.SQLType = %q, want NUMERIC(19,4)", amount.SQLType)
	}
	if !amount.IsNullZero {
		t.Error("Amount should be nullzero")
	}

	note, ok := models[0].Field("Note")
	if !ok {
		t.Fatal("Note not found")
	}
	if !note.IsPointer {
		t.Error("Note should be flagged as pointer")
	}
	if note.GoType != "*string" {
		t.Errorf("Note.GoType = %q, want *string", note.GoType)
	}

	created, ok := models[0].Field("Created")
	if !ok {
		t.Fatal("Created not found")
	}
	if !created.IsNotNull {
		t.Error("Created should be notnull")
	}
	if created.SQLType != "BIGINT" {
		t.Errorf("Created.SQLType = %q, want BIGINT", created.SQLType)
	}
}

func TestParsePackage_M2MRelations(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type Profile struct {
	bun.BaseModel `+"`"+`bun:"table:profiles,alias:pr" json:"-"`+"`"+`

	ID string `+"`"+`json:"id" bun:"id,pk,type:VARCHAR(100)"`+"`"+`

	DocumentTypes []*Profile `+"`"+`json:"documentTypes,omitempty" bun:"m2m:profile_document_types,join:Profile=DocumentType"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	m := models[0]
	if len(m.Relations) != 0 {
		t.Errorf("m2m must not appear in Relations, got %v", m.Relations)
	}
	if len(m.M2MRelations) != 1 {
		t.Fatalf("got %d m2m relations, want 1", len(m.M2MRelations))
	}
	m2m := m.M2MRelations[0]
	if m2m.GoName != "DocumentTypes" {
		t.Errorf("m2m GoName = %q, want DocumentTypes", m2m.GoName)
	}
	if m2m.ThroughTable != "profile_document_types" {
		t.Errorf("m2m ThroughTable = %q, want profile_document_types", m2m.ThroughTable)
	}
	if m2m.JoinSpec != "Profile=DocumentType" {
		t.Errorf("m2m JoinSpec = %q, want Profile=DocumentType", m2m.JoinSpec)
	}
}

func TestParsePackage_Enums(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "enums.go", `package testpkg

type Status string

const (
	StatusActive   Status = "Active"
	StatusInactive Status = "Inactive"
)

type Kind string

const (
	KindPrimary   Kind = "Primary"
	KindSecondary Kind = "Secondary"
	KindTertiary  Kind = "Tertiary"
)

type NotAnEnum int

const (
	NotAnEnumOne NotAnEnum = 1
)

const untypedConstant = "ignored"
`)

	result, err := ParsePackage(dir)
	if err != nil {
		t.Fatalf("ParsePackage() error = %v", err)
	}

	if len(result.Enums) != 2 {
		t.Fatalf("got %d enums, want 2: %+v", len(result.Enums), result.Enums)
	}

	status, ok := result.Enum("Status")
	if !ok {
		t.Fatal("Status enum not found")
	}
	if len(status.Values) != 2 || status.Values[0] != "Active" || status.Values[1] != "Inactive" {
		t.Errorf("Status values = %v, want [Active Inactive]", status.Values)
	}

	kind, ok := result.Enum("Kind")
	if !ok {
		t.Fatal("Kind enum not found")
	}
	if len(kind.Values) != 3 {
		t.Errorf("Kind values = %v, want 3 values", kind.Values)
	}

	if _, found := result.Enum("NotAnEnum"); found {
		t.Error("int-backed type must not be reported as a string enum")
	}
}

func TestParsePackage_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity_test.go", `package testpkg

import "github.com/uptrace/bun"

type TestOnlyEntity struct {
	bun.BaseModel `+"`"+`bun:"table:test_only,alias:to" json:"-"`+"`"+`
	ID string `+"`"+`json:"id" bun:"id,pk"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip _test.go files, got %d models", len(models))
	}
}

func TestParsePackage_SkipsNonBunStructs(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "plain.go", `package testpkg

type PlainStruct struct {
	Name string
	Value int
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip non-Bun structs, got %d models", len(models))
	}
}

func TestParsePackage_SkipsMissingAlias(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type NoAlias struct {
	bun.BaseModel `+"`"+`bun:"table:no_aliases" json:"-"`+"`"+`
	ID string `+"`"+`json:"id" bun:"id,pk"`+"`"+`
}
`)

	models, warnings := mustParsePackageWithWarnings(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip entities with missing alias, got %d models", len(models))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "NoAlias") ||
		!strings.Contains(warnings[0], "missing alias") {
		t.Errorf("warning should mention struct name and missing alias, got: %s", warnings[0])
	}
}

func TestParsePackage_SkipsMissingTableName(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type NoTable struct {
	bun.BaseModel `+"`"+`bun:"alias:nt" json:"-"`+"`"+`
	ID string `+"`"+`json:"id" bun:"id,pk"`+"`"+`
}
`)

	models, warnings := mustParsePackageWithWarnings(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip entities with missing table name, got %d models", len(models))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "NoTable") ||
		!strings.Contains(warnings[0], "missing table") {
		t.Errorf("warning should mention struct name and missing table, got: %s", warnings[0])
	}
}

func TestParsePackage_SkipsEntityWithNoFields(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type EmptyEntity struct {
	bun.BaseModel `+"`"+`bun:"table:empties,alias:em" json:"-"`+"`"+`

	Related *EmptyEntity `+"`"+`json:"related" bun:"rel:belongs-to,join:id=id"`+"`"+`
}
`)

	models, warnings := mustParsePackageWithWarnings(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip entities with no parseable fields, got %d models", len(models))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "EmptyEntity") ||
		!strings.Contains(warnings[0], "no parseable columns") {
		t.Errorf(
			"warning should mention struct name and no parseable columns, got: %s",
			warnings[0],
		)
	}
}

func TestParsePackage_EntityWithNoJSONTags(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type InternalEntity struct {
	bun.BaseModel `+"`"+`bun:"table:internals,alias:int"`+"`"+`

	ID        string `+"`"+`bun:"id,pk,type:VARCHAR(100)"`+"`"+`
	SeqType   string `+"`"+`bun:"sequence_type,notnull"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	m := models[0]
	if len(m.Fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(m.Fields))
	}

	if len(m.FieldMapEntries()) != 0 {
		t.Errorf(
			"FieldMap should be empty for entity with no json tags, got %d entries",
			len(m.FieldMapEntries()),
		)
	}

	if len(m.InsertableColumns()) != 2 {
		t.Errorf("InsertableColumns should have 2 entries, got %d", len(m.InsertableColumns()))
	}
}

func TestTenantFieldsAndRelations(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "entity.go", `package testpkg

import "github.com/uptrace/bun"

type TenantEntity struct {
	bun.BaseModel `+"`"+`bun:"table:tenant_entities,alias:te" json:"-"`+"`"+`

	ID             string `+"`"+`json:"id"             bun:"id,pk"`+"`"+`
	OrganizationID string `+"`"+`json:"organizationId" bun:"organization_id,pk"`+"`"+`
	BusinessUnitID string `+"`"+`json:"businessUnitId" bun:"business_unit_id,pk"`+"`"+`
	Name           string `+"`"+`json:"name"           bun:"name"`+"`"+`

	Org  *TenantEntity `+"`"+`bun:"rel:belongs-to,join:organization_id=id"`+"`"+`
	BU   *TenantEntity `+"`"+`bun:"rel:belongs-to,join:business_unit_id=id"`+"`"+`
	Items []*TenantEntity `+"`"+`bun:"rel:has-many,join:id=parent_id"`+"`"+`
}
`)

	models := mustParsePackage(t, dir)
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	m := models[0]

	if !m.HasTenantFields() {
		t.Error("should detect tenant fields (OrganizationID + BusinessUnitID)")
	}

	if len(m.Relations) != 3 {
		t.Fatalf("got %d relations, want 3 (Org, BU, Items)", len(m.Relations))
	}

	if !m.HasRelations() {
		t.Error("HasRelations() should be true")
	}

	items, ok := m.Relation("Items")
	if !ok {
		t.Fatal("Items relation not found")
	}
	if items.Kind != RelationHasMany {
		t.Errorf("Items.Kind = %q, want has-many", items.Kind)
	}

	filterable := m.FilterableFields()
	if len(filterable) != 4 {
		t.Errorf(
			"got %d filterable fields, want 4 (ID, OrganizationID, BusinessUnitID, Name)",
			len(filterable),
		)
	}
}

func TestParsePackage_ConversionStyleEnums(t *testing.T) {
	dir := t.TempDir()

	writeTestFile(t, dir, "enums.go", `package testpkg

type Phase string

const (
	PhaseOne = Phase("One")
	PhaseTwo = Phase("Two")
)
`)

	result, err := ParsePackage(dir)
	if err != nil {
		t.Fatalf("ParsePackage() error = %v", err)
	}

	phase, ok := result.Enum("Phase")
	if !ok {
		t.Fatal("Phase enum not found")
	}
	if len(phase.Values) != 2 || phase.Values[0] != "One" || phase.Values[1] != "Two" {
		t.Errorf("Phase values = %v, want [One Two]", phase.Values)
	}
}
