package hazmatsegregationrule

// SegregationType defines the type of segregation required
type SegregationType string

const (
	// SegregationTypeProhibited indicates materials cannot be on the same vehicle/container
	SegregationTypeProhibited = SegregationType("Prohibited")
	// SegregationTypeSeparated indicates materials must be in different compartments
	SegregationTypeSeparated = SegregationType("Separated")
	// SegregationTypeDistance indicates materials must maintain minimum distance
	SegregationTypeDistance = SegregationType("Distance")
	// SegregationTypeBarrier indicates materials require protective barriers between them
	SegregationTypeBarrier = SegregationType("Barrier")
)
