package common

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/settlement/app/settlementshared"
	"github.com/emoss08/trenova/shared/pulid"
)

const ParamSettlementID = "settlementId"

const ParamPaymentMethod = "paymentMethod"

const ParamPaymentReference = "paymentReference"

const maxPaymentMethodChars = 50

const maxPaymentReferenceChar = 100

var ErrSettlementNeedsAPerson = errors.New(
	"a settlement is approved, posted, voided or marked paid only once a person approves " +
		"the proposal",
)

type SettlementTotal struct {
	Label string `json:"-"`
	Minor int64  `json:"-"`
}

type SettlementFacts struct {
	ID       pulid.ID          `json:"-"`
	Number   string            `json:"-"`
	Version  int64             `json:"-"`
	Status   string            `json:"-"`
	Currency string            `json:"-"`
	Payee    string            `json:"-"`
	Totals   []SettlementTotal `json:"-"`
}

type SettlementLineFacts struct {
	AmountMinor  int64 `json:"-"`
	Manual       bool  `json:"-"`
	AddedByAgent bool  `json:"-"`
}

type SettlementLedger[E any] struct {
	Resource  permission.Resource                                                                  `json:"-"`
	Kind      permission.RecordKind                                                                `json:"-"`
	Artifact  string                                                                               `json:"-"`
	Noun      string                                                                               `json:"-"`
	Sources   string                                                                               `json:"-"`
	ByName    string                                                                               `json:"-"`
	Sensitive []string                                                                             `json:"-"`
	Facts     func(*E) *SettlementFacts                                                            `json:"-"`
	Editable  func(*E) bool                                                                        `json:"-"`
	Line      func(*E, pulid.ID) (SettlementLineFacts, bool)                                       `json:"-"`
	Effects   func(action settlementshared.Action, plan *settlementshared.ActionPlan[*E]) []string `json:"-"`
}

func ClassifyAsMoney(
	agentports.ToolExecuteParams,
) agentports.CallPolicy {
	return agentports.CallPolicy{Egress: agent.EgressMoney}
}

func BoundedString(
	params map[string]any,
	key string,
	maxLength int,
	required bool,
) (string, error) {
	value := strings.TrimSpace(OptionalString(params, key))
	if value == "" && required {
		if _, err := RequireString(params, key); err != nil {
			return "", err
		}
	}
	if len(value) > maxLength {
		return "", fmt.Errorf(
			"%s is %d characters; it keeps at most %d",
			key,
			len(value),
			maxLength,
		)
	}

	return value, nil
}

func FillSettlementPayment(
	methods agenttoolschema.EnumSource[string],
) func(map[string]any, *settlementshared.ActionRequest) error {
	return func(params map[string]any, req *settlementshared.ActionRequest) error {
		method, err := BoundedString(params, ParamPaymentMethod, maxPaymentMethodChars, true)
		if err != nil {
			return err
		}
		if !slices.Contains(methods.Values, method) {
			return ErrUnknownValue(ParamPaymentMethod, method, methods.Names())
		}
		reference, err := BoundedString(
			params,
			ParamPaymentReference,
			maxPaymentReferenceChar,
			false,
		)
		if err != nil {
			return err
		}
		req.PaymentMethod = method
		req.PaymentReference = reference

		return nil
	}
}

func SettlementPaymentProperties(methods agenttoolschema.EnumSource[string]) map[string]any {
	return map[string]any{
		ParamPaymentMethod: agenttoolschema.Enum(
			"How it was paid. Take it from the person; never assume one.", methods,
		),
		ParamPaymentReference: StringProperty("The check number, ACH trace or batch "+
			"reference the payment went out under, when the person gave one.",
			maxPaymentReferenceChar),
	}
}
