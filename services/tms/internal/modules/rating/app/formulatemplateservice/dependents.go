package formulatemplateservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/app/tariff"
	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	DependentRule     = "rule"
	DependentShipment = "shipment"
	DependentFallback = "billingFallback"

	maxDependentSets      = 2_000
	maxReportedDependents = 20
	fieldInputs           = "inputs"
	usageBillingFallback  = "billing_control_fallback"
)

type DependentFailure struct {
	Kind        string   `json:"kind"`
	ExampleID   pulid.ID `json:"exampleId,omitempty"`
	ExampleName string   `json:"exampleName,omitempty"`
	Count       int      `json:"count"`
	Path        string   `json:"path,omitempty"`
	Message     string   `json:"message"`
}

type dependentsGate struct {
	Checked  bool               `json:"checked"`
	Skipped  string             `json:"skipped,omitempty"`
	Sets     int                `json:"sets"`
	Failures []DependentFailure `json:"failures"`
}

func (g *dependentsGate) blocking() bool {
	return len(g.Failures) > 0
}

func (s *Service) checkDependents(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	content *contentCheck,
) (*dependentsGate, error) {
	gate := &dependentsGate{Failures: []DependentFailure{}}
	tenant := templateTenant(template)

	base, err := s.versionRepo.GetLatestByStatus(
		ctx,
		&formulatemplateports.GetLatestVersionByStatusRequest{
			TenantInfo: tenant,
			TemplateID: template.ID,
			Status:     formulatemplate.StatusActive,
		},
	)
	if err != nil {
		return nil, err
	}
	declared := tarifftypes.InputDeclsFrom(content.checked.Check.Inputs)
	if base != nil && tarifftypes.DeclsEqual(base.Inputs, declared) {
		gate.Skipped = "The inputs are declared as in the approved version, so every contract still fits"
		return gate, nil
	}

	sets, err := s.dependents.ListInputsByTemplate(
		ctx,
		&rateagreementports.ListInputsByTemplateRequest{
			TenantInfo: tenant,
			TemplateID: template.ID,
			Limit:      maxDependentSets,
		},
	)
	if err != nil {
		return nil, err
	}

	usage, err := s.repo.CountUsages(ctx, &formulatemplateports.GetTemplateUsageRequest{
		TemplateID: template.ID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if usesBillingFallback(usage) {
		sets = append(sets, &rateagreementports.TemplateInputs{
			Kind:  rateagreementports.TemplateInputsKind(DependentFallback),
			Count: 1,
			Rule:  tarifftypes.Inputs{},
		})
	}

	gate.Checked = true
	gate.Sets = len(sets)
	locale := i18n.FromContext(ctx)
	for _, set := range sets {
		check, checkErr := s.checker.CheckWithContext(ctx, &tariff.ContextCheckRequest{
			Source:  template.Source,
			Kind:    rules.TariffKindTariff,
			Context: content.checked.Context,
			Values: &rules.TariffInputValues{
				Rule:      map[string]rules.RawJSON(set.Rule),
				Agreement: map[string]rules.RawJSON(set.Agreement),
			},
		})
		if checkErr != nil {
			return nil, checkErr
		}
		if failure := valueFailure(locale, set, check); failure != nil {
			gate.Failures = append(gate.Failures, *failure)
		}
	}

	return gate, nil
}

func usesBillingFallback(usage *formulatemplateports.GetTemplateUsageResponse) bool {
	if usage == nil {
		return false
	}
	for _, item := range usage.Usages {
		if item.Type == usageBillingFallback && item.Count > 0 {
			return true
		}
	}
	return false
}

func valueFailure(
	locale i18n.Locale,
	set *rateagreementports.TemplateInputs,
	check *rules.TariffCheck,
) *DependentFailure {
	for i := range check.Diagnostics {
		diagnostic := &check.Diagnostics[i]
		if diagnostic.Severity != rules.SeverityError ||
			(diagnostic.Path == "" && diagnostic.Code != tarifferrors.CodeInputValue) {
			continue
		}
		return &DependentFailure{
			Kind:        string(set.Kind),
			ExampleID:   set.ExampleID,
			ExampleName: set.ExampleName,
			Count:       set.Count,
			Path:        diagnostic.Path,
			Message:     diagnostic.Message.Localize(locale),
		}
	}
	return nil
}

func dependentsRefusal(gate *dependentsGate) error {
	if !gate.blocking() {
		return nil
	}

	total := 0
	examples := make([]string, 0, min(len(gate.Failures), maxReportedDependents))
	for i := range gate.Failures {
		failure := &gate.Failures[i]
		total += failure.Count
		if len(examples) < maxReportedDependents {
			examples = append(examples, describeDependent(failure))
		}
	}

	return errortypes.NewValidationError(
		fieldInputs,
		errortypes.ErrInvalid,
		"This version's inputs no longer accept what {0} already set: {1}. Keep the inputs those contracts use, or change the contracts first",
		stringutils.CountNoun(total, "contract or shipment", "contracts and shipments"),
		strings.Join(examples, "; "),
	)
}

func describeDependent(failure *DependentFailure) string {
	name := failure.ExampleName
	if name == "" {
		name = failure.Kind
	}
	if failure.Count > 1 {
		name = fmt.Sprintf("%s and %d more", name, failure.Count-1)
	}
	if failure.Path != "" {
		return fmt.Sprintf("%s (%s: %s)", name, failure.Path, failure.Message)
	}
	return fmt.Sprintf("%s (%s)", name, failure.Message)
}
