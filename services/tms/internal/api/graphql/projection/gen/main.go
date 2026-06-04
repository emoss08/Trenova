package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/emoss08/trenova/pkg/buncolgen"
)

var (
	manifestPath = flag.String("manifest", "", "Path to projection manifest")
	schemaDir    = flag.String("schema", "", "Path to GraphQL schema directory")
	outputPath   = flag.String("output", "", "Path to generated Go file")
)

func main() {
	flag.Parse()

	if *manifestPath == "" || *schemaDir == "" || *outputPath == "" {
		fmt.Fprintln(
			os.Stderr,
			"Usage: projectiongen -manifest=<file> -schema=<dir> -output=<file>",
		)
		os.Exit(1)
	}

	if err := run(generatorOptions{
		ManifestPath: *manifestPath,
		SchemaDir:    *schemaDir,
		OutputPath:   *outputPath,
		FieldMaps:    realFieldMaps(),
	}); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating GraphQL projections: %v\n", err)
		os.Exit(1)
	}
}

func realFieldMaps() map[string]map[string]string {
	return map[string]map[string]string{
		"BusinessUnit":          buncolgen.BusinessUnitFieldMap,
		"EquipmentContinuity":   buncolgen.EquipmentContinuityFieldMap,
		"EquipmentManufacturer": buncolgen.EquipmentManufacturerFieldMap,
		"EquipmentType":         buncolgen.EquipmentTypeFieldMap,
		"FleetCode":             buncolgen.FleetCodeFieldMap,
		"Location":              buncolgen.LocationFieldMap,
		"LocationCategory":      buncolgen.LocationCategoryFieldMap,
		"Organization":          buncolgen.OrganizationFieldMap,
		"Tractor":               buncolgen.TractorFieldMap,
		"Trailer":               buncolgen.TrailerFieldMap,
		"User":                  buncolgen.UserFieldMap,
		"UsState":               buncolgen.UsStateFieldMap,
		"Worker":                buncolgen.WorkerFieldMap,
	}
}
