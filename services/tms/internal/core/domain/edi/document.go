package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const DefaultX12204Version = "004010"

type AcknowledgmentConfig struct {
	Expected           bool               `json:"expected"`
	Type               AcknowledgmentType `json:"type"`
	SLAInMinutes       int64              `json:"slaInMinutes"`
	MissingAckSeverity ValidationSeverity `json:"missingAckSeverity"`
}

type X12EnvelopeSettings struct {
	InterchangeSenderQualifier   string `json:"interchangeSenderQualifier"`
	InterchangeSenderID          string `json:"interchangeSenderId"`
	InterchangeReceiverQualifier string `json:"interchangeReceiverQualifier"`
	InterchangeReceiverID        string `json:"interchangeReceiverId"`
	ApplicationSenderCode        string `json:"applicationSenderCode"`
	ApplicationReceiverCode      string `json:"applicationReceiverCode"`
	InterchangeUsageIndicator    string `json:"interchangeUsageIndicator"`
	ElementSeparator             string `json:"elementSeparator"`
	SegmentTerminator            string `json:"segmentTerminator"`
	ComponentSeparator           string `json:"componentSeparator"`
	RepetitionSeparator          string `json:"repetitionSeparator"`
}

func DefaultX12EnvelopeSettings() X12EnvelopeSettings {
	return X12EnvelopeSettings{
		InterchangeSenderQualifier:   "ZZ",
		InterchangeSenderID:          "TRENOVA",
		InterchangeReceiverQualifier: "ZZ",
		InterchangeReceiverID:        "PARTNER",
		ApplicationSenderCode:        "TRENOVA",
		ApplicationReceiverCode:      "PARTNER",
		InterchangeUsageIndicator:    "T",
		ElementSeparator:             "*",
		SegmentTerminator:            "~",
		ComponentSeparator:           ">",
		RepetitionSeparator:          "^",
	}
}

type TemplateElementSource string

const (
	TemplateElementSourceConstant       = TemplateElementSource("constant")
	TemplateElementSourceFieldPath      = TemplateElementSource("fieldPath")
	TemplateElementSourcePartnerSetting = TemplateElementSource("partnerSetting")
	TemplateElementSourceMapping        = TemplateElementSource("mapping")
	TemplateElementSourceRuntime        = TemplateElementSource("runtime")
	TemplateElementSourceRepeat         = TemplateElementSource("repeat")
	TemplateElementSourceTransform      = TemplateElementSource("transform")
	TemplateElementSourceStarlark       = TemplateElementSource("starlark")
)

