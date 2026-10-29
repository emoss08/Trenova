package formulatemplateservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type ContentCheck struct {
	Checked *tariff.SourceCheck
	tests   *rules.TariffTestRun
}

func (c *ContentCheck) Checks() bool {
	return c.Checked != nil && c.Checked.OK()
}

func (c *ContentCheck) TestsPass() bool {
	return c.tests != nil && c.tests.Passed()
}

func (c *ContentCheck) PassedTests() int {
	if c.tests == nil {
		return 0
	}
	passed := 0
	for i := range c.tests.Tests {
		if c.tests.Tests[i].Status == rules.TariffTestPassed {
			passed++
		}
	}
	return passed
}

func (c *ContentCheck) FailingTests() []string {
	if c.tests == nil {
		return nil
	}
	names := make([]string, 0, len(c.tests.Tests))
	for i := range c.tests.Tests {
		if c.tests.Tests[i].Status != rules.TariffTestPassed {
			names = append(names, c.tests.Tests[i].Name)
		}
	}
	return names
}

func (c *ContentCheck) TestCount() int32 {
	if c.tests == nil {
		return 0
	}
	return intutils.SafeToInt32(len(c.tests.Tests))
}

func (c *ContentCheck) HeaderName() string {
	if c.Checked == nil || c.Checked.Check == nil || c.Checked.Check.Header == nil {
		return ""
	}
	return c.Checked.Check.Header.Name
}

func (c *ContentCheck) Refusal() error {
	if c.Checked == nil {
		return nil
	}
	return tarifferrors.CheckRefusal(c.Checked.Check, tarifferrors.CheckFields{Source: FieldSource})
}

type ContentRequest struct {
	Tenant pagination.TenantInfo
	Source string
	Mode   tariff.LibraryMode
	Pins   []tarifftypes.Import
	Tables []tarifftypes.TablePin
}

func (s *Service) CheckContent(ctx context.Context, req *ContentRequest) (*ContentCheck, error) {
	checked, err := s.checker.CheckSource(ctx, &tariff.SourceCheckRequest{
		TenantInfo: req.Tenant,
		Source:     req.Source,
		Kind:       rules.TariffKindTariff,
		Mode:       req.Mode,
		Pins:       req.Pins,
		TablePins:  req.Tables,
	})
	if err != nil {
		return nil, err
	}

	tests, err := s.checker.TestChecked(ctx, checked, req.Source)
	if err != nil {
		return nil, err
	}

	return &ContentCheck{Checked: checked, tests: tests}, nil
}

func TemplateTenant(template *formulatemplate.FormulaTemplate) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: template.OrganizationID, BuID: template.BusinessUnitID}
}

func (s *Service) PrepareContent(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	multiErr := errortypes.NewMultiError()
	formulatemplate.ValidateContentSizes(
		multiErr,
		entity.Source,
		entity.Description,
		entity.Metadata,
	)
	if multiErr.HasErrors() {
		return multiErr
	}

	content, err := s.CheckContent(ctx, &ContentRequest{
		Tenant: TemplateTenant(entity),
		Source: entity.Source,
		Mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return err
	}
	if refusal := content.Refusal(); refusal != nil {
		return refusal
	}

	entity.Name = content.HeaderName()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	ApplyContent(entity, content)
	return nil
}

func (s *Service) Renamed(ctx context.Context, source, name string) (string, error) {
	return s.checker.SetName(ctx, source, name)
}

func ApplyContent(entity *formulatemplate.FormulaTemplate, content *ContentCheck) {
	entity.Inputs = tarifftypes.InputDeclsFrom(content.Checked.Check.Inputs)
	entity.Imports = content.Checked.Imports
	entity.Tables = content.Checked.Tables
	entity.TestCount = content.TestCount()
}

func validateBulkTemplateIDs(ids []pulid.ID) error {
	switch {
	case len(ids) == 0:
		return errortypes.NewValidationError(
			"templateIds",
			errortypes.ErrRequired,
			"At least one template is required",
		)
	case len(ids) > maxBulkTemplateIDs:
		return errortypes.NewValidationError(
			"templateIds",
			errortypes.ErrInvalid,
			"Cannot act on more than {0} templates at once", maxBulkTemplateIDs,
		)
	default:
		return nil
	}
}
