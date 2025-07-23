// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package reportmetadata

type QueryType string

const (
	QueryTypeSQL         = QueryType("SQL")
	QueryTypeAggregation = QueryType("Aggregation")
	QueryTypeAPI         = QueryType("API")
)

// VisualizationType defines how the report will be presented.
type VisualizationType string

const (
	// VisualizationTable is a table of data.
	VisualizationTable = VisualizationType("Table")

	// VisualizationChart is a chart of data.
	VisualizationChart = VisualizationType("Chart")

	// VisualizationCustom is a custom visualization of data.
	VisualizationCustom = VisualizationType("Custom")
)
