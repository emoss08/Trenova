package development

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/buncolgen/permissioncol"
	"github.com/emoss08/trenova/pkg/buncolgen/tenantcol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantbootstrap"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/seedhelpers"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const (
	seedTeammatePassword = "admin123!" // #nosec G101 -- the published development login.

	seedTemplateETA      = "ETA update"
	seedTemplateWaiver   = "Detention waiver"
	seedTemplateAskPOD   = "Ask for the POD"
	seedTemplateAccept   = "Accept the tender"
	seedProviderIDPrefix = "seed-triage-"
	seedDisputeSender    = "claims@freshhaul.example"
	seedDuplicatePrefix  = "dup:"
	seedOverlapPrefix    = "load:"
	seedReadDelay        = int64(25)
	seedRuleDelay        = int64(40)
)

type seededMessage struct {
	message     *inboundmessage.InboundMessage
	attachments []*inboundmessage.InboundAttachment
	activities  []*triage.Activity
	replies     []*triage.Reply
	documents   []seededDocument
	corrections []*aicorrection.Correction
}

type seededDocument struct {
	document   *document.Document
	content    *documentcontent.Content
	draft      *documentshipmentdraft.DocumentShipmentDraft
	extraction *documentaiextraction.Extraction
}

type seedTeammateSpec struct {
	name     string
	username string
	email    string
}

var seedTeammateSpecs = []seedTeammateSpec{
	{name: "Dana Whitfield", username: "dana.whitfield", email: "dana.whitfield@trenova.app"},
	{name: "Marcus Lee", username: "marcus.lee", email: "marcus.lee@trenova.app"},
	{name: "Priya Shah", username: "priya.shah", email: "priya.shah@trenova.app"},
}

type seedTeammateParams struct {
	sc       *seedhelpers.SeedContext
	org      *tenant.Organization
	admin    *tenant.User
	role     *permission.Role
	spec     seedTeammateSpec
	seedName string
}

func seedTeammates(
	ctx context.Context,
	tx bun.Tx,
	sc *seedhelpers.SeedContext,
	org *tenant.Organization,
	admin *tenant.User,
	seedName string,
) (seedTeam, error) {
	role, err := administratorRole(ctx, tx, org)
	if err != nil {
		return seedTeam{}, err
	}

	users := make([]*tenant.User, 0, len(seedTeammateSpecs))
	for _, spec := range seedTeammateSpecs {
		user, userErr := seedTeammate(ctx, tx, seedTeammateParams{
			sc:       sc,
			org:      org,
			admin:    admin,
			role:     role,
			spec:     spec,
			seedName: seedName,
		})
		if userErr != nil {
			return seedTeam{}, userErr
		}
		users = append(users, user)
	}

	return seedTeam{dana: users[0], marcus: users[1], priya: users[2]}, nil
}

func administratorRole(
	ctx context.Context,
	tx bun.Tx,
	org *tenant.Organization,
) (*permission.Role, error) {
	cols := permissioncol.RoleColumns
	role := new(permission.Role)
	if err := tx.NewSelect().
		Model(role).
		Where(cols.OrganizationID.Eq(), org.ID).
		Where(cols.BusinessUnitID.Eq(), org.BusinessUnitID).
		Where(cols.Name.Eq(), tenantbootstrap.AdministratorRoleName).
		Limit(1).
		Scan(ctx); err != nil {
		return nil, fmt.Errorf("find the administrator role: %w", err)
	}

	return role, nil
}

func seedTeammate(ctx context.Context, tx bun.Tx, p seedTeammateParams) (*tenant.User, error) {
	existing := new(tenant.User)
	err := tx.NewSelect().
		Model(existing).
		Where(tenantcol.UserColumns.Username.Eq(), p.spec.username).
		Limit(1).
		Scan(ctx)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("look up %s: %w", p.spec.username, err)
	}

	user, err := p.sc.CreateUser(ctx, tx, &seedhelpers.UserOptions{
		OrganizationID: p.org.ID,
		BusinessUnitID: p.org.BusinessUnitID,
		Name:           p.spec.name,
		Username:       p.spec.username,
		Email:          p.spec.email,
		Password:       seedTeammatePassword,
		Status:         domaintypes.StatusActive,
		Timezone:       p.org.Timezone,
	}, p.seedName)
	if err != nil {
		return nil, err
	}

	now := user.CreatedAt
	membership := &tenant.OrganizationMembership{
		BusinessUnitID: p.org.BusinessUnitID,
		OrganizationID: p.org.ID,
		UserID:         user.ID,
		JoinedAt:       now,
		GrantedByID:    p.admin.ID,
		IsDefault:      true,
	}
	if _, err = tx.NewInsert().Model(membership).Exec(ctx); err != nil {
		return nil, fmt.Errorf("add %s to the organization: %w", p.spec.username, err)
	}
	if err = p.sc.TrackCreated(
		ctx, "user_organization_memberships", membership.ID, p.seedName,
	); err != nil {
		return nil, err
	}

	if err = tenantbootstrap.AssignRole(ctx, tx, tenantbootstrap.RoleAssignmentParams{
		UserID:         user.ID,
		OrganizationID: p.org.ID,
		RoleID:         p.role.ID,
		AssignedBy:     p.admin.ID,
		AssignedAt:     now,
		Record: func(ctx context.Context, table string, id pulid.ID) error {
			return p.sc.TrackCreated(ctx, table, id, p.seedName)
		},
	}); err != nil {
		return nil, err
	}

	return user, nil
}

type triageSeedRules struct {
	pods        *triage.Rule
	invoices    *triage.Rule
	newsletters *triage.Rule
	disputes    *triage.Rule
}

type seedRuleSpec struct {
	class    inboundmessage.Classification
	scope    triage.SenderScope
	sender   string
	mailbox  *inboundmessage.Mailbox
	action   inboundmessage.NovaAction
	assignee *tenant.User
	matches  int64
	enabled  bool
	by       *tenant.User
}

