package captureprofile

// SeparatorStrategy is one way a stack of paper says where a document ends.
// A profile may combine them; any one of them firing splits the stack.
type SeparatorStrategy string

const (
	// SeparatorPatchCode splits on a patch-code sheet the scanner detected.
	SeparatorPatchCode = SeparatorStrategy("PatchCode")
	// SeparatorCoverSheet splits on a Trenova cover sheet and routes what
	// follows it to the record the sheet names.
	SeparatorCoverSheet = SeparatorStrategy("CoverSheet")
	// SeparatorBlankPage splits on a page with no content on it.
	SeparatorBlankPage = SeparatorStrategy("BlankPage")
	// SeparatorFixedPageCount splits every N pages.
	SeparatorFixedPageCount = SeparatorStrategy("FixedPageCount")
)

func (s SeparatorStrategy) IsValid() bool {
	switch s {
	case SeparatorPatchCode, SeparatorCoverSheet, SeparatorBlankPage, SeparatorFixedPageCount:
		return true
	default:
		return false
	}
}

func AllSeparatorStrategies() []SeparatorStrategy {
	return []SeparatorStrategy{
		SeparatorPatchCode,
		SeparatorCoverSheet,
		SeparatorBlankPage,
		SeparatorFixedPageCount,
	}
}

// ProfileStatus is whether a profile is offered to people who scan.
type ProfileStatus string

const (
	ProfileActive   = ProfileStatus("Active")
	ProfileInactive = ProfileStatus("Inactive")
)

func (s ProfileStatus) IsValid() bool {
	switch s {
	case ProfileActive, ProfileInactive:
		return true
	default:
		return false
	}
}

func AllProfileStatuses() []ProfileStatus {
	return []ProfileStatus{ProfileActive, ProfileInactive}
}
