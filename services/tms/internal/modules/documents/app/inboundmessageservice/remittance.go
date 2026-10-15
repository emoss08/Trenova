package inboundmessageservice

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const remittanceFactKey = "factorRemittance"

type IngestRemittancesResult struct {
	RemittanceIDs []pulid.ID `json:"remittanceIds"`
	Problem       string     `json:"problem"`
}

// IngestRemittances hands a factor's remittance mail to billing, which opens
// a remittance for each file it can read, and notes on the message what came
// of it. It is idempotent: billing finds the remittance an attachment already
// opened, so a retried workflow opens nothing twice.
func (s *Service) IngestRemittances(
	ctx context.Context,
	messageID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*IngestRemittancesResult, error) {
	result := &IngestRemittancesResult{RemittanceIDs: []pulid.ID{}}
	if s.remittances == nil {
		return result, nil
	}
	message, err := s.messageRepo.GetByID(ctx, inboundmessageports.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenantInfo,
		IncludeAttachments: true,
	})
	if err != nil {
		return nil, err
	}
	if message.Classification != inboundmessage.ClassificationFactorRemittance {
		return result, nil
	}

	attachments := make(
		[]inboundmessageports.RemittanceIntakeAttachment,
		0,
		len(message.Attachments),
	)
	for _, attachment := range message.Attachments {
		if attachment == nil || attachment.DocumentID.IsNil() {
			continue
		}
		attachments = append(attachments, inboundmessageports.RemittanceIntakeAttachment{
			DocumentID:  attachment.DocumentID,
			FileName:    attachment.FileName,
			ContentType: attachment.ContentType,
		})
	}
	ingested, err := s.remittances.IngestRemittances(
		ctx,
		&inboundmessageports.RemittanceIntakeRequest{
			TenantInfo:  tenantInfo,
			MessageID:   message.ID,
			FromAddress: message.FromAddress,
			ReceivedAt:  message.ReceivedAt,
			Attachments: attachments,
		},
	)
	if err != nil {
		return nil, err
	}
	result.RemittanceIDs = ingested.RemittanceIDs
	result.Problem = ingested.Problem

	message.Facts = withRemittanceFact(message.Facts, remittanceFact(ingested))
	updated, err := s.messageRepo.Update(ctx, message)
	if err != nil {
		return nil, err
	}
	s.l.Info("handed a factor remittance message to billing",
		zap.String("messageId", updated.ID.String()),
		zap.Int("remittances", len(ingested.RemittanceIDs)),
		zap.String("problem", ingested.Problem))
	s.notify.MessagesChanged(ctx, updated)
	return result, nil
}

func remittanceFact(ingested *inboundmessageports.RemittanceIntakeResult) *inboundmessage.Fact {
	fact := &inboundmessage.Fact{
		Key:   remittanceFactKey,
		Label: "Remittance",
		Tone:  inboundmessage.FactToneNeutral,
	}
	switch count := len(ingested.RemittanceIDs); {
	case ingested.Problem != "" && count == 0:
		fact.Value = ingested.Problem
		fact.Tone = inboundmessage.FactToneWarning
	case count == 1:
		fact.Value = "Opened for " + ingested.FactorName
	default:
		fact.Value = strconv.Itoa(count) + " opened for " + ingested.FactorName
	}
	return fact
}

func withRemittanceFact(
	facts []inboundmessage.Fact,
	fact *inboundmessage.Fact,
) []inboundmessage.Fact {
	out := make([]inboundmessage.Fact, 0, len(facts)+1)
	for _, existing := range facts {
		if existing.Key != remittanceFactKey {
			out = append(out, existing)
		}
	}
	if len(out) >= MaxFacts {
		out = out[:MaxFacts-1]
	}
	return append(out, *fact)
}