func seedRules(refs *inboundSeedRefs, boxes *seedMailboxes) []*triage.Rule {
	specs := []seedRuleSpec{
		{
			class:   inboundmessage.ClassificationProofOfDelivery,
			scope:   triage.SenderDomain,
			sender:  "drivers.trenova.example",
			mailbox: boxes.intake,
			action:  inboundmessage.NovaActionFileOnShipment,
			matches: 142,
			enabled: true,
			by:      refs.team.dana,
		},
		{
			class:   inboundmessage.ClassificationInvoice,
			scope:   triage.SenderAnyone,
			mailbox: boxes.billing,
			action:  inboundmessage.NovaActionFileAndQueuePay,
			matches: 61,
			enabled: true,
			by:      refs.team.dana,
		},
		{
			class:   inboundmessage.ClassificationOther,
			scope:   triage.SenderDomain,
			sender:  "freightweekly.example",
			mailbox: boxes.intake,
			action:  inboundmessage.NovaActionIgnore,
			matches: 210,
			enabled: true,
			by:      refs.admin,
		},
		{
			class:   inboundmessage.ClassificationStatusRequest,
			scope:   triage.SenderAnyCustomer,
			mailbox: boxes.intake,
			action:  inboundmessage.NovaActionDraftReply,
			by:      refs.admin,
		},
		{
			class:    inboundmessage.ClassificationDetentionDispute,
			scope:    triage.SenderAddress,
			sender:   seedDisputeSender,
			mailbox:  boxes.intake,
			action:   inboundmessage.NovaActionAssign,
			assignee: refs.team.dana,
			matches:  9,
			enabled:  true,
			by:       refs.team.dana,
		},
	}

	rules := make([]*triage.Rule, 0, len(specs))
	for _, spec := range specs {
		assignee := pulid.Nil
		if spec.assignee != nil {
			assignee = spec.assignee.ID
		}
		rules = append(rules, &triage.Rule{
			ID:             pulid.MustNew("trule_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			Classification: spec.class,
			SenderScope:    spec.scope,
			Sender:         spec.sender,
			MailboxID:      spec.mailbox.ID,
			Action:         spec.action,
			AssigneeID:     assignee,
			Enabled:        spec.enabled,
			CreatedByID:    spec.by.ID,
			MatchCount:     spec.matches,
		})
	}

	return rules
}

func seedRuleSet(rules []*triage.Rule) triageSeedRules {
	return triageSeedRules{
		pods:        rules[0],
		invoices:    rules[1],
		newsletters: rules[2],
		disputes:    rules[4],
	}
}

func seedReplyTemplates(refs *inboundSeedRefs) []*triage.ReplyTemplate {
	template := func(name, body string, by *tenant.User) *triage.ReplyTemplate {
		return &triage.ReplyTemplate{
			ID:             pulid.MustNew("ttpl_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			Name:           name,
			Body:           body,
			CreatedByID:    by.ID,
		}
	}

	return []*triage.ReplyTemplate{
		template(seedTemplateETA,
			"Hi {first} — {pro} ({po}) is {status}. Current ETA is {eta}. "+
				"We'll tell you if that moves.\n\nThanks,\nTrenova",
			refs.team.priya),
		template(seedTemplateWaiver,
			"Hi {first} — you're right. We've removed the detention on {pro} for {customer}; "+
				"a corrected invoice follows today.\n\nThanks,\nTrenova Billing",
			refs.team.dana),
		template(seedTemplateAskPOD,
			"Hi {first} — could you send the signed POD for {pro}? "+
				"A clear photo of each page works; reply to this email from {sender}."+
				"\n\nThanks,\nTrenova",
			refs.admin),
		template(seedTemplateAccept,
			"Hi {first} — we'll take it for {customer}. Confirming {lane}. "+
				"Rate confirmation to follow.\n\nThanks,\nTrenova Dispatch",
			refs.team.marcus),
	}
}

func triageSettingsFor(refs *inboundSeedRefs) *triage.Settings {
	settings := triage.DefaultSettings(refs.org.ID, refs.org.BusinessUnitID)
	settings.OnCallUserID = refs.team.marcus.ID

	return settings
}

func seedViews(refs *inboundSeedRefs) []*triage.View {
	view := func(name, query string, position int) *triage.View {
		return &triage.View{
			ID:             pulid.MustNew("tview_"),
			OrganizationID: refs.org.ID,
			BusinessUnitID: refs.org.BusinessUnitID,
			UserID:         refs.admin.ID,
			Name:           name,
			Query:          query,
			Position:       position,
		}
	}

	return []*triage.View{
		view("Acme Manufacturing", "acme", 0),
		view("Has a PRO", "SEED-", 1),
	}
}

type seedCorrection struct {
	domain string
	from   inboundmessage.Classification
	to     inboundmessage.Classification
}

var seedCorrectionsThisWeek = []seedCorrection{
	{
		domain: "peakdistributing.example",
		from:   inboundmessage.ClassificationOther,
		to:     inboundmessage.ClassificationStatusRequest,
	},
	{
		domain: "summitcarriers.example",
		from:   inboundmessage.ClassificationRateConfirmation,
		to:     inboundmessage.ClassificationInvoice,
	},
	{
		domain: "rangelogistics.example",
		from:   inboundmessage.ClassificationStatusRequest,
		to:     inboundmessage.ClassificationTender,
	},
}

var seedCorrectionLastWeek = seedCorrection{
	domain: "globaltrade.example",
	from:   inboundmessage.ClassificationInvoice,
	to:     inboundmessage.ClassificationRateConfirmation,
}

func (r *inboundSeedRefs) correctedAt(index int) int64 {
	span := r.now - r.weekStart
	return r.weekStart + span*int64(index+1)/int64(len(seedCorrectionsThisWeek)+1)
}

func (r *inboundSeedRefs) lastWeekCorrectedAt() int64 {
	return r.weekStart - 3*inboundSeedDay + 10*inboundSeedHour
}

func seedSenderHints(refs *inboundSeedRefs) []*triage.SenderHint {
	hints := make([]*triage.SenderHint, 0, len(seedCorrectionsThisWeek)+1)
	hint := func(c seedCorrection, at int64) *triage.SenderHint {
		return &triage.SenderHint{
			OrganizationID:  refs.org.ID,
			BusinessUnitID:  refs.org.BusinessUnitID,
			SenderDomain:    c.domain,
			Classification:  c.to,
			Corrections:     1,
			LastCorrectedAt: at + inboundSeedHour,
		}
	}
	for i, correction := range seedCorrectionsThisWeek {
		hints = append(hints, hint(correction, refs.correctedAt(i)))
	}

	return append(hints, hint(seedCorrectionLastWeek, refs.lastWeekCorrectedAt()))
}

func pausedRules(rules []*triage.Rule) []*triage.Rule {
	paused := make([]*triage.Rule, 0, 1)
	for _, rule := range rules {
		if !rule.Enabled {
			paused = append(paused, rule)
		}
	}

	return paused
}

func pauseRules(ctx context.Context, tx bun.Tx, paused []*triage.Rule) error {
	cols := triagecol.RuleColumns
	for _, rule := range paused {
		rule.Enabled = false
		if _, err := tx.NewUpdate().
			Model((*triage.Rule)(nil)).
			Set(cols.Enabled.Set(), false).
			Where(cols.ID.Eq(), rule.ID).
			Where(cols.OrganizationID.Eq(), rule.OrganizationID).
			Where(cols.BusinessUnitID.Eq(), rule.BusinessUnitID).
			Exec(ctx); err != nil {
			return fmt.Errorf("pause rule %s: %w", rule.ID, err)
		}
	}

	return nil
}

func matchedBy(rule *triage.Rule, message *inboundmessage.InboundMessage) bool {
	if rule.Action == inboundmessage.NovaActionAssign {
		return rule.Covers(message) && message.OwnerID == rule.AssigneeID
	}

	return message.HandledByRuleID == rule.ID
}

func markRuleMatches(
	ctx context.Context,
	tx bun.Tx,
	rules []*triage.Rule,
	messages []seededMessage,
) error {
	cols := triagecol.RuleColumns
	for _, rule := range rules {
		var latest *inboundmessage.InboundMessage
		for _, seeded := range messages {
			message := seeded.message
			if !matchedBy(rule, message) {
				continue
			}
			if latest == nil || message.ReceivedAt > latest.ReceivedAt {
				latest = message
			}
		}
		if latest == nil {
			continue
		}

		if _, err := tx.NewUpdate().
			Model((*triage.Rule)(nil)).
			Set(cols.LastMatchedAt.Set(), latest.ReceivedAt+seedRuleDelay).
			Set(cols.LastMessageID.Set(), latest.ID).
			Where(cols.ID.Eq(), rule.ID).
			Where(cols.OrganizationID.Eq(), rule.OrganizationID).
			Where(cols.BusinessUnitID.Eq(), rule.BusinessUnitID).
			Exec(ctx); err != nil {
			return fmt.Errorf("mark rule %s matched: %w", rule.ID, err)
		}
	}

	return nil
}

type triageSeed struct {
	refs      *inboundSeedRefs
	boxes     *seedMailboxes
	rules     triageSeedRules
	templates map[string]*triage.ReplyTemplate
	out       []seededMessage
}

type seedMail struct {
	mailbox    *inboundmessage.Mailbox
	key        string
	from       string
	fromName   string
	subject    string
	body       string
	receivedAt int64
	class      inboundmessage.Classification
	confidence float64
}

func (s *InboundMessageSeed) messages(
	refs *inboundSeedRefs,
	boxes *seedMailboxes,
	rules triageSeedRules,
	templates []*triage.ReplyTemplate,
) []seededMessage {
	byName := make(map[string]*triage.ReplyTemplate, len(templates))
	for _, template := range templates {
		byName[template.Name] = template
	}

	b := &triageSeed{
		refs:      refs,
		boxes:     boxes,
		rules:     rules,
		templates: byName,
		out:       make([]seededMessage, 0, 64),
	}
	b.waitingOnYou()
	b.threads()
	b.parked()
	b.settledEarlier()
	b.settledByNovaToday()
	b.correctedThisWeek()
	b.lastWeek()

	return b.out
}

func (b *triageSeed) minutesAgo(minutes int64) int64 {
	return b.refs.now - minutes*inboundSeedMinute
}

func (b *triageSeed) shipment(pro string) *shipment.Shipment {
	return b.refs.shipments[pro]
}

func (b *triageSeed) customer(code string) *customer.Customer {
	return b.refs.customers[code]
}

func (b *triageSeed) docType(code string) *documenttype.DocumentType {
	return b.refs.docTypes[code]
}

func (b *triageSeed) mail(m seedMail) *inboundmessage.InboundMessage {
	domain := emailutils.Domain(m.from)
	message := &inboundmessage.InboundMessage{
		ID:                pulid.MustNew("imsg_"),
		OrganizationID:    b.refs.org.ID,
		BusinessUnitID:    b.refs.org.BusinessUnitID,
		MailboxID:         m.mailbox.ID,
		ProviderMessageID: seedProviderIDPrefix + m.key,
		MessageID:         "<" + m.key + "@" + domain + ">",
		FromAddress:       m.from,
		FromName:          m.fromName,
		ToAddresses:       []string{m.mailbox.Address},
		Subject:           m.subject,
		TextBody:          m.body,
		ReceivedAt:        m.receivedAt,
		Classification:    m.class,
		Confidence:        m.confidence,
		Status:            inboundmessage.StatusInReview,
		SenderDomain:      domain,
		NovaRead:          m.class != "",
	}
	message.ThreadKey = inboundmessage.DeriveThreadKey(message.MessageID, message.ID)

	return message
}

func continues(message, parent, root *inboundmessage.InboundMessage) {
	message.InReplyTo = parent.MessageID
	message.References = []string{root.MessageID}
	if parent != root {
		message.References = append(message.References, parent.MessageID)
	}
	message.ThreadKey = root.ThreadKey
}

func matched(
	message *inboundmessage.InboundMessage,
	ship *shipment.Shipment,
	cust *customer.Customer,
	reason string,
) {
	if ship != nil {
		message.MatchedShipmentID = ship.ID
	}
	if cust != nil {
		message.MatchedCustomerID = cust.ID
	}
	message.MatchReason = reason
}

func seedActivity(
	message *inboundmessage.InboundMessage,
	kind triage.ActivityKind,
	at int64,
) *triage.Activity {
	activity := triage.NewActivity(message, kind)
	activity.CreatedAt = at

	return activity
}

func (b *triageSeed) add(seeded seededMessage) {
	message := seeded.message
	for _, attachment := range seeded.attachments {
		attachment.MessageID = message.ID
		attachment.OrganizationID = message.OrganizationID
		attachment.BusinessUnitID = message.BusinessUnitID
	}
	if message.NovaRead {
		read := seedActivity(message, triage.ActivityNovaRead, message.ReceivedAt+seedReadDelay).
			ByNova().
			About(string(message.Classification), "")
		seeded.activities = append([]*triage.Activity{read}, seeded.activities...)
	}
	if message.NeedsReview() && message.DueAt == 0 {
		message.ApplyResponseTarget(triage.DefaultTargetMinutes(message.Classification))
	}
	b.out = append(b.out, seeded)
}

func (b *triageSeed) assigned(
	message *inboundmessage.InboundMessage,
	owner *tenant.User,
	at int64,
) *triage.Activity {
	message.OwnerID = owner.ID
	activity := seedActivity(message, triage.ActivityAssigned, at).
		ByPerson(b.refs.admin.ID).
		About(owner.Name, "")
	activity.RecordID = owner.ID

	return activity
}

func settledBy(
	message *inboundmessage.InboundMessage,
	user *tenant.User,
	at int64,
	note string,
) *triage.Activity {
	kind := triage.ActivityHandled
	if message.Status == inboundmessage.StatusIgnored {
		kind = triage.ActivityIgnored
	}
	message.ReviewedBy = user.ID
	message.ReviewedAt = at
	message.ReviewNote = note

	return seedActivity(message, kind, at).ByPerson(user.ID).About("", note)
}

func settledByRule(
	message *inboundmessage.InboundMessage,
	rule *triage.Rule,
	note string,
) *triage.Activity {
	at := message.ReceivedAt + seedRuleDelay
	message.Status = inboundmessage.StatusActioned
	if rule.Action == inboundmessage.NovaActionIgnore {
		message.Status = inboundmessage.StatusIgnored
	}
	message.HandledByRuleID = rule.ID
	message.HandledAction = rule.Action
	message.ReviewedAt = at
	message.ReviewNote = note

	activity := seedActivity(message, triage.ActivityRuleApplied, at).
		ByNova().
		About(string(rule.Action), message.SenderDomain)
	activity.RecordID = rule.ID

	return activity
}

type seedFiling struct {
	message    *inboundmessage.InboundMessage
	attachment *inboundmessage.InboundAttachment
	shipment   *shipment.Shipment
	docType    *documenttype.DocumentType
	by         *tenant.User
	at         int64
}

func filed(f seedFiling) *triage.Activity {
	f.attachment.FiledShipmentID = f.shipment.ID
	f.attachment.FiledAs = f.docType.Name
	f.attachment.FiledAt = f.at
	activity := seedActivity(f.message, triage.ActivityFiled, f.at).
		About(f.attachment.FileName, f.docType.Name)
	activity.RecordID = f.shipment.ID
	if f.by == nil {
		return activity.ByNova()
	}
	f.attachment.FiledBy = f.by.ID

	return activity.ByPerson(f.by.ID)
}

func corrected(
	message *inboundmessage.InboundMessage,
	correction seedCorrection,
	by *tenant.User,
	at int64,
) *triage.Activity {
	message.Classification = correction.to
	message.ClassificationCorrected = true
	activity := seedActivity(message, triage.ActivityCorrected, at).
		ByPerson(by.ID).
		About(string(correction.to), message.SenderDomain)
	activity.FromClassification = correction.from
	activity.ToClassification = correction.to

	return activity
}

func fact(key, label, value, quote string) inboundmessage.Fact {
	return inboundmessage.Fact{
		Key:   key,
		Label: label,
		Value: value,
		Quote: quote,
		Tone:  inboundmessage.FactToneNeutral,
	}
}

func warning(key, label, value, quote string) inboundmessage.Fact {
	f := fact(key, label, value, quote)
	f.Tone = inboundmessage.FactToneWarning

	return f
}

func (b *triageSeed) dateLabel(at int64) string {
	return time.Unix(at, 0).In(b.refs.loc).Format("Jan 2")
}

func (b *triageSeed) tomorrowAt(hour int) int64 {
	today := time.Unix(b.refs.now, 0).In(b.refs.loc)

	return time.Date(today.Year(), today.Month(), today.Day()+1, hour, 0, 0, 0, b.refs.loc).Unix()
}

func (b *triageSeed) waitingOnYou() {
	b.carrierInvoiceWaiting()
	b.tenderWithoutAttachment()
	b.unreadableEDI()
	b.heldBack()
	b.rateConfirmationOnFile()
	b.proofOfDeliveryWithAShortage()
	b.tenderWithLoads()
	b.blockedSender()
}

func (b *triageSeed) tenderWithoutAttachment() {
	body := "Hi team —\n\nWe have a dry van load ready Thursday morning. 42,000 lbs, " +
		"no appointment needed at origin, delivery is a 2-hour window. " +
		"Rate confirmation to follow once you accept.\n\n" +
		"Origin is our Chicago DC, 2400 S Halsted and it delivers to Columbus OH.\n\n" +
		"Thanks,\nMaya — Range Logistics dispatch"
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "tender-chicago-columbus",
		from:       "dispatch@rangelogistics.example",
		fromName:   "Range Logistics Dispatch",
		subject:    "New load: Chicago IL to Columbus OH, pickup Thursday 08:00",
		body:       body,
		receivedAt: b.minutesAgo(1102),
		class:      inboundmessage.ClassificationTender,
		confidence: 0.68,
	})
	matched(message, nil, b.customer("RNGE"),
		"Sender domain rangelogistics.example belongs to Range Logistics, a customer.")
	message.ReviewNote = "Creating a load is not something Nova does on its own."
	message.Facts = []inboundmessage.Fact{
		fact("equipment", "Equipment", "Dry van", "dry van"),
		fact("pickup", "Pickup", "Thursday 08:00", "Thursday morning"),
		fact("weight", "Weight", "42,000 lbs", "42,000 lbs"),
		fact("delivery", "Delivery window", "2-hour window", "2-hour window"),
		fact("origin", "Origin", "Chicago DC, 2400 S Halsted", "Chicago DC, 2400 S Halsted"),
		fact("destination", "Destination", "Columbus OH", "Columbus OH"),
	}

	b.add(seededMessage{message: message})
}

