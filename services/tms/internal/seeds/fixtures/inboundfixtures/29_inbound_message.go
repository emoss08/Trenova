package inboundfixtures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection/aicorrectiondb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction/documentaiextractiondb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft/documentshipmentdraftdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/internal/seeds/fixtures/tenantfixtures"
	"github.com/emoss08/trenova/internal/seeds/fixtures/triagefixtures"
	"github.com/emoss08/trenova/internal/seeds/seedenv"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

var seedInboundDocumentTypeCodes = []string{"INVOICE", documenttype.CodePOD, "RATECONF", "BOL"}

type stringSealer interface {
	EncryptStringWithAAD(value string, aad encryptionservice.AAD) (string, error)
}

func mailboxSigningSecret(sealer stringSealer, mailbox *inboundmessage.Mailbox) (string, error) {
	sealed, err := sealer.EncryptStringWithAAD(
		triagefixtures.SeedInboundMailboxSigningSecret,
		encryptionservice.InboundMailboxSecretAAD(
			mailbox.OrganizationID,
			mailbox.BusinessUnitID,
			mailbox.ID,
			inboundmessage.SigningSecretField,
		),
	)
	if errors.Is(err, encryptionservice.ErrKeyManagerDisabled) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("seal the mailbox signing secret: %w", err)
	}

	return sealed, nil
}

type InboundMessageSeed struct {
}

func (s *InboundMessageSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	sealer, err := encryptionservice.New(
		encryptionservice.Params{SecurityConfig: &seedenv.Config(ctx).Security},
	)
	if errors.Is(err, encryptionservice.ErrLocalKeyRequired) {
		sealer = encryptionservice.NewWithKeyManager(encryptionservice.DisabledKeyManager{})
	} else if err != nil {
		return err
	}
	org, err := tenantfixtures.DefaultOrganization(ctx, sr)
	if err != nil {
		return err
	}

	refs, err := s.loadRefs(ctx, db, sr, org)
	if err != nil {
		return err
	}
	if err = ensureExceptionHoldReason(ctx, db, org); err != nil {
		return err
	}
	if err = routeInboundClassification(ctx, db, org); err != nil {
		return err
	}

	boxes, err := s.insertMailboxes(ctx, db, refs, sealer)
	if err != nil {
		return err
	}

	return s.insertTriage(ctx, db, refs, boxes)
}

