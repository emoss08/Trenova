package shipmentsuggestion

import (
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/shared/pulid"
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

type DecisionRecord struct {
	keel.Table `keel:"shipment_suggestion_decisions"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:ssd_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	UserID         pulid.ID `json:"userId" keel:"user_id"`
	SuggestionKey  string   `json:"suggestionKey" keel:"suggestion_key"`
	Decision       Decision `json:"decision" keel:"decision"`
	DecidedAt      int64    `json:"decidedAt" keel:"decided_at"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}
