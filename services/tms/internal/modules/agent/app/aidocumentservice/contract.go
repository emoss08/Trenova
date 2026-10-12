package aidocumentservice

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentaiextractionports"
)

var _ aicorrectionports.ExtractionContract = Contract{}

type Contract struct{}

func NewContract() aicorrectionports.ExtractionContract { return Contract{} }

func (Contract) CompletionRequest(
	fileName string,
	pages []documentaiextractionports.AIDocumentPage,
) *agentports.StructuredCompletionRequest {
	return newExtractCall(&documentaiextractionports.AIExtractRequest{
		FileName: fileName,
		Pages:    pages,
	}).request()
}

func (Contract) PageLimit() int { return extractPageLimit }

func (Contract) FieldKeys() []string { return slices.Clone(extractFieldKeys) }

func (Contract) ParseReply(text string) (*documentaiextractionports.AIExtractResult, error) {
	parsed := new(extractResponse)
	if err := decodeStructured(text, parsed); err != nil {
		return nil, err
	}

	return convertExtractResponse(parsed), nil
}
