package inboundmessageservice

import (
	"context"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// maxReferenceCandidates bounds how many tokens are looked up.
//
// A quoted thread is full of things that look like references — ticket numbers,
// tracking codes, the sender's own order ids — and checking all of them would
// turn one message into a hundred queries for no better answer. The ones worth
// having are near the top.
const maxReferenceCandidates = 12

// referenceToken is a run of characters that could be a pro number or a BOL.
//
// It is deliberately loose. Pro numbers are free text the organization formats
// however it likes, so there is no pattern to match against — the extractor
// proposes and the database disposes. A candidate that matches no shipment is
// simply not a match, which costs one indexed lookup and nothing else.
var referenceToken = regexp.MustCompile(`\b[A-Za-z0-9]+(?:[-_/][A-Za-z0-9]+)*\b`)

// hasDigit is what separates a reference from a word. "Thursday" and "confirm"
// are not reference numbers; "88213" and "SEED-SHP-001" are.
var hasDigit = regexp.MustCompile(`[0-9]`)

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
	seen := make(map[string]struct{}, maxReferenceCandidates)
	candidates := make([]string, 0, maxReferenceCandidates)

	for _, source := range []string{subject, body} {
		for _, token := range referenceToken.FindAllString(source, -1) {
			if len(candidates) >= maxReferenceCandidates {
				return candidates
			}
			if !plausibleReference(token) {
				continue
			}

			key := strings.ToUpper(token)
			if _, repeated := seen[key]; repeated {
				continue
			}
			seen[key] = struct{}{}
			candidates = append(candidates, token)
		}
	}

	return candidates
}

// plausibleReference keeps the tokens that could name a record.
//
// A reference has a digit in it and is long enough not to be a quantity. "2" and
// "48" are pallet counts and hours; "88213" and "BOL-2026-0001" are things to
// look up.
func plausibleReference(token string) bool {
	const minReferenceLength = 4

	if len(token) < minReferenceLength || len(token) > 100 {
		return false
	}

	return hasDigit.MatchString(token)
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
