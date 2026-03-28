package tenant

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*DocumentControl)(nil)
	_ validationframework.TenantedEntity = (*DocumentControl)(nil)
)

type DocumentControl struct {
	bun.BaseModel `bun:"table:document_controls,alias:dc" json:"-"`

	ID                              pulid.ID `json:"id"                              bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID                  pulid.ID `json:"businessUnitId"                  bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID                  pulid.ID `json:"organizationId"                  bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	EnableDocumentIntelligence      bool     `json:"enableDocumentIntelligence"      bun:"enable_document_intelligence,type:BOOLEAN,notnull,default:true"`
	EnableOCR                       bool     `json:"enableOcr"                       bun:"enable_ocr,type:BOOLEAN,notnull,default:true"`
	EnableAutoClassification        bool     `json:"enableAutoClassification"        bun:"enable_auto_classification,type:BOOLEAN,notnull,default:true"`
	EnableAutoDocumentTypeAssociate bool     `json:"enableAutoDocumentTypeAssociate" bun:"enable_auto_document_type_associate,type:BOOLEAN,notnull,default:true"`
	EnableAutoCreateDocumentTypes   bool     `json:"enableAutoCreateDocumentTypes"   bun:"enable_auto_create_document_types,type:BOOLEAN,notnull,default:true"`
	EnableShipmentDraftExtraction   bool     `json:"enableShipmentDraftExtraction"   bun:"enable_shipment_draft_extraction,type:BOOLEAN,notnull,default:true"`
	EnableAIAssistedClassification  bool     `json:"enableAiAssistedClassification"  bun:"enable_ai_assisted_classification,type:BOOLEAN,notnull,default:false"`
	EnableAIAssistedExtraction      bool     `json:"enableAiAssistedExtraction"      bun:"enable_ai_assisted_extraction,type:BOOLEAN,notnull,default:false"`
	ShipmentDraftAllowedResources   []string `json:"shipmentDraftAllowedResources"   bun:"shipment_draft_allowed_resources,type:VARCHAR(100)[],notnull,default:'{}'"`
	EnableFullTextIndexing          bool     `json:"enableFullTextIndexing"          bun:"enable_full_text_indexing,type:BOOLEAN,notnull,default:true"`
	Version                         int64    `json:"version"                         bun:"version,type:BIGINT"`
	CreatedAt                       int64    `json:"createdAt"                       bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                       int64    `json:"updatedAt"                       bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func NewDefaultDocumentControl(orgID, buID pulid.ID) *DocumentControl {
	return &DocumentControl{
		OrganizationID:                  orgID,
		BusinessUnitID:                  buID,
		EnableDocumentIntelligence:      true,
		EnableOCR:                       true,
		EnableAutoClassification:        true,
		EnableAutoDocumentTypeAssociate: true,
		EnableAutoCreateDocumentTypes:   true,
		EnableShipmentDraftExtraction:   true,
		EnableAIAssistedClassification:  true,
		EnableAIAssistedExtraction:      true,
		ShipmentDraftAllowedResources:   []string{"shipment"},
		EnableFullTextIndexing:          true,
	}
}

func (dc *DocumentControl) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		dc,
		validation.Field(
			&dc.ShipmentDraftAllowedResources,
			validation.Each(validation.By(func(value any) error {
				resourceType, ok := value.(string)
				if !ok {
					return errors.New("invalid resource type")
				}
				if !isAllowedDocumentControlResource(resourceType) {
					return errors.New("unsupported resource type")
				}
				return nil
			})),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func isAllowedDocumentControlResource(resourceType string) bool {
	switch strings.ToLower(strings.TrimSpace(resourceType)) {
	case "shipment", "trailer", "tractor", "worker":
		return true
	default:
		return false
	}
}

func (dc *DocumentControl) AllowsShipmentDraftResource(resourceType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(resourceType))
	values := make([]string, 0, len(dc.ShipmentDraftAllowedResources))
	for _, value := range dc.ShipmentDraftAllowedResources {
		values = append(values, strings.ToLower(strings.TrimSpace(value)))
	}
	return slices.Contains(values, normalized)
}

func (dc *DocumentControl) GetID() pulid.ID {
	return dc.ID
}

func (dc *DocumentControl) GetTableName() string {
	return "document_controls"
}

func (dc *DocumentControl) GetOrganizationID() pulid.ID {
	return dc.OrganizationID
}

func (dc *DocumentControl) GetBusinessUnitID() pulid.ID {
	return dc.BusinessUnitID
}

func (dc *DocumentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dc.ID.IsNil() {
			dc.ID = pulid.MustNew("docc_")
		}
		if len(dc.ShipmentDraftAllowedResources) == 0 {
			dc.ShipmentDraftAllowedResources = []string{"shipment"}
		}
		dc.CreatedAt = now
		dc.UpdatedAt = now
	case *bun.UpdateQuery:
		dc.UpdatedAt = now
	}

	return nil
}
