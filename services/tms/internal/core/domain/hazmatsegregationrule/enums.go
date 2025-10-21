package hazmatsegregationrule

type SegregationType string

const (
	SegregationTypeProhibited = SegregationType("Prohibited")
	SegregationTypeSeparated  = SegregationType("Separated")
	SegregationTypeDistance   = SegregationType("Distance")
	SegregationTypeBarrier    = SegregationType("Barrier")
)
