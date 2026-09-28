package inboundmessageservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/referenceutils"
	"go.uber.org/zap"
)

// maxReferenceCandidates bounds how many tokens are looked up.
//
// A quoted thread is full of things that look like references — ticket numbers,
// tracking codes, the sender's own order ids — and checking all of them would
// turn one message into a hundred queries for no better answer. The ones worth
// having are near the top.
const maxReferenceCandidates = 12

// ShipmentFinder and PartyFinder are the lookups matching and linking need.
// They are the repository ports under the names this package reads them by.
type (
	ShipmentFinder = repositories.InboundShipmentFinder
	PartyFinder    = repositories.InboundPartyFinder
)

// Match is what the message turned out to be about, and why.
//
// Reason is not decoration. A shipment matched on a pro number in the subject
// is a different claim from one matched because the sender's company has only
// one load open, and a person deciding whether to trust the match needs to know
// which they are looking at.
type Match struct {
	CustomerID pulid.ID
	CarrierID  pulid.ID
	ShipmentID pulid.ID
	Reason     string
}

// referenceCandidates pulls the tokens worth looking up out of a message.
//
// The subject comes first because that is where people put the number they are
// writing about; the body follows, in order, so the earliest mention wins over
// something buried in a quoted reply from last week.
func referenceCandidates(subject, body string) []string {
	return referenceutils.Candidates(maxReferenceCandidates, subject, body)
}

// Match works out what a message is about.
//
// A failed lookup is never fatal. An unmatched message is still a message a
// person can read and link by hand, and refusing to store it because one query
// failed would lose the message rather than the match.
func (s *Service) Match(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	tenantInfo pagination.TenantInfo,
) Match {
	match := Match{}
	reasons := make([]string, 0, 2)

	if party := s.matchSender(ctx, message, tenantInfo); party.Reason != "" {
		match.CustomerID = party.CustomerID
		match.CarrierID = party.CarrierID
		reasons = append(reasons, party.Reason)
	}

	if shipmentID, reference, ok := s.matchShipment(ctx, message, tenantInfo); ok {
		match.ShipmentID = shipmentID
		reasons = append(reasons, "the reference "+reference+" in the message")
	}

	if len(reasons) > 0 {
		match.Reason = "Matched on " + strings.Join(reasons, ", and ") + "."
	}

	return match
}

func (s *Service) matchSender(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	tenantInfo pagination.TenantInfo,
) Match {
	if s.parties == nil || message.FromAddress == "" {
		return Match{}
	}

	if customerID, found, err := s.parties.FindCustomerByEmail(
		ctx, tenantInfo, message.FromAddress,
	); err != nil {
		s.l.Warn("failed to match an inbound sender to a customer",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	} else if found {
		return Match{
			CustomerID: customerID,
			Reason:     "the sender " + message.FromAddress + " being a customer contact",
		}
	}

	if carrierID, found, err := s.parties.FindCarrierByEmail(
		ctx, tenantInfo, message.FromAddress,
	); err != nil {
		s.l.Warn("failed to match an inbound sender to a carrier",
			zap.String("messageId", message.ID.String()), zap.Error(err))
	} else if found {
		return Match{
			CarrierID: carrierID,
			Reason:    "the sender " + message.FromAddress + " being a carrier contact",
		}
	}

	return Match{}
}

// matchShipment takes the first candidate that resolves.
//
// First rather than best: the candidates are already in the order a person
// wrote them, and a message naming two shipments is one a person should read
// rather than one to pick a winner from.
func (s *Service) matchShipment(
	ctx context.Context,
	message *inboundmessage.InboundMessage,
	tenantInfo pagination.TenantInfo,
) (pulid.ID, string, bool) {
	if s.shipments == nil {
		return pulid.Nil, "", false
	}

	for _, candidate := range referenceCandidates(message.Subject, message.TextBody) {
		shipmentID, found, err := s.shipments.FindByReference(ctx, tenantInfo, candidate)
		if err != nil {
			s.l.Warn("failed to look up an inbound message reference",
				zap.String("messageId", message.ID.String()), zap.Error(err))

			continue
		}
		if found {
			return shipmentID, candidate, true
		}
	}

	return pulid.Nil, "", false
}
