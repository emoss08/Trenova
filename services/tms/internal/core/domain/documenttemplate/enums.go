package documenttemplate

type PageSize string

const (
	PageSizeLetter = PageSize("Letter") // 8.5 x 11 inches (US standard)
	PageSizeA4     = PageSize("A4")     // 210 x 297 mm (international standard)
	PageSizeLegal  = PageSize("Legal")  // 8.5 x 14 inches
)

func (ps PageSize) String() string {
	return string(ps)
}

func (ps PageSize) Dimensions() (width, height float64) {
	switch ps {
	case PageSizeLetter:
		return 215.9, 279.4 // 8.5 x 11 inches in mm
	case PageSizeA4:
		return 210, 297
	case PageSizeLegal:
		return 215.9, 355.6 // 8.5 x 14 inches in mm
	default:
		return 215.9, 279.4 // Default to Letter
	}
}

type Orientation string

const (
	OrientationPortrait  Orientation = "Portrait"
	OrientationLandscape Orientation = "Landscape"
)

func (o Orientation) String() string {
	return string(o)
}

type TemplateStatus string

const (
	TemplateStatusDraft    = TemplateStatus("Draft")    // Template is being edited
	TemplateStatusActive   = TemplateStatus("Active")   // Template is available for use
	TemplateStatusArchived = TemplateStatus("Archived") // Template is no longer in use
)

func (ts TemplateStatus) String() string {
	return string(ts)
}

type GenerationStatus string

const (
	GenerationStatusPending    = GenerationStatus("Pending")    // Queued for generation
	GenerationStatusProcessing = GenerationStatus("Processing") // Currently being generated
	GenerationStatusCompleted  = GenerationStatus("Completed")  // Successfully generated
	GenerationStatusFailed     = GenerationStatus("Failed")     // Generation failed
)

func (gs GenerationStatus) String() string {
	return string(gs)
}

type DeliveryMethod string

const (
	DeliveryMethodNone     = DeliveryMethod("None")     // Not delivered, just stored
	DeliveryMethodEmail    = DeliveryMethod("Email")    // Sent via email
	DeliveryMethodDownload = DeliveryMethod("Download") // Downloaded by user
	DeliveryMethodPortal   = DeliveryMethod("Portal")   // Available via customer portal
)

func (dm DeliveryMethod) String() string {
	return string(dm)
}
