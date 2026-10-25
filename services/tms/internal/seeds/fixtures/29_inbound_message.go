package fixtures

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft/documentshipmentdraftdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction/documentaiextractiondb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent/documentcontentdb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage/triagedb"
	"github.com/emoss08/trenova/internal/seeds/seedenv"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/seed"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection/aicorrectiondb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	inboundSeedMinute = int64(60)
	inboundSeedHour   = 60 * inboundSeedMinute
	inboundSeedDay    = 24 * inboundSeedHour

	SeedInboundMailboxToken         = "trenova-development-inbound-token"
	SeedInboundMailboxAddress       = "intake@dev.trenova.app"
	SeedInboundMailboxSigningSecret = "whsec_dHJlbm92YS1kZXZlbG9wbWVudC1pbmJvdW5kLXNpZ25pbmcta2V5" // #nosec G101 -- a published development fixture, not a credential.

	seedBillingMailboxToken   = "trenova-development-billing-token"
	seedBillingMailboxAddress = "ap@dev.trenova.app"
	seedClaimsMailboxToken    = "trenova-development-claims-token"
	seedClaimsMailboxAddress  = "claims@dev.trenova.app"
)

var seedRoutinePros = []string{
	"SEED-PAY-001",
	"SEED-PAY-002",
	"SEED-SHP-002",
	"SEED-SHP-003",
	"SEED-SHP-005",
}

var seedInboundShipmentPros = append(
	append([]string(nil), seedRoutinePros...),
	seedOpenLanePro,
	seedDetentionPro,
)

var seedInboundDocumentTypeCodes = []string{"INVOICE", documenttype.CodePOD, "RATECONF", "BOL"}

type stringSealer interface {
	EncryptStringWithAAD(value string, aad encryptionservice.AAD) (string, error)
}