type TemplateValidationRule struct {
	Required  bool   `json:"required"`
	MaxLength int    `json:"maxLength"`
	MinLength int    `json:"minLength"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type TemplateElementBaseSource struct {
	Source             TemplateElementSource `json:"source"`
	Value              string                `json:"value,omitempty"`
	FieldPath          string                `json:"fieldPath,omitempty"`
	PartnerSettingPath string                `json:"partnerSettingPath,omitempty"`
	MappingEntityType  MappingEntityType     `json:"mappingEntityType,omitempty"`
	MappingSourcePath  string                `json:"mappingSourcePath,omitempty"`
	RuntimeKey         string                `json:"runtimeKey,omitempty"`
	RepeatPath         string                `json:"repeatPath,omitempty"`
	Default            string                `json:"default,omitempty"`
}

type TemplateTransformStep struct {
	Operation string         `json:"operation"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

type TemplateElement struct {
	Position                int                        `json:"position"`
	Name                    string                     `json:"name"`
	Source                  TemplateElementSource      `json:"source"`
	Value                   string                     `json:"value"`
	FieldPath               string                     `json:"fieldPath"`
	PartnerSettingPath      string                     `json:"partnerSettingPath"`
	MappingEntityType       MappingEntityType          `json:"mappingEntityType,omitempty"`
	MappingSourcePath       string                     `json:"mappingSourcePath"`
	RuntimeKey              string                     `json:"runtimeKey"`
	RepeatPath              string                     `json:"repeatPath"`
	BaseSource              *TemplateElementBaseSource `json:"baseSource,omitempty"`
	TransformPipeline       []TemplateTransformStep    `json:"transformPipeline,omitempty"`
	StarlarkFunction        string                     `json:"starlarkFunction"`
	StarlarkScript          string                     `json:"starlarkScript"`
	Default                 string                     `json:"default"`
	Condition               string                     `json:"condition"`
	Validation              TemplateValidationRule     `json:"validation"`
	ImplementationGuideNote string                     `json:"implementationGuideNote"`
}

type EDIDocumentType struct {
	bun.BaseModel `json:"-" bun:"table:edi_document_types,alias:edt"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100),notnull"`
	Code             string            `json:"code"             bun:"code,type:VARCHAR(20),notnull"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(200),notnull"`
	Standard         EDIStandard       `json:"standard"         bun:"standard,type:edi_standard_enum,notnull"`
	TransactionSet   TransactionSet    `json:"transactionSet"   bun:"transaction_set,type:edi_transaction_set_enum,notnull"`
	TransactionSetID pulid.ID          `json:"transactionSetId" bun:"transaction_set_id,type:VARCHAR(100),notnull"`
	Direction        DocumentDirection `json:"direction"        bun:"direction,type:edi_document_direction_enum,notnull"`
	DefaultVersion   string            `json:"defaultVersion"   bun:"default_version,type:VARCHAR(20),notnull"`
	Status           DocumentStatus    `json:"status"           bun:"status,type:edi_document_status_enum,notnull"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	TransactionSetRef *EDITransactionSet `json:"transactionSetRef,omitempty" bun:"rel:belongs-to,join:transaction_set_id=id"`
}

func (d *EDIDocumentType) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if d.ID.IsNil() {
			d.ID = pulid.MustNew("edidt_")
		}
		d.CreatedAt = now
	case *bun.UpdateQuery:
		d.UpdatedAt = now
	}
	return nil
}

type EDITemplateVersion struct {
	bun.BaseModel `json:"-" bun:"table:edi_template_versions,alias:etv"`

	ID                 pulid.ID       `json:"id"                 bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID     pulid.ID       `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	TemplateID         pulid.ID       `json:"templateId"         bun:"template_id,type:VARCHAR(100),notnull"`
	SourceVersionID    pulid.ID       `json:"sourceVersionId"    bun:"source_version_id,type:VARCHAR(100),nullzero"`
	VersionNumber      int64          `json:"versionNumber"      bun:"version_number,type:BIGINT,notnull"`
	X12Version         string         `json:"x12Version"         bun:"x12_version,type:VARCHAR(20),notnull"`
	FunctionalGroupID  string         `json:"functionalGroupId"  bun:"functional_group_id,type:VARCHAR(2),notnull"`
	Status             TemplateStatus `json:"status"             bun:"status,type:edi_template_status_enum,notnull"`
	IsActive           bool           `json:"isActive"           bun:"is_active,type:BOOLEAN,notnull,default:false"`
	Notes              string         `json:"notes"              bun:"notes,type:TEXT,nullzero"`
	CertificationNotes string         `json:"certificationNotes" bun:"certification_notes,type:TEXT,nullzero"`
	ActivationNotes    string         `json:"activationNotes"    bun:"activation_notes,type:TEXT,nullzero"`
	ArchiveNotes       string         `json:"archiveNotes"       bun:"archive_notes,type:TEXT,nullzero"`
	DeprecatedNotes    string         `json:"deprecatedNotes"    bun:"deprecated_notes,type:TEXT,nullzero"`
	SupersededNotes    string         `json:"supersededNotes"    bun:"superseded_notes,type:TEXT,nullzero"`
	CertifiedByID      pulid.ID       `json:"certifiedById"      bun:"certified_by_id,type:VARCHAR(100),nullzero"`
	ActivatedByID      pulid.ID       `json:"activatedById"      bun:"activated_by_id,type:VARCHAR(100),nullzero"`
	ArchivedByID       pulid.ID       `json:"archivedById"       bun:"archived_by_id,type:VARCHAR(100),nullzero"`
	DeprecatedByID     pulid.ID       `json:"deprecatedById"     bun:"deprecated_by_id,type:VARCHAR(100),nullzero"`
	SupersededByID     pulid.ID       `json:"supersededById"     bun:"superseded_by_id,type:VARCHAR(100),nullzero"`
	CertifiedAt        *int64         `json:"certifiedAt"        bun:"certified_at,type:BIGINT,nullzero"`
	ActivatedAt        *int64         `json:"activatedAt"        bun:"activated_at,type:BIGINT,nullzero"`
	ArchivedAt         *int64         `json:"archivedAt"         bun:"archived_at,type:BIGINT,nullzero"`
	DeprecatedAt       *int64         `json:"deprecatedAt"       bun:"deprecated_at,type:BIGINT,nullzero"`
	SupersededAt       *int64         `json:"supersededAt"       bun:"superseded_at,type:BIGINT,nullzero"`
	Version            int64          `json:"version"            bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt          int64          `json:"createdAt"          bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64          `json:"updatedAt"          bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Template        *EDITemplate                `json:"template,omitempty"        bun:"rel:belongs-to,join:template_id=id"`
	SourceVersion   *EDITemplateVersion         `json:"sourceVersion,omitempty"   bun:"rel:belongs-to,join:source_version_id=id"`
	Segments        []*EDITemplateSegment       `json:"segments,omitempty"        bun:"rel:has-many,join:id=template_version_id"`
	ScriptLibraries []*EDITemplateScriptLibrary `json:"scriptLibraries,omitempty" bun:"rel:has-many,join:id=template_version_id"`
}

func (v *EDITemplateVersion) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if v.ID.IsNil() {
			v.ID = pulid.MustNew("editv_")
		}
		v.CreatedAt = now
	case *bun.UpdateQuery:
		v.UpdatedAt = now
	}
	return nil
}

