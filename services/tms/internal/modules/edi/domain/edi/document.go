package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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
	keel.Table `keel:"edi_document_types"`

	ID               pulid.ID          `json:"id" keel:"id,pk,autoid:edidt_"`
	Code             string            `json:"code" keel:"code"`
	Name             string            `json:"name" keel:"name"`
	Standard         EDIStandard       `json:"standard" keel:"standard"`
	TransactionSet   TransactionSet    `json:"transactionSet" keel:"transaction_set"`
	TransactionSetID pulid.ID          `json:"transactionSetId" keel:"transaction_set_id"`
	Direction        DocumentDirection `json:"direction" keel:"direction"`
	DefaultVersion   string            `json:"defaultVersion" keel:"default_version"`
	Status           DocumentStatus    `json:"status" keel:"status"`
	CreatedAt        int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt        int64             `json:"updatedAt" keel:"updated_at,autoupdate"`

	TransactionSetRef *EDITransactionSet `json:"transactionSetRef,omitempty" keel:"belongs:transaction_set_id"`
}

type EDITemplateVersion struct {
	keel.Table `keel:"edi_template_versions"`

	ID                 pulid.ID       `json:"id" keel:"id,pk,autoid:editv_"`
	BusinessUnitID     pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	TemplateID         pulid.ID       `json:"templateId" keel:"template_id"`
	SourceVersionID    pulid.ID       `json:"sourceVersionId" keel:"source_version_id,nullzero"`
	VersionNumber      int64          `json:"versionNumber" keel:"version_number"`
	X12Version         string         `json:"x12Version" keel:"x12_version"`
	FunctionalGroupID  string         `json:"functionalGroupId" keel:"functional_group_id"`
	Status             TemplateStatus `json:"status" keel:"status"`
	IsActive           bool           `json:"isActive" keel:"is_active"`
	Notes              string         `json:"notes" keel:"notes,nullzero"`
	CertificationNotes string         `json:"certificationNotes" keel:"certification_notes,nullzero"`
	ActivationNotes    string         `json:"activationNotes" keel:"activation_notes,nullzero"`
	ArchiveNotes       string         `json:"archiveNotes" keel:"archive_notes,nullzero"`
	DeprecatedNotes    string         `json:"deprecatedNotes" keel:"deprecated_notes,nullzero"`
	SupersededNotes    string         `json:"supersededNotes" keel:"superseded_notes,nullzero"`
	CertifiedByID      pulid.ID       `json:"certifiedById" keel:"certified_by_id,nullzero"`
	ActivatedByID      pulid.ID       `json:"activatedById" keel:"activated_by_id,nullzero"`
	ArchivedByID       pulid.ID       `json:"archivedById" keel:"archived_by_id,nullzero"`
	DeprecatedByID     pulid.ID       `json:"deprecatedById" keel:"deprecated_by_id,nullzero"`
	SupersededByID     pulid.ID       `json:"supersededById" keel:"superseded_by_id,nullzero"`
	CertifiedAt        *int64         `json:"certifiedAt" keel:"certified_at"`
	ActivatedAt        *int64         `json:"activatedAt" keel:"activated_at"`
	ArchivedAt         *int64         `json:"archivedAt" keel:"archived_at"`
	DeprecatedAt       *int64         `json:"deprecatedAt" keel:"deprecated_at"`
	SupersededAt       *int64         `json:"supersededAt" keel:"superseded_at"`
	Version            int64          `json:"version" keel:"version,version"`
	CreatedAt          int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64          `json:"updatedAt" keel:"updated_at,autoupdate"`

	Template        *EDITemplate                `json:"template,omitempty" keel:"belongs:template_id"`
	SourceVersion   *EDITemplateVersion         `json:"sourceVersion,omitempty" keel:"belongs:source_version_id"`
	Segments        []*EDITemplateSegment       `json:"segments,omitempty" keel:"many:template_version_id"`
	ScriptLibraries []*EDITemplateScriptLibrary `json:"scriptLibraries,omitempty" keel:"many:template_version_id"`
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
	keel.Table `keel:"edi_template_segments"`

	ID                pulid.ID          `json:"id" keel:"id,pk,autoid:edisg_"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID          `json:"organizationId" keel:"organization_id,pk,tenant"`
	TemplateVersionID pulid.ID          `json:"templateVersionId" keel:"template_version_id"`
	SegmentID         string            `json:"segmentId" keel:"segment_id"`
	Name              string            `json:"name" keel:"name"`
	Sequence          int64             `json:"sequence" keel:"sequence"`
	LoopID            string            `json:"loopId" keel:"loop_id,nullzero"`
	RepeatPath        string            `json:"repeatPath" keel:"repeat_path,nullzero"`
	Condition         string            `json:"condition" keel:"condition,nullzero"`
	Required          bool              `json:"required" keel:"required"`
	MaxUse            int64             `json:"maxUse" keel:"max_use"`
	Elements          []TemplateElement `json:"elements" keel:"elements,jsonb"`
	UsageNotes        string            `json:"usageNotes" keel:"usage_notes,nullzero"`
	CreatedAt         int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt         int64             `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (s *EDITemplateSegment) BeforeUpdate(ctx context.Context) error {
	if err := s.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (s *EDITemplateSegment) BeforeCreate(ctx context.Context) error {
	if err := s.BeforeSave(ctx); err != nil {
		return err
	}
	if s.MaxUse == 0 {
		s.MaxUse = 1
	}

	return nil
}

func (s *EDITemplateSegment) BeforeSave(_ context.Context) error {
	if s.Elements == nil {
		s.Elements = []TemplateElement{}
	}
	return nil
}

type EDITemplateScriptLibrary struct {
	keel.Table `keel:"edi_template_script_libraries"`

	ID                pulid.ID       `json:"id" keel:"id,pk,autoid:edisl_"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	TemplateVersionID pulid.ID       `json:"templateVersionId" keel:"template_version_id"`
	Name              string         `json:"name" keel:"name"`
	Description       string         `json:"description" keel:"description,nullzero"`
	Language          ScriptLanguage `json:"language" keel:"language"`
	Script            string         `json:"script" keel:"script"`
	Status            TemplateStatus `json:"status" keel:"status"`
	Version           int64          `json:"version" keel:"version,version"`
	CreatedAt         int64          `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt         int64          `json:"updatedAt" keel:"updated_at,autoupdate"`

	FunctionNames   []string            `json:"functionNames"`
	TemplateVersion *EDITemplateVersion `json:"templateVersion,omitempty" keel:"belongs:template_version_id"`
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
	// Tenancy and the owning version are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(s,
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
	// Tenancy and the owning version are stamped when the parent is written, so
	// they are not the caller's to supply.
	multiErr.AddOzzoError(validation.ValidateStruct(l,
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
