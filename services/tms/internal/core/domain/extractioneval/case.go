package extractioneval

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*ExtractionCase)(nil)
	_ pagination.CursorEntity        = (*ExtractionCase)(nil)
	_ domaintypes.PostgresSearchable = (*ExtractionCase)(nil)
)

const (
	MaxTitleRunes    = 200
	MaxNotesRunes    = 2000
	MaxFileNameRunes = 255
	MaxPages         = 100
	MaxPageRunes     = 20000
	InputHashLength  = 64
)

type Page struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}

type ExtractionCase struct {
	bun.BaseModel `bun:"table:extraction_eval_cases,alias:eec" json:"-"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Task                aicorrection.Task      `json:"task"                bun:"task,type:VARCHAR(50),notnull"`
	Status              CaseStatus             `json:"status"              bun:"status,type:VARCHAR(20),notnull,default:'Candidate'"`
	Title               string                 `json:"title"               bun:"title,type:VARCHAR(200),notnull"`
	DocumentKind        string                 `json:"documentKind"        bun:"document_kind,type:VARCHAR(100),nullzero"`
	DocumentFingerprint string                 `json:"documentFingerprint" bun:"document_fingerprint,type:VARCHAR(255),nullzero"`
	FileName            string                 `json:"fileName"            bun:"file_name,type:VARCHAR(255),nullzero"`
	Pages               []Page                 `json:"pages"               bun:"pages,type:JSONB,notnull"`
	PageCount           int                    `json:"pageCount"           bun:"page_count,type:INTEGER,notnull,default:0"`
	InputHash           string                 `json:"inputHash"           bun:"input_hash,type:VARCHAR(64),notnull"`
	Expected            *aicorrection.Snapshot `json:"expected"            bun:"expected,type:JSONB,notnull"`
	ExpectedFieldCount  int                    `json:"expectedFieldCount"  bun:"expected_field_count,type:INTEGER,notnull,default:0"`
	SourceCorrectionID  *pulid.ID              `json:"sourceCorrectionId"  bun:"source_correction_id,type:VARCHAR(100),nullzero"`
	SourceDocumentID    *pulid.ID              `json:"sourceDocumentId"    bun:"source_document_id,type:VARCHAR(100),nullzero"`
	Notes               string                 `json:"notes"               bun:"notes,type:TEXT,nullzero"`
	CreatedByID         pulid.ID               `json:"createdById"         bun:"created_by_id,type:VARCHAR(100),notnull"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func ComputeInputHash(task aicorrection.Task, fileName string, pages []Page) string {
	var b strings.Builder
	b.WriteString(task.String())
	b.WriteByte(0)
	b.WriteString(strings.TrimSpace(fileName))
	for i := range pages {
		b.WriteByte(0)
		b.WriteString(strconv.Itoa(pages[i].Number))
		b.WriteByte(0)
		b.WriteString(pages[i].Text)
	}

	return hashutils.SHA256Hex(b.String())
}

func (c *ExtractionCase) Normalize() {
	c.Title = strings.TrimSpace(c.Title)
	c.Notes = strings.TrimSpace(c.Notes)
	c.FileName = strings.TrimSpace(c.FileName)
	c.PageCount = len(c.Pages)
	if c.Expected != nil {
		c.ExpectedFieldCount = len(c.Expected.Fields)
	}
	c.InputHash = ComputeInputHash(c.Task, c.FileName, c.Pages)
}

func (c *ExtractionCase) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&c.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&c.Task,
			validation.Required.Error("Task is required"),
			domainvalidation.ValidEnum[aicorrection.Task]("Task is invalid"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[CaseStatus]("Status is invalid"),
		),
		validation.Field(&c.Title,
			validation.Required.Error("Title is required"),
			validation.RuneLength(1, MaxTitleRunes).
				Error(fmt.Sprintf("Title must be at most %d characters", MaxTitleRunes)),
		),
		validation.Field(&c.Notes,
			validation.RuneLength(0, MaxNotesRunes).
				Error(fmt.Sprintf("Notes must be at most %d characters", MaxNotesRunes)),
		),
		validation.Field(&c.FileName,
			validation.RuneLength(0, MaxFileNameRunes).
				Error(fmt.Sprintf("File name must be at most %d characters", MaxFileNameRunes)),
		),
		validation.Field(&c.Pages,
			validation.Required.Error("A case needs the document's text to evaluate against"),
			validation.Length(1, MaxPages).
				Error(fmt.Sprintf("A case holds at most %d pages", MaxPages)),
		),
		validation.Field(&c.Expected, validation.NotNil.Error("Expected values are required")),
		validation.Field(&c.CreatedByID, validation.Required.Error("Created by is required")),
		validation.Field(&c.InputHash,
			validation.Required.Error("Input hash is required"),
			validation.Length(InputHashLength, InputHashLength).Error("Input hash is invalid"),
		),
	))

	for i := range c.Pages {
		if len([]rune(c.Pages[i].Text)) > MaxPageRunes {
			multiErr.Add(
				fmt.Sprintf("pages[%d].text", i),
				errortypes.ErrInvalid,
				fmt.Sprintf("A page holds at most %d characters", MaxPageRunes),
			)
		}
	}

	if c.Status == CaseStatusActive && (c.Expected == nil ||
		(len(c.Expected.Fields) == 0 && len(c.Expected.Stops) == 0)) {
		multiErr.Add(
			"expected",
			errortypes.ErrInvalid,
			"An active case needs at least one confirmed value to score against",
		)
	}
}

func (c *ExtractionCase) GetID() pulid.ID { return c.ID }

func (c *ExtractionCase) GetCreatedAt() int64 { return c.CreatedAt }

func (c *ExtractionCase) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *ExtractionCase) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *ExtractionCase) GetTableName() string { return "extraction_eval_cases" }

func (c *ExtractionCase) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "eec",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "title", Type: domaintypes.FieldTypeText},
			{Name: "file_name", Type: domaintypes.FieldTypeText},
			{Name: "document_kind", Type: domaintypes.FieldTypeText},
			{Name: "document_fingerprint", Type: domaintypes.FieldTypeText},
			{Name: "status", Type: domaintypes.FieldTypeEnum},
		},
	}
}

func (c *ExtractionCase) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("eec_")
		}
		if c.CreatedAt == 0 {
			c.CreatedAt = now
		}
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
