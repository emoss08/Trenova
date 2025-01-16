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
