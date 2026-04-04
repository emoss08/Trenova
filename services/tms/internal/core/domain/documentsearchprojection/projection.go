package documentsearchprojection

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const maxIndexedContentChars = 100000

type Projection struct {
	bun.BaseModel `bun:"table:document_search_projections,alias:dsp" json:"-"`

	ID                  pulid.ID `json:"id"                  bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID      pulid.ID `json:"organizationId"      bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID      pulid.ID `json:"businessUnitId"      bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	ResourceID          string   `json:"resourceId"          bun:"resource_id,type:VARCHAR(100),notnull"`
	ResourceType        string   `json:"resourceType"        bun:"resource_type,type:VARCHAR(100),notnull"`
	FileName            string   `json:"fileName"            bun:"file_name,type:VARCHAR(255),notnull"`
	OriginalName        string   `json:"originalName"        bun:"original_name,type:VARCHAR(255),notnull"`
	Description         string   `json:"description"         bun:"description,type:TEXT,nullzero"`
	Tags                []string `json:"tags"                bun:"tags,type:VARCHAR(100)[],default:'{}'"`
	Status              string   `json:"status"              bun:"status,type:VARCHAR(100),notnull"`
	ContentStatus       string   `json:"contentStatus"       bun:"content_status,type:VARCHAR(100),notnull"`
	DetectedKind        string   `json:"detectedKind"        bun:"detected_kind,type:VARCHAR(100),nullzero"`
	ShipmentDraftStatus string   `json:"shipmentDraftStatus" bun:"shipment_draft_status,type:VARCHAR(100),notnull"`
	ContentText         string   `json:"contentText"         bun:"content_text,type:TEXT,nullzero"`
	CreatedAt           int64    `json:"createdAt"           bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt           int64    `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull"`
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

func (p *Projection) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if p.CreatedAt == 0 {
			p.CreatedAt = now
		}
		if p.UpdatedAt == 0 {
			p.UpdatedAt = now
		}
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}

	return nil
}
