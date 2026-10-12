package development

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/documenttypecol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"

	"github.com/emoss08/trenova/internal/infrastructure/database/common"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentaiextraction"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/operations/domain/documentshipmentdraft"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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
	EncryptString(value string) (string, error)
}

func mailboxSigningSecret(sealer stringSealer) (string, error) {
	sealed, err := sealer.EncryptString(SeedInboundMailboxSigningSecret)
	if errors.Is(err, encryptionservice.ErrKeyManagerDisabled) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("seal the mailbox signing secret: %w", err)
	}

	return sealed, nil
}

type InboundMessageSeed struct {
	seedhelpers.BaseSeed
}

func NewInboundMessageSeed() *InboundMessageSeed {
	seed := &InboundMessageSeed{}
	seed.BaseSeed = *seedhelpers.NewBaseSeed(
		"InboundMessage",
		"2.0.0",
		"Seeds three monitored addresses and a Triage queue in every state the design shows",
		[]common.Environment{common.EnvDevelopment},
	)
	seed.SetDependencies(
		seedhelpers.SeedShipment,
		seedhelpers.SeedDriverPay,
		seedhelpers.SeedDocumentType,
		seedhelpers.SeedOrganizationRoles,
		seedhelpers.SeedCarrier,
		seedhelpers.SeedWorker,
		seedhelpers.SeedAIProvider,
		seedhelpers.SeedDetention,
		seedhelpers.SeedHoldReason,
	)

	return seed
}

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

func (s *InboundMessageSeed) Run(ctx context.Context, tx bun.Tx) error {
	return seedhelpers.RunInTransaction(
		ctx,
		tx,
		s.Name(),
		nil,
		func(ctx context.Context, tx bun.Tx, sc *seedhelpers.SeedContext) error {
			secret, err := mailboxSigningSecret(
				encryptionservice.New(encryptionservice.Params{Config: sc.Config()}),
			)
			if err != nil {
				return err
			}
			if secret == "" {
				sc.Logger().Warn(
					"no encryption key is configured, so the seeded mailboxes have no signing secret and will refuse deliveries",
				)
			}

			org, err := sc.GetDefaultOrganization(ctx)
			if err != nil {
				return err
			}
			seeded, err := triageAlreadySeeded(ctx, tx, org)
			if err != nil {
				return err
			}
			if seeded {
				return backfillSigningSecret(ctx, tx, org, secret)
			}

			refs, err := s.loadRefs(ctx, tx, sc, org)
			if err != nil {
				return err
			}
			if err = ensureExceptionHoldReason(ctx, tx, org); err != nil {
				return err
			}
			if err = routeInboundClassification(ctx, tx, org); err != nil {
				return err
			}
			if err = retireFirstEditionSeed(ctx, tx, org); err != nil {
				return err
			}

			boxes, err := s.insertMailboxes(ctx, tx, refs, secret)
			if err != nil {
				return err
			}

			return s.insertTriage(ctx, tx, refs, boxes)
		},
	)
}

func triageAlreadySeeded(ctx context.Context, tx bun.Tx, org *tenant.Organization) (bool, error) {
	cols := inboundmessagecol.MailboxColumns

	return tx.NewSelect().
		Model((*inboundmessage.Mailbox)(nil)).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.TokenHash.Eq(), hashutils.SHA256Hex(seedBillingMailboxToken)).
		Exists(ctx)
}

func backfillSigningSecret(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
	secret string,
) error {
	if secret == "" {
		return nil
	}

	cols := inboundmessagecol.MailboxColumns
	if _, err := tx.NewUpdate().
		Model((*inboundmessage.Mailbox)(nil)).
		Set(cols.SigningSecret.Set(), secret).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.TokenHash.In(), bun.List([]string{
			hashutils.SHA256Hex(SeedInboundMailboxToken),
			hashutils.SHA256Hex(seedBillingMailboxToken),
			hashutils.SHA256Hex(seedClaimsMailboxToken),
		})).
		Where(cols.SigningSecret.IsNull()).
		Exec(ctx); err != nil {
		return fmt.Errorf("backfill the mailbox signing secrets: %w", err)
	}

	return nil
}