func (v *EDITemplateVersion) GetID() pulid.ID {
	return v.ID
}

func (v *EDITemplateVersion) GetOrganizationID() pulid.ID {
	return v.OrganizationID
}

func (v *EDITemplateVersion) GetBusinessUnitID() pulid.ID {
	return v.BusinessUnitID
}

type EDITemplateSegment struct {
	bun.BaseModel `json:"-" bun:"table:edi_template_segments,alias:ets"`

	ID                pulid.ID          `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID          `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	TemplateVersionID pulid.ID          `json:"templateVersionId" bun:"template_version_id,type:VARCHAR(100),notnull"`
	SegmentID         string            `json:"segmentId"         bun:"segment_id,type:VARCHAR(10),notnull"`
	Name              string            `json:"name"              bun:"name,type:VARCHAR(200),notnull"`
	Sequence          int64             `json:"sequence"          bun:"sequence,type:BIGINT,notnull"`
	LoopID            string            `json:"loopId"            bun:"loop_id,type:VARCHAR(50),nullzero"`
	RepeatPath        string            `json:"repeatPath"        bun:"repeat_path,type:TEXT,nullzero"`
	Condition         string            `json:"condition"         bun:"condition,type:TEXT,nullzero"`
	Required          bool              `json:"required"          bun:"required,type:BOOLEAN,notnull,default:false"`
	MaxUse            int64             `json:"maxUse"            bun:"max_use,type:BIGINT,notnull,default:1"`
	Elements          []TemplateElement `json:"elements"          bun:"elements,type:JSONB,notnull,default:'[]'::jsonb"`
	UsageNotes        string            `json:"usageNotes"        bun:"usage_notes,type:TEXT,nullzero"`
	CreatedAt         int64             `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64             `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (s *EDITemplateSegment) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if s.Elements == nil {
		s.Elements = []TemplateElement{}
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("edisg_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}
	return nil
}

type EDITemplateScriptLibrary struct {
	bun.BaseModel `json:"-" bun:"table:edi_template_script_libraries,alias:etsl"`

	ID                pulid.ID       `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID    pulid.ID       `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	TemplateVersionID pulid.ID       `json:"templateVersionId" bun:"template_version_id,type:VARCHAR(100),notnull"`
	Name              string         `json:"name"              bun:"name,type:VARCHAR(200),notnull"`
	Description       string         `json:"description"       bun:"description,type:TEXT,nullzero"`
	Language          ScriptLanguage `json:"language"          bun:"language,type:edi_script_language_enum,notnull"`
	Script            string         `json:"script"            bun:"script,type:TEXT,notnull"`
	Status            TemplateStatus `json:"status"            bun:"status,type:edi_template_status_enum,notnull"`
	Version           int64          `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt         int64          `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64          `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	FunctionNames   []string            `json:"functionNames"             bun:"-"`
	TemplateVersion *EDITemplateVersion `json:"templateVersion,omitempty" bun:"rel:belongs-to,join:template_version_id=id"`
}

func (l *EDITemplateScriptLibrary) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("edisl_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}
	return nil
}

func (t *EDIDocumentType) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Code cannot be longer than 20 characters"),
		),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&t.Standard,
			validation.Required.Error("Standard is required"),
			domainvalidation.ValidEnum[EDIStandard]("Standard is invalid"),
		),
		validation.Field(&t.TransactionSet,
			validation.Required.Error("Transaction set is required"),
			domainvalidation.ValidEnum[TransactionSet]("Transaction set is invalid"),
		),
		validation.Field(&t.TransactionSetID,
			validation.Required.Error("Transaction set definition is required"),
		),
		validation.Field(&t.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[DocumentDirection]("Direction is invalid"),
		),
		validation.Field(&t.DefaultVersion,
			validation.Required.Error("Default version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("Default version cannot be longer than 20 characters"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[DocumentStatus]("Status is invalid"),
		),
	))
}

func (v *EDITemplateVersion) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(v,
		validation.Field(&v.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&v.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&v.TemplateID, validation.Required.Error("Template is required")),
		validation.Field(&v.VersionNumber,
			validation.Required.Error("Version number is required"),
			validation.Min(int64(1)).Error("Version number must be at least one"),
		),
		validation.Field(&v.X12Version,
			validation.Required.Error("X12 version is required"),
			validation.Length(1, maxEDICodeLength).
				Error("X12 version cannot be longer than 20 characters"),
		),
		// The functional group id is the two character code that goes on the GS
		// segment, so anything wider could not be written to the envelope.
		validation.Field(&v.FunctionalGroupID,
			validation.Required.Error("Functional group is required"),
			validation.Length(1, functionalGroupIDLength).
				Error("Functional group must be at most 2 characters"),
		),
		validation.Field(&v.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TemplateStatus]("Status is invalid"),
		),
		validation.Field(&v.CertifiedAt,
			validation.When(v.CertifiedByID.IsNotNil(), validation.Required.Error(
				"A certified version must record when it was certified",
			)),
		),
		validation.Field(&v.ActivatedAt,
			validation.When(v.ActivatedByID.IsNotNil(), validation.Required.Error(
				"An activated version must record when it was activated",
			)),
		),
		validation.Field(&v.ArchivedAt,
			validation.When(v.ArchivedByID.IsNotNil(), validation.Required.Error(
				"An archived version must record when it was archived",
			)),
		),
		// Only one version of a template can be the live one, and only a
		// certified or active version is allowed to be it.
		validation.Field(&v.IsActive,
			validation.When(
				v.Status != TemplateStatusActive && v.Status != TemplateStatusCertified,
				validation.Empty.Error("Only a certified or active version can be live"),
			),
		),
	))

	for i, segment := range v.Segments {
		if segment == nil {
			continue
		}
		segment.Validate(multiErr.WithIndex("segments", i))
	}

	for i, library := range v.ScriptLibraries {
		if library == nil {
			continue
		}
		library.Validate(multiErr.WithIndex("scriptLibraries", i))
	}
}

func (s *EDITemplateSegment) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.TemplateVersionID,
			validation.Required.Error("Template version is required"),
		),
		validation.Field(&s.SegmentID,
			validation.Required.Error("Segment is required"),
			validation.Length(1, maxSegmentIDLength).
				Error("Segment cannot be longer than 10 characters"),
		),
		validation.Field(&s.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&s.Sequence,
			validation.Min(int64(0)).Error("Sequence cannot be negative"),
		),
		validation.Field(&s.LoopID,
			validation.Length(0, maxLoopIDLength).
				Error("Loop cannot be longer than 50 characters"),
		),
		// A segment allowed zero uses would never be written, which is a way of
		// disabling it that leaves no trace of the intent.
		validation.Field(&s.MaxUse,
			validation.Required.Error("Max use is required"),
			validation.Min(int64(1)).Error("Max use must be at least one"),
		),
	))
}

func (l *EDITemplateScriptLibrary) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&l.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&l.TemplateVersionID,
			validation.Required.Error("Template version is required"),
		),
		validation.Field(&l.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxEDINameLength).
				Error("Name cannot be longer than 200 characters"),
		),
		validation.Field(&l.Language,
			validation.Required.Error("Language is required"),
			domainvalidation.ValidEnum[ScriptLanguage]("Language is invalid"),
		),
		validation.Field(&l.Script, validation.Required.Error("Script is required")),
		validation.Field(&l.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TemplateStatus]("Status is invalid"),
		),
	))
}

const (
	maxEDICodeLength        = 20
	maxEDINameLength        = 200
	functionalGroupIDLength = 2
	maxSegmentIDLength      = 10
	maxLoopIDLength         = 50
)
