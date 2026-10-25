//nolint:gocritic // value-shaped domain/codegen APIs are intentionally stable
package documentparsingrule

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*RuleSet)(nil)
	_ validationframework.TenantedEntity = (*RuleVersion)(nil)
	_ validationframework.TenantedEntity = (*Fixture)(nil)
)

type RuleSet struct {
	keel.Table `keel:"document_parsing_rule_sets"`

	ID                 pulid.ID     `json:"id"                 keel:"id,pk,autoid:dprs_"`
	OrganizationID     pulid.ID     `json:"organizationId"     keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID     `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	Name               string       `json:"name"               keel:"name"`
	Description        string       `json:"description"        keel:"description,nullzero"`
	DocumentKind       DocumentKind `json:"documentKind"       keel:"document_kind"`
	Priority           int          `json:"priority"           keel:"priority"`
	PublishedVersionID *pulid.ID    `json:"publishedVersionId" keel:"published_version_id"`
	Version            int64        `json:"version"            keel:"version,version"`
	CreatedAt          int64        `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64        `json:"updatedAt"          keel:"updated_at,autoupdate"`
}

func (r *RuleSet) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		r,
		validation.Field(
			&r.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255),
		),
		validation.Field(
			&r.DocumentKind,
			validation.Required.Error("Document kind is required"),
			validation.In(DocumentKindRateConfirmation).Error("Document kind must be valid"),
		),
		validation.Field(&r.Priority, validation.Min(0).Error("Priority must be zero or greater")),
	))
}

func (r *RuleSet) GetID() pulid.ID { return r.ID }

func (r *RuleSet) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RuleSet) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RuleSet) GetTableName() string { return "document_parsing_rule_sets" }

func (r *RuleSet) BeforeCreate(ctx context.Context) error {
	if r.Priority == 0 {
		r.Priority = 100
	}

	return nil
}

func (RuleSet) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_document_parsing_rule_sets_name_kind_scope",
			Field:   "name",
			Message: "A parsing rule set with this name already exists for this document kind",
		},
		{
			Name:    "fk_document_parsing_rule_sets_published_version",
			Message: "Published version does not exist",
		},
	}
}

type RuleVersion struct {
	keel.Table `keel:"document_parsing_rule_versions"`

	ID                pulid.ID       `json:"id"                keel:"id,pk,autoid:dprv_"`
	RuleSetID         pulid.ID       `json:"ruleSetId"         keel:"rule_set_id"`
	OrganizationID    pulid.ID       `json:"organizationId"    keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	VersionNumber     int            `json:"versionNumber"     keel:"version_number"`
	Status            VersionStatus  `json:"status"            keel:"status"`
	Label             string         `json:"label"             keel:"label,nullzero"`
	ParserMode        ParserMode     `json:"parserMode"        keel:"parser_mode"`
	MatchConfig       MatchConfig    `json:"matchConfig"       keel:"match_config,jsonb"`
	RuleDocument      RuleDocument   `json:"ruleDocument"      keel:"rule_document,jsonb"`
	ValidationSummary map[string]any `json:"validationSummary" keel:"validation_summary,jsonb"`
	PublishedAt       *int64         `json:"publishedAt"       keel:"published_at"`
	PublishedByID     *pulid.ID      `json:"publishedById"     keel:"published_by_id"`
	Version           int64          `json:"version"           keel:"version,version"`
	CreatedAt         int64          `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64          `json:"updatedAt"         keel:"updated_at,autoupdate"`
	RuleSet           *RuleSet       `json:"ruleSet,omitempty" keel:"belongs:rule_set_id"`
}

func (r *RuleVersion) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		r,
		validation.Field(&r.RuleSetID, validation.Required.Error("Rule set is required")),
		validation.Field(
			&r.VersionNumber,
			validation.Min(1).Error("Version number must be greater than zero"),
		),
		validation.Field(
			&r.Status,
			validation.Required.Error("Status is required"),
			validation.In(VersionStatusDraft, VersionStatusPublished, VersionStatusArchived).
				Error("Status must be valid"),
		),
		validation.Field(
			&r.ParserMode,
			validation.Required.Error("Parser mode is required"),
			validation.In(ParserModeMergeWithBase, ParserModeOverrideBase).
				Error("Parser mode must be valid"),
		),
	)
	if err != nil {
		// A non-ozzo error carries no field, so it is reported whole rather
		// than dropped; either way the document checks below are skipped
		// because they would be judging a shape that is already wrong.
		if multiErr.AddOzzoError(err) {
			return
		}
		multiErr.Add("", errortypes.ErrInvalid, err.Error())
	}
	if err = r.MatchConfig.Validate(); err != nil {
		multiErr.Add("matchConfig", errortypes.ErrInvalid, err.Error())
	}
	if err = r.RuleDocument.Validate(); err != nil {
		multiErr.Add("ruleDocument", errortypes.ErrInvalid, err.Error())
	}
}

func (r *RuleVersion) GetID() pulid.ID { return r.ID }

func (r *RuleVersion) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *RuleVersion) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *RuleVersion) GetTableName() string { return "document_parsing_rule_versions" }

func (r *RuleVersion) BeforeCreate(ctx context.Context) error {
	if r.Status == "" {
		r.Status = VersionStatusDraft
	}
	if r.ParserMode == "" {
		r.ParserMode = ParserModeMergeWithBase
	}
	if r.ValidationSummary == nil {
		r.ValidationSummary = map[string]any{}
	}

	return nil
}

func (RuleVersion) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_document_parsing_rule_versions_rule_set",
			Message: "Parsing rule set does not exist",
		},
		{
			Name:    "uq_document_parsing_rule_versions_rule_set_version",
			Message: "A parsing rule version was created concurrently. Retry the request.",
		},
	}
}

type Fixture struct {
	keel.Table `keel:"document_parsing_rule_fixtures"`

	ID                  pulid.ID          `json:"id"                  keel:"id,pk,autoid:dprf_"`
	RuleSetID           pulid.ID          `json:"ruleSetId"           keel:"rule_set_id"`
	OrganizationID      pulid.ID          `json:"organizationId"      keel:"organization_id,pk,tenant"`
	BusinessUnitID      pulid.ID          `json:"businessUnitId"      keel:"business_unit_id,pk,tenant"`
	Name                string            `json:"name"                keel:"name"`
	Description         string            `json:"description"         keel:"description,nullzero"`
	FileName            string            `json:"fileName"            keel:"file_name,nullzero"`
	ProviderFingerprint string            `json:"providerFingerprint" keel:"provider_fingerprint,nullzero"`
	TextSnapshot        string            `json:"textSnapshot"        keel:"text_snapshot"`
	PageSnapshots       []PageSnapshot    `json:"pageSnapshots"       keel:"page_snapshots,jsonb"`
	Assertions          FixtureAssertions `json:"assertions"          keel:"assertions,jsonb"`
	Version             int64             `json:"version"             keel:"version,version"`
	CreatedAt           int64             `json:"createdAt"           keel:"created_at,autocreate"`
	UpdatedAt           int64             `json:"updatedAt"           keel:"updated_at,autoupdate"`
}

func (f *Fixture) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		f,
		validation.Field(&f.RuleSetID, validation.Required.Error("Rule set is required")),
		validation.Field(
			&f.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255),
		),
		validation.Field(&f.TextSnapshot, validation.Required.Error("Text snapshot is required")),
	)
	if err != nil {
		// A non-ozzo error carries no field, so it is reported whole rather
		// than dropped; either way the document checks below are skipped
		// because they would be judging a shape that is already wrong.
		if multiErr.AddOzzoError(err) {
			return
		}
		multiErr.Add("", errortypes.ErrInvalid, err.Error())
	}
	if err = f.Assertions.Validate(); err != nil {
		multiErr.Add("assertions", errortypes.ErrInvalid, err.Error())
	}
}

func (f *Fixture) GetID() pulid.ID { return f.ID }

func (f *Fixture) GetOrganizationID() pulid.ID { return f.OrganizationID }

func (f *Fixture) GetBusinessUnitID() pulid.ID { return f.BusinessUnitID }

func (f *Fixture) GetTableName() string { return "document_parsing_rule_fixtures" }

func (f *Fixture) BeforeCreate(ctx context.Context) error {
	if f.PageSnapshots == nil {
		f.PageSnapshots = []PageSnapshot{}
	}

	return nil
}

func (Fixture) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_document_parsing_rule_fixtures_rule_set",
			Message: "Parsing rule set does not exist",
		},
	}
}

type MatchConfig struct {
	ProviderFingerprints []string `json:"providerFingerprints,omitempty"`
	FileNameContains     []string `json:"fileNameContains,omitempty"`
	RequiresAll          []string `json:"requiresAll,omitempty"`
	RequiresAny          []string `json:"requiresAny,omitempty"`
	SectionAnchors       []string `json:"sectionAnchors,omitempty"`
}

func (m MatchConfig) Validate() error {
	return nil
}

type RuleDocument struct {
	Sections []SectionRule `json:"sections,omitempty"`
	Fields   []FieldRule   `json:"fields,omitempty"`
	Stops    []StopRule    `json:"stops,omitempty"`
}

func (d RuleDocument) Validate() error {
	if len(d.Fields) == 0 && len(d.Stops) == 0 {
		return errors.New("rule document must define at least one field or stop rule")
	}

	fieldKeys := make(map[string]struct{}, len(d.Fields))
	for _, field := range d.Fields {
		if err := field.Validate(); err != nil {
			return err
		}
		if _, ok := fieldKeys[field.Key]; ok {
			return fmt.Errorf("duplicate field rule key %q", field.Key)
		}
		fieldKeys[field.Key] = struct{}{}
	}

	sectionNames := make(map[string]struct{}, len(d.Sections))
	for _, section := range d.Sections {
		if err := section.Validate(); err != nil {
			return err
		}
		sectionNames[strings.ToLower(strings.TrimSpace(section.Name))] = struct{}{}
	}

	for _, stop := range d.Stops {
		if err := stop.Validate(sectionNames); err != nil {
			return err
		}
	}

	return nil
}

type SectionRule struct {
	Name             string   `json:"name"`
	StartAnchors     []string `json:"startAnchors"`
	EndAnchors       []string `json:"endAnchors,omitempty"`
	CaptureBlankLine bool     `json:"captureBlankLine,omitempty"`
	AllowMultiple    bool     `json:"allowMultiple,omitempty"`
}

func (s SectionRule) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return errors.New("section name is required")
	}
	if len(s.StartAnchors) == 0 {
		return fmt.Errorf("section %q must define at least one start anchor", s.Name)
	}
	return nil
}

type FieldRule struct {
	Key          string   `json:"key"`
	Label        string   `json:"label"`
	SectionNames []string `json:"sectionNames,omitempty"`
	Aliases      []string `json:"aliases,omitempty"`
	Patterns     []string `json:"patterns,omitempty"`
	Normalizer   string   `json:"normalizer,omitempty"`
	Required     bool     `json:"required,omitempty"`
	Confidence   float64  `json:"confidence,omitempty"`
}

func (f FieldRule) Validate() error {
	if strings.TrimSpace(f.Key) == "" {
		return errors.New("field key is required")
	}
	if strings.TrimSpace(f.Label) == "" {
		return fmt.Errorf("field %q label is required", f.Key)
	}
	if len(f.Aliases) == 0 && len(f.Patterns) == 0 {
		return fmt.Errorf("field %q must define aliases or patterns", f.Key)
	}
	for _, pattern := range f.Patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("field %q has invalid pattern: %w", f.Key, err)
		}
	}
	return nil
}

type StopRule struct {
	Role                string          `json:"role"`
	Required            bool            `json:"required,omitempty"`
	SectionNames        []string        `json:"sectionNames,omitempty"`
	StartAnchors        []string        `json:"startAnchors,omitempty"`
	EndAnchors          []string        `json:"endAnchors,omitempty"`
	AllowMultiple       bool            `json:"allowMultiple,omitempty"`
	SequenceStart       int             `json:"sequenceStart,omitempty"`
	Extractors          []StopFieldRule `json:"extractors"`
	AppointmentPatterns []string        `json:"appointmentPatterns,omitempty"`
}

func (s StopRule) Validate(sectionNames map[string]struct{}) error {
	role := strings.ToLower(strings.TrimSpace(s.Role))
	if role != "pickup" && role != "delivery" && role != "stop" {
		return fmt.Errorf("stop rule role %q must be pickup, delivery, or stop", s.Role)
	}
	if len(s.SectionNames) == 0 && len(s.StartAnchors) == 0 {
		return fmt.Errorf("stop rule %q must define section names or start anchors", s.Role)
	}
	for _, name := range s.SectionNames {
		if len(sectionNames) == 0 {
			break
		}
		if _, ok := sectionNames[strings.ToLower(strings.TrimSpace(name))]; !ok {
			return fmt.Errorf("stop rule %q references unknown section %q", s.Role, name)
		}
	}
	if len(s.Extractors) == 0 {
		return fmt.Errorf("stop rule %q must define extractors", s.Role)
	}
	for _, extractor := range s.Extractors {
		if err := extractor.Validate(); err != nil {
			return fmt.Errorf("stop rule %q: %w", s.Role, err)
		}
	}
	for _, pattern := range s.AppointmentPatterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("stop rule %q has invalid appointment pattern: %w", s.Role, err)
		}
	}
	return nil
}

type StopFieldRule struct {
	FieldKey   string   `json:"fieldKey"`
	Aliases    []string `json:"aliases,omitempty"`
	Patterns   []string `json:"patterns,omitempty"`
	Normalizer string   `json:"normalizer,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	Required   bool     `json:"required,omitempty"`
}

