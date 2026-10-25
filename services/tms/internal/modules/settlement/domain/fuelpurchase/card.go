package fuelpurchase

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MinCancelReasonLength = 10
	maxCardLabelLength    = 100
	maxExternalCardIDLen  = 100
)

var (
	_ domaintypes.PostgresSearchable     = (*FuelCard)(nil)
	_ pagination.CursorEntity            = (*FuelCard)(nil)
	_ validationframework.TenantedEntity = (*FuelCard)(nil)
)

// Fuel cards issued to drivers and units
type FuelCard struct {
	keel.Table `keel:"fuel_cards,report=Equipment"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:fcard_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Provider          CardProvider `json:"provider"          keel:"provider"`
	LastFour          string       `json:"lastFour"          keel:"last_four"`
	Label             string       `json:"label"             keel:"label,label=Card Label"`
	ExternalCardID    string       `json:"externalCardId"    keel:"external_card_id,nullzero"`
	AssignedWorkerID  *pulid.ID    `json:"assignedWorkerId"  keel:"assigned_worker_id"`
	AssignedTractorID *pulid.ID    `json:"assignedTractorId" keel:"assigned_tractor_id"`
	Status            CardStatus   `json:"status"            keel:"status"`
	ExpiresAt         *int64       `json:"expiresAt"         keel:"expires_at"`
	CancelledAt       *int64       `json:"cancelledAt"       keel:"cancelled_at"`
	CancelReason      string       `json:"cancelReason"      keel:"cancel_reason,nullzero"`
	DiscoveredAt      *int64       `json:"discoveredAt"      keel:"discovered_at"`
	Notes             string       `json:"notes"             keel:"notes,nullzero"`

	SearchVector string `json:"-"         keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         keel:"rank,computed"`
	Version      int64  `json:"version"   keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	AssignedWorker  *worker.Worker   `json:"assignedWorker,omitempty"  keel:"belongs:assigned_worker_id,label=Assigned Driver"`
	AssignedTractor *tractor.Tractor `json:"assignedTractor,omitempty" keel:"belongs:assigned_tractor_id"`
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
		},
	}
}

func (c *FuelCard) BeforeCreate(ctx context.Context) error {
	if c.Status == "" {
		c.Status = CardStatusActive
	}

	return nil
}