func (b *triageSeed) unreadableEDI() {
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "edi-status-20260919-0441",
		from:       "edi@partnercarrier.example",
		fromName:   "Partner Carrier EDI",
		subject:    "Automated: load status file 20260919-0441",
		body:       "ST*214*0441~B10*PC0441**SCAC~LX*1~AT7*X3*NS***20260919*0412~MS1*COLUMBUS*OH~",
		receivedAt: b.minutesAgo(1462),
		class:      inboundmessage.ClassificationStatusRequest,
		confidence: 0.41,
	})
	message.FailureCode = "BODY_TRUNCATED"
	message.FailureText = "The body was cut off by the sending system, " +
		"so Nova could not read which load it is about."
	message.ReviewNote = "Nova could not tell what this is about, so it is waiting on a person."

	failedAt := message.ReceivedAt + 35*inboundSeedMinute
	bounced := b.reply(seedReply{
		message: message,
		body: "Hi — your status file 20260919-0441 arrived cut off after the first stop. " +
			"Could you send it again?\n\nThanks,\nTrenova",
		at:        failedAt,
		failure:   "550 5.1.1 <edi@partnercarrier.example>: Recipient address rejected: " +
			"this mailbox only sends.",
		suggested: "ops@partnercarrier.example",
		by:        b.refs.team.marcus,
	})
	failed := seedActivity(message, triage.ActivityReplyFailed, failedAt).
		ByPerson(b.refs.team.marcus.ID).
		About(bounced.ToAddress, bounced.FailureText)
	failed.RecordID = bounced.ID

	b.add(seededMessage{
		message:    message,
		replies:    []*triage.Reply{bounced},
		activities: []*triage.Activity{failed},
	})
}

func (b *triageSeed) heldBack() {
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "verify-account",
		from:     "noreply@unknownsender.example",
		fromName: "",
		subject:  "",
		body: "Your shipping account requires verification. Sign in to keep receiving loads: " +
			"hxxp://trenova-verify.example/login",
		receivedAt: b.minutesAgo(5302),
		class:      inboundmessage.ClassificationOther,
		confidence: 0.22,
	})
	message.Status = inboundmessage.StatusQuarantined
	message.SpamScore = 7.4
	message.ReviewNote = "Held back: the sender is not on this mailbox's allow list and " +
		"the message carried a link to a sign-in page."

	b.add(seededMessage{message: message})
}

