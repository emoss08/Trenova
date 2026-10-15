package factoring

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	RemittanceDocumentResourceType = "factoring_remittance"
	MaxRemittanceBytes             = 32 << 20
	MaxRemittanceLines             = 2000
	MaxRemittanceErrorLength       = 1000
	MaxRemittanceReasonLength      = 500
	MaxRemittanceSuggestions       = 3
	MaxRemittanceCellLength        = 255
	remittanceStatusField          = "status"
)

var (
	_ bun.BeforeAppendModelHook = (*Remittance)(nil)
	_ bun.BeforeAppendModelHook = (*RemittanceLine)(nil)
	_ bun.BeforeAppendModelHook = (*RemittanceProfile)(nil)
)

type Remittance struct {
	bun.BaseModel             `bun:"table:factoring_remittances,alias:facrem" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                   json:"-"`

	ID                 pulid.ID           `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID           `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID           `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	FactoringCompanyID pulid.ID           `json:"factoringCompanyId" bun:"factoring_company_id,type:VARCHAR(100),notnull"`
	DocumentID         pulid.ID           `json:"documentId"         bun:"document_id,type:VARCHAR(100),nullzero"`
	FileName           string             `json:"fileName"           bun:"file_name,type:VARCHAR(255),nullzero"`
	Source             RemittanceSource   `json:"source"             bun:"source,type:VARCHAR(20),notnull,default:'Upload'"`
	InboundMessageID   pulid.ID           `json:"inboundMessageId"   bun:"inbound_message_id,type:VARCHAR(100),nullzero"`
	SourceFormat       RemittanceFormat   `json:"sourceFormat"       bun:"source_format,type:VARCHAR(10),nullzero"`
	Status             RemittanceStatus   `json:"status"             bun:"status,type:VARCHAR(30),notnull,default:'Pending'"`
	DefaultKind        RemittanceLineKind `json:"defaultKind"        bun:"default_kind,type:VARCHAR(30),notnull,default:'Funding'"`
	RemittedOn         *int64             `json:"remittedOn"         bun:"remitted_on,type:BIGINT,nullzero"`
	CurrencyCode       string             `json:"currencyCode"       bun:"currency_code,type:VARCHAR(3),notnull,default:'USD'"`
	Mapping            map[string]int     `json:"mapping"            bun:"mapping,type:JSONB,notnull,default:'{}'"`
	Headers            []string           `json:"headers"            bun:"headers,type:JSONB,notnull,default:'[]'"`
	UnmappedHeaders    []string           `json:"unmappedHeaders"    bun:"unmapped_headers,type:JSONB,notnull,default:'[]'"`
	LineCount          int                `json:"lineCount"          bun:"line_count,type:INTEGER,notnull,default:0"`
	MatchedCount       int                `json:"matchedCount"       bun:"matched_count,type:INTEGER,notnull,default:0"`
	ExceptionCount     int                `json:"exceptionCount"     bun:"exception_count,type:INTEGER,notnull,default:0"`
	AppliedCount       int                `json:"appliedCount"       bun:"applied_count,type:INTEGER,notnull,default:0"`
	TotalCashMinor     int64              `json:"totalCashMinor"     bun:"total_cash_minor,type:BIGINT,notnull,default:0"`
	Error              string             `json:"error"              bun:"error,type:VARCHAR(1000),nullzero"`
	StagedAt           *int64             `json:"stagedAt"           bun:"staged_at,type:BIGINT,nullzero"`
	AppliedAt          *int64             `json:"appliedAt"          bun:"applied_at,type:BIGINT,nullzero"`
	AppliedByID        pulid.ID           `json:"appliedById"        bun:"applied_by_id,type:VARCHAR(100),nullzero"`
	DiscardedAt        *int64             `json:"discardedAt"        bun:"discarded_at,type:BIGINT,nullzero"`
	DiscardReason      string             `json:"discardReason"      bun:"discard_reason,type:VARCHAR(500),nullzero"`
	CreatedByID        pulid.ID           `json:"createdById"        bun:"created_by_id,type:VARCHAR(100),nullzero"`
	Version            int64              `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64              `json:"createdAt"          bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt          int64              `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" bun:"rel:belongs-to,join:factoring_company_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	Lines            []*RemittanceLine                  `json:"lines,omitempty"            bun:"rel:has-many,join:id=remittance_id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type RemittanceSuggestion struct {
	ItemID        pulid.ID              `json:"itemId"`
	SubmissionID  pulid.ID              `json:"submissionId"`
	InvoiceNumber string                `json:"invoiceNumber"`
	Score         int                   `json:"score"`
	Reason        RemittanceMatchReason `json:"reason"`
}

type RemittanceLine struct {
	bun.BaseModel             `bun:"table:factoring_remittance_lines,alias:facrml" json:"-"`
	pagination.CursorValueSet `bun:",embed"                                        json:"-"`

	ID                pulid.ID                `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID                `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID                `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	RemittanceID      pulid.ID                `json:"remittanceId"     bun:"remittance_id,type:VARCHAR(100),notnull"`
	RowNumber         int                     `json:"rowNumber"        bun:"row_number,type:INTEGER,notnull"`
	Cells             []string                `json:"cells"            bun:"cells,type:JSONB,notnull,default:'[]'"`
	Kind              RemittanceLineKind      `json:"kind"             bun:"kind,type:VARCHAR(30),notnull,default:'Funding'"`
	InvoiceNumber     string                  `json:"invoiceNumber"    bun:"invoice_number,type:VARCHAR(100),nullzero"`
	NormalizedInvoice string                  `json:"-"                bun:"normalized_invoice,type:VARCHAR(100),nullzero"`
	FactorReference   string                  `json:"factorReference"  bun:"factor_reference,type:VARCHAR(100),nullzero"`
	DebtorName        string                  `json:"debtorName"       bun:"debtor_name,type:VARCHAR(255),nullzero"`
	FaceMinor         *int64                  `json:"faceMinor"        bun:"face_minor,type:BIGINT,nullzero"`
	AdvanceMinor      *int64                  `json:"advanceMinor"     bun:"advance_minor,type:BIGINT,nullzero"`
	FeeMinor          *int64                  `json:"feeMinor"         bun:"fee_minor,type:BIGINT,nullzero"`
	ReserveMinor      *int64                  `json:"reserveMinor"     bun:"reserve_minor,type:BIGINT,nullzero"`
	CashMinor         *int64                  `json:"cashMinor"        bun:"cash_minor,type:BIGINT,nullzero"`
	OccurredOn        *int64                  `json:"occurredOn"       bun:"occurred_on,type:BIGINT,nullzero"`
	MatchStatus       RemittanceMatchStatus   `json:"matchStatus"      bun:"match_status,type:VARCHAR(20),notnull,default:'Unmatched'"`
	MatchedBy         RemittanceMatchedBy     `json:"matchedBy"        bun:"matched_by,type:VARCHAR(10),nullzero"`
	SubmissionItemID  pulid.ID                `json:"submissionItemId" bun:"submission_item_id,type:VARCHAR(100),nullzero"`
	Score             int                     `json:"score"            bun:"score,type:INTEGER,notnull,default:0"`
	Reason            RemittanceMatchReason   `json:"reason"           bun:"reason,type:VARCHAR(255),nullzero"`
	Suggestions       []*RemittanceSuggestion `json:"suggestions"      bun:"suggestions,type:JSONB,notnull,default:'[]'"`
	Error             string                  `json:"error"            bun:"error,type:VARCHAR(500),nullzero"`
	IgnoredReason     string                  `json:"ignoredReason"    bun:"ignored_reason,type:VARCHAR(500),nullzero"`
	ReserveEntryID    pulid.ID                `json:"reserveEntryId"   bun:"reserve_entry_id,type:VARCHAR(100),nullzero"`
	CreatedAt         int64                   `json:"createdAt"        bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt         int64                   `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull"`

	SubmissionItem *SubmissionItem `json:"submissionItem,omitempty" bun:"rel:belongs-to,join:submission_item_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
}

type RemittanceProfile struct {
	bun.BaseModel `bun:"table:factoring_remittance_profiles,alias:facrmp" json:"-"`

	ID                 pulid.ID       `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	FactoringCompanyID pulid.ID       `json:"factoringCompanyId" bun:"factoring_company_id,type:VARCHAR(100),notnull"`
	Mapping            map[string]int `json:"mapping"            bun:"mapping,type:JSONB,notnull,default:'{}'"`
	HeaderSignature    string         `json:"headerSignature"    bun:"header_signature,type:VARCHAR(64),notnull"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull"`
}