func (s StopFieldRule) Validate() error {
	switch strings.TrimSpace(s.FieldKey) {
	case "name",
		"addressLine1",
		"addressLine2",
		"city",
		"state",
		"postalCode",
		"date",
		"timeWindow":
	default:
		return fmt.Errorf("unsupported stop field key %q", s.FieldKey)
	}
	if len(s.Aliases) == 0 && len(s.Patterns) == 0 {
		return fmt.Errorf("extractor %q must define aliases or patterns", s.FieldKey)
	}
	for _, pattern := range s.Patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("extractor %q has invalid pattern: %w", s.FieldKey, err)
		}
	}
	return nil
}

type PageSnapshot struct {
	PageNumber int    `json:"pageNumber"`
	Text       string `json:"text"`
}

type FixtureFieldAssertionOperator string

const (
	FixtureFieldAssertionOperatorExists       FixtureFieldAssertionOperator = "exists"
	FixtureFieldAssertionOperatorNotEmpty     FixtureFieldAssertionOperator = "not_empty"
	FixtureFieldAssertionOperatorEquals       FixtureFieldAssertionOperator = "equals"
	FixtureFieldAssertionOperatorMatchesRegex FixtureFieldAssertionOperator = "matches_regex"
	FixtureFieldAssertionOperatorOneOf        FixtureFieldAssertionOperator = "one_of"
)

