package errortypes

import (
	"errors"
	"strconv"
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
	Capability string `json:"capability"`
	Reason     string `json:"reason"`
	Plan       string `json:"plan"`
}

func NewPlanRestrictionError(capability, reason, plan string) *PlanRestrictionError {
	if reason == "" {
		reason = PlanRestrictionReasonPlan
	}

	return &PlanRestrictionError{
		BaseError: BaseError{
			Code:    ErrPlanRestricted,
			Message: planRestrictionMessage(reason),
		},
		Capability: capability,
		Reason:     reason,
		Plan:       plan,
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
	ReadOnly      BaseError
	Expired       BaseError
	SignupsPaused BaseError
}{
	Plan: BaseError{
		Message: "This feature is not available on your organization's plan",
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
		"capability": e.Capability,
		"reason":     e.Reason,
		"plan":       e.Plan,
	}
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
	fields["plan_capability"] = e.Capability
	fields["plan_reason"] = e.Reason
	fields["plan_key"] = e.Plan
	return fields
}
