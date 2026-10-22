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

const fieldSource = "source"

type contentCheck struct {
	checked *tariff.SourceCheck
	tests   *rules.TariffTestRun
}

func (c *contentCheck) checks() bool {
	return c.checked != nil && c.checked.OK()
}

func (c *contentCheck) testsPass() bool {
	return c.tests != nil && c.tests.Passed()
}

func (c *contentCheck) passedTests() int {
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

func (c *contentCheck) failingTests() []string {
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

func (c *contentCheck) testCount() int32 {
	if c.tests == nil {
		return 0
	}
	return intutils.SafeToInt32(len(c.tests.Tests))
}

func (c *contentCheck) refusal() error {
	if c.checked == nil {
		return nil
	}
	return tarifferrors.CheckRefusal(c.checked.Check, tarifferrors.CheckFields{Source: fieldSource})
}

type contentRequest struct {
	tenant pagination.TenantInfo
	source string
	mode   tariff.LibraryMode
	pins   []tarifftypes.Import
}

func (s *Service) checkContent(ctx context.Context, req *contentRequest) (*contentCheck, error) {
	checked, err := s.checker.CheckSource(ctx, &tariff.SourceCheckRequest{
		TenantInfo: req.tenant,
		Source:     req.source,
		Kind:       rules.TariffKindTariff,
		Mode:       req.mode,
		Pins:       req.pins,
	})
	if err != nil {
		return nil, err
	}

	tests, err := s.checker.TestChecked(ctx, checked, req.source)
	if err != nil {
		return nil, err
	}

	return &contentCheck{checked: checked, tests: tests}, nil
}

func templateTenant(template *formulatemplate.FormulaTemplate) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: template.OrganizationID, BuID: template.BusinessUnitID}
}

func (s *Service) prepareContent(
	ctx context.Context,
	entity *formulatemplate.FormulaTemplate,
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	content, err := s.checkContent(ctx, &contentRequest{
		tenant: templateTenant(entity),
		source: entity.Source,
		mode:   tariff.LibrariesLatest,
	})
	if err != nil {
		return err
	}
	if refusal := content.refusal(); refusal != nil {
		return refusal
	}

	applyContent(entity, content)
	return nil
}

func applyContent(entity *formulatemplate.FormulaTemplate, content *contentCheck) {
	entity.Inputs = tarifftypes.InputDeclsFrom(content.checked.Check.Inputs)
	entity.Imports = content.checked.Imports
	entity.TestCount = content.testCount()
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
