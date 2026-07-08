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
	Type *worker.WorkerType `+"`json:\"type,omitempty\"`"+`
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
	Status         string   `+"`json:\"status\"`"+`
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
	Type           WorkerType `+"`json:\"type\"`"+`
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

	helpersOutput, err := os.ReadFile(filepath.Join(outputDir, "helpers_gen.go"))
	if err != nil {
		t.Fatalf("reading helpers output: %v", err)
	}
	helpersGenerated := string(helpersOutput)
	mustContain(t, helpersGenerated, "// Code generated by resolver/mappergen; DO NOT EDIT.")
	mustContain(t, helpersGenerated, "func optionalID(")

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

	workerOutput, err := os.ReadFile(filepath.Join(outputDir, "worker_mapping_gen.go"))
	if err != nil {
		t.Fatalf("reading worker output: %v", err)
	}
	workerGenerated := string(workerOutput)
	mustContain(t, workerGenerated, "func ApplyWorkerPatch(")
	mustContain(t, workerGenerated, "entity.Type = *input.Type")

	_, err = os.Stat(filepath.Join(outputDir, "organization_mapping_gen.go"))
	if !os.IsNotExist(err) {
		t.Fatalf("expected organization mapper to be skipped, stat error = %v", err)
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
