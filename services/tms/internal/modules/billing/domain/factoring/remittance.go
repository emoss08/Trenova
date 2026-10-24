package factoring

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

// The funding and reserve reports factors sent, how far each was read, matched and booked
type Remittance struct {
	keel.Table `                                                        keel:"factoring_remittances,report=Billing,label=Factoring Remittance"`

	ID                 pulid.ID           `json:"id" keel:"id,pk,autoid:facrem_"`
	OrganizationID     pulid.ID           `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID           `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID pulid.ID           `json:"factoringCompanyId" keel:"factoring_company_id"`
	DocumentID         pulid.ID           `json:"documentId" keel:"document_id,nullzero"`
	FileName           string             `json:"fileName" keel:"file_name,nullzero"`
	Source             RemittanceSource   `json:"source" keel:"source"`
	InboundMessageID   pulid.ID           `json:"inboundMessageId" keel:"inbound_message_id,nullzero"`
	SourceFormat       RemittanceFormat   `json:"sourceFormat" keel:"source_format,nullzero,label=Format"`
	Status             RemittanceStatus   `json:"status" keel:"status"`
	DefaultKind        RemittanceLineKind `json:"defaultKind" keel:"default_kind,label=Report Of"`
	RemittedOn         *int64             `json:"remittedOn" keel:"remitted_on,type=epoch"`
	CurrencyCode       string             `json:"currencyCode" keel:"currency_code,label=Currency"`
	Mapping            map[string]int     `json:"mapping" keel:"mapping,jsonb"`
	Headers            []string           `json:"headers" keel:"headers,jsonb"`
	UnmappedHeaders    []string           `json:"unmappedHeaders" keel:"unmapped_headers,jsonb"`
	LineCount          int                `json:"lineCount" keel:"line_count,label=Lines,format=count"`
	MatchedCount       int                `json:"matchedCount" keel:"matched_count,label=Matched Lines,format=count"`
	ExceptionCount     int                `json:"exceptionCount" keel:"exception_count,label=Lines to Resolve,format=count"`
	AppliedCount       int                `json:"appliedCount" keel:"applied_count,label=Booked Lines,format=count"`
	TotalCashMinor     int64              `json:"totalCashMinor" keel:"total_cash_minor,label=Total Cash (Minor)"`
	Error              string             `json:"error" keel:"error,nullzero"`
	StagedAt           *int64             `json:"stagedAt" keel:"staged_at,label=Read At"`
	AppliedAt          *int64             `json:"appliedAt" keel:"applied_at"`
	AppliedByID        pulid.ID           `json:"appliedById" keel:"applied_by_id,nullzero"`
	DiscardedAt        *int64             `json:"discardedAt" keel:"discarded_at"`
	DiscardReason      string             `json:"discardReason" keel:"discard_reason,nullzero"`
	CreatedByID        pulid.ID           `json:"createdById" keel:"created_by_id,nullzero"`
	Version            int64              `json:"version" keel:"version,version"`
	CreatedAt          int64              `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64              `json:"updatedAt" keel:"updated_at,autoupdate"`

	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
	Lines            []*RemittanceLine                  `json:"lines,omitempty" keel:"many:remittance_id"`
}

type RemittanceSuggestion struct {
	ItemID        pulid.ID              `json:"itemId"`
	SubmissionID  pulid.ID              `json:"submissionId"`
	InvoiceNumber string                `json:"invoiceNumber"`
	Score         int                   `json:"score"`
	Reason        RemittanceMatchReason `json:"reason"`
}

// Each row of a factor's remittance, its amounts as printed and the schedule item it was matched to
type RemittanceLine struct {
	keel.Table `                                                             keel:"factoring_remittance_lines,report=Billing,label=Factoring Remittance Line"`

	ID                pulid.ID                `json:"id" keel:"id,pk,autoid:facrml_"`
	OrganizationID    pulid.ID                `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID                `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	RemittanceID      pulid.ID                `json:"remittanceId" keel:"remittance_id"`
	RowNumber         int                     `json:"rowNumber" keel:"row_number,label=Row,dimension"`
	Cells             []string                `json:"cells" keel:"cells,jsonb"`
	Kind              RemittanceLineKind      `json:"kind" keel:"kind"`
	InvoiceNumber     string                  `json:"invoiceNumber" keel:"invoice_number,nullzero"`
	NormalizedInvoice string                  `json:"-" keel:"normalized_invoice,nullzero"`
	FactorReference   string                  `json:"factorReference" keel:"factor_reference,nullzero"`
	DebtorName        string                  `json:"debtorName" keel:"debtor_name,nullzero,label=Debtor"`
	FaceMinor         *int64                  `json:"faceMinor" keel:"face_minor,label=Invoice Amount (Minor)"`
	AdvanceMinor      *int64                  `json:"advanceMinor" keel:"advance_minor,label=Advance (Minor)"`
	FeeMinor          *int64                  `json:"feeMinor" keel:"fee_minor,label=Fee (Minor)"`
	ReserveMinor      *int64                  `json:"reserveMinor" keel:"reserve_minor,label=Reserve (Minor)"`
	CashMinor         *int64                  `json:"cashMinor" keel:"cash_minor,label=Cash (Minor)"`
	OccurredOn        *int64                  `json:"occurredOn" keel:"occurred_on,type=epoch"`
	MatchStatus       RemittanceMatchStatus   `json:"matchStatus" keel:"match_status"`
	MatchedBy         RemittanceMatchedBy     `json:"matchedBy" keel:"matched_by,nullzero"`
	SubmissionItemID  pulid.ID                `json:"submissionItemId" keel:"submission_item_id,nullzero,label=Schedule Invoice ID"`
	Score             int                     `json:"score" keel:"score,label=Match Score,dimension"`
	Reason            RemittanceMatchReason   `json:"reason" keel:"reason,nullzero,label=Match Reason"`
	Suggestions       []*RemittanceSuggestion `json:"suggestions" keel:"suggestions,jsonb"`
	Error             string                  `json:"error" keel:"error,nullzero"`
	IgnoredReason     string                  `json:"ignoredReason" keel:"ignored_reason,nullzero"`
	ReserveEntryID    pulid.ID                `json:"reserveEntryId" keel:"reserve_entry_id,nullzero"`
	MatchedInvoice    string                  `json:"matchedInvoice" keel:"matched_invoice_number,computed"`
	MatchedSubmission pulid.ID                `json:"-" keel:"matched_submission_id,computed"`
	CreatedAt         int64                   `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt         int64                   `json:"updatedAt" keel:"updated_at,autoupdate"`

	SubmissionItem *SubmissionItem `json:"submissionItem,omitempty" keel:"belongs:submission_item_id"`
}

