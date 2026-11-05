package workerptoservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/orgholidayports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptoledgerports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptopolicyports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/zap"
)

type unassignedPolicies struct {
	ptopolicyports.PTOPolicyRepository
}

func (unassignedPolicies) GetActiveAssignment(
	context.Context,
	*ptopolicyports.GetPTOAssignmentRequest,
) (*pto.WorkerPTOPolicyAssignment, error) {
	return nil, errortypes.NewNotFoundError("PTO assignment not found")
}

type utcOrganizations struct {
	tenantports.OrganizationRepository
}

func (utcOrganizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{}, nil
}

type noHolidays struct {
	orgholidayports.OrgHolidayRepository
}

func (noHolidays) List(
	context.Context,
	*orgholidayports.ListOrgHolidaysRequest,
) ([]*pto.OrgHoliday, error) {
	return nil, nil
}

type emptyLedger struct {
	ptoledgerports.PTOLedgerRepository
}

func (emptyLedger) HasEntry(
	context.Context,
	*ptoledgerports.HasPTOLedgerEntryRequest,
) (bool, error) {
	return false, nil
}

func untrackedLedger() *ptoledgerservice.Service {
	return ptoledgerservice.New(ptoledgerservice.Params{
		Logger:     zap.NewNop(),
		LedgerRepo: emptyLedger{},
		PolicyRepo: unassignedPolicies{},
		OrgRepo:    utcOrganizations{},
		Holidays:   noHolidays{},
	})
}
