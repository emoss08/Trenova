package shipmentevent

const (
	MetadataStopNumber       = "stopNumber"
	MetadataStopSequence     = "stopSequence"
	MetadataStopType         = "stopType"
	MetadataLocationName     = "locationName"
	MetadataActualAt         = "actualAt"
	MetadataPreviousActualAt = "previousActualAt"
	MetadataFromStopActual   = "fromStopActual"
)

func (e *Event) FromStopActual() bool {
	if e == nil || e.Metadata == nil {
		return false
	}
	from, ok := e.Metadata[MetadataFromStopActual].(bool)
	return ok && from
}
