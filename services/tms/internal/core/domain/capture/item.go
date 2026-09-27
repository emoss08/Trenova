package capture

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const maxSuggestionReasonLength = 500

// CaptureItem is a proposed document inside a batch: an ordered run of its pages,
// with a suggestion of where it goes, until a person files it or throws it
// away.
type CaptureItem struct {
	bun.BaseModel `bun:"table:capture_items,alias:citm" json:"-"`

	ID                   pulid.ID         `json:"id"                   bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID       pulid.ID         `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID         `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BatchID              pulid.ID         `json:"batchId"              bun:"batch_id,type:VARCHAR(100),notnull"`
	Position             int              `json:"position"             bun:"position,type:INTEGER,notnull"`
	Status               ItemStatus       `json:"status"               bun:"status,type:VARCHAR(20),notnull"`
	PageIDs              []pulid.ID       `json:"pageIds"              bun:"page_ids,type:TEXT[],array,notnull"`
	SuggestedType        string           `json:"suggestedType"        bun:"suggested_type,type:VARCHAR(50),nullzero"`
	SuggestedID          *pulid.ID        `json:"suggestedId"          bun:"suggested_id,type:VARCHAR(100),nullzero"`
	SuggestedDocTypeID   *pulid.ID        `json:"suggestedDocTypeId"   bun:"suggested_doc_type_id,type:VARCHAR(100),nullzero"`
	SuggestionSource     SuggestionSource `json:"suggestionSource"     bun:"suggestion_source,type:VARCHAR(20),nullzero"`
	SuggestionConfidence *float64         `json:"suggestionConfidence" bun:"suggestion_confidence,type:NUMERIC(4,3),nullzero"`
	SuggestionReason     string           `json:"suggestionReason"     bun:"suggestion_reason,type:VARCHAR(500),nullzero"`
	CoverSheetID         *pulid.ID        `json:"coverSheetId"         bun:"cover_sheet_id,type:VARCHAR(100),nullzero"`
	DetectedKind         string           `json:"detectedKind"         bun:"detected_kind,type:VARCHAR(50),nullzero"`
	FiledType            string           `json:"filedType"            bun:"filed_type,type:VARCHAR(50),nullzero"`
	FiledID              *pulid.ID        `json:"filedId"              bun:"filed_id,type:VARCHAR(100),nullzero"`
	FiledDocTypeID       *pulid.ID        `json:"filedDocTypeId"       bun:"filed_doc_type_id,type:VARCHAR(100),nullzero"`
	DocumentID           *pulid.ID        `json:"documentId"           bun:"document_id,type:VARCHAR(100),nullzero"`
	UploadSessionID      *pulid.ID        `json:"-"                    bun:"upload_session_id,type:VARCHAR(100),nullzero"`
	FiledByID            *pulid.ID        `json:"filedById"            bun:"filed_by_id,type:VARCHAR(100),nullzero"`
	FiledAt              *int64           `json:"filedAt"              bun:"filed_at,type:BIGINT,nullzero"`
	FailureMessage       string           `json:"failureMessage"       bun:"failure_message,type:VARCHAR(500),nullzero"`
	Version              int64            `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64            `json:"createdAt"            bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64            `json:"updatedAt"            bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (i *CaptureItem) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.BatchID, validation.Required.Error("Batch is required")),
		validation.Field(&i.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ItemStatus]("Status is not an item status"),
		),
		validation.Field(&i.PageIDs,
			validation.Required.Error("A document needs at least one page"),
			validation.Length(1, MaxBatchPages),
		),
		validation.Field(
			&i.SuggestionSource,
			domainvalidation.ValidEnum[SuggestionSource](
				"Suggestion source is not one Trenova recognises",
			),
		),
		validation.Field(&i.SuggestionReason, validation.Length(0, maxSuggestionReasonLength)),
		validation.Field(&i.FailureMessage, validation.Length(0, maxFailureMessageLength)),
	))

	if i.SuggestionConfidence != nil &&
		(*i.SuggestionConfidence < 0 || *i.SuggestionConfidence > 1) {
		multiErr.Add("suggestionConfidence", errortypes.ErrInvalid,
			"Confidence must be between 0 and 1")
	}

	i.Suggestion().Validate(multiErr, TargetFields{Type: "suggestedType", ID: "suggestedId"}, false)
}

// Suggestion is where the item was proposed to go.
func (i *CaptureItem) Suggestion() Target {
	return Target{
		ResourceType:   i.SuggestedType,
		ResourceID:     i.SuggestedID,
		DocumentTypeID: i.SuggestedDocTypeID,
	}
}

// Suggest records a proposed destination and where it came from.
func (i *CaptureItem) Suggest(
	target Target,
	source SuggestionSource,
	confidence float64,
	reason string,
) {
	i.SuggestedType = target.ResourceType
	i.SuggestedID = target.ResourceID
	i.SuggestedDocTypeID = target.DocumentTypeID
	i.SuggestionSource = source
	i.SuggestionConfidence = &confidence
	i.SuggestionReason = reason
}

// PageCount is how many pages the item holds.
func (i *CaptureItem) PageCount() int {
	return len(i.PageIDs)
}

func (i *CaptureItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("citm_")
		}
		if i.Status == "" {
			i.Status = ItemProposed
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (i *CaptureItem) GetID() pulid.ID      { return i.ID }
func (i *CaptureItem) GetCreatedAt() int64  { return i.CreatedAt }
func (i *CaptureItem) GetTableName() string { return "capture_items" }