func mailboxSigningSecret(sealer stringSealer, mailbox *inboundmessage.Mailbox) (string, error) {
	sealed, err := sealer.EncryptStringWithAAD(
		SeedInboundMailboxSigningSecret,
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

var InboundMessage = Seeds.Add(seed.Seed{
	Name:        "inbound_message",
	Kind:        seed.Fixture,
	Description: "Seeds three monitored addresses and a Triage queue in every state the design shows",
	After:       []*seed.Seed{Shipment, DriverPay, AdminAccount, Carrier, Worker, AIProvider, Detention, HoldReason},
	Run:         (&InboundMessageSeed{}).Run,
})

type inboundSeedRefs struct {
	org       *tenant.Organization
	admin     *tenant.User
	team      seedTeam
	customers map[string]*customer.Customer
	broker    *customer.Customer
	carriers  map[string]*carrier.Carrier
	driver    *worker.Worker
	lane      *seedLane
	shipments map[string]*shipment.Shipment
	docTypes  map[string]*documenttype.DocumentType
	loc       *time.Location
	now       int64
	dayStart  int64
	weekStart int64
}

type seedTeam struct {
	dana   *tenant.User
	marcus *tenant.User
	priya  *tenant.User
}

type seedMailboxes struct {
	intake  *inboundmessage.Mailbox
	billing *inboundmessage.Mailbox
	claims  *inboundmessage.Mailbox
}

func (s *InboundMessageSeed) Run(ctx context.Context, sr *seed.Run) error {
	db := sr.DB
	sealer, err := encryptionservice.New(encryptionservice.Params{Config: seedenv.Config(ctx)})
	if errors.Is(err, encryptionservice.ErrLocalKeyRequired) {
		sealer = encryptionservice.NewWithKeyManager(encryptionservice.DisabledKeyManager{})
	} else if err != nil {
		return err
	}
	org, err := defaultOrganization(ctx, sr)
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
) (*inboundSeedRefs, error) {
	admin, err := userByUsername(ctx, db, "admin")
	if err != nil {
		return nil, fmt.Errorf("get admin user: %w", err)
	}

	now := timeutils.NowUnix()
	loc, err := time.LoadLocation(org.Timezone)
	if err != nil {
		loc = time.UTC
	}
	refs := &inboundSeedRefs{
		org:       org,
		admin:     admin,
		loc:       loc,
		now:       now,
		dayStart:  timeutils.DayStart(now, loc),
		weekStart: timeutils.WeekStart(now, loc),
	}

	if refs.customers, err = loadSeedCustomers(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.shipments, err = loadSeedShipments(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.docTypes, err = loadSeedDocumentTypes(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.team, err = seedTeammates(ctx, db, org, admin); err != nil {
		return nil, err
	}
	if refs.carriers, err = loadSeedCarriers(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.driver, err = loadSeedDriver(ctx, db, org); err != nil {
		return nil, err
	}
	if refs.broker, err = ensureSeedBroker(ctx, db, sr, org); err != nil {
		return nil, err
	}
	if refs.lane, err = loadOpenLane(ctx, db, refs.shipments[seedOpenLanePro]); err != nil {
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
			return nil, fmt.Errorf("need seeded customer %s: %w", code, errFixtureMissing)
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
		shipmentdb.Shipments.ProNumber.In(seedInboundShipmentPros...),
	).All(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("load shipments: %w", err)
	}

	byPro := make(map[string]*shipment.Shipment, len(rows))
	for _, row := range rows {
		byPro[row.ProNumber] = row
	}
	for _, pro := range seedInboundShipmentPros {
		if _, ok := byPro[pro]; !ok {
			return nil, fmt.Errorf("need seeded shipment %s: %w", pro, errFixtureMissing)
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
			return nil, fmt.Errorf("need document type %s: %w", code, errFixtureMissing)
		}
	}

	return byCode, nil
}

func (s *InboundMessageSeed) insertMailboxes(
	ctx context.Context,
	db *keel.DB,
	refs *inboundSeedRefs,
	sealer stringSealer,
) (*seedMailboxes, error) {
	boxes := seedMailboxesFor(refs)
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.intake, boxes.billing, boxes.claims} {
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

func seedMailboxesFor(refs *inboundSeedRefs) *seedMailboxes {
	mailbox := func(name, address, token, purpose string) *inboundmessage.Mailbox {
		return &inboundmessage.Mailbox{
			ID:             pulid.MustNew("imbx_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			Name:           name,
			Address:        address,
			Provider:       inboundmessage.ProviderResend,
			TokenHash:      hashutils.SHA256Hex(token),
			Purpose:        purpose,
			ReviewPolicy:   inboundmessage.ReviewBelowConfidence,
			MinConfidence:  0.75,
			Status:         inboundmessage.MailboxActive,
		}
	}

	intake := mailbox("Intake", SeedInboundMailboxAddress, SeedInboundMailboxToken,
		"Tenders, rate confirmations, PODs and status requests")
	intake.BlockedSenders = []string{"deals.loadblaster.example"}

	billing := mailbox("Billing", seedBillingMailboxAddress, seedBillingMailboxToken,
		"Carrier invoices and payment questions")
	billing.MinConfidence = 0.85
	billing.TestSentAt = refs.now - 3*inboundSeedDay
	billing.TestReceivedAt = refs.now - 3*inboundSeedDay + 40

	claims := mailbox("Claims", seedClaimsMailboxAddress, seedClaimsMailboxToken,
		"Detention disputes, shortages and damage claims")
	claims.ReviewPolicy = inboundmessage.ReviewAlways
	claims.MinConfidence = 0
	claims.LastReceivedAt = refs.now - 7*inboundSeedDay - 2*inboundSeedHour

	return &seedMailboxes{intake: intake, billing: billing, claims: claims}
}

func (s *InboundMessageSeed) insertTriage(
	ctx context.Context,
	db *keel.DB,
	refs *inboundSeedRefs,
	boxes *seedMailboxes,
) error {
	rules := seedRules(refs, boxes)
	if err := triagedb.Rules.CreateMany(ctx, db, rules); err != nil {
		return fmt.Errorf("insert triage rules: %w", err)
	}

	templates := seedReplyTemplates(refs)
	if err := triagedb.ReplyTemplates.CreateMany(ctx, db, templates); err != nil {
		return fmt.Errorf("insert reply templates: %w", err)
	}

	messages := s.messages(refs, boxes, seedRuleSet(rules), templates)
	if err := insertSeededMessages(ctx, db, messages); err != nil {
		return err
	}
	if err := markRuleMatches(ctx, db, rules, messages); err != nil {
		return err
	}
	if err := markMailboxesReceived(ctx, db, boxes, messages); err != nil {
		return err
	}

	if err := triagedb.Settingses.Create(ctx, db, triageSettingsFor(refs)); err != nil {
		return fmt.Errorf("insert triage settings: %w", err)
	}
	if err := triagedb.Views.CreateMany(ctx, db, seedViews(refs)); err != nil {
		return fmt.Errorf("insert triage views: %w", err)
	}
	if err := triagedb.SenderHints.CreateMany(ctx, db, seedSenderHints(refs)); err != nil {
		return fmt.Errorf("insert triage sender hints: %w", err)
	}

	return nil
}

func insertSeededMessages(ctx context.Context, db *keel.DB, messages []seededMessage) error {
	if err := insertSeededDocuments(ctx, db, messages); err != nil {
		return err
	}

	activities := make([]*triage.Activity, 0, len(messages)*3)
	replies := make([]*triage.Reply, 0, 4)
	corrections := make([]*aicorrection.Correction, 0, 4)
	for _, seeded := range messages {
		if err := inboundmessagedb.InboundMessages.Create(ctx, db, seeded.message); err != nil {
			return fmt.Errorf("insert message %q: %w", seeded.message.Subject, err)
		}
		if err := inboundmessagedb.InboundAttachments.CreateMany(ctx, db, seeded.attachments); err != nil {
			return fmt.Errorf("insert attachments for %q: %w", seeded.message.Subject, err)
		}
		activities = append(activities, seeded.activities...)
		replies = append(replies, seeded.replies...)
		corrections = append(corrections, seeded.corrections...)
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

func insertSeededDocuments(ctx context.Context, db *keel.DB, messages []seededMessage) error {
	documents := make([]*document.Document, 0, 8)
	contents := make([]*documentcontent.Content, 0, 8)
	drafts := make([]*documentshipmentdraft.DocumentShipmentDraft, 0, 4)
	extractions := make([]*documentaiextraction.Extraction, 0, 2)
	for _, seeded := range messages {
		for _, staged := range seeded.documents {
			documents = append(documents, staged.document)
			if staged.content != nil {
				contents = append(contents, staged.content)
			}
			if staged.draft != nil {
				drafts = append(drafts, staged.draft)
			}
			if staged.extraction != nil {
				extractions = append(extractions, staged.extraction)
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
	boxes *seedMailboxes,
	messages []seededMessage,
) error {
	latest := make(map[pulid.ID]int64, 3)
	for _, seeded := range messages {
		if seeded.message.ReceivedAt > latest[seeded.message.MailboxID] {
			latest[seeded.message.MailboxID] = seeded.message.ReceivedAt
		}
	}

	mb := inboundmessagedb.Mailboxes
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.intake, boxes.billing, boxes.claims} {
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
			mb.LastReceivedAt.ToExpr(keel.DynFunc(
				"greatest",
				keel.DynOf(mb.LastReceivedAt).Coalesce(0),
				keel.DynLit(at).Cast(keel.TypeInt64),
			)),
		); err != nil {
			return fmt.Errorf("mark mailbox %s received: %w", mailbox.Name, err)
		}
	}

	return nil
}
