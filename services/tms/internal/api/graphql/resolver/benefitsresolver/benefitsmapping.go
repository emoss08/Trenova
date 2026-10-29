package benefitsresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/paybenefit"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/pulid"
)

type benefitPlanFields struct {
	code              string
	name              string
	description       *string
	planType          paybenefit.BenefitPlanType
	carrier           *string
	policyNumber      *string
	payCodeID         string
	planYear          int
	employeeCostMinor int
	employerCostMinor int
	waitingPeriodDays *int
	status            *domaintypes.Status
}

func benefitPlanFromInput(
	f *benefitPlanFields,
	tenantInfo pagination.TenantInfo,
) (*paybenefit.BenefitPlan, error) {
	payCodeID, err := pulid.MustParse(f.payCodeID)
	if err != nil {
		return nil, errortypes.NewValidationError(
			"payCodeId",
			errortypes.ErrInvalid,
			"Pay code is invalid",
		)
	}

	entity := &paybenefit.BenefitPlan{
		OrganizationID:    tenantInfo.OrgID,
		BusinessUnitID:    tenantInfo.BuID,
		Code:              f.code,
		Name:              f.name,
		Description:       gqlscalar.StringValue(f.description),
		PlanType:          f.planType,
		Carrier:           gqlscalar.StringValue(f.carrier),
		PolicyNumber:      gqlscalar.StringValue(f.policyNumber),
		PayCodeID:         payCodeID,
		PlanYear:          int16(f.planYear), //nolint:gosec // a calendar year
		EmployeeCostMinor: int64(f.employeeCostMinor),
		EmployerCostMinor: int64(f.employerCostMinor),
		WaitingPeriodDays: intutils.SafeToInt32(gqlscalar.IntValue(f.waitingPeriodDays)),
	}
	if f.status != nil {
		entity.Status = *f.status
	}

	return entity, nil
}