func retireFirstEditionSeed(ctx context.Context, tx bun.Tx, org *tenant.Organization) error {
	mailboxCols := inboundmessagecol.MailboxColumns
	if _, err := tx.NewDelete().
		Model((*inboundmessage.Mailbox)(nil)).
		Where(mailboxCols.OrganizationID.Eq(), org.ID).
		Where(mailboxCols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(mailboxCols.TokenHash.Eq(), hashutils.SHA256Hex(SeedInboundMailboxToken)).
		Exec(ctx); err != nil {
		return fmt.Errorf("retire the first seeded mailbox: %w", err)
	}

	return nil
}

func (s *InboundMessageSeed) loadRefs(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
) (*inboundSeedRefs, error) {
	admin, err := sc.GetUserByUsername(ctx, "admin")
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

	if refs.customers, err = loadSeedCustomers(ctx, tx, org); err != nil {
		return nil, err
	}
	if refs.shipments, err = loadSeedShipments(ctx, tx, org); err != nil {
		return nil, err
	}
	if refs.docTypes, err = loadSeedDocumentTypes(ctx, tx, org); err != nil {
		return nil, err
	}
	if refs.team, err = seedTeammates(ctx, tx, sc, org, admin, s.Name()); err != nil {
		return nil, err
	}
	if refs.carriers, err = loadSeedCarriers(ctx, tx, org); err != nil {
		return nil, err
	}
	if refs.driver, err = loadSeedDriver(ctx, tx, org); err != nil {
		return nil, err
	}
	if refs.broker, err = ensureSeedBroker(ctx, tx, sc, org); err != nil {
		return nil, err
	}
	if refs.lane, err = loadOpenLane(ctx, tx, refs.shipments[seedOpenLanePro]); err != nil {
		return nil, err
	}

	return refs, nil
}

func loadSeedCustomers(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (map[string]*customer.Customer, error) {
	cols := customercol.CustomerColumns
	rows := make([]*customer.Customer, 0, 8)
	if err := tx.NewSelect().
		Model(&rows).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Code.In(), bun.List([]string{"ACME", "FRSH", "GLBL", "RNGE", "PEAK"})).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load customers: %w", err)
	}

	byCode := make(map[string]*customer.Customer, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row
	}
	for _, code := range []string{"ACME", "FRSH", "GLBL", "RNGE", "PEAK"} {
		if _, ok := byCode[code]; !ok {
			return nil, fmt.Errorf("need seeded customer %s: %w", code, seedhelpers.ErrEntityNotFound)
		}
	}

	return byCode, nil
}

func loadSeedShipments(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (map[string]*shipment.Shipment, error) {
	cols := shipmentcol.ShipmentColumns
	rows := make([]*shipment.Shipment, 0, len(seedInboundShipmentPros))
	if err := tx.NewSelect().
		Model(&rows).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.ProNumber.In(), bun.List(seedInboundShipmentPros)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load shipments: %w", err)
	}

	byPro := make(map[string]*shipment.Shipment, len(rows))
	for _, row := range rows {
		byPro[row.ProNumber] = row
	}
	for _, pro := range seedInboundShipmentPros {
		if _, ok := byPro[pro]; !ok {
			return nil, fmt.Errorf("need seeded shipment %s: %w", pro, seedhelpers.ErrEntityNotFound)
		}
	}

	return byPro, nil
}

func loadSeedDocumentTypes(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (map[string]*documenttype.DocumentType, error) {
	cols := documenttypecol.DocumentTypeColumns
	rows := make([]*documenttype.DocumentType, 0, len(seedInboundDocumentTypeCodes))
	if err := tx.NewSelect().
		Model(&rows).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Code.In(), bun.List(seedInboundDocumentTypeCodes)).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("load document types: %w", err)
	}

	byCode := make(map[string]*documenttype.DocumentType, len(rows))
	for _, row := range rows {
		byCode[row.Code] = row
	}
	for _, code := range seedInboundDocumentTypeCodes {
		if _, ok := byCode[code]; !ok {
			return nil, fmt.Errorf("need document type %s: %w", code, seedhelpers.ErrEntityNotFound)
		}
	}

	return byCode, nil
}

func (s *InboundMessageSeed) insertMailboxes(
	ctx context.Context,
	tx bun.Tx,
	refs *inboundSeedRefs,
	secret string,
) (*seedMailboxes, error) {
	boxes := seedMailboxesFor(refs, secret)
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.intake, boxes.billing, boxes.claims} {
		if _, err := tx.NewInsert().Model(mailbox).Exec(ctx); err != nil {
			return nil, fmt.Errorf("insert mailbox %s: %w", mailbox.Name, err)
		}
	}

	return boxes, nil
}

