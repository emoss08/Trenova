package workerhandler_test

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/ptopolicyports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

type emptyCredentialRepo struct {
	workercredentialports.WorkerCredentialRepository
}

func (emptyCredentialRepo) EnsureSystemTypes(
	context.Context,
	pagination.TenantInfo,
	[]*workercredential.WorkerCredentialType,
) (int, error) {
	return 0, nil
}

func (emptyCredentialRepo) ListActiveTypes(
	context.Context,
	pagination.TenantInfo,
) ([]*workercredential.WorkerCredentialType, error) {
	return nil, nil
}

func (emptyCredentialRepo) ListForWorker(
	context.Context,
	*workercredentialports.ListWorkerCredentialsRequest,
) ([]*workercredential.WorkerCredential, error) {
	return nil, nil
}

type noDefaultPTOPolicyRepo struct {
	ptopolicyports.PTOPolicyRepository
}

func (noDefaultPTOPolicyRepo) GetDefault(
	context.Context,
	pagination.TenantInfo,
) (*pto.PTOPolicy, error) {
	return nil, errortypes.NewNotFoundError("no default PTO policy")
}
