package aidocumentservice

import (
	"slices"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

var _ serviceports.ExtractionContract = Contract{}

type Contract struct{}

func NewContract() serviceports.ExtractionContract { return Contract{} }

func (Contract) CompletionRequest(
	fileName string,
	pages []serviceports.AIDocumentPage,
) *serviceports.StructuredCompletionRequest {
	return newExtractCall(&serviceports.AIExtractRequest{
		FileName: fileName,
		Pages:    pages,
	}).request()
}

func (Contract) PageLimit() int { return extractPageLimit }

func (Contract) FieldKeys() []string { return slices.Clone(extractFieldKeys) }

func (Contract) ParseReply(text string) (*serviceports.AIExtractResult, error) {
	parsed := new(extractResponse)
	if err := decodeStructured(text, parsed); err != nil {
		return nil, err
	}

	return convertExtractResponse(parsed), nil
}
