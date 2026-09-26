package extractionevaljobs

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
)

var _ services.ExtractionReplyReader = ReplyReader{}

type ReplyReader struct {
	contract services.ExtractionContract
}

func NewReplyReader(contract services.ExtractionContract) ReplyReader {
	return ReplyReader{contract: contract}
}

func (r ReplyReader) ReadReply(text string) (*aicorrection.Prediction, error) {
	result, err := r.contract.ParseReply(text)
	if err != nil {
		return nil, err
	}

	return aicorrection.ReadPrediction(documentintelligencejobs.ShipmentDraftDataFromAIExtract(result)), nil
}
