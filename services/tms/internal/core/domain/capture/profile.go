package capture

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel             `bun:"table:capture_profiles,alias:cprf" json:"-"`
	pagination.CursorValueSet `bun:",embed"                            json:"-"`

	ID                  pulid.ID            `json:"id"                  bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID      pulid.ID            `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID            `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name                string              `json:"name"                bun:"name,type:VARCHAR(100),notnull"`
	Description         string              `json:"description"         bun:"description,type:VARCHAR(500),nullzero"`
	Status              ProfileStatus       `json:"status"              bun:"status,type:VARCHAR(20),notnull"`
	IsDefault           bool                `json:"isDefault"           bun:"is_default,type:BOOLEAN,notnull,default:false"`
	DPI                 int                 `json:"dpi"                 bun:"dpi,type:INTEGER,notnull"`
	PixelType           PixelType           `json:"pixelType"           bun:"pixel_type,type:VARCHAR(20),notnull"`
	Duplex              bool                `json:"duplex"              bun:"duplex,type:BOOLEAN,notnull,default:true"`
	UseFeeder           bool                `json:"useFeeder"           bun:"use_feeder,type:BOOLEAN,notnull,default:true"`
	DiscardBlankPages   bool                `json:"discardBlankPages"   bun:"discard_blank_pages,type:BOOLEAN,notnull,default:true"`
	JPEGQuality         int                 `json:"jpegQuality"         bun:"jpeg_quality,type:INTEGER,notnull"`
	ShowDriverUI        bool                `json:"showDriverUi"        bun:"show_driver_ui,type:BOOLEAN,notnull,default:false"`
	SeparatorStrategies []SeparatorStrategy `json:"separatorStrategies" bun:"separator_strategies,type:VARCHAR(30)[],notnull,default:'{}',array"`
	FixedPageCount      int                 `json:"fixedPageCount"      bun:"fixed_page_count,type:INTEGER,notnull,default:0"`
	Version             int64               `json:"version"             bun:"version,type:BIGINT"`
	CreatedAt           int64               `json:"createdAt"           bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64               `json:"updatedAt"           bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
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

func (p *CaptureProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("cprf_")
		}
		p.ApplyDefaults()
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
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
