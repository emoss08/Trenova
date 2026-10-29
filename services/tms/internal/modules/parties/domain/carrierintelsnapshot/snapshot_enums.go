package carrierintelsnapshot

type SnapshotSource string

const (
	SnapshotSourcePrimary         = SnapshotSource("Primary")
	SnapshotSourceFallback        = SnapshotSource("Fallback")
	SnapshotSourceChangeFeedPatch = SnapshotSource("ChangeFeedPatch")
)

func (s SnapshotSource) String() string { return string(s) }

func (s SnapshotSource) IsValid() bool {
	switch s {
	case SnapshotSourcePrimary, SnapshotSourceFallback, SnapshotSourceChangeFeedPatch:
		return true
	default:
		return false
	}
}

type ReviewState string

const (
	ReviewStateNone        = ReviewState("None")
	ReviewStateNeedsReview = ReviewState("NeedsReview")
	ReviewStateReviewed    = ReviewState("Reviewed")
)

func (s ReviewState) String() string { return string(s) }

func (s ReviewState) IsValid() bool {
	switch s {
	case ReviewStateNone, ReviewStateNeedsReview, ReviewStateReviewed:
		return true
	default:
		return false
	}
}
