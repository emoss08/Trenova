package fuelpurchase

import (
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const maxFeedErrorLength = 2000

type FeedType string

const (
	FeedTypeTransactions = FeedType("Transactions")
	FeedTypeCards        = FeedType("Cards")
)

func (t FeedType) String() string { return string(t) }

func (t FeedType) IsValid() bool {
	return t == FeedTypeTransactions || t == FeedTypeCards
}

func (t FeedType) Label() string {
	switch t {
	case FeedTypeTransactions:
		return "Transactions"
	case FeedTypeCards:
		return "Cards"
	default:
		return string(t)
	}
}

// CardFeedState is how far one organization's feed has been read for a provider. The
// window is always re-requested with an overlap, because a provider can post a
// transaction late; the transaction reference unique index is what actually stops
// a purchase landing twice.
type CardFeedState struct {
	bun.BaseModel `bun:"table:fuel_card_feed_states,alias:fcfs" json:"-"`

	OrganizationID pulid.ID     `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Provider       CardProvider `json:"provider"       bun:"provider,pk,type:fuel_card_provider_enum,notnull"`
	FeedType       FeedType     `json:"feedType"       bun:"feed_type,pk,type:VARCHAR(32),notnull"`
	Cursor         string       `json:"cursor"         bun:"cursor,type:TEXT,nullzero"`
	LastPolledAt   int64        `json:"lastPolledAt"   bun:"last_polled_at,type:BIGINT,nullzero"`
	LastSuccessAt  int64        `json:"lastSuccessAt"  bun:"last_success_at,type:BIGINT,nullzero"`
	FailureCount   int          `json:"failureCount"   bun:"failure_count,type:INTEGER,notnull"`
	LastError      string       `json:"lastError"      bun:"last_error,type:TEXT,nullzero"`
}

func (f *CardFeedState) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(&f.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&f.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&f.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[CardProvider]("Provider is not valid"),
		),
		validation.Field(&f.FeedType,
			validation.Required.Error("Feed type is required"),
			domainvalidation.ValidEnum[FeedType]("Feed type is not valid"),
		),
		validation.Field(&f.FailureCount,
			validation.Min(0).Error("Failure count cannot be negative"),
		),
	))
}

// RecordSuccess advances the watermark. It is the only thing that moves
// LastSuccessAt, so a run that failed part way never shrinks the window the next
// run asks for.
func (f *CardFeedState) RecordSuccess(at int64, cursor string) {
	f.LastPolledAt = at
	f.LastSuccessAt = at
	f.Cursor = cursor
	f.FailureCount = 0
	f.LastError = ""
}

// RecordFailure counts the attempt without moving the watermark, so the failed
// window is read again on the next run.
func (f *CardFeedState) RecordFailure(at int64, err error) {
	f.LastPolledAt = at
	f.FailureCount++
	if err == nil {
		f.LastError = ""
		return
	}
	message := err.Error()
	if len(message) > maxFeedErrorLength {
		message = message[:maxFeedErrorLength]
	}
	f.LastError = message
}

// Since is the instant a run should read from, backing off by overlap so a
// transaction posted late is still picked up. A feed that has never succeeded
// reads from fallback instead.
func (f *CardFeedState) Since(overlap, fallback int64) int64 {
	if f.LastSuccessAt <= 0 {
		return fallback
	}
	since := f.LastSuccessAt - overlap
	if since < 0 {
		return 0
	}
	return since
}

func (f *CardFeedState) GetOrganizationID() pulid.ID { return f.OrganizationID }

func (f *CardFeedState) GetBusinessUnitID() pulid.ID { return f.BusinessUnitID }

func (f *CardFeedState) GetTableName() string { return "fuel_card_feed_states" }