func (r *Remittance) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(&r.OrganizationID, validation.Required),
		validation.Field(&r.BusinessUnitID, validation.Required),
		validation.Field(
			&r.FactoringCompanyID,
			validation.Required.Error("Factoring company is required"),
		),
		validation.Field(
			&r.Source,
			validation.Required,
			domainvalidation.ValidEnum[RemittanceSource]("Invalid remittance source"),
		),
		validation.Field(
			&r.Status,
			validation.Required,
			domainvalidation.ValidEnum[RemittanceStatus]("Invalid remittance status"),
		),
		validation.Field(
			&r.DefaultKind,
			validation.Required,
			domainvalidation.ValidEnum[RemittanceLineKind]("Choose what the report records"),
		),
		validation.Field(
			&r.DiscardReason,
			validation.Length(0, MaxRemittanceReasonLength).
				Error("Reason cannot be longer than 500 characters"),
		),
	))
	if r.SourceFormat != "" && !r.SourceFormat.IsValid() {
		multiErr.Add("sourceFormat", errortypes.ErrInvalid, "Invalid remittance file format")
	}
	if r.DefaultKind == RemittanceLineOther {
		multiErr.Add(
			"defaultKind",
			errortypes.ErrInvalid,
			"Choose a funding, rejection, reserve release or chargeback report",
		)
	}
}

