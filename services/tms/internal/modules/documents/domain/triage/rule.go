package triage

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const maxSenderLength = 255

type SenderScope string

const (
	SenderAnyone      = SenderScope("Anyone")
	SenderAnyCustomer = SenderScope("AnyCustomer")
	SenderAnyCarrier  = SenderScope("AnyCarrier")
	SenderDomain      = SenderScope("Domain")
	SenderAddress     = SenderScope("Address")
)

func (s SenderScope) IsValid() bool {
	switch s {
	case SenderAnyone, SenderAnyCustomer, SenderAnyCarrier, SenderDomain, SenderAddress:
		return true
	default:
		return false
	}
}

type Rule struct {
	keel.Table `keel:"triage_rules"`

	ID             pulid.ID                      `json:"id"             keel:"id,pk,autoid:trule_"`
	BusinessUnitID pulid.ID                      `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID                      `json:"organizationId" keel:"organization_id,pk,tenant"`
	Classification inboundmessage.Classification `json:"classification" keel:"classification"`
	SenderScope    SenderScope                   `json:"senderScope"    keel:"sender_scope"`
	Sender         string                        `json:"sender"         keel:"sender,nullzero"`
	MailboxID      pulid.ID                      `json:"mailboxId"      keel:"mailbox_id,nullzero"`
	Action         inboundmessage.NovaAction     `json:"action"         keel:"action"`
	AssigneeID     pulid.ID                      `json:"assigneeId"     keel:"assignee_id,nullzero"`
	Enabled        bool                          `json:"enabled"        keel:"enabled"`
	CreatedByID    pulid.ID                      `json:"createdById"    keel:"created_by_id,nullzero"`
	MatchCount     int64                         `json:"matchCount"     keel:"match_count"`
	LastMatchedAt  int64                         `json:"lastMatchedAt"  keel:"last_matched_at,nullzero"`
	LastMessageID  pulid.ID                      `json:"lastMessageId"  keel:"last_message_id,nullzero"`
	Version        int64                         `json:"version"        keel:"version,version"`
	CreatedAt      int64                         `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64                         `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (r *Rule) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.Classification,
			validation.Required.Error("Kind of message is required"),
			domainvalidation.ValidEnum[inboundmessage.Classification]("Kind is not a known one"),
		),
		validation.Field(&r.SenderScope,
			validation.Required.Error("Sender is required"),
			domainvalidation.ValidEnum[SenderScope]("Sender is not a known one"),
		),
		validation.Field(&r.Action,
			validation.Required.Error("Action is required"),
			domainvalidation.ValidEnum[inboundmessage.NovaAction]("Action is not a known one"),
		),
		validation.Field(&r.Sender, validation.Length(0, maxSenderLength)),
	))

	switch r.SenderScope {
	case SenderDomain:
		if err := emailutils.ValidateDomain(r.Sender); err != nil {
			multiErr.Add("sender", errortypes.ErrInvalid, "Sender must be a domain")
		}
	case SenderAddress:
		if err := emailutils.Validate(r.Sender); err != nil {
			multiErr.Add("sender", errortypes.ErrInvalid, "Sender must be an email address")
		}
	case SenderAnyone, SenderAnyCustomer, SenderAnyCarrier:
		if r.Sender != "" {
			multiErr.Add("sender", errortypes.ErrInvalid,
				"A sender is only named for a domain or an address")
		}
	}

	if r.Action == inboundmessage.NovaActionAssign && r.AssigneeID.IsNil() {
		multiErr.Add("assigneeId", errortypes.ErrRequired, "Say who to assign it to")
	}
	if r.Action != inboundmessage.NovaActionAssign && r.AssigneeID.IsNotNil() {
		multiErr.Add("assigneeId", errortypes.ErrInvalid,
			"Only an assigning rule names a person")
	}
}

func (r *Rule) Normalize() {
	r.Sender = strings.ToLower(strings.TrimSpace(r.Sender))
}

func (r *Rule) Matches(m *inboundmessage.InboundMessage) bool {
	if !r.Enabled {
		return false
	}

	return r.Covers(m)
}

func (r *Rule) Covers(m *inboundmessage.InboundMessage) bool {
	if m.Classification != r.Classification {
		return false
	}
	if r.MailboxID.IsNotNil() && r.MailboxID != m.MailboxID {
		return false
	}

	return r.MatchesSender(m)
}

func (r *Rule) MatchesSender(m *inboundmessage.InboundMessage) bool {
	switch r.SenderScope {
	case SenderAnyone:
		return true
	case SenderAnyCustomer:
		return m.MatchedCustomerID.IsNotNil()
	case SenderAnyCarrier:
		return m.MatchedCarrierID.IsNotNil()
	case SenderDomain:
		return emailutils.DomainAllowed(emailutils.Domain(m.FromAddress), []string{r.Sender})
	case SenderAddress:
		return strings.EqualFold(strings.TrimSpace(m.FromAddress), r.Sender)
	default:
		return false
	}
}

func (r *Rule) Specificity() int {
	score := 0
	switch r.SenderScope {
	case SenderAddress:
		score += 4
	case SenderDomain:
		score += 3
	case SenderAnyCustomer, SenderAnyCarrier:
		score++
	case SenderAnyone:
	}
	if r.MailboxID.IsNotNil() {
		score += 2
	}

	return score
}

func FirstMatch(rules []*Rule, m *inboundmessage.InboundMessage) *Rule {
	var best *Rule
	for _, rule := range rules {
		if !rule.Matches(m) {
			continue
		}
		if best == nil || rule.Specificity() > best.Specificity() ||
			(rule.Specificity() == best.Specificity() && rule.CreatedAt < best.CreatedAt) {
			best = rule
		}
	}

	return best
}

func (r *Rule) BeforeCreate(ctx context.Context) error {
	r.Normalize()

	return nil
}

func (r *Rule) BeforeUpdate(ctx context.Context) error {
	r.Normalize()

	return nil
}

func (r *Rule) GetID() pulid.ID      { return r.ID }
func (r *Rule) GetCreatedAt() int64  { return r.CreatedAt }
func (r *Rule) GetTableName() string { return "triage_rules" }

func (Rule) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_triage_rules_mailbox",
			Message: "Mailbox does not exist",
		},
	}
}
