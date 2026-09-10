package fuelpurchase

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MinCancelReasonLength = 10
	maxCardLabelLength    = 100
	maxExternalCardIDLen  = 100
)

var (
	_ bun.BeforeAppendModelHook          = (*FuelCard)(nil)
	_ domaintypes.PostgresSearchable     = (*FuelCard)(nil)
	_ pagination.CursorEntity            = (*FuelCard)(nil)
	_ validationframework.TenantedEntity = (*FuelCard)(nil)
)

type FuelCard struct {
	bun.BaseModel             `bun:"table:fuel_cards,alias:fcard" json:"-"`
	pagination.CursorValueSet `bun:",embed"                       json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Provider          CardProvider `json:"provider"          bun:"provider,type:fuel_card_provider_enum,notnull"`
	LastFour          string       `json:"lastFour"          bun:"last_four,type:VARCHAR(4),notnull"`
	Label             string       `json:"label"             bun:"label,type:VARCHAR(100),notnull"`
	ExternalCardID    string       `json:"externalCardId"    bun:"external_card_id,type:VARCHAR(100),nullzero"`
	AssignedWorkerID  *pulid.ID    `json:"assignedWorkerId"  bun:"assigned_worker_id,type:VARCHAR(100),nullzero"`
	AssignedTractorID *pulid.ID    `json:"assignedTractorId" bun:"assigned_tractor_id,type:VARCHAR(100),nullzero"`
	Status            CardStatus   `json:"status"            bun:"status,type:fuel_card_status_enum,notnull,default:'Active'"`
	ExpiresAt         *int64       `json:"expiresAt"         bun:"expires_at,type:BIGINT,nullzero"`
	CancelledAt       *int64       `json:"cancelledAt"       bun:"cancelled_at,type:BIGINT,nullzero"`
	CancelReason      string       `json:"cancelReason"      bun:"cancel_reason,type:TEXT,nullzero"`
	DiscoveredAt      *int64       `json:"discoveredAt"      bun:"discovered_at,type:BIGINT,nullzero"`
	Notes             string       `json:"notes"             bun:"notes,type:TEXT,nullzero"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	AssignedWorker  *worker.Worker   `json:"assignedWorker,omitempty"  bun:"rel:belongs-to,join:assigned_worker_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	AssignedTractor *tractor.Tractor `json:"assignedTractor,omitempty" bun:"rel:belongs-to,join:assigned_tractor_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

func (c *FuelCard) Normalize() {
	c.LastFour = strings.TrimSpace(c.LastFour)
	c.Label = strings.TrimSpace(c.Label)
	c.ExternalCardID = strings.TrimSpace(c.ExternalCardID)
	c.CancelReason = strings.TrimSpace(c.CancelReason)
	c.Notes = strings.TrimSpace(c.Notes)
	if c.Status == "" {
		c.Status = CardStatusActive
	}
}

func (c *FuelCard) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		c,
		validation.Field(
			&c.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[CardProvider]("Provider is not valid"),
		),
		validation.Field(
			&c.LastFour,
			validation.Required.Error("The last four digits are required"),
			validation.Match(domaintypes.CardLastFourRegex).
				Error("The last four must be exactly four digits"),
		),
		validation.Field(
			&c.Label,
			validation.Required.Error("Label is required"),
			validation.Length(1, maxCardLabelLength).
				Error("Label must be between 1 and 100 characters"),
		),
		validation.Field(
			&c.ExternalCardID,
			validation.Length(0, maxExternalCardIDLen).
				Error("External card ID cannot exceed 100 characters"),
		),
		validation.Field(
			&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[CardStatus]("Status is not valid"),
		),
	))

	if c.ExpiresAt != nil && *c.ExpiresAt <= 0 {
		multiErr.Add("expiresAt", errortypes.ErrInvalid, "Expiry must be a valid date")
	}

	c.validateCancellation(multiErr)
}

func (c *FuelCard) validateCancellation(multiErr *errortypes.MultiError) {
	if c.Status != CardStatusCancelled {
		if c.CancelledAt != nil {
			multiErr.Add(
				"cancelledAt",
				errortypes.ErrInvalid,
				"Only a cancelled card carries a cancellation date",
			)
		}
		return
	}

	if c.CancelledAt == nil || *c.CancelledAt <= 0 {
		multiErr.Add(
			"cancelledAt",
			errortypes.ErrRequired,
			"A cancelled card must record when it was cancelled",
		)
	}
	if len(c.CancelReason) < MinCancelReasonLength {
		multiErr.Add(
			"cancelReason",
			errortypes.ErrRequired,
			"A cancellation reason of at least 10 characters is required",
		)
	}
}

func (c *FuelCard) IsActive() bool { return c.Status == CardStatusActive }

// IsUnassigned reports whether the card is not yet tied to a tractor or a driver.
// A feed creates cards in this state the first time it sees a transaction on one,
// and they stay unusable for matching until somebody assigns them.
func (c *FuelCard) IsUnassigned() bool {
	return (c.AssignedTractorID == nil || c.AssignedTractorID.IsNil()) &&
		(c.AssignedWorkerID == nil || c.AssignedWorkerID.IsNil())
}

// WasDiscovered reports whether a feed created this card rather than a person.
func (c *FuelCard) WasDiscovered() bool {
	return c.DiscoveredAt != nil && *c.DiscoveredAt > 0
}

func (c *FuelCard) IsCancelled() bool { return c.Status == CardStatusCancelled }

func (c *FuelCard) IsExpiredAt(now int64) bool {
	return c.ExpiresAt != nil && *c.ExpiresAt > 0 && *c.ExpiresAt < now
}

func (c *FuelCard) CanRecordPurchase(now int64) bool {
	return c.IsActive() && !c.IsExpiredAt(now)
}

func (c *FuelCard) Cancel(at int64, reason string) {
	c.Status = CardStatusCancelled
	c.CancelledAt = &at
	c.CancelReason = strings.TrimSpace(reason)
}

func (c *FuelCard) GetID() pulid.ID { return c.ID }

func (c *FuelCard) GetCreatedAt() int64 { return c.CreatedAt }

func (c *FuelCard) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *FuelCard) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *FuelCard) GetTableName() string { return "fuel_cards" }

func (c *FuelCard) GetResourceType() string { return "fuel_card" }

func (c *FuelCard) GetResourceID() string { return c.ID.String() }

func (c *FuelCard) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "fcard",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "label", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "last_four", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
			{
				Name:   "external_card_id",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (c *FuelCard) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("fcard_")
		}
		if c.Status == "" {
			c.Status = CardStatusActive
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