type FixtureFieldAssertion struct {
	Operator FixtureFieldAssertionOperator `json:"operator"`
	Value    string                        `json:"value,omitempty"`
	Values   []string                      `json:"values,omitempty"`
	Pattern  string                        `json:"pattern,omitempty"`
}

func (f FixtureFieldAssertion) Validate(fieldKey string) error {
	switch f.Operator {
	case FixtureFieldAssertionOperatorExists, FixtureFieldAssertionOperatorNotEmpty:
		return nil
	case FixtureFieldAssertionOperatorEquals:
		if strings.TrimSpace(f.Value) == "" {
			return fmt.Errorf(
				"field assertion for %q with operator %q requires a value",
				fieldKey,
				f.Operator,
			)
		}
		return nil
	case FixtureFieldAssertionOperatorMatchesRegex:
		if strings.TrimSpace(f.Pattern) == "" {
			return fmt.Errorf(
				"field assertion for %q with operator %q requires a pattern",
				fieldKey,
				f.Operator,
			)
		}
		if _, err := regexp.Compile(f.Pattern); err != nil {
			return fmt.Errorf("field assertion for %q has invalid regex: %w", fieldKey, err)
		}
		return nil
	case FixtureFieldAssertionOperatorOneOf:
		if len(f.Values) == 0 {
			return fmt.Errorf(
				"field assertion for %q with operator %q requires values",
				fieldKey,
				f.Operator,
			)
		}
		return nil
	default:
		return fmt.Errorf("field assertion for %q has invalid operator %q", fieldKey, f.Operator)
	}
}

