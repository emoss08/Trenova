package supportlogports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/shared/pulid"
)

type Requester struct {
	UserID pulid.ID
	Name   string
	Email  string
}

type OverdueRequest struct {
	RequestedBefore  int64
	CollectingBefore int64
	Limit            int
}

type Repository interface {
	Create(ctx context.Context, req *supportlog.LogRequest) (*supportlog.LogRequest, error)
	Recent(ctx context.Context, limit int) ([]*supportlog.LogRequest, error)
	Requester(ctx context.Context, userID pulid.ID) (*Requester, error)
	InstanceLoad(ctx context.Context, since int64) (*supportlog.Load, error)
	Waiting(ctx context.Context, limit int) ([]*supportlog.LogRequest, error)
	ForOperator(ctx context.Context, id pulid.ID) (*supportlog.LogRequest, error)
	SaveReport(ctx context.Context, req *supportlog.LogRequest) (*supportlog.LogRequest, error)
	Overdue(ctx context.Context, req *OverdueRequest) ([]*supportlog.LogRequest, error)
}

type FinishedNotice struct {
	Request *supportlog.LogRequest
}

type Notifier interface {
	LogRequestFinished(ctx context.Context, notice *FinishedNotice) error
}