func (r *Remittance) requireStageable() error {
	if r.Status.CanStage() {
		return nil
	}
	return errortypes.NewBusinessError(
		"Remittance {0} is {1}; it cannot be read again",
		r.FileName,
		string(r.Status),
	)
}

type RemittanceStaging struct {
	DocumentID      pulid.ID
	FileName        string
	Format          RemittanceFormat
	DefaultKind     RemittanceLineKind
	RemittedOn      *int64
	Headers         []string
	UnmappedHeaders []string
	Mapping         map[string]int
	At              int64
}

func (r *Remittance) stage(in *RemittanceStaging) error {
	if err := r.requireStageable(); err != nil {
		return err
	}
	r.DocumentID = in.DocumentID
	r.FileName = stringutils.TruncateRunes(strings.TrimSpace(in.FileName), MaxRemittanceCellLength)
	r.SourceFormat = in.Format
	if in.DefaultKind != "" {
		r.DefaultKind = in.DefaultKind
	}
	if in.RemittedOn != nil {
		r.RemittedOn = in.RemittedOn
	}
	r.Headers = nonNilStrings(in.Headers)
	r.UnmappedHeaders = nonNilStrings(in.UnmappedHeaders)
	r.Mapping = nonNilMapping(in.Mapping)
	r.Error = ""
	r.StagedAt = &in.At
	return nil
}

func (r *Remittance) MarkNeedsMapping(in *RemittanceStaging) error {
	if err := r.stage(in); err != nil {
		return err
	}
	r.Status = RemittanceStatusNeedsMapping
	r.Lines = nil
	r.Recount()
	return nil
}

func (r *Remittance) MarkParsed(in *RemittanceStaging, lines []*RemittanceLine) error {
	if err := r.stage(in); err != nil {
		return err
	}
	if len(lines) == 0 {
		return errortypes.NewBusinessError("The remittance has no lines to read")
	}
	if len(lines) > MaxRemittanceLines {
		return errortypes.NewBusinessError(
			"A remittance can hold at most {0} lines",
			MaxRemittanceLines,
		)
	}
	r.Status = RemittanceStatusParsed
	r.Lines = lines
	for _, line := range lines {
		line.RemittanceID = r.ID
		line.OrganizationID = r.OrganizationID
		line.BusinessUnitID = r.BusinessUnitID
	}
	r.Recount()
	return nil
}

func (r *Remittance) MarkFailed(documentID pulid.ID, fileName, cause string, at int64) error {
	if err := r.requireStageable(); err != nil {
		return err
	}
	if !documentID.IsNil() {
		r.DocumentID = documentID
	}
	if fileName = strings.TrimSpace(fileName); fileName != "" {
		r.FileName = stringutils.TruncateRunes(fileName, MaxRemittanceCellLength)
	}
	r.Status = RemittanceStatusFailed
	r.Error = stringutils.TruncateRunes(strings.TrimSpace(cause), MaxRemittanceErrorLength)
	if r.Error == "" {
		r.Error = "The remittance could not be read"
	}
	r.StagedAt = &at
	r.Lines = nil
	r.Recount()
	return nil
}

func (r *Remittance) Recount() {
	r.LineCount = 0
	r.MatchedCount = 0
	r.ExceptionCount = 0
	r.AppliedCount = 0
	r.TotalCashMinor = 0
	for _, line := range r.Lines {
		if line == nil {
			continue
		}
		r.LineCount++
		switch {
		case line.MatchStatus == RemittanceLineMatched:
			r.MatchedCount++
		case line.MatchStatus == RemittanceLineApplied:
			r.AppliedCount++
		case line.MatchStatus.IsException():
			r.ExceptionCount++
		}
		if line.MatchStatus != RemittanceLineIgnored && line.MatchStatus != RemittanceLineError {
			r.TotalCashMinor += line.SignedCash()
		}
	}
}

