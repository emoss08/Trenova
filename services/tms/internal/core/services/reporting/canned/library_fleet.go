package canned

import (
	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/pkg/reportcatalog"
)

func driverProductivity() *Entry {
	return &Entry{
		Key:     "driver-activity",
		Version: initialVersion,
		Name:    "Driver Productivity",
		Description: "Moves, loaded and empty miles, and last activity per driver — the " +
			"asset-based view of who is turning miles",
		Category:      categoryOperations,
		Tags:          []string{tagDrivers, tagMiles, "productivity", tagAssets},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipmentMove,
			Columns: []report.ColumnSpec{
				dimCol("first_name", "First Name", firstNameField,
					assignmentEdge, primaryWorkerEdge),
				dimCol("last_name", "Last Name", lastNameField,
					assignmentEdge, primaryWorkerEdge),
				dimCol("driver_type", "Driver Type", "driverType",
					assignmentEdge, primaryWorkerEdge),
				dimCol("fleet_code", "Fleet", codeField,
					assignmentEdge, primaryWorkerEdge, fleetCodeEdge),
				countMeasure(moveCountColumnID, "Moves"),
				sumMiles(totalMilesColumnID, totalMilesLabel),
				sumMiles(loadedMilesColumnID, milesLabel),
				avgMovesMiles(avgMilesColumnID),
				timestampMeasure("last_move", "Last Move",
					reportcatalog.AggMax, createdAtFieldKey),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Pivot:      loadedPivot(loadedMilesColumnID),
			Sort:       []report.SortSpec{desc(totalMilesColumnID)},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func tractorUtilization() *Entry {
	return &Entry{
		Key:     "tractor-utilization",
		Version: initialVersion,
		Name:    "Tractor Utilization",
		Description: "Miles, moves, loaded versus empty split, and last activity per " +
			"power unit — spot idle and under-worked tractors",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, tagUtilizatn, tagAssets, tagMiles},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipmentMove,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Tractor Unit", codeField, assignmentEdge, tractorEdge),
				dimCol("fleet_code", "Fleet", codeField,
					assignmentEdge, tractorEdge, fleetCodeEdge),
				dimCol("equipment_type", "Equipment Type", codeField,
					assignmentEdge, tractorEdge, equipmentTypeEdge),
				countMeasure(moveCountColumnID, "Moves"),
				sumMiles(totalMilesColumnID, totalMilesLabel),
				sumMiles(loadedMilesColumnID, milesLabel),
				avgMovesMiles(avgMilesColumnID),
				distinctMeasure("driver_count", "Drivers Used", "primaryWorkerId",
					assignmentEdge),
				timestampMeasure("last_move", "Last Move",
					reportcatalog.AggMax, createdAtFieldKey),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Pivot:      loadedPivot(loadedMilesColumnID),
			Sort:       []report.SortSpec{desc(totalMilesColumnID)},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func trailerUtilization() *Entry {
	return &Entry{
		Key:     "trailer-utilization",
		Version: initialVersion,
		Name:    "Trailer Utilization",
		Description: "Turns and miles per trailer with last activity — find trailers " +
			"sitting idle or over-cycled",
		Category:      categoryFleet,
		Tags:          []string{tagTrailers, tagUtilizatn, tagAssets},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipmentMove,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Trailer Unit", codeField, assignmentEdge, trailerEdge),
				dimCol("fleet_code", "Fleet", codeField,
					assignmentEdge, trailerEdge, fleetCodeEdge),
				dimCol("equipment_type", "Equipment Type", codeField,
					assignmentEdge, trailerEdge, equipmentTypeEdge),
				countMeasure(moveCountColumnID, "Moves"),
				sumMiles(totalMilesColumnID, totalMilesLabel),
				avgMovesMiles(avgMilesColumnID),
				timestampMeasure("first_move", "First Move",
					reportcatalog.AggMin, createdAtFieldKey),
				timestampMeasure("last_move", "Last Move",
					reportcatalog.AggMax, createdAtFieldKey),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{desc(totalMilesColumnID)},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func deadheadAnalysis() *Entry {
	return &Entry{
		Key:     "deadhead-analysis",
		Version: initialVersion,
		Name:    "Loaded vs Empty Miles by Fleet",
		Description: "Deadhead exposure by fleet and tractor — loaded miles against " +
			"empty miles for the selected window",
		Category:      categoryFleet,
		Tags:          []string{"deadhead", tagMiles, tagAssets, tagUtilizatn},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityShipmentMove,
			Columns: []report.ColumnSpec{
				dimCol("fleet_code", "Fleet", codeField,
					assignmentEdge, tractorEdge, fleetCodeEdge),
				dimCol("unit", "Tractor Unit", codeField, assignmentEdge, tractorEdge),
				countMeasure(moveCountColumnID, "Moves"),
				sumMiles(totalMilesColumnID, totalMilesLabel),
				sumMiles(loadedMilesColumnID, milesLabel),
				avgMovesMiles(avgMilesColumnID),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Pivot:      loadedPivot(loadedMilesColumnID),
			Sort:       []report.SortSpec{desc(totalMilesColumnID)},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func equipmentUtilization() *Entry {
	return &Entry{
		Key:     "equipment-utilization",
		Version: initialVersion,
		Name:    "Equipment Assignment History",
		Description: "Assignment counts per tractor with the drivers and trailers it ran " +
			"with, plus first and last assignment dates",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, tagUtilizatn, "assignments"},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityAssignment,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Tractor Unit", codeField, tractorEdge),
				dimCol("fleet_code", "Fleet", codeField, tractorEdge, fleetCodeEdge),
				dimCol("equipment_type", "Equipment Type", codeField,
					tractorEdge, equipmentTypeEdge),
				countMeasure("assignment_count", "Assignments"),
				distinctMeasure("driver_count", "Drivers Used", "primaryWorkerId"),
				distinctMeasure("trailer_count", "Trailers Pulled", "trailerId"),
				timestampMeasure("first_assignment", "First Assignment",
					reportcatalog.AggMin, createdAtFieldKey),
				timestampMeasure("last_assignment", "Last Assignment",
					reportcatalog.AggMax, createdAtFieldKey),
			},
			Filters:    andFilters(windowFilter(createdAtFieldKey), excludeCanceled()),
			Sort:       []report.SortSpec{desc("assignment_count")},
			Parameters: []report.ParameterDef{windowParam(30)},
		},
	}
}

func fleetRoster() *Entry {
	return &Entry{
		Key:     "fleet-roster",
		Version: initialVersion,
		Name:    "Fleet Roster — Tractors",
		Description: "Full power-unit roster: identity, specification, registration, " +
			"fleet assignment, and the drivers currently attached",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, tagFleet, tagRoster, tagAssets},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityTractor,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Unit Number", codeField),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("year", "Year", "year"),
				dimCol("make", "Make", "make"),
				dimCol("model", "Model", "model"),
				styledDim("vin", "VIN", "vin", emptyDash()),
				styledDim("license_plate", "License Plate", "licensePlateNumber", emptyDash()),
				styledDim(
					"registration_number",
					"Registration Number",
					"registrationNumber",
					emptyDash(),
				),
				dateDim("registration_expiry", "Registration Expiry", "registrationExpiry"),
				dimCol("equipment_type", "Equipment Type", codeField, equipmentTypeEdge),
				dimCol("manufacturer", "Manufacturer", nameField, "equipmentManufacturer"),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
				dimCol("primary_driver_first", "Primary Driver First Name",
					firstNameField, primaryWorkerEdge),
				dimCol("primary_driver_last", "Primary Driver Last Name",
					lastNameField, primaryWorkerEdge),
				dimCol("secondary_driver_last", "Co-Driver Last Name",
					lastNameField, secondaryWorkerKey),
			},
			Filters: andFilters(inParam(statusFieldKey, statusParam)),
			Sort:    []report.SortSpec{asc("unit")},
			Parameters: []report.ParameterDef{
				equipmentStatusParam(),
			},
		},
	}
}

func trailerRoster() *Entry {
	return &Entry{
		Key:     "trailer-roster",
		Version: initialVersion,
		Name:    "Fleet Roster — Trailers",
		Description: "Trailer roster with specification, capacity, registration, and " +
			"last inspection date",
		Category:      categoryFleet,
		Tags:          []string{tagTrailers, tagFleet, tagRoster, tagAssets},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityTrailer,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Unit Number", codeField),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("year", "Year", "year"),
				dimCol("make", "Make", "make"),
				dimCol("model", "Model", "model"),
				styledDim("vin", "VIN", "vin", emptyDash()),
				styledDim("license_plate", "License Plate", "licensePlateNumber", emptyDash()),
				dimCol("equipment_type", "Equipment Type", codeField, equipmentTypeEdge),
				dimCol("manufacturer", "Manufacturer", nameField, "equipmentManufacturer"),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
				dimCol("max_load_weight", "Max Load Weight", "maxLoadWeight"),
				dateDim("last_inspection", "Last Inspection", "lastInspectionDate"),
				dateDim("registration_expiry", "Registration Expiry", "registrationExpiry"),
			},
			Filters: andFilters(inParam(statusFieldKey, statusParam)),
			Sort:    []report.SortSpec{asc("unit")},
			Parameters: []report.ParameterDef{
				equipmentStatusParam(),
			},
		},
	}
}

func expiringTractorRegistrations() *Entry {
	return &Entry{
		Key:     "expiring-tractor-registrations",
		Version: initialVersion,
		Name:    "Expiring Tractor Registrations",
		Description: "Power units whose plates expire inside the horizon, with the " +
			"drivers and fleet affected",
		Category:      categoryFleet,
		Tags:          []string{tagTractors, "registrations", tagCompliance},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityTractor,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Unit Number", codeField),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("year", "Year", "year"),
				dimCol("make", "Make", "make"),
				dimCol("model", "Model", "model"),
				styledDim("license_plate", "License Plate", "licensePlateNumber", emptyDash()),
				styledDim(
					"registration_number",
					"Registration Number",
					"registrationNumber",
					emptyDash(),
				),
				dateDim("registration_expiry", "Registration Expiry", "registrationExpiry"),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
				dimCol("primary_driver_first", "Primary Driver First Name",
					firstNameField, primaryWorkerEdge),
				dimCol("primary_driver_last", "Primary Driver Last Name",
					lastNameField, primaryWorkerEdge),
			},
			Filters:    andFilters(horizonFilter("registrationExpiry")),
			Sort:       []report.SortSpec{asc("registration_expiry")},
			Parameters: []report.ParameterDef{horizonParam(60)},
		},
	}
}

