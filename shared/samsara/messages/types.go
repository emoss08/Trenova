package messages

import samsaraspec "github.com/emoss08/trenova/shared/samsara/internal/samsaraspec"

type Message = samsaraspec.V1MessageResponse

type SentMessage = samsaraspec.V1Message

type ListResponse = samsaraspec.InlineResponse2005

type CreateResponse = samsaraspec.InlineResponse2006

type CreateRequest struct {
	DriverIDs []string
	Text      string
}

type createRequestBody struct {
	DriverIDs []int64 `json:"driverIds"`
	Text      string  `json:"text"`
}