func (b *triageSeed) rateConfirmationOnFile() {
	ship := b.shipment("SEED-SHP-003")
	rateCon := b.docType("RATECONF")
	body := "Please find the signed rate confirmation for SEED-SHP-003 attached. " +
		"$1,240 all-in, Columbus OH → Louisville KY.\n\n" +
		"Sign and return within 2 hours to hold the truck."
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "ratecon-seed-shp-003",
		from:       seedBrokerSender,
		fromName:   "Blue Ridge Brokerage",
		subject:    "Rate con — SEED-SHP-003 Columbus to Louisville",
		body:       body,
		receivedAt: b.minutesAgo(54),
		class:      inboundmessage.ClassificationRateConfirmation,
		confidence: 0.83,
	})
	matched(message, ship, b.refs.broker,
		"The PRO number SEED-SHP-003 is in the subject, and the attached PDF names the same lane.")
	message.ReviewNote = "Filing a document on a shipment waits for a person on this mailbox."
	message.Facts = []inboundmessage.Fact{
		fact("shipment", "Shipment", "SEED-SHP-003", "SEED-SHP-003"),
		fact("rate", "Rate", "$1,240.00 all-in", "$1,240 all-in"),
		fact("lane", "Lane", "Columbus OH → Louisville KY", "Columbus OH → Louisville KY"),
	}

	attachment := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    "RateCon_SEED-SHP-003_signed.pdf",
		ContentType: "application/pdf",
		ByteSize:    212 * 1024,
		Kind:        inboundmessage.AttachmentRateConfirmation,
		PageCount:   2,
		Pages: []inboundmessage.AttachmentPage{
			{Number: 1, Label: "Rate confirmation", SuggestedType: rateCon.Name},
			{Number: 2, Label: "Signed BOL", SuggestedType: b.docType("BOL").Name},
		},
	}
	incoming := b.stage(seedStaging{
		message:    message,
		attachment: attachment,
		docType:    rateCon,
		kind:       "RateConfirmation",
		rate:       "$1,240.00",
		fields: []documentcontent.FoundField{
			{Key: "load", Label: "Load", Value: "SEED-SHP-003", Confidence: 0.97},
			{Key: "rate", Label: "Rate", Value: "$1,240.00 all-in", Confidence: 0.96},
			{Key: "equipment", Label: "Equipment", Value: "53' dry van", Confidence: 0.92},
			{Key: "signed", Label: "Signed", Value: "Yes", Confidence: 0.9},
		},
	})
	onFile := b.stage(seedStaging{
		message:  message,
		resource: ship,
		docType:  rateCon,
		kind:     "RateConfirmation",
		rate:     "$1,180.00",
		file: seedFile{
			name:        "RateCon_SEED-SHP-003.pdf",
			contentType: "application/pdf",
			size:        198 * 1024,
			pages:       1,
		},
		uploadedAt: b.minutesAgo(1440 + 300),
		fields: []documentcontent.FoundField{
			{Key: "load", Label: "Load", Value: "SEED-SHP-003", Confidence: 0.97},
			{Key: "rate", Label: "Rate", Value: "$1,180.00 all-in", Confidence: 0.95},
		},
	})
	message.Conflicts = []inboundmessage.Conflict{{
		ID:                 seedDuplicatePrefix + attachment.ID.String(),
		Kind:               inboundmessage.ConflictDuplicateDocument,
		AttachmentID:       attachment.ID,
		ExistingDocumentID: onFile.document.ID,
		ExistingShipmentID: ship.ID,
		ExistingFiledAt:    onFile.document.CreatedAt,
		ExistingAmount:     "1180.00",
		NewAmount:          "1240.00",
	}}

	b.add(seededMessage{
		message:     message,
		attachments: []*inboundmessage.InboundAttachment{attachment},
		documents:   []seededDocument{incoming, onFile},
	})
}

func (b *triageSeed) proofOfDeliveryWithAShortage() {
	driver := b.refs.driver
	driverName := driver.FirstName + " " + driver.LastName
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "pod-seed-shp-005",
		from:     driver.Email,
		fromName: driverName,
		subject:  "POD SEED-SHP-005",
		body: "Delivered, signed by receiving. 2 pallets short-shipped noted on the BOL.\n\n" +
			"Sent from my phone",
		receivedAt: b.minutesAgo(196),
		class:      inboundmessage.ClassificationProofOfDelivery,
		confidence: 0.94,
	})
	message.CcAddresses = []string{"dispatch@dev.trenova.app"}
	matched(message, b.shipment("SEED-SHP-005"), b.customer("ACME"),
		"The driver is assigned to SEED-SHP-005 and the subject names it.")
	message.ReviewNote = "Nova held this back from the POD rule: one file was refused and " +
		"the driver noted a shortage."
	message.Facts = []inboundmessage.Fact{
		fact("shipment", "Shipment", "SEED-SHP-005", "SEED-SHP-005"),
		warning("exception", "Exceptions", "2 pallets short",
			"2 pallets short-shipped noted on the BOL"),
	}

	photo := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    "IMG_4410.jpg",
		ContentType: "image/jpeg",
		ByteSize:    2_400 * 1024,
		Kind:        inboundmessage.AttachmentProofOfDelivery,
		PageCount:   1,
	}
	reading := b.stage(seedStaging{
		message:    message,
		attachment: photo,
		docType:    b.docType(documenttype.CodePOD),
		kind:       "ProofOfDelivery",
		fields: []documentcontent.FoundField{
			{Key: "bol", Label: "BOL", Value: "BOL-2026-0005", Confidence: 0.91},
			{Key: "received_by", Label: "Received by", Value: "R. Patel", Confidence: 0.88},
			{Key: "signature", Label: "Signature", Value: "Present", Confidence: 0.93},
			{
				Key:            "exceptions",
				Label:          "Exceptions",
				Value:          "2 pallets short",
				Confidence:     0.86,
				ReviewRequired: true,
			},
		},
	})

	b.add(seededMessage{
		message: message,
		attachments: []*inboundmessage.InboundAttachment{
			photo,
			{
				ID:          pulid.MustNew("imsga_"),
				FileName:    "IMG_4411.HEIC",
				ContentType: "image/heic",
				ByteSize:    31_800 * 1024,
				Kind:        inboundmessage.AttachmentUnknown,
				FailureText: "Refused: larger than this mailbox's 25 MB limit per file.",
			},
		},
		documents: []seededDocument{reading},
		activities: []*triage.Activity{
			b.assigned(message, b.refs.admin, message.ReceivedAt+12*inboundSeedMinute),
		},
	})
}

