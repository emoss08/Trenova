package extractionevaljobs

import (
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/temporaljobs/documentintelligencejobs"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
)

var _ aicorrectionports.ExtractionReplyReader = ReplyReader{}

type ReplyReader struct {
	contract aicorrectionports.ExtractionContract
}

func NewReplyReader(contract aicorrectionports.ExtractionContract) ReplyReader {
	return ReplyReader{contract: contract}
}

func (r ReplyReader) ReadReply(text string) (*aicorrection.Prediction, error) {
	result, err := r.contract.ParseReply(text)
	if err != nil {
		return nil, err
	}

	return aicorrection.ReadPrediction(
		documentintelligencejobs.ShipmentDraftDataFromAIExtract(result),
	), nil
}
