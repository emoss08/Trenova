package extractioneval

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
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
	Number   int    `json:"number"`
	Text     string `json:"text"`
	Markdown bool   `json:"markdown,omitempty"`
}

type ExtractionCase struct {
	keel.Table `keel:"extraction_eval_cases"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:eec_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Task                aicorrection.Task      `json:"task" keel:"task"`
	Status              CaseStatus             `json:"status" keel:"status"`
	Title               string                 `json:"title" keel:"title"`
	DocumentKind        string                 `json:"documentKind" keel:"document_kind,nullzero"`
	DocumentFingerprint string                 `json:"documentFingerprint" keel:"document_fingerprint,nullzero"`
	FileName            string                 `json:"fileName" keel:"file_name,nullzero"`
	Pages               []Page                 `json:"pages" keel:"pages,jsonb"`
	PageCount           int                    `json:"pageCount" keel:"page_count"`
	InputHash           string                 `json:"inputHash" keel:"input_hash"`
	Expected            *aicorrection.Snapshot `json:"expected" keel:"expected,jsonb"`
	ExpectedFieldCount  int                    `json:"expectedFieldCount" keel:"expected_field_count"`
	SourceCorrectionID  *pulid.ID              `json:"sourceCorrectionId" keel:"source_correction_id"`
	SourceDocumentID    *pulid.ID              `json:"sourceDocumentId" keel:"source_document_id"`
	Notes               string                 `json:"notes" keel:"notes,nullzero"`
	CreatedByID         pulid.ID               `json:"createdById" keel:"created_by_id"`

	Version   int64 `json:"version" keel:"version,version"`
	CreatedAt int64 `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt int64 `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
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

func (c *ExtractionCase) BeforeCreate(_ context.Context) error {
	if c.Status == "" {
		c.Status = CaseStatusCandidate
	}

	return nil
}