func (r *Remittance) requireEditable() error {
	if r.Status.CanEditLines() {
		return nil
	}
	return errortypes.NewBusinessError(
		"Remittance {0} is {1}; its lines cannot be changed",
		r.FileName,
		string(r.Status),
	)
}

func (r *Remittance) RequireApplicable() error {
	if !r.Status.CanApply() {
		return errortypes.NewBusinessError(
			"Remittance {0} is {1}; only a read remittance can be applied",
			r.FileName,
			string(r.Status),
		)
	}
	for _, line := range r.Lines {
		if line != nil && line.MatchStatus == RemittanceLineMatched {
			return nil
		}
	}
	return errortypes.NewBusinessError(
		"Nothing on remittance {0} is matched yet; match a line or ignore the rest",
		r.FileName,
	)
}

func (r *Remittance) Line(id pulid.ID) *RemittanceLine {
	for _, line := range r.Lines {
		if line != nil && line.ID == id {
			return line
		}
	}
	return nil
}

func (r *Remittance) EditableLine(id pulid.ID) (*RemittanceLine, error) {
	if err := r.requireEditable(); err != nil {
		return nil, err
	}
	line := r.Line(id)
	if line == nil {
		return nil, errortypes.NewValidationError(
			"lineId",
			errortypes.ErrInvalid,
			"That line is not on this remittance",
		)
	}
	if line.MatchStatus == RemittanceLineApplied {
		return nil, errortypes.NewBusinessError(
			"Row {0} is already booked; it cannot be changed",
			line.RowNumber,
		)
	}
	return line, nil
}

func (r *Remittance) MarkApplied(at int64, by pulid.ID) {
	r.Recount()
	r.AppliedAt = &at
	r.AppliedByID = by
	r.Status = RemittanceStatusPartiallyApplied
	if r.LineCount > 0 && r.settled() {
		r.Status = RemittanceStatusApplied
	}
}

func (r *Remittance) settled() bool {
	for _, line := range r.Lines {
		if line != nil && !line.MatchStatus.IsSettled() {
			return false
		}
	}
	return true
}

func (r *Remittance) SettleIfDone() {
	r.Recount()
	if r.Status == RemittanceStatusPartiallyApplied && r.settled() {
		r.Status = RemittanceStatusApplied
	}
}

func (r *Remittance) Discard(at int64, reason string) error {
	if !r.Status.CanDiscard() {
		if r.Status == RemittanceStatusPartiallyApplied {
			return errortypes.NewBusinessError(
				"Part of remittance {0} is booked; ignore the lines that are left instead",
				r.FileName,
			)
		}
		return errortypes.NewBusinessError(
			"Remittance {0} is {1}; it cannot be discarded",
			r.FileName,
			string(r.Status),
		)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the remittance is being discarded",
		)
	}
	r.Status = RemittanceStatusDiscarded
	r.DiscardedAt = &at
	r.DiscardReason = stringutils.TruncateRunes(reason, MaxRemittanceReasonLength)
	return nil
}

func (r *Remittance) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("facrem_")
		}
		if r.Status == "" {
			r.Status = RemittanceStatusPending
		}
		if r.Source == "" {
			r.Source = RemittanceSourceUpload
		}
		if r.DefaultKind == "" {
			r.DefaultKind = RemittanceLineFunding
		}
		if r.CurrencyCode == "" {
			r.CurrencyCode = "USD"
		}
		r.Mapping = nonNilMapping(r.Mapping)
		r.Headers = nonNilStrings(r.Headers)
		r.UnmappedHeaders = nonNilStrings(r.UnmappedHeaders)
		r.CreatedAt = now
		r.UpdatedAt = now
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}
	return nil
}

func (r *Remittance) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "facrem",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "file_name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   remittanceStatusField,
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (r *Remittance) GetID() pulid.ID             { return r.ID }
func (r *Remittance) GetCreatedAt() int64         { return r.CreatedAt }
func (r *Remittance) GetOrganizationID() pulid.ID { return r.OrganizationID }
func (r *Remittance) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }
func (r *Remittance) GetTableName() string        { return "factoring_remittances" }

func (l *RemittanceLine) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "facrml",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "invoice_number",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "factor_reference",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "debtor_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (l *RemittanceLine) GetID() pulid.ID             { return l.ID }
func (l *RemittanceLine) GetCreatedAt() int64         { return l.CreatedAt }
func (l *RemittanceLine) GetOrganizationID() pulid.ID { return l.OrganizationID }
func (l *RemittanceLine) GetBusinessUnitID() pulid.ID { return l.BusinessUnitID }
func (l *RemittanceLine) GetTableName() string        { return "factoring_remittance_lines" }

