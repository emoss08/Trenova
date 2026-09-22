package development

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	inboundSeedHour = int64(3600)
	inboundSeedDay  = 24 * inboundSeedHour

	// SeedInboundMailboxToken is the webhook token the seeded mailbox listens
	// on. A real mailbox shows its token once and stores only the hash; this
	// one is published here so a developer can post a delivery to it without
	// rotating anything. The seed runs in development only, and the token is
	// the same in every checkout — so it identifies nothing worth protecting.
	SeedInboundMailboxToken = "trenova-development-inbound-token"
	// SeedInboundMailboxAddress is where the seeded mail is addressed.
	SeedInboundMailboxAddress = "intake@dev.trenova.app"
)

type InboundMessageSeed struct {
	seedhelpers.BaseSeed
}

// InboundMessageSeed writes one monitored address and the mail that came to
// it: a tender waiting on a person, a rate confirmation the desk matched and
// handled, a proof of delivery with its attachment read, a status request
// answered without anybody looking, a message that failed mid-pipeline, and
// one nobody could make sense of.
//
// Between them the inbox has something in every lane — waiting, handled,
// ignored and quarantined — and the watchtower has items to show, because the
// ones that need a person are exactly the ones that project onto it.
//
// The classifications and confidences are written rather than computed: this
// seed exists to show what the pages look like without a provider being
// reachable. Real mail arriving on the same mailbox is classified for real.
//
// Depends on:
//   - Shipment: the loads the messages are matched to
func NewInboundMessageSeed() *InboundMessageSeed {
	seed := &InboundMessageSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"InboundMessage",
		"1.0.0",
		"Seeds a monitored mailbox and mail in every lane of the inbox",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(seedhelpers.SeedShipment)

	return seed
}

type inboundSeedRefs struct {
	org       *tenant.Organization
	admin     *tenant.User
	customers []*customer.Customer
	shipments []*shipment.Shipment
	now       int64
}

func (s *InboundMessageSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			refs, err := s.loadRefs(ctx, tx, sc)
			if err != nil {
				return err
			}

			cols := buncolgen.MailboxColumns
			count, err := tx.NewSelect().
				Model((*inboundmessage.Mailbox)(nil)).
				Where(cols.OrganizationID.Eq(), refs.org.ID).
				Where(cols.BusinessUnitID.Eq(), refs.org.BusinessUnitID).
				Count(ctx)
			if err != nil {
				return fmt.Errorf("count existing mailboxes: %w", err)
			}
			if count > 0 {
				return nil
			}

			mailbox, err := s.insertMailbox(ctx, tx, refs)
			if err != nil {
				return err
			}

			return s.insertMessages(ctx, tx, refs, mailbox)
		},
	)
}