func (b *triageSeed) tenderWithLoads() {
	lane := b.refs.lane
	firstDay := b.dateOnly(lane.pickupAt - inboundSeedDay)
	secondDay := lane.pickupDay(b.refs.loc)
	thirdDay := b.dateOnly(lane.pickupAt + 2*inboundSeedDay)
	weekOf := b.dateLabel(lane.pickupAt)
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "tender-4471",
		from:     "logistics@acme.example",
		fromName: "Acme Manufacturing",
		subject:  "Tender 4471 — 3 loads week of " + weekOf,
		body: "Attached is tender 4471 for three dry van loads the week of " + weekOf +
			". Please confirm by end of day Thursday.",
		receivedAt: b.minutesAgo(312),
		class:      inboundmessage.ClassificationTender,
		confidence: 0.72,
	})
	matched(message, nil, b.customer("ACME"),
		"Sender domain acme.example matches Acme Manufacturing.")
	message.ReviewNote = "Creating a load is not something Nova does on its own."
	message.Facts = []inboundmessage.Fact{
		fact("tender", "Tender", "4471", "tender 4471"),
		fact("loads", "Loads", "3", "three dry van loads"),
		fact("respondBy", "Respond by", "Thursday, end of day", "end of day Thursday"),
	}

	attachment := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    "tender_4471.pdf",
		ContentType: "application/pdf",
		ByteSize:    164 * 1024,
		Kind:        inboundmessage.AttachmentOther,
		PageCount:   3,
		Pages: []inboundmessage.AttachmentPage{
			{Number: 1, Label: "Cover letter", SuggestedType: "Other"},
			{Number: 2, Label: "Load sheet", SuggestedType: "Tender"},
			{Number: 3, Label: "BOL template", SuggestedType: b.docType("BOL").Name},
		},
	}
	reading := b.stage(seedStaging{
		message:    message,
		attachment: attachment,
		kind:       "Tender",
		fields: []documentcontent.FoundField{
			{Key: "tender", Label: "Tender", Value: "4471", Confidence: 0.95, PageNumber: 1},
			{Key: "loads", Label: "Loads", Value: "3", Confidence: 0.93, PageNumber: 2},
			{Key: "equipment", Label: "Equipment", Value: "Dry van", Confidence: 0.9, PageNumber: 2},
			{Key: "respond_by", Label: "Respond by", Value: "Thursday", Confidence: 0.82, PageNumber: 1},
		},
		stops: []documentcontent.FoundStop{
			{Sequence: 1, Role: "pickup", City: "Chicago", State: "IL", Date: firstDay, PageNumber: 2},
			{Sequence: 2, Role: "delivery", City: "Columbus", State: "OH", PageNumber: 2},
			{
				Sequence:   3,
				Role:       "pickup",
				City:       lane.originCity,
				State:      lane.originState,
				Date:       secondDay,
				PageNumber: 2,
			},
			{
				Sequence:   4,
				Role:       "delivery",
				City:       lane.destinationCity,
				State:      lane.destinationState,
				PageNumber: 2,
			},
			{Sequence: 5, Role: "pickup", City: "Toledo", State: "OH", Date: thirdDay, PageNumber: 2},
			{Sequence: 6, Role: "delivery", City: "Lansing", State: "MI", PageNumber: 2},
		},
	})
	draftedAt := message.ReceivedAt + 2*inboundSeedMinute
	message.LoadDraft = &inboundmessage.LoadDraft{
		Source: attachment.FileName,
		Fields: []inboundmessage.LoadDraftField{
			{Key: inboundmessage.LoadFieldCustomer, Value: "Acme Manufacturing"},
			{Key: inboundmessage.LoadFieldReference, Value: "Tender 4471"},
			{Key: inboundmessage.LoadFieldLoads, Value: "3"},
			{Key: inboundmessage.LoadFieldWeekOf, Value: weekOf},
			{Key: inboundmessage.LoadFieldEquipment, Value: "Dry van"},
			{Key: inboundmessage.LoadFieldRespondBy, Value: "Thursday, end of day"},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Chicago, IL"},
			{Key: inboundmessage.LoadFieldDestination, Value: "Columbus, OH"},
			{Key: inboundmessage.LoadFieldPickup, Value: firstDay},
			{Key: inboundmessage.LoadFieldOrigin, Value: lane.origin},
			{Key: inboundmessage.LoadFieldDestination, Value: lane.destination},
			{Key: inboundmessage.LoadFieldPickup, Value: secondDay},
			{Key: inboundmessage.LoadFieldOrigin, Value: "Toledo, OH"},
			{Key: inboundmessage.LoadFieldDestination, Value: "Lansing, MI"},
			{Key: inboundmessage.LoadFieldPickup, Value: thirdDay},
		},
		DraftedAt:     draftedAt,
		DraftedByNova: true,
	}
	message.Conflicts = []inboundmessage.Conflict{{
		ID:                 seedOverlapPrefix + "2",
		Kind:               inboundmessage.ConflictOverlappingLoad,
		ExistingShipmentID: lane.shipment.ID,
		LoadIndex:          2,
	}}

	b.add(seededMessage{
		message:     message,
		attachments: []*inboundmessage.InboundAttachment{attachment},
		documents:   []seededDocument{reading},
		activities: []*triage.Activity{
			seedActivity(message, triage.ActivityLoadDrafted, draftedAt).
				ByNova().
				About(attachment.FileName, "15 fields"),
		},
	})
}

func (b *triageSeed) blockedSender() {
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "loadblaster-hot-loads",
		from:       "deals@deals.loadblaster.example",
		fromName:   "LoadBlaster Deals",
		subject:    "500 hot loads waiting — upgrade today",
		body:       "Unlock 500 hot loads in your lanes. Upgrade to LoadBlaster Pro today.",
		receivedAt: b.minutesAgo(140),
	})
	message.Status = inboundmessage.StatusIgnored
	message.ReviewNote = "This sender is blocked on this address, so the message was ignored."

	b.add(seededMessage{
		message: message,
		activities: []*triage.Activity{
			seedActivity(message, triage.ActivityIgnored, message.ReceivedAt).
				About(message.FromAddress, "blocked"),
		},
	})
}

func (b *triageSeed) threads() {
	b.detentionThread()
	b.statusThread()
}

func (b *triageSeed) detentionThread() {
	ship := b.shipment(seedDetentionPro)
	freshHaul := b.customer("FRSH")

	appointment := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "freshhaul-appointment-seed-det-008",
		from:       "ops@freshhaul.example",
		fromName:   "FreshHaul Foods",
		subject:    "Re: delivery appointment for " + seedDetentionPro,
		body:       "Can we confirm the cold storage appointment for " + seedDetentionPro + "?",
		receivedAt: b.minutesAgo(1640),
		class:      inboundmessage.ClassificationStatusRequest,
		confidence: 0.8,
	})
	matched(appointment, ship, freshHaul,
		"The PRO number "+seedDetentionPro+" is in the subject line.")
	appointment.Status = inboundmessage.StatusActioned
	b.add(seededMessage{
		message: appointment,
		activities: []*triage.Activity{
			settledBy(appointment, b.refs.team.dana, b.minutesAgo(1580),
				"Confirmed the appointment with Phoenix Cold Storage."),
		},
	})

	dispute := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "freshhaul-detention-seed-det-008",
		from:     seedDisputeSender,
		fromName: "FreshHaul Foods Claims",
		subject:  "Disputing detention on " + seedDetentionPro,
		body: "Hello,\n\nWe show the driver arriving after the appointment window " +
			"at Phoenix Cold Storage, so we are disputing the $1,200.00 detention on invoice " +
			"INV-20931.\n\nPlease send the gate log or ELD record if you believe otherwise." +
			"\n\nRegards,\nFreshHaul Claims",
		receivedAt: b.minutesAgo(1222),
		class:      inboundmessage.ClassificationDetentionDispute,
		confidence: 0.91,
	})
	continues(dispute, appointment, appointment)
	matched(dispute, ship, freshHaul,
		"The PRO number "+seedDetentionPro+" is in the subject line.")
	dispute.ReviewNote = "A charge is being disputed; that is a person's call."
	dispute.Facts = []inboundmessage.Fact{
		fact("lateArrival", "Late arrival claim", "After the appointment window",
			"arriving after the appointment window"),
		fact("disputedAmount", "Disputed amount", "$1,200.00", "$1,200.00 detention"),
		fact("invoice", "Invoice", "INV-20931", "INV-20931"),
	}

	invoice := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    "INV-20931.pdf",
		ContentType: "application/pdf",
		ByteSize:    88 * 1024,
		Kind:        inboundmessage.AttachmentInvoice,
		PageCount:   1,
	}
	reading := b.stage(seedStaging{
		message:    dispute,
		attachment: invoice,
		docType:    b.docType("INVOICE"),
		kind:       "Invoice",
		fields: []documentcontent.FoundField{
			{Key: "invoice_number", Label: "Invoice", Value: "INV-20931", Confidence: 0.98},
			{Key: "linehaul", Label: "Linehaul", Value: "$1,420.00", Confidence: 0.95},
			{Key: "detention", Label: "Detention", Value: "$1,200.00", Confidence: 0.94},
			{Key: "total", Label: "Total", Value: "$2,620.00", Confidence: 0.97},
		},
	})
	assignedAt := dispute.ReceivedAt + seedRuleDelay
	applied := seedActivity(dispute, triage.ActivityRuleApplied, assignedAt).
		ByNova().
		About(string(b.rules.disputes.Action), dispute.SenderDomain)
	applied.RecordID = b.rules.disputes.ID
	dispute.OwnerID = b.refs.team.dana.ID
	assigned := seedActivity(dispute, triage.ActivityAssigned, assignedAt).
		ByNova().
		About(b.refs.team.dana.Name, "")
	assigned.RecordID = b.refs.team.dana.ID

	b.add(seededMessage{
		message:     dispute,
		attachments: []*inboundmessage.InboundAttachment{invoice},
		documents:   []seededDocument{reading},
		activities:  []*triage.Activity{applied, assigned},
	})
}

func (b *triageSeed) statusThread() {
	ship := b.shipment("SEED-SHP-002")
	peak := b.customer("PEAK")
	reason := "PO 88213 is on SEED-SHP-002 for Peak Distributing."

	pickup := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "peak-po-88213-pickup",
		from:       "transport@peakdistributing.example",
		fromName:   "Peak Distributing",
		subject:    "PO 88213 — pickup confirmed?",
		body:       "Just checking pickup for PO 88213 is still Monday morning?",
		receivedAt: b.minutesAgo(3020),
		class:      inboundmessage.ClassificationStatusRequest,
		confidence: 0.86,
	})
	matched(pickup, ship, peak, reason)
	pickup.Status = inboundmessage.StatusActioned
	b.add(seededMessage{
		message: pickup,
		activities: []*triage.Activity{
			settledBy(pickup, b.refs.team.priya, b.minutesAgo(2990),
				"Confirmed pickup for Mon 09:00."),
		},
	})

	whereIs := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "peak-po-88213-where",
		from:     "transport@peakdistributing.example",
		fromName: "Peak Distributing",
		subject:  "Where is PO 88213?",
		body: "Hi, can you give me an ETA on PO 88213? " +
			"Our receiving closes at 17:00 today.",
		receivedAt: b.minutesAgo(22),
		class:      inboundmessage.ClassificationStatusRequest,
		confidence: 0.88,
	})
	continues(whereIs, pickup, pickup)
	matched(whereIs, ship, peak, reason)
	whereIs.ReviewNote = "Nova drafted an answer from the shipment; it waits for you to send it."
	whereIs.Facts = []inboundmessage.Fact{
		fact("purchaseOrder", "Purchase order", "PO 88213", "PO 88213"),
		fact("receivingCutoff", "Receiving cutoff", "17:00 today", "17:00"),
	}

	draftedAt := whereIs.ReceivedAt + 3*inboundSeedMinute
	draft := b.reply(seedReply{
		message:  whereIs,
		template: b.templates[seedTemplateETA],
		body: "Hi Peak — SEED-SHP-002 is running about two hours behind. " +
			"Current ETA is 16:10 today. We'll tell you if that moves.\n\nThanks,\nTrenova",
		at: draftedAt,
	})
	drafted := seedActivity(whereIs, triage.ActivityReplyDrafted, draftedAt).
		ByNova().
		About(draft.Subject, draft.ToAddress)
	drafted.RecordID = draft.ID

	b.add(seededMessage{
		message:    whereIs,
		replies:    []*triage.Reply{draft},
		activities: []*triage.Activity{drafted},
	})
}

