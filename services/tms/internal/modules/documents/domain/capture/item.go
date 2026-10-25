package capture

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const maxSuggestionReasonLength = 500

// CaptureItem is a proposed document inside a batch: an ordered run of its pages,
// with a suggestion of where it goes, until a person files it or throws it
// away.
type CaptureItem struct {
	keel.Table `keel:"capture_items"`

	ID                   pulid.ID         `json:"id"                   keel:"id,pk,autoid:citm_"`
	BusinessUnitID       pulid.ID         `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID         `json:"organizationId"       keel:"organization_id,pk,tenant"`
	BatchID              pulid.ID         `json:"batchId"              keel:"batch_id"`
	Position             int              `json:"position"             keel:"position"`
	Status               ItemStatus       `json:"status"               keel:"status"`
	PageIDs              []pulid.ID       `json:"pageIds"              keel:"page_ids"`
	SuggestedType        string           `json:"suggestedType"        keel:"suggested_type,nullzero"`
	SuggestedID          *pulid.ID        `json:"suggestedId"          keel:"suggested_id"`
	SuggestedDocTypeID   *pulid.ID        `json:"suggestedDocTypeId"   keel:"suggested_doc_type_id"`
	SuggestionSource     SuggestionSource `json:"suggestionSource"     keel:"suggestion_source,nullzero"`
	SuggestionConfidence *float64         `json:"suggestionConfidence" keel:"suggestion_confidence"`
	SuggestionReason     string           `json:"suggestionReason"     keel:"suggestion_reason,nullzero"`
	CoverSheetID         *pulid.ID        `json:"coverSheetId"         keel:"cover_sheet_id"`
	DetectedKind         string           `json:"detectedKind"         keel:"detected_kind,nullzero"`
	FiledType            string           `json:"filedType"            keel:"filed_type,nullzero"`
	FiledID              *pulid.ID        `json:"filedId"              keel:"filed_id"`
	FiledDocTypeID       *pulid.ID        `json:"filedDocTypeId"       keel:"filed_doc_type_id"`
	DocumentID           *pulid.ID        `json:"documentId"           keel:"document_id"`
	UploadSessionID      *pulid.ID        `json:"-"                    keel:"upload_session_id"`
	FiledByID            *pulid.ID        `json:"filedById"            keel:"filed_by_id"`
	FiledAt              *int64           `json:"filedAt"              keel:"filed_at"`
	FailureMessage       string           `json:"failureMessage"       keel:"failure_message,nullzero"`
	Version              int64            `json:"version"              keel:"version,version"`
	CreatedAt            int64            `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64            `json:"updatedAt"            keel:"updated_at,autoupdate"`
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

func (i *CaptureItem) GetID() pulid.ID      { return i.ID }
func (i *CaptureItem) GetCreatedAt() int64  { return i.CreatedAt }
func (i *CaptureItem) GetTableName() string { return "capture_items" }

func (i *CaptureItem) BeforeCreate(ctx context.Context) error {
	if i.Status == "" {
		i.Status = ItemProposed
	}

	return nil
}
