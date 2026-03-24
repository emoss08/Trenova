package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func mustParsePackage(t *testing.T, dir string) []ModelInfo {
	t.Helper()
	result, err := ParsePackage(dir)
	if err != nil {
		t.Fatalf("ParsePackage() error = %v", err)
	}
	return result.Models
}

func mustParsePackageWithWarnings(t *testing.T, dir string) ([]ModelInfo, []string) {
	t.Helper()
	result, err := ParsePackage(dir)
	if err != nil {
		t.Fatalf("ParsePackage() error = %v", err)
	}
	return result.Models, result.Warnings
}

func TestParseBunFieldTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		wantCol  string
		wantRel  bool
		wantScan bool
		wantPK   bool
	}{
		{
			name:    "simple column with pk",
			tag:     "id,pk,type:VARCHAR(100)",
			wantCol: "id",
			wantPK:  true,
		},
		{
			name:    "column with type and notnull",
			tag:     "business_unit_id,type:VARCHAR(100),notnull,pk",
			wantCol: "business_unit_id",
			wantPK:  true,
		},
		{
			name:    "column with default",
			tag:     "status,type:status_enum,notnull,default:'Active'",
			wantCol: "status",
		},
		{
			name:     "scanonly column",
			tag:      "search_vector,type:TSVECTOR,scanonly",
			wantCol:  "search_vector",
			wantScan: true,
		},
		{
			name:    "relationship belongs-to",
			tag:     "rel:belongs-to,join:state_id=id",
			wantRel: true,
		},
		{
			name:    "relationship has-many",
			tag:     "rel:has-many,join:id=worker_id,join:organization_id=organization_id",
			wantRel: true,
		},
		{
			name:    "relationship has-one with composite join",
			tag:     "rel:has-one,join:id=worker_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id",
			wantRel: true,
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
			name:    "JSONB column",
			tag:     "options,type:JSONB",
			wantCol: "options",
		},
		{
			name:    "column with nullzero",
			tag:     "fleet_code_id,type:VARCHAR(100),nullzero",
			wantCol: "fleet_code_id",
		},
		{
			name:     "rank scanonly",
			tag:      "rank,type:VARCHAR(100),scanonly",
			wantCol:  "rank",
			wantScan: true,
		},
		{
			name:    "keyword-only tag returns empty column",
			tag:     "pk,notnull",
			wantCol: "",
			wantPK:  true,
		},
		{
			name:     "scanonly-only tag returns empty column",
			tag:      "scanonly",
			wantCol:  "",
			wantScan: true,
		},
		{
			name:    "nullzero-only tag returns empty column",
			tag:     "nullzero",
			wantCol: "",
		},
		{
			name:    "column with default containing quotes",
			tag:     "created_at,notnull,default:extract(epoch from current_timestamp)::bigint",
			wantCol: "created_at",
		},
		{
			name:    "column with soft_delete keyword",
			tag:     "deleted_at,soft_delete",
			wantCol: "deleted_at",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			col, isRel, isScan, isPK := parseBunFieldTag(tt.tag)
			if col != tt.wantCol {
				t.Errorf("column = %q, want %q", col, tt.wantCol)
			}
			if isRel != tt.wantRel {
				t.Errorf("isRelation = %v, want %v", isRel, tt.wantRel)
			}
			if isScan != tt.wantScan {
				t.Errorf("isScanOnly = %v, want %v", isScan, tt.wantScan)
			}
			if isPK != tt.wantPK {
				t.Errorf("isPK = %v, want %v", isPK, tt.wantPK)
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

func TestParsePackage(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

import "github.com/uptrace/bun"

type TestEntity struct {
	bun.BaseModel ` + "`" + `bun:"table:test_entities,alias:te" json:"-"` + "`" + `

	ID        string ` + "`" + `json:"id"        bun:"id,pk,type:VARCHAR(100)"` + "`" + `
	Name      string ` + "`" + `json:"name"      bun:"name,type:VARCHAR(100),notnull"` + "`" + `
	Status    string ` + "`" + `json:"status"    bun:"status,type:VARCHAR(50),notnull"` + "`" + `
	Hidden    string ` + "`" + `json:"-"         bun:"hidden_col,type:TEXT"` + "`" + `
	Computed  string ` + "`" + `json:"-"         bun:"computed,type:TSVECTOR,scanonly"` + "`" + `
	Excluded  map[string]any ` + "`" + `json:"excluded" bun:"-"` + "`" + `

	Related *TestEntity ` + "`" + `json:"related,omitempty" bun:"rel:belongs-to,join:related_id=id"` + "`" + `
}

type SecondEntity struct {
	bun.BaseModel ` + "`" + `bun:"table:second_entities,alias:se" json:"-"` + "`" + `

	ID   string ` + "`" + `json:"id"   bun:"id,pk,type:VARCHAR(100)"` + "`" + `
	Code string ` + "`" + `json:"code" bun:"code,type:VARCHAR(50),notnull"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	models := mustParsePackage(t, dir)

	if len(models) != 2 {
		t.Fatalf("got %d models, want 2", len(models))
	}

	var te *ModelInfo
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

	// Should have: ID, Name, Status, Hidden, Computed (5 fields)
	// Excluded (bun:"-") and Related (rel:belongs-to) should be skipped
	if len(te.Fields) != 5 {
		t.Fatalf("got %d fields, want 5. Fields: %+v", len(te.Fields), te.Fields)
	}

	if !te.Fields[0].IsPK {
		t.Error("ID should be PK")
	}

	var hidden *FieldInfo
	for i := range te.Fields {
		if te.Fields[i].GoName == "Hidden" {
			hidden = &te.Fields[i]
		}
	}
	if hidden == nil {
		t.Fatal("Hidden field not found")
	}
	if hidden.JSONName != "" {
		t.Errorf("Hidden.JSONName = %q, want empty", hidden.JSONName)
	}

	var computed *FieldInfo
	for i := range te.Fields {
		if te.Fields[i].GoName == "Computed" {
			computed = &te.Fields[i]
		}
	}
	if computed == nil {
		t.Fatal("Computed field not found")
	}
	if !computed.IsScanOnly {
		t.Error("Computed should be scanonly")
	}

	fieldMapEntries := te.FieldMapEntries()
	if len(fieldMapEntries) != 3 {
		t.Errorf("got %d FieldMap entries, want 3 (ID, Name, Status)", len(fieldMapEntries))
	}

	// Check relations are captured
	if len(te.Relations) != 1 {
		t.Fatalf("got %d relations, want 1 (Related)", len(te.Relations))
	}
	if te.Relations[0].GoName != "Related" {
		t.Errorf("relation GoName = %q, want %q", te.Relations[0].GoName, "Related")
	}

	// TestEntity has no tenant fields (no OrganizationID/BusinessUnitID)
	if te.HasTenantFields() {
		t.Error("TestEntity should not have tenant fields")
	}

	// FilterableFields = FieldMap entries minus scanonly
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

	var se *ModelInfo
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

func TestParsePackage_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()

	testContent := `package testpkg

import "github.com/uptrace/bun"

type TestOnlyEntity struct {
	bun.BaseModel ` + "`" + `bun:"table:test_only,alias:to" json:"-"` + "`" + `
	ID string ` + "`" + `json:"id" bun:"id,pk"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity_test.go"), []byte(testContent), 0o644); err != nil {
		t.Fatal(err)
	}

	models := mustParsePackage(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip _test.go files, got %d models", len(models))
	}
}

func TestParsePackage_SkipsNonBunStructs(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

type PlainStruct struct {
	Name string
	Value int
}
`
	if err := os.WriteFile(filepath.Join(dir, "plain.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	models := mustParsePackage(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip non-Bun structs, got %d models", len(models))
	}
}

func TestParsePackage_SkipsMissingAlias(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

import "github.com/uptrace/bun"

type NoAlias struct {
	bun.BaseModel ` + "`" + `bun:"table:no_aliases" json:"-"` + "`" + `
	ID string ` + "`" + `json:"id" bun:"id,pk"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	models, warnings := mustParsePackageWithWarnings(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip entities with missing alias, got %d models", len(models))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "NoAlias") || !strings.Contains(warnings[0], "missing alias") {
		t.Errorf("warning should mention struct name and missing alias, got: %s", warnings[0])
	}
}

func TestParsePackage_SkipsMissingTableName(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

import "github.com/uptrace/bun"

type NoTable struct {
	bun.BaseModel ` + "`" + `bun:"alias:nt" json:"-"` + "`" + `
	ID string ` + "`" + `json:"id" bun:"id,pk"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	models, warnings := mustParsePackageWithWarnings(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip entities with missing table name, got %d models", len(models))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "NoTable") || !strings.Contains(warnings[0], "missing table") {
		t.Errorf("warning should mention struct name and missing table, got: %s", warnings[0])
	}
}

func TestParsePackage_SkipsEntityWithNoFields(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

import "github.com/uptrace/bun"

type EmptyEntity struct {
	bun.BaseModel ` + "`" + `bun:"table:empties,alias:em" json:"-"` + "`" + `

	Related *EmptyEntity ` + "`" + `json:"related" bun:"rel:belongs-to,join:id=id"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	models, warnings := mustParsePackageWithWarnings(t, dir)
	if len(models) != 0 {
		t.Errorf("should skip entities with no parseable fields, got %d models", len(models))
	}
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "EmptyEntity") || !strings.Contains(warnings[0], "no parseable columns") {
		t.Errorf("warning should mention struct name and no parseable columns, got: %s", warnings[0])
	}
}

func TestParsePackage_EntityWithNoJSONTags(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

import "github.com/uptrace/bun"

type InternalEntity struct {
	bun.BaseModel ` + "`" + `bun:"table:internals,alias:int"` + "`" + `

	ID        string ` + "`" + `bun:"id,pk,type:VARCHAR(100)"` + "`" + `
	SeqType   string ` + "`" + `bun:"sequence_type,notnull"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	models := mustParsePackage(t, dir)
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}

	m := models[0]
	if len(m.Fields) != 2 {
		t.Fatalf("got %d fields, want 2", len(m.Fields))
	}

	// Fields exist in Columns but FieldMap should be empty (no json tags)
	if len(m.FieldMapEntries()) != 0 {
		t.Errorf("FieldMap should be empty for entity with no json tags, got %d entries", len(m.FieldMapEntries()))
	}

	// InsertableColumns should still work
	if len(m.InsertableColumns()) != 2 {
		t.Errorf("InsertableColumns should have 2 entries, got %d", len(m.InsertableColumns()))
	}
}

func TestTenantFieldsAndRelations(t *testing.T) {
	dir := t.TempDir()

	content := `package testpkg

import "github.com/uptrace/bun"

type TenantEntity struct {
	bun.BaseModel ` + "`" + `bun:"table:tenant_entities,alias:te" json:"-"` + "`" + `

	ID             string ` + "`" + `json:"id"             bun:"id,pk"` + "`" + `
	OrganizationID string ` + "`" + `json:"organizationId" bun:"organization_id,pk"` + "`" + `
	BusinessUnitID string ` + "`" + `json:"businessUnitId" bun:"business_unit_id,pk"` + "`" + `
	Name           string ` + "`" + `json:"name"           bun:"name"` + "`" + `

	Org  *TenantEntity ` + "`" + `bun:"rel:belongs-to,join:organization_id=id"` + "`" + `
	BU   *TenantEntity ` + "`" + `bun:"rel:belongs-to,join:business_unit_id=id"` + "`" + `
	Items []*TenantEntity ` + "`" + `bun:"rel:has-many,join:id=parent_id"` + "`" + `
}
`
	if err := os.WriteFile(filepath.Join(dir, "entity.go"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

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

	filterable := m.FilterableFields()
	if len(filterable) != 4 {
		t.Errorf("got %d filterable fields, want 4 (ID, OrganizationID, BusinessUnitID, Name)", len(filterable))
	}
}
