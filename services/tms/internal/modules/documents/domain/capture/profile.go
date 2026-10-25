package capture

import (
	"context"
	"slices"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	maxProfileNameLength        = 100
	maxProfileDescriptionLength = 500
	minProfileDPI               = 100
	maxProfileDPI               = 600
	minJPEGQuality              = 30
	maxJPEGQuality              = 95
	maxFixedPageCount           = 500
)

// CaptureProfile is a named set of scan settings an organization offers the people
// who scan, so a billing clerk picks "PODs" rather than a resolution.
type CaptureProfile struct {
	keel.Table `keel:"capture_profiles"`

	ID                  pulid.ID            `json:"id"                  keel:"id,pk,autoid:cprf_"`
	BusinessUnitID      pulid.ID            `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	OrganizationID      pulid.ID            `json:"organizationId"      keel:"organization_id,pk,tenant"`
	Name                string              `json:"name"                keel:"name"`
	Description         string              `json:"description"         keel:"description,nullzero"`
	Status              ProfileStatus       `json:"status"              keel:"status"`
	IsDefault           bool                `json:"isDefault"           keel:"is_default"`
	DPI                 int                 `json:"dpi"                 keel:"dpi"`
	PixelType           PixelType           `json:"pixelType"           keel:"pixel_type"`
	Duplex              bool                `json:"duplex"              keel:"duplex"`
	UseFeeder           bool                `json:"useFeeder"           keel:"use_feeder"`
	DiscardBlankPages   bool                `json:"discardBlankPages"   keel:"discard_blank_pages"`
	JPEGQuality         int                 `json:"jpegQuality"         keel:"jpeg_quality"`
	ShowDriverUI        bool                `json:"showDriverUi"        keel:"show_driver_ui"`
	SeparatorStrategies []SeparatorStrategy `json:"separatorStrategies" keel:"separator_strategies"`
	FixedPageCount      int                 `json:"fixedPageCount"      keel:"fixed_page_count"`
	Version             int64               `json:"version"             keel:"version,version"`
	CreatedAt           int64               `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64               `json:"updatedAt"           keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
}

func (p *CaptureProfile) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxProfileNameLength),
		),
		validation.Field(&p.Description, validation.Length(0, maxProfileDescriptionLength)),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ProfileStatus]("Status must be Active or Inactive"),
		),
		validation.Field(&p.DPI,
			validation.Required.Error("Resolution is required"),
			validation.Min(minProfileDPI).Error("Resolution must be at least 100 DPI"),
			validation.Max(maxProfileDPI).Error("Resolution must be at most 600 DPI"),
		),
		validation.Field(&p.PixelType,
			validation.Required.Error("Colour mode is required"),
			domainvalidation.ValidEnum[PixelType]("Colour mode is not one of the three"),
		),
		validation.Field(&p.JPEGQuality,
			validation.Required.Error("Image quality is required"),
			validation.Min(minJPEGQuality).Error("Image quality must be at least 30"),
			validation.Max(maxJPEGQuality).Error("Image quality must be at most 95"),
		),
		validation.Field(&p.SeparatorStrategies,
			validation.Each(domainvalidation.ValidEnum[SeparatorStrategy](
				"Separator is not one Trenova recognises",
			)),
		),
	))

	if p.UsesSeparator(SeparatorFixedPageCount) {
		if p.FixedPageCount < 1 || p.FixedPageCount > maxFixedPageCount {
			multiErr.Add("fixedPageCount", errortypes.ErrInvalid,
				"Pages per document must be between 1 and {0}", maxFixedPageCount)
		}
	} else if p.FixedPageCount != 0 {
		multiErr.Add("fixedPageCount", errortypes.ErrInvalid,
			"Pages per document only applies when splitting by page count")
	}

	if p.IsDefault && p.Status != ProfileActive {
		multiErr.Add("isDefault", errortypes.ErrInvalid,
			"An inactive profile cannot be the default")
	}
}

// UsesSeparator reports whether the profile splits on the given strategy.
func (p *CaptureProfile) UsesSeparator(strategy SeparatorStrategy) bool {
	return slices.Contains(p.SeparatorStrategies, strategy)
}

// ApplyDefaults is what a profile is before anybody tunes it: 300 DPI black and
// white, both sides, blank pages dropped. That is the setting scanner vendors
// recommend for paperwork that is mostly text, and it keeps a POD to tens of
// kilobytes a page.
func (p *CaptureProfile) ApplyDefaults() {
	if p.Status == "" {
		p.Status = ProfileActive
	}
	if p.DPI == 0 {
		p.DPI = 300
	}
	if p.PixelType == "" {
		p.PixelType = PixelBlackWhite
	}
	if p.JPEGQuality == 0 {
		p.JPEGQuality = 80
	}
	if p.SeparatorStrategies == nil {
		p.SeparatorStrategies = []SeparatorStrategy{}
	}
}

func (p *CaptureProfile) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "cprf",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (p *CaptureProfile) GetID() pulid.ID      { return p.ID }
func (p *CaptureProfile) GetCreatedAt() int64  { return p.CreatedAt }
func (p *CaptureProfile) GetTableName() string { return "capture_profiles" }

func (p *CaptureProfile) BeforeCreate(ctx context.Context) error {
	p.ApplyDefaults()

	return nil
}
