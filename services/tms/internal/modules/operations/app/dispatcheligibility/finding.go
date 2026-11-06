package dispatcheligibility

import (
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchfinding"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Evaluation struct {
	Findings []dispatchfinding.Finding `json:"findings"`
}

func NewEvaluation(capacity int) *Evaluation {
	return &Evaluation{Findings: make([]dispatchfinding.Finding, 0, capacity)}
}

func (e *Evaluation) Add(f *dispatchfinding.Finding) {
	e.Findings = append(e.Findings, *f)
}

func (e *Evaluation) Merge(other *Evaluation) {
	if other == nil || len(other.Findings) == 0 {
		return
	}
	e.Findings = append(e.Findings, other.Findings...)
}

func (e *Evaluation) Blocked() bool {
	if e == nil {
		return false
	}
	for i := range e.Findings {
		if e.Findings[i].Severity == dispatchfinding.SeverityBlock {
			return true
		}
	}
	return false
}

func (e *Evaluation) AppendToMultiError(multiErr *errortypes.MultiError, prefix string) {
	if e == nil || multiErr == nil || len(e.Findings) == 0 {
		return
	}

	target := multiErr
	if prefix != "" {
		target = multiErr.WithPrefix(prefix)
	}

	for i := range e.Findings {
		if e.Findings[i].Severity == dispatchfinding.SeverityInfo {
			continue
		}
		target.Add(
			e.Findings[i].Field,
			severityErrorCode(e.Findings[i].Severity),
			e.Findings[i].Message,
		)
	}
}

func (e *Evaluation) ToMultiError(prefix string) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()
	e.AppendToMultiError(multiErr, prefix)
	if multiErr.HasErrors() {
		return multiErr
	}
	return nil
}

func severityErrorCode(severity dispatchfinding.Severity) errortypes.ErrorCode {
	if severity == dispatchfinding.SeverityBlock {
		return errortypes.ErrComplianceViolation
	}
	return errortypes.ErrInvalid
}

func SeverityForEnforcement(
	level dispatchcontrol.ComplianceEnforcementLevel,
) dispatchfinding.Severity {
	if level.ShouldBlock() {
		return dispatchfinding.SeverityBlock
	}
	return dispatchfinding.SeverityWarn
}

func ComplianceErrorCode(level dispatchcontrol.ComplianceEnforcementLevel) errortypes.ErrorCode {
	return severityErrorCode(SeverityForEnforcement(level))
}

func enforcementSeverity(
	level dispatchcontrol.ComplianceEnforcementLevel,
) dispatchfinding.Severity {
	return SeverityForEnforcement(level)
}
