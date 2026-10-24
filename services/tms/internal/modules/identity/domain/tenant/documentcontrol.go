package tenant

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/versionutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*DocumentControl)(nil)
)

type DocumentControl struct {
	keel.Table `keel:"document_controls"`

	ID                              pulid.ID `json:"id" keel:"id,pk,autoid:docc_"`
	BusinessUnitID                  pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID                  pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	EnableDocumentIntelligence      bool     `json:"enableDocumentIntelligence" keel:"enable_document_intelligence"`
	EnableOCR                       bool     `json:"enableOcr" keel:"enable_ocr"`
	EnableAutoClassification        bool     `json:"enableAutoClassification" keel:"enable_auto_classification"`
	EnableAutoDocumentTypeAssociate bool     `json:"enableAutoDocumentTypeAssociate" keel:"enable_auto_document_type_associate"`
	EnableAutoCreateDocumentTypes   bool     `json:"enableAutoCreateDocumentTypes" keel:"enable_auto_create_document_types"`
	EnableShipmentDraftExtraction   bool     `json:"enableShipmentDraftExtraction" keel:"enable_shipment_draft_extraction"`
	EnableAIAssistedClassification  bool     `json:"enableAiAssistedClassification" keel:"enable_ai_assisted_classification"`
	EnableAIAssistedExtraction      bool     `json:"enableAiAssistedExtraction" keel:"enable_ai_assisted_extraction"`
	ShipmentDraftAllowedResources   []string `json:"shipmentDraftAllowedResources" keel:"shipment_draft_allowed_resources"`
	EnableFullTextIndexing          bool     `json:"enableFullTextIndexing" keel:"enable_full_text_indexing"`
	EnableCapture                   bool     `json:"enableCapture" keel:"enable_capture"`
	CaptureAutoFileCoverSheets      bool     `json:"captureAutoFileCoverSheets" keel:"capture_auto_file_cover_sheets"`
	CaptureRetentionDays            int      `json:"captureRetentionDays" keel:"capture_retention_days"`
	CaptureMinAgentVersion          string   `json:"captureMinAgentVersion" keel:"capture_min_agent_version,nullzero"`
	CaptureAllowAutoUpdate          bool     `json:"captureAllowAutoUpdate" keel:"capture_allow_auto_update"`
	Version                         int64    `json:"version" keel:"version,version"`
	CreatedAt                       int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt                       int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
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
		EnableCapture:                   true,
		CaptureAutoFileCoverSheets:      true,
		CaptureRetentionDays:            DefaultCaptureRetentionDays,
		CaptureAllowAutoUpdate:          true,
	}
}

const (
	// DefaultCaptureRetentionDays is how long an unfiled scan waits in intake
	// before it is discarded. A month covers a slow billing cycle; longer and
	// the queue fills with paper somebody decided not to file.
	DefaultCaptureRetentionDays = 30
	minCaptureRetentionDays     = 1
	maxCaptureRetentionDays     = 365
)

func (dc *DocumentControl) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		dc,
		validation.Field(
			&dc.CaptureRetentionDays,
			validation.Required.Error("Intake retention is required"),
			validation.Min(minCaptureRetentionDays).
				Error("Intake retention must be at least 1 day"),
			validation.Max(maxCaptureRetentionDays).
				Error("Intake retention must be at most 365 days"),
		),
		validation.Field(
			&dc.CaptureMinAgentVersion,
			validation.By(func(value any) error {
				raw, _ := value.(string)
				if raw == "" || versionutils.IsValid(raw) {
					return nil
				}
				return errors.New("minimum version must look like 1.4.0")
			}),
		),
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
	))
}

func isAllowedDocumentControlResource(resourceType string) bool {
	switch normalizeDocumentControlResource(resourceType) {
	case "shipment", "trailer", "tractor", "worker":
		return true
	default:
		return false
	}
}

func (dc *DocumentControl) AllowsShipmentDraftResource(resourceType string) bool {
	normalized := normalizeDocumentControlResource(resourceType)
	values := make([]string, 0, len(dc.ShipmentDraftAllowedResources))
	for _, value := range dc.ShipmentDraftAllowedResources {
		values = append(values, normalizeDocumentControlResource(value))
	}
	return slices.Contains(values, normalized)
}

func normalizeDocumentControlResource(resourceType string) string {
	normalized := strings.ToLower(strings.TrimSpace(resourceType))
	switch normalized {
	// Each of these is a shipment arriving by another road: an import, a
	// tender by email, and a rate confirmation handed to an agent in a
	// conversation. A tenant that allows drafts on shipments means it allows
	// them on the paperwork that becomes shipments, and making them opt in
	// again per channel is asking the same question twice.
	case "shipment_import", "inbound_message", "assistant_thread":
		return "shipment"
	default:
		return normalized
	}
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

func (dc *DocumentControl) BeforeCreate(_ context.Context) error {
	if len(dc.ShipmentDraftAllowedResources) == 0 {
		dc.ShipmentDraftAllowedResources = []string{"shipment"}
	}
	if dc.CaptureRetentionDays == 0 {
		dc.CaptureRetentionDays = DefaultCaptureRetentionDays
	}
	if dc.CaptureRetentionDays == 0 {
		dc.CaptureRetentionDays = 30
	}

	return nil
}