type FixtureAssertions struct {
	ExpectedFields    map[string]string                  `json:"expectedFields,omitempty"`
	FieldAssertions   map[string][]FixtureFieldAssertion `json:"fieldAssertions,omitempty"`
	RequiredStopRoles []string                           `json:"requiredStopRoles,omitempty"`
	MinimumStopCount  int                                `json:"minimumStopCount,omitempty"`
	ReviewStatus      string                             `json:"reviewStatus,omitempty"`
}

func (f FixtureAssertions) Validate() error {
	if len(f.ExpectedFields) == 0 &&
		len(f.FieldAssertions) == 0 &&
		len(f.RequiredStopRoles) == 0 &&
		f.MinimumStopCount == 0 &&
		strings.TrimSpace(f.ReviewStatus) == "" {
		return errors.New("fixture assertions must define at least one expectation")
	}
	if f.MinimumStopCount < 0 {
		return errors.New("minimum stop count must be zero or greater")
	}
	for fieldKey, assertions := range f.FieldAssertions {
		if strings.TrimSpace(fieldKey) == "" {
			return errors.New("field assertion key is required")
		}
		if len(assertions) == 0 {
			return fmt.Errorf("field assertion for %q must define at least one rule", fieldKey)
		}
		for _, assertion := range assertions {
			if err := assertion.Validate(fieldKey); err != nil {
				return err
			}
		}
	}
	for _, role := range f.RequiredStopRoles {
		normalized := strings.ToLower(strings.TrimSpace(role))
		if !slices.Contains([]string{"pickup", "delivery", "stop"}, normalized) {
			return fmt.Errorf("required stop role %q is invalid", role)
		}
	}
	switch status := strings.TrimSpace(f.ReviewStatus); status {
	case "", "Ready", "NeedsReview", "Unavailable":
	default:
		return fmt.Errorf("review status %q is invalid", status)
	}
	return nil
}
