package triage

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const maxActivityTextLength = 500

type ActivityKind string

const (
	ActivityAssigned         = ActivityKind("Assigned")
	ActivityUnassigned       = ActivityKind("Unassigned")
	ActivitySnoozed          = ActivityKind("Snoozed")
	ActivityWoken            = ActivityKind("Woken")
	ActivityWaitingOnReply   = ActivityKind("WaitingOnReply")
	ActivityStoppedWaiting   = ActivityKind("StoppedWaiting")
	ActivityTheyReplied      = ActivityKind("TheyReplied")
	ActivityReplyDrafted     = ActivityKind("ReplyDrafted")
	ActivityReplied          = ActivityKind("Replied")
	ActivityReplyFailed      = ActivityKind("ReplyFailed")
	ActivityLinked           = ActivityKind("Linked")
	ActivityUnlinked         = ActivityKind("Unlinked")
	ActivityFiled            = ActivityKind("Filed")
	ActivityUnfiled          = ActivityKind("Unfiled")
	ActivitySplit            = ActivityKind("Split")
	ActivityCorrected        = ActivityKind("Corrected")
	ActivityFactFlagged      = ActivityKind("FactFlagged")
	ActivityHandled          = ActivityKind("Handled")
	ActivityIgnored          = ActivityKind("Ignored")
	ActivityReopened         = ActivityKind("Reopened")
	ActivityLoadDrafted      = ActivityKind("LoadDrafted")
	ActivityLoadCreated      = ActivityKind("LoadCreated")
	ActivityExceptionLogged  = ActivityKind("ExceptionLogged")
	ActivitySenderBlocked    = ActivityKind("SenderBlocked")
	ActivityStopsDecided     = ActivityKind("StopsDecided")
	ActivityConflictResolved = ActivityKind("ConflictResolved")
	ActivityConflictReopened = ActivityKind("ConflictReopened")
	ActivityHandedOff        = ActivityKind("HandedOff")
	ActivityRuleApplied      = ActivityKind("RuleApplied")
	ActivityRuleCreated      = ActivityKind("RuleCreated")
	ActivityDueChanged       = ActivityKind("DueChanged")
	ActivityTestReceived     = ActivityKind("TestReceived")
	ActivityNovaRead         = ActivityKind("NovaRead")
)

type ActorKind string

const (
	ActorPerson = ActorKind("Person")
	ActorNova   = ActorKind("Nova")
	ActorSystem = ActorKind("System")
)

type Activity struct {
	bun.BaseModel `bun:"table:inbound_message_activities,alias:iact" json:"-"`

	ID                 pulid.ID                      `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID     pulid.ID                      `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID                      `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	MessageID          pulid.ID                      `json:"messageId"          bun:"message_id,type:VARCHAR(100),notnull"`
	Kind               ActivityKind                  `json:"kind"               bun:"kind,type:VARCHAR(30),notnull"`
	ActorKind          ActorKind                     `json:"actorKind"          bun:"actor_kind,type:VARCHAR(20),notnull"`
	ActorID            pulid.ID                      `json:"actorId"            bun:"actor_id,type:VARCHAR(100),nullzero"`
	Subject            string                        `json:"subject"            bun:"subject,type:VARCHAR(500),nullzero"`
	Detail             string                        `json:"detail"             bun:"detail,type:VARCHAR(500),nullzero"`
	Until              int64                         `json:"until"              bun:"until,type:BIGINT,nullzero"`
	FromClassification inboundmessage.Classification `json:"fromClassification" bun:"from_classification,type:VARCHAR(30),nullzero"`
	ToClassification   inboundmessage.Classification `json:"toClassification"   bun:"to_classification,type:VARCHAR(30),nullzero"`
	RecordID           pulid.ID                      `json:"recordId"           bun:"record_id,type:VARCHAR(100),nullzero"`
	CreatedAt          int64                         `json:"createdAt"          bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (a *Activity) BeforeAppendModel(_ context.Context, query bun.Query) error {
	a.Subject = stringutils.TruncateRunes(stringutils.StripNUL(a.Subject), maxActivityTextLength)
	a.Detail = stringutils.TruncateRunes(stringutils.StripNUL(a.Detail), maxActivityTextLength)
	if _, ok := query.(*bun.InsertQuery); ok {
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("iact_")
		}
		if a.CreatedAt == 0 {
			a.CreatedAt = timeutils.NowUnix()
		}
	}

	return nil
}

func (a *Activity) GetID() pulid.ID      { return a.ID }
func (a *Activity) GetCreatedAt() int64  { return a.CreatedAt }
func (a *Activity) GetTableName() string { return "inbound_message_activities" }

func NewActivity(m *inboundmessage.InboundMessage, kind ActivityKind) *Activity {
	return &Activity{
		OrganizationID: m.OrganizationID,
		BusinessUnitID: m.BusinessUnitID,
		MessageID:      m.ID,
		Kind:           kind,
		ActorKind:      ActorSystem,
	}
}

func (a *Activity) ByPerson(userID pulid.ID) *Activity {
	a.ActorKind = ActorPerson
	a.ActorID = userID

	return a
}

func (a *Activity) ByNova() *Activity {
	a.ActorKind = ActorNova
	a.ActorID = pulid.Nil

	return a
}

func (a *Activity) About(subject, detail string) *Activity {
	a.Subject = subject
	a.Detail = detail

	return a
}