type seedReply struct {
	message   *inboundmessage.InboundMessage
	template  *triage.ReplyTemplate
	body      string
	at        int64
	sentBy    *tenant.User
	by        *tenant.User
	failure   string
	suggested string
}

func (b *triageSeed) reply(r seedReply) *triage.Reply {
	templateID := pulid.Nil
	if r.template != nil {
		templateID = r.template.ID
	}
	author := b.refs.admin
	if r.by != nil {
		author = r.by
	}
	mailbox := b.mailboxOf(r.message)
	reply := &triage.Reply{
		ID:             pulid.MustNew("irpl_"),
		OrganizationID: r.message.OrganizationID,
		BusinessUnitID: r.message.BusinessUnitID,
		MessageID:      r.message.ID,
		MailboxID:      mailbox.ID,
		ThreadKey:      r.message.ThreadKey,
		Status:         triage.ReplyDraft,
		FromAddress:    mailbox.Address,
		ToAddress:      r.message.FromAddress,
		Subject:        "Re: " + r.message.Subject,
		Body:           r.body,
		DraftedByNova:  r.template != nil,
		TemplateID:     templateID,
		CreatedByID:    author.ID,
		CreatedAt:      r.at,
		UpdatedAt:      r.at,
	}
	if r.sentBy != nil {
		reply.Status = triage.ReplySent
		reply.SentByID = r.sentBy.ID
		reply.SentAt = r.at
		reply.MessageKey = inboundmessage.NormalizeMessageID(
			"<seed-reply-" + reply.ID.String() + "@" + emailutils.Domain(mailbox.Address) + ">",
		)
		reply.ProviderMessageID = "seed-reply-" + reply.ID.String()
	}
	if r.failure != "" {
		reply.Status = triage.ReplyFailed
		reply.FailureText = r.failure
		reply.SuggestedAddress = r.suggested
		reply.SentByID = author.ID
	}

	return reply
}

func (b *triageSeed) mailboxOf(message *inboundmessage.InboundMessage) *inboundmessage.Mailbox {
	switch message.MailboxID {
	case b.boxes.billing.ID:
		return b.boxes.billing
	case b.boxes.claims.ID:
		return b.boxes.claims
	default:
		return b.boxes.intake
	}
}

func (b *triageSeed) parked() {
	b.snoozedUntilTomorrow()
	b.snoozedUntilDelivered()
	b.snoozedUntilPOD()
	b.waitingOnThem()
}

func (b *triageSeed) snooze(
	message *inboundmessage.InboundMessage,
	snooze inboundmessage.Snooze,
	at int64,
) *triage.Activity {
	message.Snooze(snooze, at)
	detail := ""
	if snooze.DocumentTypeID.IsNotNil() {
		detail = b.docType(documenttype.CodePOD).Name
	}
	activity := seedActivity(message, triage.ActivitySnoozed, at).
		ByPerson(snooze.By).
		About(string(snooze.Event), detail)
	activity.Until = snooze.Until
	activity.RecordID = snooze.WatchID

	return activity
}

func (b *triageSeed) snoozedUntilTomorrow() {
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "globaltrade-return-quote",
		from:     "dispatch@globaltrade.example",
		fromName: "GlobalTrade Imports Dispatch",
		subject:  "Return load quote — Phoenix to Denver next week",
		body: "Could you quote a return load Phoenix to Denver next week? " +
			"Two reefers, 38,000 lbs each.",
		receivedAt: b.minutesAgo(260),
		class:      inboundmessage.ClassificationTender,
		confidence: 0.77,
	})
	matched(message, nil, b.customer("GLBL"),
		"Sender domain globaltrade.example belongs to GlobalTrade Imports.")
	message.Facts = []inboundmessage.Fact{
		fact("lane", "Lane", "Phoenix → Denver", "Phoenix to Denver"),
		fact("equipment", "Equipment", "Reefer × 2", "Two reefers"),
	}
	assigned := b.assigned(message, b.refs.admin, message.ReceivedAt+30*inboundSeedMinute)
	snoozed := b.snooze(message, inboundmessage.Snooze{
		Until: b.tomorrowAt(8),
		By:    b.refs.admin.ID,
	}, message.ReceivedAt+45*inboundSeedMinute)

	b.add(seededMessage{message: message, activities: []*triage.Activity{assigned, snoozed}})
}

func (b *triageSeed) snoozedUntilDelivered() {
	ship := b.shipment("SEED-SHP-003")
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "globaltrade-delivery-seed-shp-003",
		from:     "receiving@globaltrade.example",
		fromName: "GlobalTrade Imports Receiving",
		subject:  "Delivery receipt for SEED-SHP-003",
		body: "Please send the signed delivery receipt for SEED-SHP-003 " +
			"as soon as it delivers.",
		receivedAt: b.minutesAgo(380),
		class:      inboundmessage.ClassificationStatusRequest,
		confidence: 0.9,
	})
	matched(message, ship, b.customer("GLBL"),
		"The PRO number SEED-SHP-003 is in the subject line.")
	assigned := b.assigned(message, b.refs.team.marcus, message.ReceivedAt+20*inboundSeedMinute)
	snoozed := b.snooze(message, inboundmessage.Snooze{
		Event:   inboundmessage.SnoozeEventShipmentDelivered,
		WatchID: ship.ID,
		By:      b.refs.team.marcus.ID,
	}, message.ReceivedAt+25*inboundSeedMinute)

	b.add(seededMessage{message: message, activities: []*triage.Activity{assigned, snoozed}})
}

func (b *triageSeed) snoozedUntilPOD() {
	ship := b.shipment("SEED-PAY-001")
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "acme-detention-seed-pay-001",
		from:     "ap@acme.example",
		fromName: "Acme Manufacturing AP",
		subject:  "Detention on SEED-PAY-001 — signed POD to follow",
		body: "We will review the detention on SEED-PAY-001 once we have the signed POD " +
			"showing the out time.",
		receivedAt: b.minutesAgo(1900),
		class:      inboundmessage.ClassificationDetentionDispute,
		confidence: 0.86,
	})
	matched(message, ship, b.customer("ACME"),
		"The PRO number SEED-PAY-001 is in the subject line.")
	assigned := b.assigned(message, b.refs.team.dana, message.ReceivedAt+inboundSeedHour)
	snoozed := b.snooze(message, inboundmessage.Snooze{
		Event:          inboundmessage.SnoozeEventDocumentAttached,
		WatchID:        ship.ID,
		DocumentTypeID: b.docType(documenttype.CodePOD).ID,
		By:             b.refs.team.dana.ID,
	}, message.ReceivedAt+inboundSeedHour+5*inboundSeedMinute)

	b.add(seededMessage{message: message, activities: []*triage.Activity{assigned, snoozed}})
}

func (b *triageSeed) waitingOnThem() {
	message := b.mail(seedMail{
		mailbox:  b.boxes.intake,
		key:      "ironhorse-seed-pay-001-delivered",
		from:     "dispatch@ironhorsefreight.example",
		fromName: "Ironhorse Freight Dispatch",
		subject:  "SEED-PAY-001 delivered",
		body: "Delivered this morning. The driver forgot to photograph the signed BOL; " +
			"we'll chase it.",
		receivedAt: b.minutesAgo(1500),
		class:      inboundmessage.ClassificationProofOfDelivery,
		confidence: 0.82,
	})
	matched(message, b.shipment("SEED-PAY-001"), nil,
		"The PRO number SEED-PAY-001 is in the subject line.")
	message.MatchedCarrierID = b.refs.carriers["IHF"].ID

	draftedAt := message.ReceivedAt + 30*inboundSeedMinute
	sentAt := message.ReceivedAt + 40*inboundSeedMinute
	reply := b.reply(seedReply{
		message:  message,
		template: b.templates[seedTemplateAskPOD],
		body: "Hi Ironhorse — could you send the signed POD for SEED-PAY-001? " +
			"A clear photo of each page works.\n\nThanks,\nTrenova",
		at:     sentAt,
		sentBy: b.refs.admin,
	})
	reply.CreatedAt = draftedAt

	drafted := seedActivity(message, triage.ActivityReplyDrafted, draftedAt).
		ByNova().
		About(reply.Subject, reply.ToAddress)
	drafted.RecordID = reply.ID
	replied := seedActivity(message, triage.ActivityReplied, sentAt).
		ByPerson(b.refs.admin.ID).
		About(reply.FromAddress, reply.ToAddress)
	replied.RecordID = reply.ID
	until := sentAt + 2*inboundSeedDay
	message.AwaitReply(until, sentAt)
	waiting := seedActivity(message, triage.ActivityWaitingOnReply, sentAt).
		ByPerson(b.refs.admin.ID).
		About(message.FromAddress, "")
	waiting.Until = until

	b.add(seededMessage{
		message:    message,
		replies:    []*triage.Reply{reply},
		activities: []*triage.Activity{drafted, replied, waiting},
	})
}

