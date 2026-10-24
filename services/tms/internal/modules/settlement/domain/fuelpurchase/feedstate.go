package fuelpurchase

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table `keel:"fuel_card_feed_states"`

	OrganizationID pulid.ID     `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID     `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider       CardProvider `json:"provider" keel:"provider,pk"`
	FeedType       FeedType     `json:"feedType" keel:"feed_type,pk"`
	Cursor         string       `json:"cursor" keel:"cursor,nullzero"`
	LastPolledAt   int64        `json:"lastPolledAt" keel:"last_polled_at,nullzero"`
	LastSuccessAt  int64        `json:"lastSuccessAt" keel:"last_success_at,nullzero"`
	FailureCount   int          `json:"failureCount" keel:"failure_count"`
	LastError      string       `json:"lastError" keel:"last_error,nullzero"`
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
