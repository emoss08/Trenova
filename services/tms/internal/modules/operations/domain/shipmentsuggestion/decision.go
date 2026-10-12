package shipmentsuggestion

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

type Decision string

const (
	DecisionDone  = Decision("Done")
	DecisionLater = Decision("Later")
)

func Decisions() []Decision {
	return []Decision{DecisionDone, DecisionLater}
}

func (d Decision) IsValid() bool {
	return slices.Contains(Decisions(), d)
}

func (d Decision) String() string { return string(d) }

type Kind string

const (
	KindCoverage       = Kind("coverage")
	KindTender         = Kind("tender")
	KindDelayNotice    = Kind("delay")
	KindHoursOfService = Kind("hos")
	KindDetention      = Kind("detention")
	KindRetender       = Kind("retender")
)

func Kinds() []Kind {
	return []Kind{
		KindCoverage,
		KindTender,
		KindDelayNotice,
		KindHoursOfService,
		KindDetention,
		KindRetender,
	}
}

func (k Kind) IsValid() bool {
	return slices.Contains(Kinds(), k)
}

const (
	keySeparator = ":"
	MaxKeyLength = 200
)

var ErrInvalidKey = errors.New("suggestion key is not one the board issues")

type Key struct {
	Kind     Kind
	RecordID pulid.ID
}

func NewKey(kind Kind, recordID pulid.ID) Key {
	return Key{Kind: kind, RecordID: recordID}
}

func (k Key) String() string {
	return string(k.Kind) + keySeparator + k.RecordID.String()
}

func ParseKey(value string) (Key, error) {
	if len(value) > MaxKeyLength {
		return Key{}, ErrInvalidKey
	}
	kind, id, ok := strings.Cut(value, keySeparator)
	if !ok || !Kind(kind).IsValid() {
		return Key{}, ErrInvalidKey
	}
	recordID, err := pulid.Parse(id)
	if err != nil {
		return Key{}, ErrInvalidKey
	}

	return NewKey(Kind(kind), recordID), nil
}

var _ bun.BeforeAppendModelHook = (*DecisionRecord)(nil)

type DecisionRecord struct {
	bun.BaseModel `bun:"table:shipment_suggestion_decisions,alias:ssd" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	SuggestionKey  string   `json:"suggestionKey"  bun:"suggestion_key,type:VARCHAR(200),notnull"`
	Decision       Decision `json:"decision"       bun:"decision,type:VARCHAR(10),notnull"`
	DecidedAt      int64    `json:"decidedAt"      bun:"decided_at,type:BIGINT,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`
}

func (d *DecisionRecord) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("ssd_")
		}
		d.CreatedAt = now
		d.UpdatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}

	return nil
}