func (s *InboundMessageSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
) (*inboundSeedRefs, error) {
	org, err := sc.GetDefaultOrganization(ctx)
	if err != nil {
		return nil, err
	}
	admin, err := sc.GetUserByUsername(ctx, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	refs := &inboundSeedRefs{org: org, admin: admin, now: timeutils.NowUnix()}

	customerCols := buncolgen.CustomerColumns
	refs.customers = make([]*customer.Customer, 0, 2)
	if err = tx.NewSelect().
		Model(&refs.customers).
		Where(customerCols.OrganizationID.Eq(), org.ID).
		Where(customerCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(customerCols.Name.OrderAsc()).
		Limit(2).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}
	if len(refs.customers) < 2 {
		return nil, fmt.Errorf("need two seeded customers: %w", seedhelpers.ErrEntityNotFound)
	}

	shipmentCols := buncolgen.ShipmentColumns
	refs.shipments = make([]*shipment.Shipment, 0, 3)
	if err = tx.NewSelect().
		Model(&refs.shipments).
		Where(shipmentCols.OrganizationID.Eq(), org.ID).
		Where(shipmentCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Order(shipmentCols.ProNumber.OrderAsc()).
		Limit(3).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load shipments: %w", err)
	}
	if len(refs.shipments) < 3 {
		return nil, fmt.Errorf("need three seeded shipments: %w", seedhelpers.ErrEntityNotFound)
	}

	return refs, nil
}

// insertMailbox writes the address in the middle policy — act when the reading
// is confident, ask when it is not. AlwaysReview would leave every seeded
// message in one lane, and AutoHandle would leave none of them waiting, so
// neither would show what the inbox is for.
func (s *InboundMessageSeed) insertMailbox(
	ctx context.Context,
	tx bun.Tx,
	refs *inboundSeedRefs,
) (*inboundmessage.Mailbox, error) {
	mailbox := &inboundmessage.Mailbox{
		ID:             pulid.MustNew("imbx_"),
		OrganizationID: refs.org.ID,
		BusinessUnitID: refs.org.BusinessUnitID,
		Name:           "Intake",
		Address:        SeedInboundMailboxAddress,
		Provider:       inboundmessage.ProviderResend,
		TokenHash:      hashutils.SHA256Hex(SeedInboundMailboxToken),
		Purpose:        "Tenders, rate confirmations, PODs and status requests",
		ReviewPolicy:   inboundmessage.ReviewBelowConfidence,
		MinConfidence:  0.75,
		Status:         inboundmessage.MailboxActive,
	}

	if _, err := tx.NewInsert().Model(mailbox).Exec(ctx); err != nil {
		return nil, fmt.Errorf("insert mailbox: %w", err)
	}

	return mailbox, nil
}

// seededMessage is one piece of mail and the files that came with it, so the
// attachment rows are written beside the message that carried them rather than
// in a second pass that could drift out of step.
type seededMessage struct {
	message     *inboundmessage.InboundMessage
	attachments []*inboundmessage.InboundAttachment
}

func (s *InboundMessageSeed) insertMessages(
	ctx context.Context,
	tx bun.Tx,
	refs *inboundSeedRefs,
	mailbox *inboundmessage.Mailbox,
) error {
	for _, seeded := range s.messages(refs, mailbox) {
		if _, err := tx.NewInsert().Model(seeded.message).Exec(ctx); err != nil {
			return fmt.Errorf("insert message %q: %w", seeded.message.Subject, err)
		}
		if len(seeded.attachments) == 0 {
			continue
		}
		for _, attachment := range seeded.attachments {
			attachment.MessageID = seeded.message.ID
			attachment.OrganizationID = seeded.message.OrganizationID
			attachment.BusinessUnitID = seeded.message.BusinessUnitID
		}
		if _, err := tx.NewInsert().Model(&seeded.attachments).Exec(ctx); err != nil {
			return fmt.Errorf("insert attachments for %q: %w", seeded.message.Subject, err)
		}
	}

	return nil
}

//nolint:funlen // One literal per message; splitting it would hide the set.
func (s *InboundMessageSeed) messages(
	refs *inboundSeedRefs,
	mailbox *inboundmessage.Mailbox,
) []seededMessage {
	base := func(
		providerID string,
		from string,
		fromName string,
		subject string,
		body string,
		receivedAt int64,
	) *inboundmessage.InboundMessage {
		return &inboundmessage.InboundMessage{
			ID:                pulid.MustNew("imsg_"),
			OrganizationID:    refs.org.ID,
			BusinessUnitID:    refs.org.BusinessUnitID,
			MailboxID:         mailbox.ID,
			ProviderMessageID: providerID,
			MessageID:         "<" + providerID + "@mail.example.com>",
			FromAddress:       from,
			FromName:          fromName,
			ToAddresses:       []string{mailbox.Address},
			Subject:           subject,
			TextBody:          body,
			ReceivedAt:        receivedAt,
		}
	}

	tender := base(
		"seed-inbound-tender",
		"dispatch@northwindfoods.example",
		"Northwind Foods Dispatch",
		"New load: Chicago IL to Columbus OH, pickup Thursday 08:00",
		"Hi team — we have a dry van load ready Thursday morning. 42,000 lbs, "+
			"no appointment needed at origin, delivery is a 2-hour window. "+
			"Rate confirmation to follow once you accept.",
		refs.now-2*inboundSeedHour,
	)
	tender.Classification = inboundmessage.ClassificationTender
	tender.Confidence = 0.68
	tender.Status = inboundmessage.StatusInReview
	tender.MatchedCustomerID = refs.customers[0].ID
	tender.MatchReason = "Sender domain matches " + refs.customers[0].Name
	tender.ReviewNote = "Creating a load is not something a desk does on its own."

	rateCon := base(
		"seed-inbound-ratecon",
		"rates@northwindfoods.example",
		"Northwind Foods Rates",
		"Rate confirmation "+refs.shipments[0].ProNumber,
		"Attached is the signed rate confirmation for "+refs.shipments[0].ProNumber+".",
		refs.now-6*inboundSeedHour,
	)
	rateCon.Classification = inboundmessage.ClassificationRateConfirmation
	rateCon.Confidence = 0.94
	rateCon.Status = inboundmessage.StatusActioned
	rateCon.MatchedCustomerID = refs.customers[0].ID
	rateCon.MatchedShipmentID = refs.shipments[0].ID
	rateCon.MatchReason = "Pro number " + refs.shipments[0].ProNumber + " in the subject"
	rateCon.ReviewedBy = refs.admin.ID
	rateCon.ReviewedAt = refs.now - 5*inboundSeedHour

	pod := base(
		"seed-inbound-pod",
		"driver.settlement@partnercarrier.example",
		"Partner Carrier Settlement",
		"POD for "+refs.shipments[1].ProNumber,
		"Delivered 14:22 yesterday, signed by R. Alvarez. Signed bill attached.",
		refs.now-inboundSeedDay,
	)
	pod.Classification = inboundmessage.ClassificationProofOfDelivery
	pod.Confidence = 0.91
	pod.Status = inboundmessage.StatusActioned
	pod.MatchedShipmentID = refs.shipments[1].ID
	pod.MatchReason = "Pro number " + refs.shipments[1].ProNumber + " in the subject"

	statusRequest := base(
		"seed-inbound-status",
		"ap@harborlinegrocers.example",
		"Harborline Grocers",
		"Where is "+refs.shipments[2].ProNumber+"?",
		"Customer is asking for an ETA on this one. Any update?",
		refs.now-90*60,
	)
	statusRequest.Classification = inboundmessage.ClassificationStatusRequest
	statusRequest.Confidence = 0.96
	statusRequest.Status = inboundmessage.StatusActioned
	statusRequest.MatchedCustomerID = refs.customers[1].ID
	statusRequest.MatchedShipmentID = refs.shipments[2].ID
	statusRequest.MatchReason = "Pro number " + refs.shipments[2].ProNumber + " in the subject"
	statusRequest.ReviewNote = "Answered from tracking; nobody had to look at it."

	dispute := base(
		"seed-inbound-detention",
		"claims@harborlinegrocers.example",
		"Harborline Grocers Claims",
		"Disputing detention on "+refs.shipments[1].ProNumber,
		"We show the driver arriving 40 minutes after the appointment window, "+
			"so we will not be paying the detention on this load.",
		refs.now-4*inboundSeedHour,
	)
	dispute.Classification = inboundmessage.ClassificationDetentionDispute
	dispute.Confidence = 0.81
	dispute.Status = inboundmessage.StatusInReview
	dispute.MatchedCustomerID = refs.customers[1].ID
	dispute.MatchedShipmentID = refs.shipments[1].ID
	dispute.MatchReason = "Pro number " + refs.shipments[1].ProNumber + " in the subject"
	dispute.ReviewNote = "A charge is being disputed; that is a person's call."

	// A message that failed mid-pipeline. It is in review rather than left at
	// Received on purpose: Received reads as still being worked on, and that is
	// the state nobody goes and checks.
	failed := base(
		"seed-inbound-failed",
		"edi@partnercarrier.example",
		"Partner Carrier EDI",
		"Automated: load status file 20260919-0441",
		"This message body was truncated by the sending system.",
		refs.now-8*inboundSeedHour,
	)
	failed.Status = inboundmessage.StatusInReview
	failed.FailureCode = "SETTLE_FAILED"
	failed.FailureText = "The classifier could not be reached after four attempts."
	failed.ReviewNote = "This message could not be processed, so it is waiting on a person."

	unreadable := base(
		"seed-inbound-unreadable",
		"noreply@unknownsender.example",
		"",
		"(no subject)",
		"",
		refs.now-3*inboundSeedDay,
	)
	unreadable.Classification = inboundmessage.ClassificationOther
	unreadable.Confidence = 0.22
	unreadable.Status = inboundmessage.StatusQuarantined
	unreadable.SpamScore = 7.4
	unreadable.ReviewNote = "Nothing in this message says what it is about."

	return []seededMessage{
		{message: tender},
		{
			message: rateCon,
			attachments: []*inboundmessage.InboundAttachment{{
				ID:          pulid.MustNew("imsga_"),
				FileName:    "rate-confirmation-" + refs.shipments[0].ProNumber + ".pdf",
				ContentType: "application/pdf",
				ByteSize:    184_320,
				Kind:        inboundmessage.AttachmentRateConfirmation,
			}},
		},
		{
			message: pod,
			attachments: []*inboundmessage.InboundAttachment{
				{
					ID:          pulid.MustNew("imsga_"),
					FileName:    "signed-bol-" + refs.shipments[1].ProNumber + ".pdf",
					ContentType: "application/pdf",
					ByteSize:    221_184,
					Kind:        inboundmessage.AttachmentProofOfDelivery,
				},
				// A file the pipeline could not read. It leaves a reason on its
				// own row rather than vanishing, because its absence is what
				// makes the message incomplete.
				{
					ID:          pulid.MustNew("imsga_"),
					FileName:    "trailer-photo.heic",
					ContentType: "image/heic",
					ByteSize:    2_097_152,
					Kind:        inboundmessage.AttachmentUnknown,
					FailureText: "This file type cannot be uploaded.",
				},
			},
		},
		{message: statusRequest},
		{message: dispute},
		{message: failed},
		{message: unreadable},
	}
}