func (l *RemittanceLine) BeforeUpdate(_ context.Context) error {
	if l.Suggestions == nil {
		l.Suggestions = []*RemittanceSuggestion{}
	}

	return nil
}

type RemittanceProfile struct {
	keel.Table `                                                                keel:"factoring_remittance_profiles"`

	ID                 pulid.ID       `json:"id" keel:"id,pk,autoid:facrmp_"`
	OrganizationID     pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	FactoringCompanyID pulid.ID       `json:"factoringCompanyId" keel:"factoring_company_id"`
	Mapping            map[string]int `json:"mapping" keel:"mapping,jsonb"`
	HeaderSignature    string         `json:"headerSignature" keel:"header_signature"`
	Version            int64          `json:"version" keel:"version,version"`
	CreatedAt          int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64          `json:"updatedAt" keel:"updated_at,autoupdate"`
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

func (r *Remittance) RequireStageable() error {
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
	if err := r.RequireStageable(); err != nil {
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

func (r *Remittance) MarkReading(in *RemittanceStaging) error {
	if err := r.stage(in); err != nil {
		return err
	}
	r.Status = RemittanceStatusPending
	r.Lines = nil
	r.Recount()
	return nil
}

func (r *Remittance) AwaitsReading() bool {
	return r.Status == RemittanceStatusPending && r.DocumentID.IsNotNil() &&
		r.SourceFormat == RemittanceFormatPdf
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
	if err := r.RequireStageable(); err != nil {
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

func (r *Remittance) RequireEditable() error {
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
	if err := r.RequireEditable(); err != nil {
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

func (r *Remittance) BeforeCreate(ctx context.Context) error {
	if r.Source == "" {
		r.Source = RemittanceSourceUpload
	}
	if r.Status == "" {
		r.Status = RemittanceStatusPending
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

func (l *RemittanceLine) BeforeCreate(ctx context.Context) error {
	if l.Kind == "" {
		l.Kind = RemittanceLineFunding
	}
	if l.MatchStatus == "" {
		l.MatchStatus = RemittanceLineUnmatched
	}
	l.Cells = nonNilStrings(l.Cells)
	if l.Suggestions == nil {
		l.Suggestions = []*RemittanceSuggestion{}
	}

	return nil
}

func (p *RemittanceProfile) BeforeCreate(ctx context.Context) error {
	p.Mapping = nonNilMapping(p.Mapping)

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
