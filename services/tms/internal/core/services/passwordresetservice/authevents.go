package passwordresetservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/iam"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	resetErrorUnknownAccount     = "unknown_account"
	resetErrorAccountUnavailable = "account_unavailable"
	resetErrorRateLimited        = "rate_limited"
	resetErrorInvalidToken       = "invalid_token"
	resetErrorInternal           = "internal_error"
)

type resetEvent struct {
	provider       string
	user           *tenant.User
	organizationID pulid.ID
	businessUnitID pulid.ID
	outcome        iam.AuthEventOutcome
	errorCode      string
}

func (e *resetEvent) deny(outcome iam.AuthEventOutcome, code string) {
	e.outcome = outcome
	e.errorCode = code
}

func (s *Service) recordResetEvent(ctx context.Context, event *resetEvent, err error) {
	if s.authEvents == nil {
		return
	}

	rec := serviceports.AuthEventRecord{
		Provider:       event.provider,
		Outcome:        event.outcome,
		ErrorCode:      event.errorCode,
		OrganizationID: event.organizationID,
		BusinessUnitID: event.businessUnitID,
		RiskOutcome:    iam.RiskOutcomeAllow,
	}
	if event.user != nil {
		rec.UserID = event.user.ID
		if rec.OrganizationID.IsNil() {
			rec.OrganizationID = event.user.CurrentOrganizationID
			rec.BusinessUnitID = event.user.BusinessUnitID
		}
	}

	switch {
	case err != nil && rec.ErrorCode == "":
		rec.Outcome = iam.AuthEventOutcomeFailed
		rec.ErrorCode = resetErrorInternal
	case rec.Outcome == "":
		rec.Outcome = iam.AuthEventOutcomeSuccess
	}
	if rec.Outcome == iam.AuthEventOutcomeDenied {
		rec.RiskOutcome = iam.RiskOutcomeDeny
	}

	s.authEvents.Record(ctx, &rec)
}
