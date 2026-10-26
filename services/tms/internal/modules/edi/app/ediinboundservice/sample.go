package ediinboundservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/edi/app/edix12inspect"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
)

type ParsedSample struct {
	SenderQualifier     string                        `json:"senderQualifier"`
	SenderID            string                        `json:"senderId"`
	ReceiverQualifier   string                        `json:"receiverQualifier"`
	ReceiverID          string                        `json:"receiverId"`
	InterchangeVersion  string                        `json:"interchangeVersion"`
	UsageIndicator      string                        `json:"usageIndicator"`
	ApplicationSender   string                        `json:"applicationSender"`
	ApplicationReceiver string                        `json:"applicationReceiver"`
	FunctionalGroupID   string                        `json:"functionalGroupId"`
	GroupVersion        string                        `json:"groupVersion"`
	Separators          edix12inspect.X12Separators   `json:"separators"`
	Inspection          edix12inspect.InspectOverview `json:"inspection"`
	Transactions        []ParsedSampleTransaction     `json:"transactions"`
}

type ParsedSampleTransaction struct {
	TransactionSet edi.TransactionSet  `json:"transactionSet"`
	ControlNumber  string              `json:"controlNumber"`
	Payload        edi.DocumentPayload `json:"payload"`
}

func ParseSample(ctx context.Context, rawX12 string) (*ParsedSample, error) {
	interchange, problem, err := parseInterchange(ctx, rawX12)
	if err != nil {
		return nil, err
	}
	if problem != nil {
		return nil, problem
	}

	sample := &ParsedSample{
		SenderQualifier:     interchange.senderQualifier,
		SenderID:            interchange.senderID,
		ReceiverQualifier:   interchange.receiverQualifier,
		ReceiverID:          interchange.receiverID,
		InterchangeVersion:  interchange.version,
		UsageIndicator:      interchange.usage,
		ApplicationSender:   interchange.applicationSender,
		ApplicationReceiver: interchange.applicationReceiver,
		FunctionalGroupID:   interchange.functionalGroupID,
		Separators:          interchange.inspection.Separators,
		Inspection:          interchange.inspection.InspectOverview,
		Transactions:        make([]ParsedSampleTransaction, 0, len(interchange.transactions)),
	}
	if len(interchange.groups) > 0 {
		sample.GroupVersion = interchange.groups[0].version
	}
	for index := range interchange.transactions {
		transaction := &interchange.transactions[index]
		sample.Transactions = append(sample.Transactions, ParsedSampleTransaction{
			TransactionSet: transaction.set,
			ControlNumber:  transaction.controlNumber,
			Payload:        transaction.documentPayload(),
		})
	}

	return sample, nil
}
