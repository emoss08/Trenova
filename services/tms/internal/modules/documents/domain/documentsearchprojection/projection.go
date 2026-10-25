package documentsearchprojection

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const maxIndexedContentChars = 100000

type Projection struct {
	keel.Table `keel:"document_search_projections"`

	ID                  pulid.ID `json:"id"                  keel:"id,pk"`
	OrganizationID      pulid.ID `json:"organizationId"      keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	ResourceID          string   `json:"resourceId"          keel:"resource_id"`
	ResourceType        string   `json:"resourceType"        keel:"resource_type"`
	FileName            string   `json:"fileName"            keel:"file_name"`
	OriginalName        string   `json:"originalName"        keel:"original_name"`
	Description         string   `json:"description"         keel:"description,nullzero"`
	Tags                []string `json:"tags"                keel:"tags"`
	Status              string   `json:"status"              keel:"status"`
	ContentStatus       string   `json:"contentStatus"       keel:"content_status"`
	DetectedKind        string   `json:"detectedKind"        keel:"detected_kind,nullzero"`
	ShipmentDraftStatus string   `json:"shipmentDraftStatus" keel:"shipment_draft_status"`
	ContentText         string   `json:"contentText"         keel:"content_text,nullzero"`
	CreatedAt           int64    `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64    `json:"updatedAt"           keel:"updated_at,autoupdate"`
}

func (p *Projection) BeforeUpdate(_ context.Context) error {
	p.stripNUL()

	return nil
}

func Build(doc *document.Document, contentText string) *Projection {
	contentText = strings.TrimSpace(contentText)
	if len(contentText) > maxIndexedContentChars {
		contentText = contentText[:maxIndexedContentChars]
	}

	return &Projection{
		ID:                  doc.ID,
		OrganizationID:      doc.OrganizationID,
		BusinessUnitID:      doc.BusinessUnitID,
		ResourceID:          doc.ResourceID,
		ResourceType:        doc.ResourceType,
		FileName:            doc.FileName,
		OriginalName:        doc.OriginalName,
		Description:         doc.Description,
		Tags:                doc.Tags,
		Status:              doc.Status.String(),
		ContentStatus:       string(doc.ContentStatus),
		DetectedKind:        doc.DetectedKind,
		ShipmentDraftStatus: string(doc.ShipmentDraftStatus),
		ContentText:         contentText,
		CreatedAt:           doc.CreatedAt,
		UpdatedAt:           doc.UpdatedAt,
	}
}

func (p *Projection) BeforeCreate(_ context.Context) error {
	p.stripNUL()
	now := timeutils.NowUnix()
	if p.CreatedAt == 0 {
		p.CreatedAt = now
	}
	if p.UpdatedAt == 0 {
		p.UpdatedAt = now
	}

	return nil
}

func (p *Projection) BeforeSave(_ context.Context) error {
	p.stripNUL()
	return nil
}

func (p *Projection) stripNUL() {
	p.FileName = stringutils.StripNUL(p.FileName)
	p.OriginalName = stringutils.StripNUL(p.OriginalName)
	p.Description = stringutils.StripNUL(p.Description)
	p.ContentText = stringutils.StripNUL(p.ContentText)
}

func (p *Projection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(p,
		validation.Field(&p.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&p.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// The projection is what search returns, so a row that cannot be traced
		// back to its document would surface a result nobody can open.
		validation.Field(&p.ResourceID,
			validation.Required.Error("Resource is required"),
			validation.Length(1, maxProjectionCodeLength).
				Error("Resource cannot be longer than 100 characters"),
		),
		validation.Field(&p.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.Length(1, maxProjectionCodeLength).
				Error("Resource type cannot be longer than 100 characters"),
		),
		validation.Field(&p.FileName,
			validation.Required.Error("File name is required"),
			validation.Length(1, maxProjectionNameLength).
				Error("File name cannot be longer than 255 characters"),
		),
		validation.Field(&p.OriginalName,
			validation.Required.Error("Original name is required"),
			validation.Length(1, maxProjectionNameLength).
				Error("Original name cannot be longer than 255 characters"),
		),
		validation.Field(&p.Status,
			validation.Required.Error("Status is required"),
			validation.Length(1, maxProjectionCodeLength).
				Error("Status cannot be longer than 100 characters"),
		),
		validation.Field(&p.ContentStatus,
			validation.Required.Error("Content status is required"),
			validation.Length(1, maxProjectionCodeLength).
				Error("Content status cannot be longer than 100 characters"),
		),
		validation.Field(&p.ShipmentDraftStatus,
			validation.Required.Error("Shipment draft status is required"),
			validation.Length(1, maxProjectionCodeLength).
				Error("Shipment draft status cannot be longer than 100 characters"),
		),
		validation.Field(&p.DetectedKind,
			validation.Length(0, maxProjectionCodeLength).
				Error("Detected kind cannot be longer than 100 characters"),
		),
		validation.Field(&p.Tags,
			validation.Each(validation.Length(1, maxProjectionCodeLength).
				Error("A tag must be between 1 and 100 characters")),
		),
	))
}

const (
	maxProjectionCodeLength = 100
	maxProjectionNameLength = 255
)
