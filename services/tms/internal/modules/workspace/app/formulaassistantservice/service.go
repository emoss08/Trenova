package formulaassistantservice

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/modules/rating/app/formulalibraryservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/formulatemplateservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

const MaxExplanationRunes = 4000

type Kind string

const (
	KindTemplate = Kind("template")
	KindLibrary  = Kind("library")
)

func (k Kind) IsValid() bool {
	return k == KindTemplate || k == KindLibrary
}

func AllKinds() []Kind {
	return []Kind{KindTemplate, KindLibrary}
}

type templateEngine interface {
	StudioContext(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*formulatemplateservice.StudioContext, error)
	TestSource(
		ctx context.Context,
		req *formulatemplateservice.TestSourceRequest,
	) (*formulatemplateservice.TestSourceResult, error)
	Preview(
		ctx context.Context,
		req *formulatemplateservice.PreviewRequest,
	) (*formulatemplateports.PreviewResult, error)
}

type libraryEngine interface {
	TestSource(
		ctx context.Context,
		req *formulalibraryservice.TestSourceRequest,
	) (*formulalibraryservice.TestSourceResult, error)
}

type catalogReader interface {
	Catalog(ctx context.Context) (*tariff.CatalogEntry, error)
}

type Params struct {
	fx.In

	Templates *formulatemplateservice.Service
	Libraries *formulalibraryservice.Service
	Checker   *tariff.Checker
}

type Service struct {
	templates templateEngine
	libraries libraryEngine
	catalog   catalogReader
}

func New(p Params) *Service {
	return &Service{templates: p.Templates, libraries: p.Libraries, catalog: p.Checker}
}

type Language struct {
	ModuleVersion string
	Catalog       rules.RawJSON
	Context       *rules.TariffContext
	Libraries     []formulatemplateservice.StudioLibrary
}

func (s *Service) Language(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*Language, error) {
	catalog, err := s.catalog.Catalog(ctx)
	if err != nil {
		return nil, err
	}
	studio, err := s.templates.StudioContext(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	return &Language{
		ModuleVersion: catalog.ModuleVersion,
		Catalog:       catalog.Catalog,
		Context:       studio.Context,
		Libraries:     studio.Libraries,
	}, nil
}

type Input struct {
	Name       string                  `json:"name"`
	Type       string                  `json:"type"`
	Required   bool                    `json:"required"`
	Default    rules.RawJSON           `json:"default,omitempty"`
	Constraint *rules.TariffConstraint `json:"constraint,omitempty"`
	Doc        string                  `json:"doc,omitempty"`
}

type Checked struct {
	OK       bool               `json:"ok"`
	Kind     string             `json:"kind,omitempty"`
	Name     string             `json:"name,omitempty"`
	Findings []tariff.Finding   `json:"findings"`
	Inputs   []Input            `json:"inputs"`
	Imports  []string           `json:"imports"`
	Charges  []string           `json:"charges"`
	Tests    tariff.TestSummary `json:"tests"`
	source   string
	check    *rules.TariffCheck
	locale   i18n.Locale
	warnings []string
}

func (c *Checked) Passing() bool {
	return c.OK && c.Tests.Error == "" && c.Tests.Passed == c.Tests.Total
}

type CheckRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       Kind
	Source     string
}

func (s *Service) Check(ctx context.Context, req *CheckRequest) (*Checked, error) {
	if !req.Kind.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Kind must be template or library",
		)
	}
	if strings.TrimSpace(req.Source) == "" {
		return nil, errortypes.NewValidationError(
			"source", errortypes.ErrRequired, "Source is required",
		)
	}

	if req.Kind == KindLibrary {
		result, err := s.libraries.TestSource(ctx, &formulalibraryservice.TestSourceRequest{
			TenantInfo: req.TenantInfo,
			Source:     req.Source,
		})
		if err != nil {
			return nil, err
		}
		return checkedOf(ctx, req.Source, result.Check, result.Imports, result.Tests), nil
	}

	result, err := s.templates.TestSource(ctx, &formulatemplateservice.TestSourceRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Source,
	})
	if err != nil {
		return nil, err
	}
	return checkedOf(ctx, req.Source, result.Check, result.Imports, result.Tests), nil
}

func checkedOf(
	ctx context.Context,
	source string,
	check *rules.TariffCheck,
	imports []tarifftypes.Import,
	tests *rules.TariffTestRun,
) *Checked {
	locale := i18n.FromContext(ctx)
	out := &Checked{
		OK:       check != nil && check.OK && check.FirstError() == nil,
		Findings: []tariff.Finding{},
		Inputs:   []Input{},
		Imports:  []string{},
		Charges:  []string{},
		Tests:    tariff.SummarizeTests(locale, source, tests),
		source:   source,
		check:    check,
		locale:   locale,
	}
	if check == nil {
		return out
	}
	if check.Header != nil {
		out.Kind = string(check.Header.Kind)
		out.Name = check.Header.Name
	}
	out.Findings = tariff.Findings(locale, source, check.Diagnostics)
	for i := range out.Findings {
		if out.Findings[i].Severity == rules.SeverityWarning {
			out.warnings = append(out.warnings, out.Findings[i].Message)
		}
	}
	for i := range check.Inputs {
		input := &check.Inputs[i]
		out.Inputs = append(out.Inputs, Input{
			Name:       input.Name,
			Type:       input.Type,
			Required:   input.Required,
			Default:    input.Default,
			Constraint: input.Constraint,
			Doc:        input.Doc,
		})
	}
	for _, imp := range imports {
		out.Imports = append(out.Imports, imp.Path)
	}
	for i := range check.Charges {
		out.Charges = append(out.Charges, check.Charges[i].Label)
	}
	return out
}