func (s *InboundMessageSeed) loadRefs(
	ctx context.Context,
	db *keel.DB,
	sr *seed.Run,
	org *tenant.Organization,
) (*triagefixtures.Refs, error) {
	admin, err := tenantfixtures.UserByUsername(ctx, db, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	now := timeutils.NowUnix()
	loc, err := time.LoadLocation(org.Timezone)
	if err != nil {
		loc = time.UTC
	}
	refs := &triagefixtures.Refs{
		Org:       org,
		Admin:     admin,
		Loc:       loc,
		Now:       now,
		DayStart:  timeutils.DayStart(now, loc),
		WeekStart: timeutils.WeekStart(now, loc),
	}

	if refs.Customers, err = loadSeedCustomers(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.Shipments, err = loadSeedShipments(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.DocTypes, err = loadSeedDocumentTypes(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.Team, err = seedTeammates(ctx, db, org, admin); err != nil {
		return nil, err
	}
	if refs.Carriers, err = loadSeedCarriers(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.Driver, err = loadSeedDriver(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.Broker, err = ensureSeedBroker(ctx, db, sr, org); err != nil {
		return nil, err
	}
	if refs.Lane, err = loadOpenLane(ctx, db, refs.Shipments[triagefixtures.OpenLanePro]); err != nil {
		return nil, err
	}

	return refs, nil
}

func loadSeedCustomers(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (map[string]*customer.Customer, error) {
	rows, err := customerdb.Customers.AllTenants().Where(
		customerdb.Customers.OrganizationID.Eq(org.ID),
		customerdb.Customers.BusinessUnitID.Eq(org.BusinessUnitID),
		customerdb.Customers.Code.In([]string{"ACME", "FRSH", "GLBL", "RNGE", "PEAK"}...),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}

	byCode := make(map[string]*customer.Customer, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row
	}
	for _, code := range []string{"ACME", "FRSH", "GLBL", "RNGE", "PEAK"} {
		if _, ok := byCode[code]; !ok {
			return nil, fmt.Errorf(
				"need seeded customer %s: %w",
				code,
				tenantfixtures.ErrFixtureMissing,
			)
		}
	}

	return byCode, nil
}

func loadSeedShipments(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (map[string]*shipment.Shipment, error) {
	rows, err := shipmentdb.Shipments.AllTenants().Where(
		shipmentdb.Shipments.OrganizationID.Eq(org.ID),
		shipmentdb.Shipments.BusinessUnitID.Eq(org.BusinessUnitID),
		shipmentdb.Shipments.ProNumber.In(triagefixtures.ShipmentPros...),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load shipments: %w", err)
	}

	byPro := make(map[string]*shipment.Shipment, len(rows))
	for _, row := range rows {
		byPro[row.ProNumber] = row
	}
	for _, pro := range triagefixtures.ShipmentPros {
		if _, ok := byPro[pro]; !ok {
			return nil, fmt.Errorf(
				"need seeded shipment %s: %w",
				pro,
				tenantfixtures.ErrFixtureMissing,
			)
		}
	}

	return byPro, nil
}

func loadSeedDocumentTypes(
	ctx context.Context,
	db *keel.DB,
	org *tenant.Organization,
) (map[string]*documenttype.DocumentType, error) {
	rows, err := documenttypedb.DocumentTypes.AllTenants().Where(
		documenttypedb.DocumentTypes.OrganizationID.Eq(org.ID),
		documenttypedb.DocumentTypes.BusinessUnitID.Eq(org.BusinessUnitID),
		documenttypedb.DocumentTypes.Code.In(seedInboundDocumentTypeCodes...),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load document types: %w", err)
	}

	byCode := make(map[string]*documenttype.DocumentType, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row
	}
	for _, code := range seedInboundDocumentTypeCodes {
		if _, ok := byCode[code]; !ok {
			return nil, fmt.Errorf(
				"need document type %s: %w",
				code,
				tenantfixtures.ErrFixtureMissing,
			)
		}
	}

	return byCode, nil
}

func (s *InboundMessageSeed) insertMailboxes(
	ctx context.Context,
	db *keel.DB,
	refs *triagefixtures.Refs,
	sealer stringSealer,
) (*triagefixtures.Mailboxes, error) {
	boxes := triagefixtures.MailboxesFor(refs)
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.Intake, boxes.Billing, boxes.Claims} {
		secret, err := mailboxSigningSecret(sealer, mailbox)
		if err != nil {
			return nil, err
		}
		mailbox.SigningSecret = secret
		if err = inboundmessagedb.Mailboxes.Create(ctx, db, mailbox); err != nil {
			return nil, fmt.Errorf("insert mailbox %s: %w", mailbox.Name, err)
		}
	}

	return boxes, nil
}

func (s *InboundMessageSeed) insertTriage(
	ctx context.Context,
	db *keel.DB,
	refs *triagefixtures.Refs,
	boxes *triagefixtures.Mailboxes,
) error {
	rules := triagefixtures.Rules(refs, boxes)
	if err := triagedb.Rules.CreateMany(ctx, db, rules); err != nil {
		return fmt.Errorf("insert triage rules: %w", err)
	}

	templates := triagefixtures.ReplyTemplates(refs)
	if err := triagedb.ReplyTemplates.CreateMany(ctx, db, templates); err != nil {
		return fmt.Errorf("insert reply templates: %w", err)
	}

	messages := triagefixtures.Messages(refs, boxes, triagefixtures.NewRuleSet(rules), templates)
	if err := insertSeededMessages(ctx, db, messages); err != nil {
		return err
	}
	if err := markRuleMatches(ctx, db, rules, messages); err != nil {
		return err
	}
	if err := markMailboxesReceived(ctx, db, boxes, messages); err != nil {
		return err
	}

	if err := triagedb.Settingses.Create(ctx, db, triagefixtures.Settings(refs)); err != nil {
		return fmt.Errorf("insert triage settings: %w", err)
	}
	if err := triagedb.Views.CreateMany(ctx, db, triagefixtures.Views(refs)); err != nil {
		return fmt.Errorf("insert triage views: %w", err)
	}
	if err := triagedb.SenderHints.CreateMany(ctx, db, triagefixtures.SenderHints(refs)); err != nil {
		return fmt.Errorf("insert triage sender hints: %w", err)
	}

	return nil
}

func insertSeededMessages(
	ctx context.Context,
	db *keel.DB,
	messages []triagefixtures.Message,
) error {
	if err := insertSeededDocuments(ctx, db, messages); err != nil {
		return err
	}

	activities := make([]*triage.Activity, 0, len(messages)*3)
	replies := make([]*triage.Reply, 0, 4)
	corrections := make([]*aicorrection.Correction, 0, 4)
	for _, seeded := range messages {
		if err := inboundmessagedb.InboundMessages.Create(ctx, db, seeded.Message); err != nil {
			return fmt.Errorf("insert message %q: %w", seeded.Message.Subject, err)
		}
		if err := inboundmessagedb.InboundAttachments.CreateMany(ctx, db, seeded.Attachments); err != nil {
			return fmt.Errorf("insert attachments for %q: %w", seeded.Message.Subject, err)
		}
		activities = append(activities, seeded.Activities...)
		replies = append(replies, seeded.Replies...)
		corrections = append(corrections, seeded.Corrections...)
	}

	if err := triagedb.Replies.CreateMany(ctx, db, replies); err != nil {
		return fmt.Errorf("insert replies: %w", err)
	}
	if err := triagedb.Activities.CreateMany(ctx, db, activities); err != nil {
		return fmt.Errorf("insert activities: %w", err)
	}
	if err := aicorrectiondb.Corrections.CreateMany(ctx, db, corrections); err != nil {
		return fmt.Errorf("insert AI corrections: %w", err)
	}

	return nil
}

func insertSeededDocuments(
	ctx context.Context,
	db *keel.DB,
	messages []triagefixtures.Message,
) error {
	documents := make([]*document.Document, 0, 8)
	contents := make([]*documentcontent.Content, 0, 8)
	drafts := make([]*documentshipmentdraft.DocumentShipmentDraft, 0, 4)
	extractions := make([]*documentaiextraction.Extraction, 0, 2)
	for _, seeded := range messages {
		for _, staged := range seeded.Documents {
			documents = append(documents, staged.Document)
			if staged.Content != nil {
				contents = append(contents, staged.Content)
			}
			if staged.Draft != nil {
				drafts = append(drafts, staged.Draft)
			}
			if staged.Extraction != nil {
				extractions = append(extractions, staged.Extraction)
			}
		}
	}

	if err := documentdb.Documents.CreateMany(ctx, db, documents); err != nil {
		return fmt.Errorf("insert documents: %w", err)
	}
	if err := documentcontentdb.Contents.CreateMany(ctx, db, contents); err != nil {
		return fmt.Errorf("insert document readings: %w", err)
	}
	if err := documentshipmentdraftdb.DocumentShipmentDrafts.CreateMany(ctx, db, drafts); err != nil {
		return fmt.Errorf("insert shipment drafts: %w", err)
	}
	if err := documentaiextractiondb.Extractions.CreateMany(ctx, db, extractions); err != nil {
		return fmt.Errorf("insert AI extractions: %w", err)
	}

	return nil
}

func markMailboxesReceived(
	ctx context.Context,
	db *keel.DB,
	boxes *triagefixtures.Mailboxes,
	messages []triagefixtures.Message,
) error {
	latest := make(map[pulid.ID]int64, 3)
	for _, seeded := range messages {
		if seeded.Message.ReceivedAt > latest[seeded.Message.MailboxID] {
			latest[seeded.Message.MailboxID] = seeded.Message.ReceivedAt
		}
	}

	mb := inboundmessagedb.Mailboxes
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.Intake, boxes.Billing, boxes.Claims} {
		at, ok := latest[mailbox.ID]
		if !ok {
			continue
		}
		if _, err := mb.AllTenants().Where(
			mb.ID.Eq(mailbox.ID),
			mb.OrganizationID.Eq(mailbox.OrganizationID),
			mb.BusinessUnitID.Eq(mailbox.BusinessUnitID),
		).Update(
			ctx,
			db,
			mb.LastReceivedAt.ToExpr(mb.LastReceivedAt.Coalesce(0).Greatest(keel.Int64(at))),
		); err != nil {
			return fmt.Errorf("mark mailbox %s received: %w", mailbox.Name, err)
		}
	}

	return nil
}
