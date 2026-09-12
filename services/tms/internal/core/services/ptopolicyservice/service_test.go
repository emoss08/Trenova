package ptopolicyservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakePolicyRepo struct {
	repositories.PTOPolicyRepository
	policies   map[pulid.ID]*worker.PTOPolicy
	codes      map[string]pulid.ID
	openCounts map[pulid.ID]int
	updated    []*worker.PTOPolicy
	created    []*worker.PTOPolicy
}

func newFakePolicyRepo() *fakePolicyRepo {
	return &fakePolicyRepo{
		policies:   map[pulid.ID]*worker.PTOPolicy{},
		codes:      map[string]pulid.ID{},
		openCounts: map[pulid.ID]int{},
	}
}

func (f *fakePolicyRepo) GetByID(
	_ context.Context,
	req *repositories.GetPTOPolicyByIDRequest,
) (*worker.PTOPolicy, error) {
	policy, ok := f.policies[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("PTOPolicy not found within your organization")
	}
	copied := *policy
	return &copied, nil
}

func (f *fakePolicyRepo) CodeExists(
	_ context.Context,
	req *repositories.PTOPolicyCodeExistsRequest,
) (bool, error) {
	id, ok := f.codes[req.Code]
	return ok && id != req.ExcludeID, nil
}

func (f *fakePolicyRepo) Create(
	_ context.Context,
	entity *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	entity.ID = pulid.MustNew("ptop_")
	f.policies[entity.ID] = entity
	f.codes[entity.Code] = entity.ID
	f.created = append(f.created, entity)
	return entity, nil
}

func (f *fakePolicyRepo) Update(
	_ context.Context,
	entity *worker.PTOPolicy,
) (*worker.PTOPolicy, error) {
	entity.Version++
	f.policies[entity.ID] = entity
	f.updated = append(f.updated, entity)
	return entity, nil
}

func (f *fakePolicyRepo) CountOpenAssignments(
	_ context.Context,
	req *repositories.CountOpenPTOAssignmentsRequest,
) (int, error) {
	return f.openCounts[req.PolicyID], nil
}

type fakeAudit struct {
	serviceports.AuditService
	ops []permission.Operation
}

func (f *fakeAudit) LogAction(
	params *serviceports.LogActionParams,
	_ ...serviceports.LogOption,
) error {
	f.ops = append(f.ops, params.Operation)
	return nil
}

func activePolicy() *worker.PTOPolicy {
	return &worker.PTOPolicy{
		ID:             pulid.MustNew("ptop_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Name:           "Standard",
		Code:           "STD",
		Status:         worker.PTOPolicyStatusActive,
		YearBasis:      worker.PTOYearBasisCalendarYear,
		CountWeekends:  true,
		Version:        2,
		Rules: []*worker.PTOPolicyRule{{
			PTOType:           worker.PTOTypeVacation,
			AccrualMethod:     worker.PTOAccrualMethodMonthly,
			AccrualAmountDays: decimal.RequireFromString("1"),
		}},
	}
}

func TestCreateRejectsDuplicateCodeAndDraftDefault(t *testing.T) {
	t.Parallel()

	repo := newFakePolicyRepo()
	repo.codes["STD"] = pulid.MustNew("ptop_")
	svc := &Service{l: zap.NewNop(), repo: repo, auditService: &fakeAudit{}}

	policy := activePolicy()
	policy.ID = pulid.Nil
	policy.IsDefault = true
	policy.Status = worker.PTOPolicyStatusDraft

	_, err := svc.Create(t.Context(), policy, pulid.MustNew("usr_"))
	require.Error(t, err)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Contains(t, fields, "code")
	assert.Contains(t, fields, "isDefault")
	assert.Empty(t, repo.created)
}

func TestArchiveIsBlockedWhileWorkersAreAssigned(t *testing.T) {
	t.Parallel()

	repo := newFakePolicyRepo()
	policy := activePolicy()
	repo.policies[policy.ID] = policy
	repo.openCounts[policy.ID] = 3
	audit := &fakeAudit{}
	svc := &Service{l: zap.NewNop(), repo: repo, auditService: audit}

	_, err := svc.Archive(t.Context(), &StatusChangeRequest{
		ID: policy.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: policy.OrganizationID,
			BuID:  policy.BusinessUnitID,
		},
		Version: policy.Version,
		UserID:  pulid.MustNew("usr_"),
	})
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "status", validationErr.Field)
	assert.Contains(t, validationErr.Error(), "3 workers are still assigned")
	assert.Empty(t, repo.updated)
	assert.Empty(t, audit.ops)
}

func TestArchiveClearsDefaultAndAuditsThenRestore(t *testing.T) {
	t.Parallel()

	repo := newFakePolicyRepo()
	policy := activePolicy()
	policy.IsDefault = true
	repo.policies[policy.ID] = policy
	audit := &fakeAudit{}
	svc := &Service{l: zap.NewNop(), repo: repo, auditService: audit}
	tenant := pagination.TenantInfo{OrgID: policy.OrganizationID, BuID: policy.BusinessUnitID}

	archived, err := svc.Archive(t.Context(), &StatusChangeRequest{
		ID: policy.ID, TenantInfo: tenant, Version: policy.Version, UserID: pulid.MustNew("usr_"),
	})
	require.NoError(t, err)
	assert.Equal(t, worker.PTOPolicyStatusInactive, archived.Status)
	assert.False(t, archived.IsDefault, "an archived policy cannot stay the default")
	assert.Len(t, archived.Rules, 1, "rules survive a status change")

	_, err = svc.Archive(t.Context(), &StatusChangeRequest{
		ID: policy.ID, TenantInfo: tenant, Version: policy.Version, UserID: pulid.MustNew("usr_"),
	})
	require.NoError(t, err, "archiving an inactive policy is a no-op")

	restored, err := svc.Restore(t.Context(), &StatusChangeRequest{
		ID: policy.ID, TenantInfo: tenant, UserID: pulid.MustNew("usr_"),
	})
	require.NoError(t, err)
	assert.Equal(t, worker.PTOPolicyStatusActive, restored.Status)

	assert.Equal(t, []permission.Operation{permission.OpArchive, permission.OpRestore}, audit.ops)
}

func TestStatusChangeRejectsStaleVersion(t *testing.T) {
	t.Parallel()

	repo := newFakePolicyRepo()
	policy := activePolicy()
	repo.policies[policy.ID] = policy
	svc := &Service{l: zap.NewNop(), repo: repo, auditService: &fakeAudit{}}

	_, err := svc.Archive(t.Context(), &StatusChangeRequest{
		ID: policy.ID,
		TenantInfo: pagination.TenantInfo{
			OrgID: policy.OrganizationID,
			BuID:  policy.BusinessUnitID,
		},
		Version: policy.Version + 1,
		UserID:  pulid.MustNew("usr_"),
	})
	require.Error(t, err)

	var validationErr *errortypes.Error
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, errortypes.ErrVersionMismatch, validationErr.Code)
}
