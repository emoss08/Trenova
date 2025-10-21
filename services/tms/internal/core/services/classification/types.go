package classification

import (
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v2"
)

const (
	locationClassificationModel       = openai.ChatModelGPT5Nano
	locationClassificationServiceTier = "default"
)

var locationClassificationSystemPrompt = `Expert transportation logistics location classifier. Analyze location data to determine primary supply chain function.

Classification patterns:
- Corporate campus/HQ facilities → Regional Distribution Center (especially for manufacturers/pharma)
- "DC" or numbered facilities (DC01, WH02) → Distribution Center or Warehouse
- Retail store names with numbers → Customer Warehouse/Location
- "Terminal" or fleet operations → Company Terminal
- Travel centers/plaza → Truck Stop
- Port/rail facilities → Consider intermodal operations

Facility types (when applicable):
- StorageWarehouse: Standard storage operations
- CrossDock: Rapid transfer, minimal storage
- ColdStorage: Temperature-controlled (food/pharma)
- HazmatFacility: Dangerous goods handling
- IntermodalFacility: Multi-mode transport hub
`

type LocationClassificationRequest struct {
	TenantOpts  pagination.TenantOptions `json:"tenantOpts"`
	Name        string                   `json:"name"`
	Description *string                  `json:"description,omitempty"`
	Address     *string                  `json:"address,omitempty"`
	City        *string                  `json:"city,omitempty"`
	State       *string                  `json:"state,omitempty"`
	PostalCode  *string                  `json:"postalCode,omitempty"`
	Code        *string                  `json:"code,omitempty"`
	PlaceID     *string                  `json:"placeId,omitempty"`
	Latitude    *float64                 `json:"latitude,omitempty"`
	Longitude   *float64                 `json:"longitude,omitempty"`
}

type LocationAlternativeCategory struct {
	Category   string  `json:"category"   jsonschema_description:"The category name"`
	CategoryID string  `json:"categoryId" jsonschema_description:"The ID of the category"`
	Confidence float64 `json:"confidence" jsonschema_description:"Confidence score between 0.0 and 1.0"`
}

type LocationClassificationResponse struct {
	Category              string                        `json:"category"`
	CategoryID            string                        `json:"categoryId"`
	FacilityType          *string                       `json:"facilityType,omitempty"`
	Confidence            float64                       `json:"confidence"`
	Reasoning             string                        `json:"reasoning"`
	AlternativeCategories []LocationAlternativeCategory `json:"alternativeCategories"`
}

type StructuredLocationResponse struct {
	Category              string                        `json:"category"              jsonschema:"required"                                                                                                jsonschema_description:"The exact category name from the list provided"`
	CategoryID            string                        `json:"categoryId"            jsonschema:"required"                                                                                                jsonschema_description:"The exact ID from the list provided"`
	FacilityType          string                        `json:"facilityType"          jsonschema:"enum=,enum=CrossDock,enum=StorageWarehouse,enum=ColdStorage,enum=HazmatFacility,enum=IntermodalFacility" jsonschema_description:"The facility type if applicable, empty string if not applicable"`
	Confidence            float64                       `json:"confidence"            jsonschema:"required"                                                                                                jsonschema_description:"Confidence score between 0.0 and 1.0"`
	Reasoning             string                        `json:"reasoning"             jsonschema:"required"                                                                                                jsonschema_description:"Brief explanation of the classification"`
	AlternativeCategories []LocationAlternativeCategory `json:"alternativeCategories" jsonschema:"required"                                                                                                jsonschema_description:"Alternative category suggestions, can be empty array"`
}

func GenerateSchema[T any]() any {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

var LocationResponseSchema = GenerateSchema[StructuredLocationResponse]()
