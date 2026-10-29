package edioutbound

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/app/edilading"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

// envelopeOf is the envelope Trenova writes to a partner, from the interchange its partner file
// declares: Trenova is the sender and the partner the receiver.
func envelopeOf(program *edilading.Program) edidocument.X12EnvelopeSettings {
	envelope := edidocument.DefaultX12EnvelopeSettings()
	partner := program.Manifest.Partner
	if partner == nil || partner.Interchange == nil {
		return envelope
	}
	interchange := partner.Interchange
	if interchange.Us != nil {
		envelope.InterchangeSenderQualifier = interchange.Us.Qualifier
		envelope.InterchangeSenderID = interchange.Us.ID
		envelope.ApplicationSenderCode = interchange.Us.ID
	}
	if interchange.Them != nil {
		envelope.InterchangeReceiverQualifier = interchange.Them.Qualifier
		envelope.InterchangeReceiverID = interchange.Them.ID
		envelope.ApplicationReceiverCode = interchange.Them.ID
	}
	if interchange.Group != nil {
		envelope.ApplicationSenderCode = stringutils.FirstNonEmpty(
			interchange.Group.Ours,
			envelope.ApplicationSenderCode,
		)
		envelope.ApplicationReceiverCode = stringutils.FirstNonEmpty(
			interchange.Group.Theirs,
			envelope.ApplicationReceiverCode,
		)
	}
	if strings.EqualFold(interchange.Usage, "production") {
		envelope.InterchangeUsageIndicator = "P"
	} else {
		envelope.InterchangeUsageIndicator = "T"
	}
	if len(interchange.Separators) == 4 {
		envelope.ElementSeparator = interchange.Separators[0]
		envelope.SegmentTerminator = interchange.Separators[1]
		envelope.ComponentSeparator = interchange.Separators[2]
		envelope.RepetitionSeparator = interchange.Separators[3]
	}
	return envelope
}

// partnerEnvelope is the envelope of the partner's active version.
func (s *Service) partnerEnvelope(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	partnerID pulid.ID,
) (edidocument.X12EnvelopeSettings, *edilading.Program, error) {
	program, err := s.lading.Active(ctx, tenantInfo, partnerID)
	if err != nil {
		return edidocument.X12EnvelopeSettings{}, nil, err
	}
	return envelopeOf(program), program, nil
}