func (b *triageSeed) settledEarlier() {
	b.brokerRateConsFiledByHand()
	b.settledByPeopleToday()
	b.invoiceFiledByRule()
	b.newsletterIgnoredByHand()
}

func (b *triageSeed) invoiceFiledByRule() {
	ship := b.shipment("SEED-SHP-002")
	message := b.mail(seedMail{
		mailbox:    b.boxes.billing,
		key:        "ironhorse-invoice-kf-5521",
		from:       "ap@ironhorsefreight.example",
		fromName:   "Ironhorse Freight AP",
		subject:    "Invoice IHF-5521 for SEED-SHP-002",
		body:       "Invoice IHF-5521 attached for SEED-SHP-002. Net 30.",
		receivedAt: b.minutesAgo(410),
		class:      inboundmessage.ClassificationInvoice,
		confidence: 0.97,
	})
	matched(message, ship, nil, "The invoice references SEED-SHP-002.")
	message.MatchedCarrierID = b.refs.carriers["IHF"].ID
	message.Facts = []inboundmessage.Fact{
		fact("invoice", "Invoice", "IHF-5521", "IHF-5521"),
		fact("terms", "Terms", "Net 30", "Net 30"),
	}
	attachment := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    "IHF-5521.pdf",
		ContentType: "application/pdf",
		ByteSize:    74 * 1024,
		Kind:        inboundmessage.AttachmentInvoice,
		PageCount:   1,
	}
	applied := settledByRule(message, b.rules.invoices,
		"Filed IHF-5521.pdf on SEED-SHP-002 and queued it for carrier pay.")
	filing := filed(seedFiling{
		message:    message,
		attachment: attachment,
		shipment:   ship,
		docType:    b.docType("INVOICE"),
		at:         message.ReviewedAt,
	})

	b.add(seededMessage{
		message:     message,
		attachments: []*inboundmessage.InboundAttachment{attachment},
		activities:  []*triage.Activity{applied, filing},
	})
}

func (b *triageSeed) newsletterIgnoredByHand() {
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "freightweekly-spot-rates-soften",
		from:       "news@freightweekly.example",
		fromName:   "FreightWeekly",
		subject:    "This week in freight: spot rates soften",
		body:       "Spot rates eased for a third week as capacity loosened across the Midwest.",
		receivedAt: b.minutesAgo(2900),
		class:      inboundmessage.ClassificationOther,
		confidence: 0.9,
	})
	message.Status = inboundmessage.StatusIgnored

	b.add(seededMessage{
		message: message,
		activities: []*triage.Activity{
			settledBy(message, b.refs.team.dana, b.minutesAgo(2800), "Newsletter."),
		},
	})
}

type seedSender struct {
	address string
	name    string
	carrier string
}

var seedDrivers = []seedSender{
	{address: "luis.ortega@drivers.trenova.example", name: "Luis Ortega"},
	{address: "ana.ruiz@drivers.trenova.example", name: "Ana Ruiz"},
	{address: "tom.becker@drivers.trenova.example", name: "Tom Becker"},
	{address: "kim.tran@drivers.trenova.example", name: "Kim Tran"},
	{address: "ray.diaz@drivers.trenova.example", name: "Ray Diaz"},
}

var seedCarrierBilling = []seedSender{
	{address: "ap@ironhorsefreight.example", name: "Ironhorse Freight AP", carrier: "IHF"},
	{address: "billing@harborpointtrucking.example", name: "Harbor Point Trucking", carrier: "HPT"},
	{address: "invoices@lonestarhaulers.example", name: "Lone Star Haulers", carrier: "LSH"},
}

var seedNewsletterSubjects = []string{
	"This morning in freight: diesel ticks up",
	"Capacity report: reefer tightens in the Southeast",
	"Spot market midday: van rates flat",
	"Webinar today: detention and the new rules",
	"FreightWeekly Daily: port volumes rebound",
	"Lane watch: Chicago to Atlanta",
	"Readers' poll: how long do your PODs take?",
	"Market close preview: flatbed demand",
	"Today's top five freight stories",
}

const (
	seedTodayPODs        = 14
	seedTodayInvoices    = 6
	seedTodayNewsletters = 9
	seedTodayRuleSettled = seedTodayPODs + seedTodayInvoices + seedTodayNewsletters
)

func (b *triageSeed) todayAt(index int) int64 {
	span := b.refs.now - b.refs.dayStart
	return b.refs.dayStart + span*int64(index+1)/int64(seedTodayRuleSettled+1)
}

func (b *triageSeed) settledByNovaToday() {
	slot := 0
	for i := range seedTodayPODs {
		pro := seedRoutinePros[i%len(seedRoutinePros)]
		b.podFiledByRule(seedPOD{
			driver:     seedDrivers[i%len(seedDrivers)],
			shipment:   b.shipment(pro),
			fileName:   fmt.Sprintf("POD_%s_%02d.jpg", pro, i+1),
			key:        fmt.Sprintf("today-pod-%02d", i+1),
			receivedAt: b.todayAt(slot),
		})
		slot++
	}
	for i := range seedTodayInvoices {
		pro := seedRoutinePros[(i+2)%len(seedRoutinePros)]
		b.invoiceQueuedByRule(seedInvoice{
			carrier:    seedCarrierBilling[i%len(seedCarrierBilling)],
			shipment:   b.shipment(pro),
			number:     fmt.Sprintf("INV-%05d", 31_200+i*7),
			key:        fmt.Sprintf("today-invoice-%02d", i+1),
			receivedAt: b.todayAt(slot),
		})
		slot++
	}
	for i := range seedTodayNewsletters {
		b.newsletterIgnoredByRule(
			seedNewsletterSubjects[i%len(seedNewsletterSubjects)],
			fmt.Sprintf("today-newsletter-%02d", i+1),
			b.todayAt(slot),
		)
		slot++
	}
}

type seedPOD struct {
	driver     seedSender
	shipment   *shipment.Shipment
	fileName   string
	key        string
	receivedAt int64
}

func (b *triageSeed) podFiledByRule(p seedPOD) {
	pro := p.shipment.ProNumber
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        p.key,
		from:       p.driver.address,
		fromName:   p.driver.name,
		subject:    "POD " + pro,
		body:       "Delivered and signed. POD for " + pro + " attached.",
		receivedAt: p.receivedAt,
		class:      inboundmessage.ClassificationProofOfDelivery,
		confidence: 0.95,
	})
	matched(message, p.shipment, nil,
		"The driver is assigned to "+pro+" and the subject names it.")
	attachment := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    p.fileName,
		ContentType: "image/jpeg",
		ByteSize:    1_800 * 1024,
		Kind:        inboundmessage.AttachmentProofOfDelivery,
		PageCount:   1,
	}
	applied := settledByRule(message, b.rules.pods, "Filed "+p.fileName+" on "+pro+".")
	filing := filed(seedFiling{
		message:    message,
		attachment: attachment,
		shipment:   p.shipment,
		docType:    b.docType(documenttype.CodePOD),
		at:         message.ReviewedAt,
	})

	b.add(seededMessage{
		message:     message,
		attachments: []*inboundmessage.InboundAttachment{attachment},
		activities:  []*triage.Activity{applied, filing},
	})
}

type seedInvoice struct {
	carrier    seedSender
	shipment   *shipment.Shipment
	number     string
	key        string
	receivedAt int64
}

