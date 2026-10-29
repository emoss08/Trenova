package graphqllisttool

// PageRecord is a list's outcome as one object, whichever page carries it:
// a page read beside other records, or one that pages by offset.
func PageRecord(data any) (map[string]any, bool) {
	switch outcome := data.(type) {
	case map[string]any:
		return outcome, true
	case *BesidePage:
		record, err := outcome.record()
		return record, err == nil
	case *OffsetPage:
		return outcome.Record, true
	}

	return nil, false
}