func (l *RemittanceLine) SignedCash() int64 {
	if l.CashMinor != nil {
		return *l.CashMinor
	}
	if l.Kind == RemittanceLineFunding && l.AdvanceMinor != nil {
		return *l.AdvanceMinor
	}
	return 0
}

func (l *RemittanceLine) MatchTo(
	item *SubmissionItem,
	by RemittanceMatchedBy,
	score int,
	reason RemittanceMatchReason,
) {
	l.MatchStatus = RemittanceLineMatched
	l.MatchedBy = by
	l.SubmissionItemID = item.ID
	l.SubmissionItem = item
	l.Score = intutils.Clamp(score, 0, 100)
	l.Reason = reason
	l.IgnoredReason = ""
}

func (l *RemittanceLine) Suggest(
	suggestions []*RemittanceSuggestion,
	reason RemittanceMatchReason,
) {
	l.MatchedBy = ""
	l.SubmissionItemID = ""
	l.SubmissionItem = nil
	l.Suggestions = trimSuggestions(suggestions)
	if len(l.Suggestions) == 0 {
		l.MatchStatus = RemittanceLineUnmatched
		l.Score = 0
		l.Reason = reason
		return
	}
	l.MatchStatus = RemittanceLineSuggested
	l.Score = l.Suggestions[0].Score
	l.Reason = reason
	if reason == "" {
		l.Reason = l.Suggestions[0].Reason
	}
}

func (l *RemittanceLine) Ignore(reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why this line is not booked",
		)
	}
	l.MatchStatus = RemittanceLineIgnored
	l.MatchedBy = ""
	l.SubmissionItemID = ""
	l.SubmissionItem = nil
	l.IgnoredReason = stringutils.TruncateRunes(reason, MaxRemittanceReasonLength)
	return nil
}

func (l *RemittanceLine) Reopen() {
	l.MatchStatus = RemittanceLineUnmatched
	l.MatchedBy = ""
	l.SubmissionItemID = ""
	l.SubmissionItem = nil
	l.Score = 0
	l.Reason = ""
	l.IgnoredReason = ""
	if strings.TrimSpace(l.Error) != "" {
		l.MatchStatus = RemittanceLineError
	}
}

func (l *RemittanceLine) SetKind(kind RemittanceLineKind) error {
	if !kind.IsValid() {
		return errortypes.NewValidationError("kind", errortypes.ErrInvalid, "Invalid line kind")
	}
	l.Kind = kind
	l.Reopen()
	return nil
}

func (l *RemittanceLine) MarkApplied(entryID pulid.ID) {
	l.MatchStatus = RemittanceLineApplied
	l.ReserveEntryID = entryID
}

func (l *RemittanceLine) Matchable() bool {
	return l.MatchStatus != RemittanceLineApplied &&
		l.MatchStatus != RemittanceLineIgnored &&
		l.MatchStatus != RemittanceLineError &&
		(l.MatchStatus != RemittanceLineMatched || l.MatchedBy != RemittanceMatchedByPerson)
}

func (l *RemittanceLine) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("facrml_")
		}
		if l.MatchStatus == "" {
			l.MatchStatus = RemittanceLineUnmatched
		}
		if l.Kind == "" {
			l.Kind = RemittanceLineFunding
		}
		l.Cells = nonNilStrings(l.Cells)
		if l.Suggestions == nil {
			l.Suggestions = []*RemittanceSuggestion{}
		}
		l.CreatedAt = now
		l.UpdatedAt = now
	case *bun.UpdateQuery:
		if l.Suggestions == nil {
			l.Suggestions = []*RemittanceSuggestion{}
		}
		l.UpdatedAt = now
	}
	return nil
}

func (p *RemittanceProfile) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("facrmp_")
		}
		p.Mapping = nonNilMapping(p.Mapping)
		p.CreatedAt = now
		p.UpdatedAt = now
	case *bun.UpdateQuery:
		p.UpdatedAt = now
	}
	return nil
}

func trimSuggestions(suggestions []*RemittanceSuggestion) []*RemittanceSuggestion {
	if len(suggestions) > MaxRemittanceSuggestions {
		suggestions = suggestions[:MaxRemittanceSuggestions]
	}
	if suggestions == nil {
		return []*RemittanceSuggestion{}
	}
	return suggestions
}

func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func nonNilMapping(mapping map[string]int) map[string]int {
	if mapping == nil {
		return map[string]int{}
	}
	return mapping
}
