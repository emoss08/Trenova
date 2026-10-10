package extractionevaljobs

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/documents/jobs/documentintelligencejobs"
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