type PreviewRequest struct {
	TenantInfo pagination.TenantInfo
	Source     string
	Inputs     tarifftypes.Inputs
	ShipmentID pulid.ID
}

type Preview struct {
	*Checked
	Bill      *tarifftypes.Bill      `json:"bill,omitempty"`
	Choices   []tarifftypes.Choice   `json:"choices,omitempty"`
	Refusal   string                 `json:"refusal,omitempty"`
	RefusalAt *tariff.SourcePosition `json:"refusalAt,omitempty"`
	Error     string                 `json:"error,omitempty"`
	ErrorAt   *tariff.SourcePosition `json:"errorAt,omitempty"`
}

func (s *Service) Preview(ctx context.Context, req *PreviewRequest) (*Preview, error) {
	if strings.TrimSpace(req.Source) == "" {
		return nil, errortypes.NewValidationError(
			"source", errortypes.ErrRequired, "Source is required",
		)
	}

	previewReq := &formulatemplateservice.PreviewRequest{
		TenantInfo: req.TenantInfo,
		Source:     req.Source,
		Inputs:     req.Inputs,
	}
	if req.ShipmentID.IsNotNil() {
		shipmentID := req.ShipmentID
		previewReq.ShipmentID = &shipmentID
	}
	result, err := s.templates.Preview(ctx, previewReq)
	if err != nil {
		return nil, err
	}

	check := &rules.TariffCheck{
		OK:          result.Valid,
		Diagnostics: result.Diagnostics,
		Charges:     result.Charges,
	}
	for i := range result.Inputs {
		input := &result.Inputs[i]
		check.Inputs = append(check.Inputs, rules.TariffInput{
			Name:       input.Name,
			Doc:        input.Doc,
			Type:       input.Type,
			Required:   input.Required,
			Default:    input.Default,
			Constraint: input.Constraint,
		})
	}
	checked := checkedOf(ctx, req.Source, check, result.Imports,
		&rules.TariffTestRun{Tests: result.Tests})
	checked.OK = result.Valid

	out := &Preview{Checked: checked, Bill: result.Bill}
	currency := ""
	if result.Bill != nil {
		currency = result.Bill.Currency
		if result.Bill.Exchange != nil {
			currency = result.Bill.Exchange.Currency
		}
	}
	out.Choices, err = tarifftypes.ChoicesFrom(&tarifftypes.ChoicesRequest{
		Choices:  result.Choices,
		Currency: currency,
		Locale:   checked.locale,
	})
	if err != nil {
		return nil, err
	}
	if result.Refusal != nil {
		out.Refusal = result.Refusal.Localize(checked.locale)
		out.RefusalAt = positionOf(req.Source, result.Refusal.Range)
	}
	if result.Error != nil {
		out.Error = result.Error.Message
		out.ErrorAt = positionOf(req.Source, result.Error.Range)
	}
	return out, nil
}

func positionOf(source string, at *rules.TariffRange) *tariff.SourcePosition {
	if at == nil {
		return nil
	}
	position := tariff.PositionOf(source, at.From)
	return &position
}

type ProposeRequest struct {
	TenantInfo  pagination.TenantInfo
	Kind        Kind
	Name        string
	Description string
	Source      string
	Explanation string
}

type Proposed struct {
	Proposal *pagedraft.SourceProposal
	Checked  *Checked
}

func (r *ProposeRequest) validate() error {
	multiErr := errortypes.NewMultiError()
	nameLimit := pagedraft.MaxTemplateNameLength
	if r.Kind == KindLibrary {
		nameLimit = pagedraft.MaxLibraryNameLength
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.Name)) > nameLimit {
		multiErr.Add("name", errortypes.ErrInvalid, "Name is too long")
	}
	if utf8.RuneCountInString(strings.TrimSpace(r.Description)) >
		pagedraft.MaxDescriptionLength {
		multiErr.Add("description", errortypes.ErrInvalid, "Description is too long")
	}
	switch explanation := strings.TrimSpace(r.Explanation); {
	case explanation == "":
		multiErr.Add("explanation", errortypes.ErrRequired, "Explain the change")
	case utf8.RuneCountInString(explanation) > MaxExplanationRunes:
		multiErr.Add("explanation", errortypes.ErrInvalid, "The explanation is too long")
	}
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func (s *Service) Propose(ctx context.Context, req *ProposeRequest) (*Proposed, error) {
	if !req.Kind.IsValid() {
		return nil, errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Kind must be template or library",
		)
	}
	if err := req.validate(); err != nil {
		return nil, err
	}

	checked, err := s.Check(ctx, &CheckRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		Source:     req.Source,
	})
	if err != nil {
		return nil, err
	}
	if !checked.Passing() || checked.Tests.Total == 0 {
		return &Proposed{Checked: checked}, nil
	}

	return &Proposed{
		Checked: checked,
		Proposal: &pagedraft.SourceProposal{
			Name:        strings.TrimSpace(req.Name),
			Description: strings.TrimSpace(req.Description),
			Source:      req.Source,
			Explanation: strings.TrimSpace(req.Explanation),
			Inputs:      inputNames(checked.Inputs),
			Imports:     checked.Imports,
			Charges:     checked.Charges,
			Warnings:    orEmpty(checked.warnings),
			Tests: pagedraft.ProposalTests{
				Total:  checked.Tests.Total,
				Passed: checked.Tests.Passed,
			},
		},
	}, nil
}

func inputNames(inputs []Input) []string {
	names := make([]string, 0, len(inputs))
	for i := range inputs {
		names = append(names, inputs[i].Name)
	}
	return names
}

func orEmpty(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}