func expiringTrailerRegistrations() *Entry {
	return &Entry{
		Key:     "expiring-trailer-registrations",
		Version: initialVersion,
		Name:    "Expiring Trailer Registrations",
		Description: "Trailers whose plates expire inside the horizon, with inspection " +
			"status and fleet assignment",
		Category:      categoryFleet,
		Tags:          []string{tagTrailers, "registrations", tagCompliance},
		DefaultFormat: report.FormatXLSX,
		Definition: &report.Definition{
			IRVersion: report.CurrentIRVersion,
			Entity:    entityTrailer,
			Columns: []report.ColumnSpec{
				dimCol("unit", "Unit Number", codeField),
				dimCol(statusFieldKey, "Status", statusFieldKey),
				dimCol("year", "Year", "year"),
				dimCol("make", "Make", "make"),
				dimCol("model", "Model", "model"),
				styledDim("license_plate", "License Plate", "licensePlateNumber", emptyDash()),
				dateDim("registration_expiry", "Registration Expiry", "registrationExpiry"),
				dateDim("last_inspection", "Last Inspection", "lastInspectionDate"),
				dimCol("equipment_type", "Equipment Type", codeField, equipmentTypeEdge),
				dimCol("fleet_code", "Fleet", codeField, fleetCodeEdge),
			},
			Filters:    andFilters(horizonFilter("registrationExpiry")),
			Sort:       []report.SortSpec{asc("registration_expiry")},
			Parameters: []report.ParameterDef{horizonParam(60)},
		},
	}
}
