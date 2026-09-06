package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGeneratesSafeGlobalMappers(t *testing.T) {
	root := t.TempDir()
	resolverDir := filepath.Join(root, "internal", "api", "graphql", "resolver")

	writeTestFile(t, root, "go.mod", "module example.com/app\n\ngo 1.26\n")
	writeTestFile(t, root, "gqlgen.yml", `models:
  EquipmentManufacturer:
    model:
      - example.com/app/internal/core/domain/equipmentmanufacturer.EquipmentManufacturer
  Tractor:
    model:
      - example.com/app/internal/core/domain/tractor.Tractor
  EquipmentType:
    model:
      - example.com/app/internal/core/domain/equipmenttype.EquipmentType
  FleetCode:
    model:
      - example.com/app/internal/core/domain/fleetcode.FleetCode
  Organization:
    model:
      - example.com/app/internal/core/domain/tenant.Organization
  Worker:
    model:
      - example.com/app/internal/core/domain/worker.Worker
`)
	writeTestFile(t, root, "internal/api/graphql/gqlmodel/models_gen.go", `package gqlmodel

import (
	"github.com/99designs/gqlgen/graphql"
	"example.com/app/internal/core/domain/worker"
)

type EquipmentManufacturerInput struct {
	Status      *string `+"`json:\"status,omitempty\"`"+`
	Name        string `+"`json:\"name\"`"+`
	Description string `+"`json:\"description\"`"+`
	Version     int    `+"`json:\"version\"`"+`
	CreatedAt   int    `+"`json:\"createdAt\"`"+`
}

type EquipmentManufacturerPatchInput struct {
	Status      *string `+"`json:\"status,omitempty\"`"+`
	Name        *string `+"`json:\"name,omitempty\"`"+`
	Description *string `+"`json:\"description,omitempty\"`"+`
	Version     *int    `+"`json:\"version,omitempty\"`"+`
}

type TractorInput struct {
	PrimaryWorkerID string `+"`json:\"primaryWorkerId\"`"+`
	Status          *string `+"`json:\"status,omitempty\"`"+`
}

type TractorPatchInput struct {
	PrimaryWorkerID *string `+"`json:\"primaryWorkerId,omitempty\"`"+`
	Code            graphql.Omittable[*string] `+"`json:\"code,omitempty\"`"+`
	Model           graphql.Omittable[*string] `+"`json:\"model,omitempty\"`"+`
	Vin             graphql.Omittable[*string] `+"`json:\"vin,omitempty\"`"+`
	StateID         graphql.Omittable[*string] `+"`json:\"stateId,omitempty\"`"+`
	Year            graphql.Omittable[*int] `+"`json:\"year,omitempty\"`"+`
	CustomFields    graphql.Omittable[map[string]any] `+"`json:\"customFields,omitempty\"`"+`
}

type EquipmentTypeInput struct {
	Code string `+"`json:\"code\"`"+`
}

type FleetCodePatchInput struct {
	Description graphql.Omittable[*string] `+"`json:\"description,omitempty\"`"+`
}

type OrganizationInput struct {
	Name string `+"`json:\"name\"`"+`
}

type WorkerPatchInput struct {
	Type graphql.Omittable[*worker.WorkerType] `+"`json:\"type,omitempty\"`"+`
}
`)
	writeTestFile(t, root, "internal/core/domain/equipmentmanufacturer/equipmentmanufacturer.go", `package equipmentmanufacturer

import "example.com/app/shared/pulid"

type EquipmentManufacturer struct {
	ID             pulid.ID `+"`json:\"id\"`"+`
	OrganizationID pulid.ID `+"`json:\"organizationId\"`"+`
	BusinessUnitID pulid.ID `+"`json:\"businessUnitId\"`"+`
	Status         string   `+"`json:\"status\"`"+`
	Name           string   `+"`json:\"name\"`"+`
	Description    string   `+"`json:\"description\"`"+`
	Version        int64    `+"`json:\"version\"`"+`
	CreatedAt      int64    `+"`json:\"createdAt\"`"+`
}
`)
	writeTestFile(t, root, "internal/core/domain/tractor/tractor.go", `package tractor

import "example.com/app/shared/pulid"

type Tractor struct {
	ID             pulid.ID `+"`json:\"id\"`"+`
	OrganizationID pulid.ID `+"`json:\"organizationId\"`"+`
	BusinessUnitID pulid.ID `+"`json:\"businessUnitId\"`"+`
	PrimaryWorkerID pulid.ID `+"`json:\"primaryWorkerId\"`"+`
	StateID        pulid.ID `+"`json:\"stateId\" bun:\"state_id,type:VARCHAR(100),nullzero\"`"+`
	Status         string   `+"`json:\"status\"`"+`
	Code           string   `+"`json:\"code\" bun:\"code,type:VARCHAR(50),notnull\"`"+`
	Model          string   `+"`json:\"model\" bun:\"model,type:VARCHAR(50),notnull\"`"+`
	Vin            string   `+"`json:\"vin\" bun:\"vin,type:VARCHAR(50),nullzero\"`"+`
	Year           *int     `+"`json:\"year\" bun:\"year,type:INT,nullzero\"`"+`
	CustomFields   map[string]any `+"`json:\"customFields,omitempty\" bun:\"-\"`"+`
}
`)
	writeTestFile(t, root, "internal/core/domain/equipmenttype/equipmenttype.go", `package equipmenttype

import "example.com/app/shared/pulid"

type EquipmentType struct {
	ID             pulid.ID `+"`json:\"id\"`"+`
	OrganizationID pulid.ID `+"`json:\"organizationId\"`"+`
	BusinessUnitID pulid.ID `+"`json:\"businessUnitId\"`"+`
	Code           string   `+"`json:\"code\"`"+`
}
`)
	writeTestFile(t, root, "internal/core/domain/fleetcode/fleetcode.go", `package fleetcode

import "example.com/app/shared/pulid"

type FleetCode struct {
	ID             pulid.ID `+"`json:\"id\"`"+`
	OrganizationID pulid.ID `+"`json:\"organizationId\"`"+`
	BusinessUnitID pulid.ID `+"`json:\"businessUnitId\"`"+`
	Description    string   `+"`json:\"description\"`"+`
}
`)
	writeTestFile(t, root, "internal/core/domain/tenant/organization.go", `package tenant

import "example.com/app/shared/pulid"

type Organization struct {
	ID             pulid.ID `+"`json:\"id\"`"+`
	BusinessUnitID pulid.ID `+"`json:\"businessUnitId\"`"+`
	Name           string   `+"`json:\"name\"`"+`
}
`)
	writeTestFile(t, root, "internal/core/domain/worker/worker.go", `package worker

import "example.com/app/shared/pulid"

type WorkerType string

type Worker struct {
	ID             pulid.ID  `+"`json:\"id\"`"+`
	OrganizationID pulid.ID  `+"`json:\"organizationId\"`"+`
	BusinessUnitID pulid.ID  `+"`json:\"businessUnitId\"`"+`
	Type           WorkerType `+"`json:\"type\" bun:\"type,type:worker_type_enum,notnull\"`"+`
}
`)
	writeTestFile(t, root, "internal/api/graphql/resolver/existing.go", `package resolver

func equipmentTypeFromInput() {}
`)
	writeTestFile(t, root, "internal/api/graphql/resolver/mappers.yml", `types:
  EquipmentManufacturer:
    imports:
      statusvalue: example.com/app/pkg/statusvalue
    defaults:
      status: statusvalue.Active
  Tractor:
    imports:
      statusvalue: example.com/app/pkg/statusvalue
    defaults:
      status: statusvalue.Available
    required:
      - vin
    clearable:
      - model
`)

	outputDir := filepath.Join(resolverDir, "mappers")
	err := run(&generatorOptions{
		ManifestPath: filepath.Join(resolverDir, "mappers.yml"),
		OutputDir:    outputDir,
		GqlgenPath:   filepath.Join(root, "gqlgen.yml"),
		ModelPath:    filepath.Join(root, "internal", "api", "graphql", "gqlmodel", "models_gen.go"),
		DomainDir:    filepath.Join(root, "internal", "core", "domain"),
		ResolverDir:  resolverDir,
		GoModPath:    filepath.Join(root, "go.mod"),
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	equipmentOutput, err := os.ReadFile(filepath.Join(outputDir, "equipment_manufacturer_mapping_gen.go"))
	if err != nil {
		t.Fatalf("reading equipment manufacturer output: %v", err)
	}
	equipmentGenerated := string(equipmentOutput)
	mustContain(t, equipmentGenerated, "// Code generated by resolver/mappergen; DO NOT EDIT.")
	mustContain(t, equipmentGenerated, "package mappers")
	mustContain(t, equipmentGenerated, "func EquipmentManufacturerFromInput(")
	mustContain(t, equipmentGenerated, "func ApplyEquipmentManufacturerPatch(")
	mustContain(t, equipmentGenerated, `"example.com/app/pkg/statusvalue"`)
	mustContain(t, equipmentGenerated, "status := statusvalue.Active")
	mustContain(t, equipmentGenerated, "if input.Status != nil {")
	mustContain(t, equipmentGenerated, "status = *input.Status")
	mustContain(t, equipmentGenerated, "Status:         status,")
	mustContain(t, equipmentGenerated, "Version:        int64(input.Version),")

	tractorOutput, err := os.ReadFile(filepath.Join(outputDir, "tractor_mapping_gen.go"))
	if err != nil {
		t.Fatalf("reading tractor output: %v", err)
	}
	tractorGenerated := string(tractorOutput)
	mustContain(t, tractorGenerated, "func TractorFromInput(")
	mustContain(t, tractorGenerated, "primaryWorkerID, err := pulid.MustParse(input.PrimaryWorkerID)")
	mustContain(t, tractorGenerated, "status := statusvalue.Available")
	mustContain(t, tractorGenerated, "primaryWorkerID, err := pulid.MustParse(*input.PrimaryWorkerID)")
	mustNotContain(t, tractorGenerated, "optionalID(input.PrimaryWorkerID)")
	mustContain(t, tractorGenerated, `"example.com/app/pkg/errortypes"`)
	mustContain(t, tractorGenerated, "if codeValue, ok := input.Code.ValueOK(); ok {")
	mustContain(t, tractorGenerated, "if codeValue == nil {")
	mustContain(t, tractorGenerated, `"code",`)
	mustContain(t, tractorGenerated, "errortypes.ErrRequired,")
	mustContain(t, tractorGenerated, `"Code cannot be cleared",`)
	mustContain(t, tractorGenerated, "entity.Code = *codeValue")
	mustContain(t, tractorGenerated, "entity.Model = StringValue(modelValue)")
	mustNotContain(t, tractorGenerated, "Model cannot be cleared")
	mustContain(t, tractorGenerated, "if vinValue == nil {")
	mustContain(t, tractorGenerated, `"Vin cannot be cleared",`)
	mustContain(t, tractorGenerated, "entity.Vin = *vinValue")
	mustContain(t, tractorGenerated, "stateID, err := optionalID(stateIDValue)")
	mustNotContain(t, tractorGenerated, "State cannot be cleared")
	mustContain(t, tractorGenerated, "entity.Year = yearValue")
	mustContain(t, tractorGenerated, "entity.CustomFields = mapValue(customFieldsValue)")

	helpersOutput, err := os.ReadFile(filepath.Join(outputDir, "helpers_gen.go"))
	if err != nil {
		t.Fatalf("reading helpers output: %v", err)
	}
	helpersGenerated := string(helpersOutput)
	mustContain(t, helpersGenerated, "// Code generated by resolver/mappergen; DO NOT EDIT.")
	mustContain(t, helpersGenerated, "func optionalID(")
	mustContain(t, helpersGenerated, "func mapValue[K comparable, V any](value map[K]V) map[K]V {")

	_, err = os.Stat(filepath.Join(outputDir, "equipment_type_mapping_gen.go"))
	if !os.IsNotExist(err) {
		t.Fatalf("expected equipment type mapper to be skipped, stat error = %v", err)
	}
	fleetCodeOutput, err := os.ReadFile(filepath.Join(outputDir, "fleet_code_mapping_gen.go"))
	if err != nil {
		t.Fatalf("reading fleet code output: %v", err)
	}
	fleetCodeGenerated := string(fleetCodeOutput)
	mustContain(t, fleetCodeGenerated, "func ApplyFleetCodePatch(")
	mustContain(t, fleetCodeGenerated, "entity.Description = StringValue(descriptionValue)")
	mustNotContain(t, fleetCodeGenerated, "errortypes")

	workerOutput, err := os.ReadFile(filepath.Join(outputDir, "worker_mapping_gen.go"))
	if err != nil {
		t.Fatalf("reading worker output: %v", err)
	}
	workerGenerated := string(workerOutput)
	mustContain(t, workerGenerated, "func ApplyWorkerPatch(")
	mustContain(t, workerGenerated, "if typeValue, ok := input.Type.ValueOK(); ok {")
	mustContain(t, workerGenerated, "if typeValue == nil {")
	mustContain(t, workerGenerated, `"type",`)
	mustContain(t, workerGenerated, `"Type cannot be cleared",`)
	mustContain(t, workerGenerated, "entity.Type = *typeValue")
	mustContain(t, workerGenerated, `"example.com/app/pkg/errortypes"`)

	_, err = os.Stat(filepath.Join(outputDir, "organization_mapping_gen.go"))
	if !os.IsNotExist(err) {
		t.Fatalf("expected organization mapper to be skipped, stat error = %v", err)
	}
}

func TestHumanizeFieldName(t *testing.T) {
	cases := map[string]string{
		"Code":                    "Code",
		"DriverType":              "Driver Type",
		"PrimaryWorkerID":         "Primary Worker",
		"EquipmentManufacturerID": "Equipment Manufacturer",
		"ID":                      "ID",
		"Vin":                     "Vin",
	}
	for input, expected := range cases {
		if actual := humanizeFieldName(input); actual != expected {
			t.Fatalf("humanizeFieldName(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func writeTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()

	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("creating directory for %s: %v", relPath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", relPath, err)
	}
}

func mustContain(t *testing.T, value, expected string) {
	t.Helper()

	if !strings.Contains(value, expected) {
		t.Fatalf("expected output to contain %q\n%s", expected, value)
	}
}

func mustNotContain(t *testing.T, value, unexpected string) {
	t.Helper()

	if strings.Contains(value, unexpected) {
		t.Fatalf("expected output not to contain %q\n%s", unexpected, value)
	}
}