func (b *triageSeed) invoiceQueuedByRule(p seedInvoice) {
	pro := p.shipment.ProNumber
	fileName := p.number + ".pdf"
	message := b.mail(seedMail{
		mailbox:    b.boxes.billing,
		key:        p.key,
		from:       p.carrier.address,
		fromName:   p.carrier.name,
		subject:    "Invoice " + p.number + " for " + pro,
		body:       "Invoice " + p.number + " attached for " + pro + ". Net 30.",
		receivedAt: p.receivedAt,
		class:      inboundmessage.ClassificationInvoice,
		confidence: 0.96,
	})
	matched(message, p.shipment, nil, "The invoice references "+pro+".")
	message.MatchedCarrierID = b.refs.carriers[p.carrier.carrier].ID
	message.Facts = []inboundmessage.Fact{
		fact("invoice", "Invoice", p.number, p.number),
		fact("terms", "Terms", "Net 30", "Net 30"),
	}
	attachment := &inboundmessage.InboundAttachment{
		ID:          pulid.MustNew("imsga_"),
		FileName:    fileName,
		ContentType: "application/pdf",
		ByteSize:    76 * 1024,
		Kind:        inboundmessage.AttachmentInvoice,
		PageCount:   1,
	}
	applied := settledByRule(message, b.rules.invoices,
		"Filed "+fileName+" on "+pro+" and queued it for carrier pay.")
	filing := filed(seedFiling{
		message:    message,
		attachment: attachment,
		shipment:   p.shipment,
		docType:    b.docType("INVOICE"),
		at:         message.ReviewedAt,
	})

	b.add(seededMessage{
		message:     message,
		attachments: []*inboundmessage.InboundAttachment{attachment},
		activities:  []*triage.Activity{applied, filing},
	})
}

func (b *triageSeed) newsletterIgnoredByRule(subject, key string, receivedAt int64) {
	message := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        key,
		from:       "news@freightweekly.example",
		fromName:   "FreightWeekly",
		subject:    subject,
		body:       subject + ". Read the full issue on freightweekly.example.",
		receivedAt: receivedAt,
		class:      inboundmessage.ClassificationOther,
		confidence: 0.93,
	})
	applied := settledByRule(message, b.rules.newsletters,
		"Ignored by the rule for freightweekly.example.")

	b.add(seededMessage{message: message, activities: []*triage.Activity{applied}})
}

func (b *triageSeed) correctedThisWeek() {
	type correctedMail struct {
		mailbox  *inboundmessage.Mailbox
		from     string
		fromName string
		subject  string
		body     string
		by       *tenant.User
		note     string
	}
	mails := []correctedMail{
		{
			mailbox:  b.boxes.intake,
			from:     "ops@peakdistributing.example",
			fromName: "Peak Distributing",
			subject:  "Quick question on SEED-SHP-002",
			body:     "Is SEED-SHP-002 still on for Monday?",
			by:       b.refs.team.priya,
			note:     "Confirmed Monday with the carrier.",
		},
		{
			mailbox:  b.boxes.billing,
			from:     "invoices@summitcarriers.example",
			fromName: "Summit Carriers",
			subject:  "Summit Carriers statement SC-8812",
			body:     "Statement SC-8812 for last week's loads, net 30.",
			by:       b.refs.team.dana,
			note:     "Queued for carrier pay.",
		},
		{
			mailbox:  b.boxes.intake,
			from:     "dispatch@rangelogistics.example",
			fromName: "Range Logistics Dispatch",
			subject:  "Can you cover Denver to Salt Lake Friday?",
			body:     "We need a truck Denver to Salt Lake City Friday, 40,000 lbs dry.",
			by:       b.refs.team.marcus,
			note:     "Built the load and confirmed.",
		},
	}

	for i, correction := range seedCorrectionsThisWeek {
		mail := mails[i]
		receivedAt := b.refs.correctedAt(i)
		message := b.mail(seedMail{
			mailbox:    mail.mailbox,
			key:        fmt.Sprintf("corrected-this-week-%d", i+1),
			from:       mail.from,
			fromName:   mail.fromName,
			subject:    mail.subject,
			body:       mail.body,
			receivedAt: receivedAt,
			class:      correction.from,
			confidence: 0.58,
		})
		message.Status = inboundmessage.StatusActioned
		correctedAt := receivedAt + inboundSeedHour
		activities := []*triage.Activity{
			corrected(message, correction, mail.by, correctedAt),
			settledBy(message, mail.by, correctedAt+10*inboundSeedMinute, mail.note),
		}
		if i == 0 {
			message.Facts = []inboundmessage.Fact{
				fact("shipment", "Shipment", "SEED-SHP-002", "SEED-SHP-002"),
				flaggedFact("pickupDay", "Pickup day", "Tuesday", "still on for Monday"),
			}
			activities = append(activities,
				flagged(message, message.Facts[1], mail.by, correctedAt+5*inboundSeedMinute))
		}
		b.add(seededMessage{
			message:     message,
			activities:  activities,
			corrections: []*aicorrection.Correction{b.aiCorrection(message, correction, mail.by, correctedAt)},
		})
	}
}

func (b *triageSeed) lastWeek() {
	weekStart := b.refs.weekStart
	pro := "SEED-SHP-005"

	b.podFiledByRule(seedPOD{
		driver:     seedDrivers[1],
		shipment:   b.shipment(pro),
		fileName:   "POD_" + pro + "_last_week.jpg",
		key:        "last-week-pod",
		receivedAt: weekStart - 2*inboundSeedDay + 9*inboundSeedHour,
	})
	b.invoiceQueuedByRule(seedInvoice{
		carrier:    seedCarrierBilling[1],
		shipment:   b.shipment("SEED-PAY-002"),
		number:     "INV-30988",
		key:        "last-week-invoice",
		receivedAt: weekStart - 4*inboundSeedDay + 11*inboundSeedHour,
	})
	b.newsletterIgnoredByRule("Weekend edition: the year in freight so far",
		"last-week-newsletter", weekStart-inboundSeedDay+7*inboundSeedHour)

	tender := b.mail(seedMail{
		mailbox:    b.boxes.intake,
		key:        "last-week-tender",
		from:       "dispatch@rangelogistics.example",
		fromName:   "Range Logistics Dispatch",
		subject:    "Two loads Denver to Phoenix",
		body:       "Two dry van loads Denver to Phoenix, pickup Tuesday.",
		receivedAt: weekStart - 5*inboundSeedDay + 8*inboundSeedHour,
		class:      inboundmessage.ClassificationTender,
		confidence: 0.81,
	})
	matched(tender, nil, b.customer("RNGE"),
		"Sender domain rangelogistics.example belongs to Range Logistics, a customer.")
	tender.Status = inboundmessage.StatusActioned
	b.add(seededMessage{
		message: tender,
		activities: []*triage.Activity{
			settledBy(tender, b.refs.team.marcus, tender.ReceivedAt+50*inboundSeedMinute,
				"Built both loads."),
		},
	})

	rateCon := b.mail(seedMail{
		mailbox:    b.boxes.billing,
		key:        "last-week-corrected",
		from:       "loads@globaltrade.example",
		fromName:   "GlobalTrade Imports",
		subject:    "Signed rate con for SEED-SHP-003",
		body:       "Signed rate confirmation for SEED-SHP-003 attached.",
		receivedAt: b.refs.lastWeekCorrectedAt(),
		class:      seedCorrectionLastWeek.from,
		confidence: 0.64,
	})
	matched(rateCon, b.shipment("SEED-SHP-003"), b.customer("GLBL"),
		"The PRO number SEED-SHP-003 is in the subject line.")
	rateCon.Facts = []inboundmessage.Fact{
		flaggedFact("rate", "Rate", "$11,800.00", "Signed rate confirmation"),
	}
	rateCon.Status = inboundmessage.StatusActioned
	b.add(seededMessage{
		message: rateCon,
		activities: []*triage.Activity{
			corrected(rateCon, seedCorrectionLastWeek, b.refs.team.dana,
				rateCon.ReceivedAt+inboundSeedHour),
			flagged(rateCon, rateCon.Facts[0], b.refs.team.dana,
				rateCon.ReceivedAt+inboundSeedHour+30),
			settledBy(rateCon, b.refs.team.dana, rateCon.ReceivedAt+inboundSeedHour+60,
				"Filed on SEED-SHP-003."),
		},
		corrections: []*aicorrection.Correction{
			b.aiCorrection(rateCon, seedCorrectionLastWeek, b.refs.team.dana,
				rateCon.ReceivedAt+inboundSeedHour),
		},
	})

	b.claimsLastHeardFrom()
}

func (b *triageSeed) claimsLastHeardFrom() {
	message := b.mail(seedMail{
		mailbox:  b.boxes.claims,
		key:      "claims-shortage-seed-shp-005",
		from:     "claims@acme.example",
		fromName: "Acme Manufacturing Claims",
		subject:  "Shortage claim on SEED-SHP-005",
		body: "Two pallets arrived short on SEED-SHP-005. " +
			"Please confirm what the driver noted at delivery.",
		receivedAt: b.boxes.claims.LastReceivedAt,
		class:      inboundmessage.ClassificationOther,
		confidence: 0.79,
	})
	matched(message, b.shipment("SEED-SHP-005"), b.customer("ACME"),
		"The PRO number SEED-SHP-005 is in the subject line.")
	message.Status = inboundmessage.StatusActioned

	b.add(seededMessage{
		message: message,
		activities: []*triage.Activity{
			settledBy(message, b.refs.team.priya, message.ReceivedAt+3*inboundSeedHour,
				"Sent the BOL notation to Acme claims."),
		},
	})
}
