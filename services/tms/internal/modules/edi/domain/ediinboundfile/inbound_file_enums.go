package ediinboundfile

type InboundFileStatus string

const (
	InboundFileStatusReceived           = InboundFileStatus("Received")
	InboundFileStatusParsed             = InboundFileStatus("Parsed")
	InboundFileStatusProcessed          = InboundFileStatus("Processed")
	InboundFileStatusPartiallyProcessed = InboundFileStatus("PartiallyProcessed")
	InboundFileStatusQuarantined        = InboundFileStatus("Quarantined")
	InboundFileStatusDuplicate          = InboundFileStatus("Duplicate")
)

func (s InboundFileStatus) IsReprocessable() bool {
	return s == InboundFileStatusQuarantined || s == InboundFileStatusPartiallyProcessed
}

func (s InboundFileStatus) IsValid() bool {
	switch s {
	case InboundFileStatusReceived,
		InboundFileStatusParsed,
		InboundFileStatusProcessed,
		InboundFileStatusPartiallyProcessed,
		InboundFileStatusQuarantined,
		InboundFileStatusDuplicate:
		return true
	default:
		return false
	}
}
