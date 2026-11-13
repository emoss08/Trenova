package pagetour

import (
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	MaxTourLength   = 64
	MaxTourVersion  = 10_000
	MaxViewsPerUser = 500
)

var tourPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)

type Outcome string

const (
	OutcomeFinished = Outcome("Finished")
	OutcomeSkipped  = Outcome("Skipped")
)

func (o Outcome) String() string { return string(o) }

func (o Outcome) IsValid() bool {
	switch o {
	case OutcomeFinished, OutcomeSkipped:
		return true
	default:
		return false
	}
}

type View struct {
	keel.Table `keel:"page_tour_views"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:ptv_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	Tour           string   `json:"tour"           keel:"tour"`
	TourVersion    int32    `json:"tourVersion"    keel:"tour_version"`
	Outcome        Outcome  `json:"outcome"        keel:"outcome"`
	SeenAt         int64    `json:"seenAt"         keel:"seen_at"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	User         *tenant.User         `json:"user,omitempty"         keel:"belongs:user_id"`
}

func (v *View) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.UserID, validation.Required.Error("User is required")),
		validation.Field(&v.Tour,
			validation.Required.Error("Tour is required"),
			validation.Length(1, MaxTourLength).Error("Tour is too long"),
			validation.Match(tourPattern).
				Error("Tour must be lowercase words joined by hyphens"),
		),
		validation.Field(&v.TourVersion,
			validation.Required.Error("Tour version is required"),
			validation.Min(int32(1)).Error("Tour version must be at least 1"),
			validation.Max(int32(MaxTourVersion)).Error("Tour version is too high"),
		),
		validation.Field(&v.Outcome,
			validation.Required.Error("Outcome is required"),
			domainvalidation.ValidEnum[Outcome]("Outcome must be Finished or Skipped"),
		),
	))
}

func (v *View) GetID() pulid.ID { return v.ID }

func (v *View) GetOrganizationID() pulid.ID { return v.OrganizationID }

func (v *View) GetBusinessUnitID() pulid.ID { return v.BusinessUnitID }

func (v *View) GetTableName() string { return "page_tour_views" }
