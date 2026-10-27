package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/board"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type RecordSummaryRequest struct {
	Tenant pagination.TenantInfo
	Refs   map[permission.Resource][]pulid.ID
	Now    int64
}

type RecordSummaries map[permission.Resource]map[pulid.ID]board.Summary

type RecordSummarizer interface {
	Summaries(ctx context.Context, req *RecordSummaryRequest) (RecordSummaries, error)
}
