package agentquerytoolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workercredentialservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	maxCatalogRows       = 100
	activeStatus         = "Active"
	paramOutstanding     = "outstandingOnly"
	paramCertOutstanding = "certificationOutstandingOnly"
	paramIncludeArchived = "includeArchived"
)

type wfCredentialReader interface {
	ListForWorker(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		workerID pulid.ID,
		includeArchived bool,
	) ([]*worker.WorkerCredential, error)
	ListTypes(
		ctx context.Context,
		req *workerports.ListCredentialTypesRequest,
	) (*pagination.CursorListResult[*worker.WorkerCredentialType], error)
}

var (
	_ wfCredentialReader = (*workercredentialservice.Service)(nil)
)

func catalogFilter(params *agentports.QueryToolParams) *pagination.QueryOptions {
	return &pagination.QueryOptions{
		TenantInfo: tenantOf(params),
		Pagination: pagination.Info{Limit: maxCatalogRows},
	}
}
