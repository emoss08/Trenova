package errortypes

import (
	"errors"
	"strconv"

	"github.com/emoss08/trenova/shared/i18n"
)

const (
	PlanRestrictionReasonPlan          = "plan_restricted"
	PlanRestrictionReasonReadOnly      = "subscription_read_only"
	PlanRestrictionReasonExpired       = "subscription_expired"
	PlanRestrictionReasonSignupsPaused = "signups_paused"
)

type QuotaExceededError struct {
	BaseError
	Meter string `json:"meter"`
	Limit int64  `json:"limit"`
	Used  int64  `json:"used"`
	Plan  string `json:"plan"`
}

func NewQuotaExceededError(meter string, limit, used int64, plan string) *QuotaExceededError {
	return &QuotaExceededError{
		BaseError: BaseError{
			Code:    ErrQuotaExceeded,
			Message: "This organization has reached the limit of its plan",
		},
		Meter: meter,
		Limit: limit,
		Used:  used,
		Plan:  plan,
	}
}

func IsQuotaExceededError(err error) bool {
	_, ok := errors.AsType[*QuotaExceededError](err)
	return ok
}

func (e *QuotaExceededError) Params() map[string]string {
	return map[string]string{
		"meter": e.Meter,
		"limit": strconv.FormatInt(e.Limit, 10),
		"used":  strconv.FormatInt(e.Used, 10),
		"plan":  e.Plan,
	}
}

func (e *QuotaExceededError) WithInternal(err error) *QuotaExceededError {
	e.Internal = err
	return e
}

func (e *QuotaExceededError) WithContext(ctx *ErrorContext) *QuotaExceededError {
	e.Context = ctx
	return e
}

func (e *QuotaExceededError) LogFields() LogFields {
	fields := e.BaseError.LogFields()
	fields["quota_meter"] = e.Meter
	fields["quota_limit"] = e.Limit
	fields["quota_used"] = e.Used
	fields["quota_plan"] = e.Plan
	return fields
}

type PlanRestrictionError struct {
	BaseError
	Feature string `json:"feature"`
	AddOn   string `json:"addOn"`
	Reason  string `json:"reason"`
	Plan    string `json:"plan"`
}

func NewPlanRestrictionError(reason, plan string) *PlanRestrictionError {
	return &PlanRestrictionError{
		BaseError: BaseError{
			Code:    ErrPlanRestricted,
			Message: planRestrictionMessage(reason),
		},
		Reason: reason,
		Plan:   plan,
	}
}

func NewFeatureRestrictionError(feature, addOn string) *PlanRestrictionError {
	base := BaseError{
		Code:    ErrPlanRestricted,
		Message: planRestrictionMessages.Plan.Message,
	}
	if addOn != "" {
		base.Message = planRestrictionMessages.AddOn.Message
		base.Args = []any{i18n.Term(addOn)}
	}

	return &PlanRestrictionError{
		BaseError: base,
		Feature:   feature,
		AddOn:     addOn,
		Reason:    PlanRestrictionReasonPlan,
	}
}

func planRestrictionMessage(reason string) string {
	switch reason {
	case PlanRestrictionReasonReadOnly:
		return planRestrictionMessages.ReadOnly.Message
	case PlanRestrictionReasonExpired:
		return planRestrictionMessages.Expired.Message
	case PlanRestrictionReasonSignupsPaused:
		return planRestrictionMessages.SignupsPaused.Message
	default:
		return planRestrictionMessages.Plan.Message
	}
}

var planRestrictionMessages = struct {
	Plan          BaseError
	AddOn         BaseError
	ReadOnly      BaseError
	Expired       BaseError
	SignupsPaused BaseError
}{
	Plan: BaseError{
		Message: "This feature is not available on your organization's plan",
	},
	AddOn: BaseError{
		Message: "This feature is part of the {0} add-on, which is not on your plan",
	},
	ReadOnly: BaseError{
		Message: "This organization's trial has ended and it is now read-only",
	},
	Expired: BaseError{
		Message: "This organization's trial has expired",
	},
	SignupsPaused: BaseError{
		Message: "Signups are paused right now. You have been added to the wait list.",
	},
}

func IsPlanRestrictionError(err error) bool {
	_, ok := errors.AsType[*PlanRestrictionError](err)
	return ok
}

func (e *PlanRestrictionError) Params() map[string]string {
	return map[string]string{
		"feature": e.Feature,
		"addOn":   e.AddOn,
		"reason":  e.Reason,
		"plan":    e.Plan,
	}
}

func (e *PlanRestrictionError) LocalizedParams(locale i18n.Locale) map[string]string {
	params := e.Params()
	params["addOn"] = i18n.Translate(locale, e.AddOn)
	return params
}

func (e *PlanRestrictionError) WithInternal(err error) *PlanRestrictionError {
	e.Internal = err
	return e
}

func (e *PlanRestrictionError) WithContext(ctx *ErrorContext) *PlanRestrictionError {
	e.Context = ctx
	return e
}

func (e *PlanRestrictionError) LogFields() LogFields {
	fields := e.BaseError.LogFields()
	fields["plan_feature"] = e.Feature
	fields["plan_add_on"] = e.AddOn
	fields["plan_reason"] = e.Reason
	fields["plan_key"] = e.Plan
	return fields
}