func seedMailboxesFor(refs *inboundSeedRefs, secret string) *seedMailboxes {
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
			SigningSecret:  secret,
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
	tx bun.Tx,
	refs *inboundSeedRefs,
	boxes *seedMailboxes,
) error {
	rules := seedRules(refs, boxes)
	if len(rules) > 0 {
		paused := pausedRules(rules)
		if _, err := tx.NewInsert().Model(&rules).Exec(ctx); err != nil {
			return fmt.Errorf("insert triage rules: %w", err)
		}
		if err := pauseRules(ctx, tx, paused); err != nil {
			return err
		}
	}

	templates := seedReplyTemplates(refs)
	if _, err := tx.NewInsert().Model(&templates).Exec(ctx); err != nil {
		return fmt.Errorf("insert reply templates: %w", err)
	}

	messages := s.messages(refs, boxes, seedRuleSet(rules), templates)
	if err := insertSeededMessages(ctx, tx, messages); err != nil {
		return err
	}
	if err := markRuleMatches(ctx, tx, rules, messages); err != nil {
		return err
	}
	if err := markMailboxesReceived(ctx, tx, boxes, messages); err != nil {
		return err
	}

	extras := []any{
		triageSettingsFor(refs),
		ptrSlice(seedViews(refs)),
		ptrSlice(seedSenderHints(refs)),
	}
	for _, model := range extras {
		if _, err := tx.NewInsert().Model(model).Exec(ctx); err != nil {
			return fmt.Errorf("insert triage fixtures: %w", err)
		}
	}

	return nil
}

func ptrSlice[T any](items []T) *[]T {
	return &items
}

func insertSeededMessages(ctx context.Context, tx bun.Tx, messages []seededMessage) error {
	if err := insertSeededDocuments(ctx, tx, messages); err != nil {
		return err
	}

	activities := make([]*triage.Activity, 0, len(messages)*3)
	replies := make([]*triage.Reply, 0, 4)
	corrections := make([]*aicorrection.Correction, 0, 4)
	for _, seeded := range messages {
		if _, err := tx.NewInsert().Model(seeded.message).Exec(ctx); err != nil {
			return fmt.Errorf("insert message %q: %w", seeded.message.Subject, err)
		}
		if len(seeded.attachments) > 0 {
			if _, err := tx.NewInsert().Model(&seeded.attachments).Exec(ctx); err != nil {
				return fmt.Errorf("insert attachments for %q: %w", seeded.message.Subject, err)
			}
		}
		activities = append(activities, seeded.activities...)
		replies = append(replies, seeded.replies...)
		corrections = append(corrections, seeded.corrections...)
	}

	batches := []struct {
		what  string
		model any
		size  int
	}{
		{what: "replies", model: &replies, size: len(replies)},
		{what: "activities", model: &activities, size: len(activities)},
		{what: "AI corrections", model: &corrections, size: len(corrections)},
	}
	for _, batch := range batches {
		if batch.size == 0 {
			continue
		}
		if _, err := tx.NewInsert().Model(batch.model).Exec(ctx); err != nil {
			return fmt.Errorf("insert %s: %w", batch.what, err)
		}
	}

	return nil
}

func insertSeededDocuments(ctx context.Context, tx bun.Tx, messages []seededMessage) error {
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

	batches := []struct {
		what  string
		model any
		size  int
	}{
		{what: "documents", model: &documents, size: len(documents)},
		{what: "document readings", model: &contents, size: len(contents)},
		{what: "shipment drafts", model: &drafts, size: len(drafts)},
		{what: "AI extractions", model: &extractions, size: len(extractions)},
	}
	for _, batch := range batches {
		if batch.size == 0 {
			continue
		}
		if _, err := tx.NewInsert().Model(batch.model).Exec(ctx); err != nil {
			return fmt.Errorf("insert %s: %w", batch.what, err)
		}
	}

	return nil
}

func markMailboxesReceived(
	ctx context.Context,
	tx bun.Tx,
	boxes *seedMailboxes,
	messages []seededMessage,
) error {
	latest := make(map[pulid.ID]int64, 3)
	for _, seeded := range messages {
		if seeded.message.ReceivedAt > latest[seeded.message.MailboxID] {
			latest[seeded.message.MailboxID] = seeded.message.ReceivedAt
		}
	}

	cols := inboundmessagecol.MailboxColumns
	for _, mailbox := range []*inboundmessage.Mailbox{boxes.intake, boxes.billing, boxes.claims} {
		at, ok := latest[mailbox.ID]
		if !ok {
			continue
		}
		if _, err := tx.NewUpdate().
			Model((*inboundmessage.Mailbox)(nil)).
			Set(cols.LastReceivedAt.SetExpr("GREATEST(COALESCE({}, 0), ?::bigint)"), at).
			Where(cols.ID.Eq(), mailbox.ID).
			Where(cols.OrganizationID.Eq(), mailbox.OrganizationID).
			Where(cols.BusinessUnitID.Eq(), mailbox.BusinessUnitID).
			Exec(ctx); err != nil {
			return fmt.Errorf("mark mailbox %s received: %w", mailbox.Name, err)
		}
	}

	return nil
}
