package rateagreementservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	codeInputValue  = "input-value"
	pathRule        = "rule."
	pathAgreement   = "agreement."
	fieldInputs     = "inputs"
	fieldTemplateID = "formulaTemplateId"
	fieldCondition  = "condition"
)

type tariffInputs struct {
	resolver tariffports.Resolver
	checker  tariffports.Checker
	now      func() int64
}

type templateKey struct {
	id   pulid.ID
	asOf int64
}

func (t *tariffInputs) checkRules(
	ctx context.Context,
	agreement *rateagreement.RateAgreement,
	ruleSet []*rateagreement.RateAgreementRule,
	multiErr *errortypes.MultiError,
) error {
	if t.resolver == nil || t.checker == nil || len(ruleSet) == 0 {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: agreement.OrganizationID, BuID: agreement.BusinessUnitID}
	templates := make(map[templateKey]*tariffports.Template, 4)
	agreementErrors := make(map[string]struct{}, 4)

	for index, rule := range ruleSet {
		if rule == nil || rule.FormulaTemplateID.IsNil() {
			continue
		}
		indexed := multiErr.WithIndex("rules", index)
		key := templateKey{id: rule.FormulaTemplateID, asOf: max(rule.EffectiveFrom, t.now())}
		template, ok := templates[key]
		if !ok {
			resolved, err := t.resolver.ResolveBilling(ctx, &tariffports.ResolveRequest{
				TenantInfo: tenant,
				TemplateID: rule.FormulaTemplateID,
				AsOf:       key.asOf,
			})
			if err != nil {
				if errortypes.IsNotFoundError(err) {
					indexed.Add(
						fieldTemplateID,
						errortypes.ErrInvalid,
						"Formula template has no approved version that prices shipments on this date",
					)
					continue
				}
				return err
			}
			template = resolved
			templates[key] = template
		}

		check, err := t.checker.Check(ctx, &tariffports.CheckRequest{
			TenantInfo: tenant,
			Source:     template.Source,
			Kind:       rules.TariffKindTariff,
			Libraries:  template.Libraries,
			Values: &rules.TariffInputValues{
				Rule:      rule.Inputs,
				Agreement: declaredValues(template.Inputs, agreement.Inputs),
			},
		})
		if err != nil {
			return err
		}
		mapDiagnostics(check, indexed, multiErr, agreementErrors)
	}
	return nil
}

func declaredValues(decls []tarifftypes.InputDecl, inputs tarifftypes.Inputs) map[string]rules.RawJSON {
	if len(inputs) == 0 {
		return nil
	}
	out := make(map[string]rules.RawJSON, len(inputs))
	for name, value := range inputs {
		if tarifftypes.FindInput(decls, name) != nil {
			out[name] = value
		}
	}
	return out
}

func mapDiagnostics(
	check *rules.TariffCheck,
	rule *errortypes.MultiError,
	agreement *errortypes.MultiError,
	reported map[string]struct{},
) {
	for i := range check.Diagnostics {
		diagnostic := &check.Diagnostics[i]
		if diagnostic.Severity != rules.SeverityError {
			continue
		}
		if diagnostic.Code != codeInputValue {
			rule.Add(
				fieldTemplateID,
				errortypes.ErrInvalid,
				"The formula template does not check: {0}",
				diagnostic.Message,
			)
			continue
		}
		switch {
		case strings.HasPrefix(diagnostic.Path, pathRule):
			field := fieldInputs + "." + strings.TrimPrefix(diagnostic.Path, pathRule)
			rule.AddError(diagnostic.Message.ValidationError(field, errortypes.ErrInvalid))
		case strings.HasPrefix(diagnostic.Path, pathAgreement):
			field := fieldInputs + "." + strings.TrimPrefix(diagnostic.Path, pathAgreement)
			if _, seen := reported[field]; seen {
				continue
			}
			reported[field] = struct{}{}
			agreement.AddError(diagnostic.Message.ValidationError(field, errortypes.ErrInvalid))
		default:
			rule.AddError(diagnostic.Message.ValidationError(fieldInputs, errortypes.ErrInvalid))
		}
	}
}

func (t *tariffInputs) checkConditions(
	ctx context.Context,
	agreement *rateagreement.RateAgreement,
	multiErr *errortypes.MultiError,
) error {
	if t.checker == nil {
		return nil
	}
	tenant := pagination.TenantInfo{OrgID: agreement.OrganizationID, BuID: agreement.BusinessUnitID}
	for index, accessorial := range agreement.Accessorials {
		if accessorial == nil || strings.TrimSpace(accessorial.Condition) == "" {
			continue
		}
		check, err := t.checker.Check(ctx, &tariffports.CheckRequest{
			TenantInfo: tenant,
			Source:     accessorial.Condition,
			Kind:       rules.TariffKindCondition,
		})
		if err != nil {
			return err
		}
		if first := check.FirstError(); first != nil {
			multiErr.Add(
				"accessorials["+strconv.Itoa(index)+"]."+fieldCondition,
				errortypes.ErrInvalid,
				"The condition does not check: {0}",
				first.Message,
			)
		}
	}
	return nil
}
