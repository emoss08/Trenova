package documentaiextractionports

type ExtractionAvailability string

const (
	ExtractionAvailable          ExtractionAvailability = "available"
	ExtractionOffOnServer        ExtractionAvailability = "off_on_server"
	ExtractionNotInPlan          ExtractionAvailability = "not_in_plan"
	ExtractionNoProviderAssigned ExtractionAvailability = "no_model_provider"
)

func (a ExtractionAvailability) Available() bool {
	return a == ExtractionAvailable
}
